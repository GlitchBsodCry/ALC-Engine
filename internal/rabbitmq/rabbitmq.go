package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"mygo_bangforai/api/model"
	"mygo_bangforai/pkg/config"
	"mygo_bangforai/pkg/logger"

	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

var (
	conn     *amqp.Connection
	channel  *amqp.Channel
	instance *RabbitMQ
)

type RabbitMQ struct {
	conn     *amqp.Connection
	channel  *amqp.Channel
	Exchange string
	Key      string
}

type MessageMQParam struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	UserName  string `json:"user_name"`
	IsUser    bool   `json:"is_user"`
}

func InitRabbitMQ() error {
	cfg := config.GetRabbitMQConfig()
	mqUrl := fmt.Sprintf(
		"amqp://%s:%s@%s:%d/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, cfg.Vhost,
	)

	var err error
	conn, err = amqp.Dial(mqUrl)
	if err != nil {
		logger.Error("RabbitMQ connection failed", zap.Error(err))
		return err
	}

	channel, err = conn.Channel()
	if err != nil {
		logger.Error("RabbitMQ channel creation failed", zap.Error(err))
		return err
	}

	instance = &RabbitMQ{
		conn:     conn,
		channel:  channel,
		Exchange: "",
		Key:      "cache_updates",
	}

	_, err = instance.channel.QueueDeclare(
		instance.Key,
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("RabbitMQ queue declaration failed", zap.Error(err))
		return err
	}

	logger.Info("RabbitMQ initialized successfully")
	return nil
}

func GetRabbitMQ() *RabbitMQ {
	if instance == nil {
		if err := InitRabbitMQ(); err != nil {
			logger.Error("Failed to initialize RabbitMQ", zap.Error(err))
			return nil
		}
	}
	return instance
}

func (r *RabbitMQ) Publish(message []byte) error {
	return r.channel.Publish(
		r.Exchange,
		r.Key,
		false,
		false,
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        message,
		},
	)
}

// GetChannel 获取 channel（导出方法供外部使用）
func (r *RabbitMQ) GetChannel() *amqp.Channel {
	return r.channel
}

func (r *RabbitMQ) Consume(ctx context.Context, handle func(msg *amqp.Delivery) error) {
	msgs, err := r.channel.Consume(
		r.Key,
		"",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		logger.Error("RabbitMQ consume failed", zap.Error(err))
		return
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				logger.Info("RabbitMQ consumer stopped")
				return
			case msg := <-msgs:
				if err := handle(&msg); err != nil {
					logger.Error("RabbitMQ message handling failed", zap.Error(err))
				}
			}
		}
	}()
}

func GenerateMessageMQParam(sessionID string, content string, userName string, isUser bool) []byte {
	param := MessageMQParam{
		SessionID: sessionID,
		Content:   content,
		UserName:  userName,
		IsUser:    isUser,
	}
	data, _ := json.Marshal(param)
	return data
}

func ParseMessageMQParam(data []byte) (*model.ChatMessage, error) {
	var param MessageMQParam
	if err := json.Unmarshal(data, &param); err != nil {
		return nil, err
	}
	return &model.ChatMessage{
		SessionID: param.SessionID,
		Content:   param.Content,
		IsUser:    param.IsUser,
	}, nil
}
