package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mygo_bangforai/api/model"
	"mygo_bangforai/internal/rabbitmq"
	"mygo_bangforai/internal/repository"
	"sync"

	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

type ConsumerCoordinator struct {
	ar    repository.ApprovalRedisRepository
	batch *ApprovedBatchService
	inbox chan *model.ApprovalResultMessage
	sf    singleflight.Group
	logger *zap.Logger
	stop  chan struct{}
	wg    sync.WaitGroup
}

func NewConsumerCoordinator(ar repository.ApprovalRedisRepository, batch *ApprovedBatchService) *ConsumerCoordinator {
	return &ConsumerCoordinator{
		ar:     ar,
		batch:  batch,
		inbox:  make(chan *model.ApprovalResultMessage, 100),
		logger: zap.L(),
		stop:   make(chan struct{}),
	}
}

func (cc *ConsumerCoordinator) Start(ctx context.Context) {
	cc.wg.Add(2)
	go cc.runConsumerListener(ctx)
	go cc.runProcessor(ctx)
	cc.logger.Info("消费协程已启动")
}

func (cc *ConsumerCoordinator) Stop() {
	cc.logger.Info("开始关闭消费协程...")
	close(cc.stop)
	cc.wg.Wait()
	cc.logger.Info("消费协程已关闭")
}

func (cc *ConsumerCoordinator) runConsumerListener(ctx context.Context) {
	defer cc.wg.Done()
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		cc.logger.Error("RabbitMQ 未初始化")
		return
	}
	msgs, err := mq.GetChannel().Consume(
		"consumer_queue",
		"consumer_coordinator",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		cc.logger.Error("监听消费队列失败", zap.Error(err))
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-cc.stop:
			return
		case msg, ok := <-msgs:
			if !ok {
				return
			}
			var m model.ApprovalResultMessage
			if err := json.Unmarshal(msg.Body, &m); err != nil {
				cc.logger.Error("解析消费消息失败", zap.Error(err))
				continue
			}
			select {
			case cc.inbox <- &m:
			case <-cc.stop:
				return
			}
		}
	}
}

func (cc *ConsumerCoordinator) runProcessor(ctx context.Context) {
	defer cc.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-cc.stop:
			return
		case msg := <-cc.inbox:
			key := fmt.Sprintf("%d:%d", msg.UserID, msg.ProjectID)
			_, err, _ := cc.sf.Do(key, func() (interface{}, error) {
				return nil, cc.handleResult(ctx, msg)
			})
			if err != nil {
				cc.logger.Error("处理审批结果失败",
					zap.Uint("user_id", msg.UserID),
					zap.Uint("project_id", msg.ProjectID),
					zap.Error(err))
			}
		}
	}
}

func (cc *ConsumerCoordinator) handleResult(ctx context.Context, msg *model.ApprovalResultMessage) error {
	if msg.Approved {
		return cc.onApproved(ctx, msg.UserID, msg.ProjectID)
	}
	return cc.onRefused(ctx, msg.UserID, msg.ProjectID)
}

func (cc *ConsumerCoordinator) onApproved(ctx context.Context, userID, projectID uint) error {
	payload, err := cc.ar.TakePreStoragePayload(ctx, userID, projectID)
	if err != nil {
		_ = cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusFailed)
		return err
	}
	if payload == nil {
		_ = cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusFailed)
		return fmt.Errorf("missing prestorage payload")
	}
	if err := cc.batch.ExecuteApproved(ctx, userID, payload); err != nil {
		_ = cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusFailed)
		return err
	}
	if err := cc.ar.RemovePendingChangeForProject(ctx, userID, projectID); err != nil {
		_ = cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusFailed)
		return err
	}
	if err := cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusCompleted); err != nil {
		return err
	}
	cc.logger.Info("批准请求消费完成", zap.Uint("user_id", userID), zap.Uint("project_id", projectID))
	return nil
}

func (cc *ConsumerCoordinator) onRefused(ctx context.Context, userID, projectID uint) error {
	_, _ = cc.ar.TakePreStoragePayload(ctx, userID, projectID)
	if err := cc.ar.RemovePendingChangeForProject(ctx, userID, projectID); err != nil {
		_ = cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusFailed)
		return err
	}
	if err := cc.ar.SetUserProjectStatus(ctx, userID, projectID, model.StatusRefused); err != nil {
		return err
	}
	cc.logger.Info("拒绝请求处理完成", zap.Uint("user_id", userID), zap.Uint("project_id", projectID))
	return nil
}

func InitConsumerQueue() error {
	mq := rabbitmq.GetRabbitMQ()
	if mq == nil {
		return fmt.Errorf("RabbitMQ 未初始化")
	}
	_, err := mq.GetChannel().QueueDeclare(
		"consumer_queue",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("声明消费队列失败: %w", err)
	}
	return nil
}
