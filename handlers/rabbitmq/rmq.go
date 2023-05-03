package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ipaas-org/image-builder/controller"
	"github.com/ipaas-org/image-builder/model"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	l             *logrus.Logger
	Error         <-chan error
	Connection    *amqp.Connection
	Channel       *amqp.Channel
	Delivery      <-chan amqp.Delivery
	uri           string
	exchangeQueue string
	Controller    *controller.Builder
}

func NewRabbitMQ(uri, exchangeQueue string, controller *controller.Builder, logger *logrus.Logger) *RabbitMQ {
	return &RabbitMQ{
		uri:           uri,
		l:             logger,
		exchangeQueue: exchangeQueue,
		Controller:    controller,
	}
}

func (r *RabbitMQ) Connect() error {
	r.l.Info("connecting to rabbitmq")
	r.l.Debug(r.uri)
	var err error
	r.Connection, err = amqp.Dial(r.uri)
	if err != nil {
		return fmt.Errorf("ampq.Dial: %w", err)
	}

	r.Channel, err = r.Connection.Channel()
	if err != nil {
		return fmt.Errorf("r.Connection.Channel: %w", err)
	}

	if err = r.Channel.Qos(1, 0, false); err != nil {
		return fmt.Errorf("r.Channel.Qos: %w", err)
	}

	q, err := r.Channel.QueueDeclare(
		r.exchangeQueue, // name
		true,            // durable
		false,           // delete when unused
		false,           // exclusive
		false,           // no-wait
		nil,             // arguments
	)
	if err != nil {
		return fmt.Errorf("r.Channel.QueueuDeclare: %w", err)
	}

	r.Delivery, err = r.Channel.Consume(
		q.Name, // queue
		"",     // consumer
		false,  // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	if err != nil {
		return fmt.Errorf("r.Channel.Consume: %w", err)
	}

	return nil
}

func (r *RabbitMQ) Close() error {
	if err := r.Channel.Close(); err != nil {
		return fmt.Errorf("r.Channel.Close: %w", err)
	}

	if err := r.Connection.Close(); err != nil {
		return fmt.Errorf("r.Connection.Close: %w", err)
	}

	return nil
}

func (r *RabbitMQ) Consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.l.Info("stopping rabbitmq consumer")
			return
		case err := <-r.Error:
			r.l.Error(err)
		case d := <-r.Delivery:
			r.l.Info("received message from rabbitmq")
			r.l.Debug(string(d.Body))

			var info model.BuildRequest
			if err := json.Unmarshal(d.Body, &info); err != nil {
				r.l.Error("r.Consume.json.Unmarshal(): %w:", err)
				r.l.Debug(string(d.Body))
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}
				continue
				//TODO: send error to rabbitmq
			}

			r.l.Debug(info)

			pulledInfo, err := r.Controller.PullRepo(info)
			if err != nil {
				r.l.Error("r.Controller.PullRepo():", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}
				continue
			}

			// metadata, err := r.Controller.GetGranularMetadata(info, github.MetaDescription)
			// if err != nil {
			// 	r.l.Error("r.Controller.GetGranularMetadata(): %w:", err)
			// 	if err := d.Nack(false, true); err != nil {
			// 		r.l.Error("r.Consume.Nack(): %w:", err)
			// 	}
			// 	continue
			// }

			// r.l.Debugf("metadata: %+v", metadata)

			imageID, buildErrorMessage, err := r.Controller.BuildImage(ctx, info, pulledInfo.Path)
			if err != nil {
				r.l.Error("r.Controller.BuildImage(): %w:", err)
				r.l.Error(buildErrorMessage)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}
				continue
			}

			imageName := r.Controller.GenerateImageName(info.UserID, pulledInfo)

			if err := r.Controller.PushImage(ctx, imageID, imageName); err != nil {
				r.l.Error("r.Controller.PushImage(): %w:", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}
				continue
			}

			if err := d.Ack(false); err != nil {
				r.l.Error("r.Consume.Ack(): %w:", err)
			}
			r.l.Infof("image %s built and pushed successfully", imageName)
		}
	}
}
