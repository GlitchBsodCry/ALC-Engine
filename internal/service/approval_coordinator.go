package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/internal/repository"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

type ApprovalCoordinator struct {
	changeReq         repository.ChangeRequestRepository
	approvalRedis     repository.ApprovalRedisRepository
	approvalChannel   chan *model.ApprovalMessage
	preStorageChannel chan *model.ApprovalPreStorageMessage
	stateRegistry     sync.Map
	logger            *zap.Logger
	shutdownChan      chan struct{}
	wg                sync.WaitGroup
}

func NewApprovalCoordinator(changeReq repository.ChangeRequestRepository, approvalRedis repository.ApprovalRedisRepository) *ApprovalCoordinator {
	return &ApprovalCoordinator{
		changeReq:         changeReq,
		approvalRedis:     approvalRedis,
		approvalChannel:   make(chan *model.ApprovalMessage, 100),
		preStorageChannel: make(chan *model.ApprovalPreStorageMessage, 100),
		logger:            zap.L(),
		shutdownChan:      make(chan struct{}),
	}
}

// Start 启动审批协调器协程
func (ac *ApprovalCoordinator) Start(ctx context.Context) {
	ac.wg.Add(4)

	go func() {
		defer ac.wg.Done()
		ac.listenApprovalQueue(ctx)
	}()

	go func() {
		defer ac.wg.Done()
		ac.listenPreStorageQueue(ctx)
	}()

	go func() {
		defer ac.wg.Done()
		ac.coordinate(ctx)
	}()

	go func() {
		defer ac.wg.Done()
		ac.cleanupExpiredStates(ctx)
	}()

	ac.logger.Info("审批协调器已启动")
}

// Stop 优雅停止审批协调器
func (ac *ApprovalCoordinator) Stop() {
	ac.logger.Info("开始关闭审批协调器...")

	// 发送关闭信号
	close(ac.shutdownChan)

	// 等待所有协程退出
	ac.wg.Wait()

	// 清空状态注册表
	ac.stateRegistry.Range(func(key, value interface{}) bool {
		ac.stateRegistry.Delete(key)
		return true
	})

	ac.logger.Info("审批协调器已优雅关闭")
}

// listenApprovalQueue 监听审批结果队列
func (ac *ApprovalCoordinator) listenApprovalQueue(ctx context.Context) {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		ac.logger.Error("RabbitMQ 未初始化")
		return
	}

	msgs, err := mq.GetChannel().Consume(
		"approval_result_queue",
		"approval_consumer",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ac.logger.Error("监听审批队列失败", zap.Error(err))
		return
	}

	for {
		select {
		case <-ctx.Done():
			ac.logger.Info("审批队列监听停止（ctx取消）")
			return
		case <-ac.shutdownChan:
			ac.logger.Info("审批队列监听停止（收到关闭信号）")
			return
		case msg := <-msgs:
			var approvalMsg model.ApprovalMessage
			if err := json.Unmarshal(msg.Body, &approvalMsg); err != nil {
				ac.logger.Error("解析审批消息失败", zap.Error(err))
				continue
			}
			select {
			case ac.approvalChannel <- &approvalMsg:
			case <-ac.shutdownChan:
				ac.logger.Info("审批消息发送被中断")
				return
			}
		}
	}
}

// listenPreStorageQueue 监听预存储完成队列
func (ac *ApprovalCoordinator) listenPreStorageQueue(ctx context.Context) {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		ac.logger.Error("RabbitMQ 未初始化")
		return
	}

	msgs, err := mq.GetChannel().Consume(
		"pre_storage_queue",
		"prestorage_consumer",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		ac.logger.Error("监听预存储队列失败", zap.Error(err))
		return
	}

	for {
		select {
		case <-ctx.Done():
			ac.logger.Info("预存储队列监听停止（ctx取消）")
			return
		case <-ac.shutdownChan:
			ac.logger.Info("预存储队列监听停止（收到关闭信号）")
			return
		case msg := <-msgs:
			var preStorageMsg model.ApprovalPreStorageMessage
			if err := json.Unmarshal(msg.Body, &preStorageMsg); err != nil {
				ac.logger.Error("解析预存储消息失败", zap.Error(err))
				continue
			}
			select {
			case ac.preStorageChannel <- &preStorageMsg:
			case <-ac.shutdownChan:
				ac.logger.Info("预存储消息发送被中断")
				return
			}
		}
	}
}

// coordinate 协调协程 - 等待两种消息并触发状态转换
func (ac *ApprovalCoordinator) coordinate(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			ac.logger.Info("审批协调协程停止（ctx取消）")
			return
		case <-ac.shutdownChan:
			ac.logger.Info("审批协调协程停止（收到关闭信号）")
			return
		case approvalMsg := <-ac.approvalChannel:
			ac.handleApprovalMessage(approvalMsg)
		case preStorageMsg := <-ac.preStorageChannel:
			ac.handlePreStorageMessage(preStorageMsg)
		}
	}
}

// handleApprovalMessage 处理审批消息
func (ac *ApprovalCoordinator) handleApprovalMessage(msg *model.ApprovalMessage) {
	key := ac.generateKey(msg.UserID, msg.ProjectID)

	state := ac.getOrCreateState(msg.UserID, msg.ProjectID)
	state.ApprovalRecv = true
	state.Approved = msg.Approved

	ac.stateRegistry.Store(key, state)
	ac.checkAndTransition(state, key)
}

// handlePreStorageMessage 处理预存储消息
func (ac *ApprovalCoordinator) handlePreStorageMessage(msg *model.ApprovalPreStorageMessage) {
	key := ac.generateKey(msg.UserID, msg.ProjectID)

	state := ac.getOrCreateState(msg.UserID, msg.ProjectID)
	state.PreStorageRecv = true
	state.PreStorageDone = msg.PreStorageDone

	if err := ac.approvalRedis.SetUserProjectStatus(context.Background(), msg.UserID, msg.ProjectID, model.StatusPreStoraged); err != nil {
		ac.logger.Error("更新状态为 pre_storaged 失败", zap.Error(err))
	}

	ac.stateRegistry.Store(key, state)
	ac.checkAndTransition(state, key)
}

// getOrCreateState 获取或创建状态记录
func (ac *ApprovalCoordinator) getOrCreateState(userID, projectID uint) *model.ApprovalState {
	key := ac.generateKey(userID, projectID)

	if val, ok := ac.stateRegistry.Load(key); ok {
		return val.(*model.ApprovalState)
	}

	return &model.ApprovalState{
		UserID:         userID,
		ProjectID:      projectID,
		ApprovalRecv:   false,
		PreStorageRecv: false,
	}
}

// generateKey 生成状态记录的 key
func (ac *ApprovalCoordinator) generateKey(userID, projectID uint) string {
	return fmt.Sprintf("approval:user:%d:project:%d", userID, projectID)
}

// checkAndTransition 检查是否可以触发状态转换
func (ac *ApprovalCoordinator) checkAndTransition(state *model.ApprovalState, key string) {
	if !state.ApprovalRecv || !state.PreStorageRecv {
		return
	}

	var newStatus model.ChangeRequestStatus
	if state.Approved {
		newStatus = model.StatusApproved
	} else {
		newStatus = model.StatusRefused
	}

	if err := ac.approvalRedis.SetUserProjectStatus(context.Background(), state.UserID, state.ProjectID, newStatus); err != nil {
		ac.logger.Error("更新状态失败", zap.Error(err))
		return
	}

	// 发送消息给消费协程（状态机保证此时预存储已完成）
	if err := ac.notifyConsumerCoordinator(state.UserID, state.ProjectID, state.Approved); err != nil {
		ac.logger.Error("通知消费协程失败", zap.Error(err))
		return
	}

	ac.stateRegistry.Delete(key)
}

// notifyConsumerCoordinator 发送批准/拒绝消息给消费协程
func (ac *ApprovalCoordinator) notifyConsumerCoordinator(userID, projectID uint, approved bool) error {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		return fmt.Errorf("RabbitMQ未初始化")
	}

	consumerMsg := model.ApprovalResultMessage{
		UserID:    userID,
		ProjectID: projectID,
		Approved:  approved,
	}

	jsonData, err := json.Marshal(consumerMsg)
	if err != nil {
		return fmt.Errorf("序列化消费消息失败: %w", err)
	}

	err = mq.GetChannel().Publish(
		"",
		"consumer_queue",
		false,
		false,
		amqp.Publishing{
			ContentType: "application/json",
			Body:        jsonData,
		},
	)
	if err != nil {
		return fmt.Errorf("发送消费消息失败: %w", err)
	}

	ac.logger.Debug("批准结果消息已发送到消费协程",
		zap.Uint("user_id", userID),
		zap.Uint("project_id", projectID),
		zap.Bool("approved", approved))

	return nil
}

// cleanupExpiredStates 清理超时状态
func (ac *ApprovalCoordinator) cleanupExpiredStates(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ac.shutdownChan:
			return
		case <-ticker.C:
			ac.stateRegistry.Range(func(key, value interface{}) bool {
				state := value.(*model.ApprovalState)
				rec, err := ac.changeReq.GetStatusRecord(context.Background(), state.UserID)
				if err == nil && rec != nil {
					return true
				}
				ac.stateRegistry.Delete(key)
				return true
			})
		}
	}
}

// InitApprovalCoordinatorQueue 初始化审批协调器所需的队列
func InitApprovalCoordinatorQueue() error {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		return fmt.Errorf("RabbitMQ 未初始化")
	}

	_, err := mq.GetChannel().QueueDeclare(
		"approval_result_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("声明审批结果队列失败: %w", err)
	}

	_, err = mq.GetChannel().QueueDeclare(
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
