package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/internal/repository"
	"mygo_bangforai/pkg/config"
	"strconv"
	"sync"

	"github.com/redis/go-redis/v9"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type PreStorageCoroutine struct {
	changeReq     repository.ChangeRequestRepository
	approvalRedis repository.ApprovalRedisRepository
	pubsubRedis   *redis.Client
	stateRegistry sync.Map
	logger        *zap.Logger
	shutdownChan  chan struct{}
	wg            sync.WaitGroup
	pendingUpdatesChan chan *PendingUpdateEvent
}

// PendingUpdateEvent 待处理更新事件
type PendingUpdateEvent struct {
	UserID    uint
	ProjectID uint
}

func NewPreStorageCoroutine(changeReq repository.ChangeRequestRepository, approvalRedis repository.ApprovalRedisRepository) *PreStorageCoroutine {
	return &PreStorageCoroutine{
		changeReq:          changeReq,
		approvalRedis:      approvalRedis,
		pubsubRedis:        config.GetRedisClient(),
		logger:             zap.L(),
		shutdownChan:       make(chan struct{}),
		pendingUpdatesChan: make(chan *PendingUpdateEvent, 100),
	}
}

// Start 启动预存储协程
func (ps *PreStorageCoroutine) Start(ctx context.Context) {
	ps.wg.Add(2)

	// 协程1：处理待处理更新事件（被动等待通知）
	go func() {
		defer ps.wg.Done()
		ps.processPendingUpdates(ctx)
	}()

	// 协程2：监听Redis键空间通知（作为被动通知的补充）
	go func() {
		defer ps.wg.Done()
		ps.listenKeyspaceNotifications(ctx)
	}()

	ps.logger.Info("预存储协程已启动（被动通知模式）")
}

func (ps *PreStorageCoroutine) Stop() {
	ps.logger.Info("开始关闭预存储协程...")

	close(ps.shutdownChan)
	ps.wg.Wait()

	ps.stateRegistry.Range(func(key, value interface{}) bool {
		ps.stateRegistry.Delete(key)
		return true
	})

	ps.logger.Info("预存储协程已优雅关闭")
}

// NotifyPendingUpdate 外部通知预存储协程处理变更请求
// 变更请求提交成功后调用此方法，实现被动通知模式
func (ps *PreStorageCoroutine) NotifyPendingUpdate(userID, projectID uint) {
	select {
	case ps.pendingUpdatesChan <- &PendingUpdateEvent{
		UserID:    userID,
		ProjectID: projectID,
	}:
		ps.logger.Debug("收到变更请求通知",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID))
	default:
		ps.logger.Warn("待处理更新通道已满，通知被丢弃",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID))
	}
}

func (ps *PreStorageCoroutine) isWaitingStatus(ctx context.Context, userID uint) bool {
	rec, err := ps.changeReq.GetStatusRecord(ctx, userID)
	if err != nil || rec == nil {
		return false
	}
	return rec.Status == model.StatusWaiting
}

// processPendingUpdates 处理待处理更新事件
func (ps *PreStorageCoroutine) processPendingUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ps.logger.Info("处理待更新协程停止（ctx取消）")
			return
		case <-ps.shutdownChan:
			ps.logger.Info("处理待更新协程停止（收到关闭信号）")
			return
		case event := <-ps.pendingUpdatesChan:
			ps.handlePendingUpdate(ctx, event.UserID, event.ProjectID)
		}
	}
}

// handlePendingUpdate 处理单个待处理更新
func (ps *PreStorageCoroutine) handlePendingUpdate(ctx context.Context, userID, projectID uint) {
	key := fmt.Sprintf("prestorage:user:%d:project:%d", userID, projectID)

	// 检查是否正在处理中（避免重复处理）
	if _, loaded := ps.stateRegistry.LoadOrStore(key, struct{}{}); loaded {
		ps.logger.Debug("变更请求正在处理中，跳过",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID))
		return
	}
	defer ps.stateRegistry.Delete(key)

	changeRequests, err := ps.changeReq.GetPendingChanges(ctx, userID)
	if err != nil {
		ps.logger.Error("获取变更请求失败",
			zap.Uint("userID", userID),
			zap.Error(err))
		return
	}

	if len(changeRequests) == 0 {
		ps.logger.Warn("未找到变更请求",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID))
		return
	}

	// 2. 找到对应项目的变更请求
	var targetRequest *model.ChangeRequest
	for i := range changeRequests {
		if changeRequests[i].ProjectID == projectID {
			targetRequest = &changeRequests[i]
			break
		}
	}

	if targetRequest == nil {
		ps.logger.Warn("未找到项目对应的变更请求",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID))
		return
	}

	// 3. 转换为PreStorageMessage格式
	preStorageMsg := ps.convertToPreStorageMessage(targetRequest)

	if err := ps.approvalRedis.StagePreStoragePayload(ctx, preStorageMsg); err != nil {
		ps.logger.Error("发送预存储消息失败",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID),
			zap.Error(err))
		return
	}

	if err := ps.approvalRedis.SetUserProjectStatus(ctx, userID, projectID, model.StatusPreStoraged); err != nil {
		ps.logger.Error("更新状态为pre_storaged失败",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID),
			zap.Error(err))
		return
	}

	// 6. 发送预存储完成消息通知批准协程
	if err := ps.notifyApprovalCoordinator(ctx, userID, projectID); err != nil {
		ps.logger.Error("通知批准协程失败",
			zap.Uint("userID", userID),
			zap.Uint("projectID", projectID),
			zap.Error(err))
		return
	}

	ps.logger.Info("预存储处理完成",
		zap.Uint("userID", userID),
		zap.Uint("projectID", projectID))
}

// convertToPreStorageMessage 将ChangeRequest转换为PreStorageMessage
// 支持临时ID引用：客户端在单次提交内为新创建的文件夹赋予临时ID(tempid)，
// move操作可以引用这个临时ID作为新的父文件夹
func (ps *PreStorageCoroutine) convertToPreStorageMessage(req *model.ChangeRequest) *model.PreStorageMessage {
	msg := &model.PreStorageMessage{
		UserID:    req.UserID,
		ProjectID: req.ProjectID,
		Ops: model.PreOps{
			Create: make([]model.PreCreateOp, len(req.Operations.Create)),
			Move:   make([]model.PreMoveOp, len(req.Operations.Move)),
			Rename: make([]model.PreRenameOp, len(req.Operations.Rename)),
			Delete: make([]model.PreDeleteOp, len(req.Operations.Delete)),
		},
	}

	// 转换Create操作（保持原有顺序）
	for i, op := range req.Operations.Create {
		// 设置默认的父文件夹类型为"enduring"（持久ID）
		fatherType := op.FatherIDType
		if fatherType == "" {
			fatherType = "enduring"
		}

		msg.Ops.Create[i] = model.PreCreateOp{
			TempID:       op.TempID,
			FatherID:     op.FatherID,
			FatherIDType: fatherType,
			Name:         op.Name,
		}
	}

	// 转换Move操作（保持原有顺序）
	for i, op := range req.Operations.Move {
		// 设置默认的新父文件夹类型为"enduring"（持久ID）
		newFatherType := op.NewFatherIDType
		if newFatherType == "" {
			newFatherType = "enduring"
		}

		msg.Ops.Move[i] = model.PreMoveOp{
			ID:              op.ID,
			OldFatherID:     op.OldFatherID,
			NewFatherID:     op.NewFatherID,
			NewFatherIDType: newFatherType,
		}
	}

	// 转换Rename操作
	for i, op := range req.Operations.Rename {
		msg.Ops.Rename[i] = model.PreRenameOp{
			ID:   op.ID,
			Name: op.Name,
		}
	}

	// 转换Delete操作
	for i, op := range req.Operations.Delete {
		msg.Ops.Delete[i] = model.PreDeleteOp{
			ID: op.ID,
		}
	}

	return msg
}

// notifyApprovalCoordinator 发送预存储完成消息通知批准协程
func (ps *PreStorageCoroutine) notifyApprovalCoordinator(_ context.Context, userID, projectID uint) error {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		return fmt.Errorf("RabbitMQ未初始化")
	}

	approvalMsg := model.ApprovalPreStorageMessage{
		UserID:         userID,
		ProjectID:      projectID,
		PreStorageDone: true,
	}

	jsonData, err := json.Marshal(approvalMsg)
	if err != nil {
		return fmt.Errorf("序列化预存储完成消息失败: %w", err)
	}

	err = mq.GetChannel().Publish(
		"",
		"pre_storage_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	)
	if err != nil {
		return fmt.Errorf("发送预存储完成消息失败: %w", err)
	}

	ps.logger.Debug("预存储完成消息已发送到批准协程",
		zap.Uint("userID", userID),
		zap.Uint("projectID", projectID))

	return nil
}

func (ps *PreStorageCoroutine) listenKeyspaceNotifications(ctx context.Context) {
	if ps.pubsubRedis == nil {
		return
	}
	pubsub := ps.pubsubRedis.Subscribe(ctx, "__keyevent@0__:hset")

	defer pubsub.Close()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ps.shutdownChan:
			return
		case msg := <-pubsub.Channel():
			var userID uint
			n, _ := fmt.Sscanf(msg.Payload, "userId:%d", &userID)
			if n == 1 {
				statusKey := fmt.Sprintf("userId:%d", userID)
				projectIDStr, err := ps.pubsubRedis.HGet(ctx, statusKey, "project").Result()
				if err != nil {
					continue
				}
				projectID, _ := strconv.ParseUint(projectIDStr, 10, 32)

				if ps.isWaitingStatus(ctx, userID) {
					select {
					case ps.pendingUpdatesChan <- &PendingUpdateEvent{
						UserID:    userID,
						ProjectID: uint(projectID),
					}:
					default:
					}
				}
			}
		}
	}
}

// InitPreStorageQueue 初始化预存储所需的RabbitMQ队列
func InitPreStorageQueue() error {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		return fmt.Errorf("RabbitMQ未初始化")
	}

	_, err := mq.GetChannel().QueueDeclare(
		"pre_storage_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("声明预存储完成队列失败: %w", err)
	}

	return nil
}
