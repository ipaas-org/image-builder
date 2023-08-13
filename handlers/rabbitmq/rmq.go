package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"github.com/ipaas-org/image-builder/controller"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/connectors/github"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	l                 *logrus.Logger
	Error             <-chan error
	Connection        *amqp.Connection
	Channel           *amqp.Channel
	ResponseQueue     amqp.Queue
	Delivery          <-chan amqp.Delivery
	uri               string
	requestQueueName  string
	responseQueueName string
	Controller        *controller.Builder
	Done              chan struct{}
	restarts          int
}

func NewRabbitMQ(uri, requestQueue, reponseQueue string, controller *controller.Builder, logger *logrus.Logger) *RabbitMQ {
	doneChan := make(chan struct{})
	return &RabbitMQ{
		uri:               uri,
		l:                 logger,
		requestQueueName:  requestQueue,
		responseQueueName: reponseQueue,
		Controller:        controller,
		Done:              doneChan,
		restarts:          0,
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

	r.ResponseQueue, err = r.Channel.QueueDeclare(
		r.responseQueueName, // name
		true,                // durable
		false,               // delete when unused
		false,               // exclusive
		false,               // no-wait
		nil,                 // arguments
	)
	if err != nil {
		return fmt.Errorf("r.Channel.QueueDeclare: %w", err)
	}

	q, err := r.Channel.QueueDeclare(
		r.requestQueueName, // name
		true,               // durable
		false,              // delete when unused
		false,              // exclusive
		false,              // no-wait
		nil,                // arguments
	)
	if err != nil {
		return fmt.Errorf("r.Channel.QueueDeclare: %w", err)
	}

	r.Delivery, err = r.Channel.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
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

func (r *RabbitMQ) Start(ctx context.Context, ID int, routineMonitor chan int) {

	defer func(restarts int) {
		r.Close()
		rec := recover()
		r.l.Debug("recover:", rec)
		if rec != nil {
			r.l.Error("rabbitmq routine panic:", rec)
			r.l.Error(string(debug.Stack()))
		}
		if ctx.Err() == nil && restarts <= 5 {
			if restarts > 0 {
				time.Sleep(3 * time.Second)
			}
			routineMonitor <- ID
		} else {
			r.l.Infof("rabbitmq routine [ID=%d] not restarting", ID)
			r.Done <- struct{}{}
		}
	}(r.restarts)

	r.l.Infof("starting rabbitmq routine [ID=%d]", ID)
	if err := r.Connect(); err != nil {
		r.l.Error("r.Connect():", err)
		r.restarts++
		return
	}

	r.l.Infof("rabbitmq routine [ID=%d] connected", ID)

	r.Consume(ctx)
	r.l.Info("rabbitmq done consuming")
}

func (r *RabbitMQ) Consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.l.Info("stopping rabbitmq consumer, context cancelled")
			return
		case err := <-r.Error:
			r.l.Error(err)
		case d := <-r.Delivery:
			r.l.Info("received message from rabbitmq")
			r.l.Debug(string(d.Body))

			var response model.BuildResponse
			var buildErr *model.BuildError
			response.Status = model.ResponseStatusFailed
			response.Error = buildErr
			var info model.BuildRequest
			if err := json.Unmarshal(d.Body, &info); err != nil {
				r.l.Error("r.Consume.json.Unmarshal(): %w:", err)
				r.l.Debug(string(d.Body))
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}
				buildErr.Message = err.Error()
				buildErr.Fault = model.ResponseErrorFaultService
				r.SendResponse(response)
				continue
			}

			r.l.Debug(info)

			response.UUID = info.UUID
			response.Repo = info.Repo
			pulledInfo, err := r.Controller.PullRepo(info)
			if err != nil {
				r.l.Error("r.Controller.PullRepo():", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}

				buildErr.Message = err.Error()
				buildErr.Fault = model.ResponseErrorFaultService
				r.SendResponse(response)
				continue
			}
			response.LatestCommit = pulledInfo.LastCommit

			metadata, err := r.Controller.GetGranularMetadata(info, github.MetaDescription)
			if err != nil {
				r.l.Error("r.Controller.GetGranularMetadata(): %w:", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}

				buildErr.Message = err.Error()
				buildErr.Fault = model.ResponseErrorFaultService
				r.SendResponse(response)
				continue
			}

			r.l.Debugf("metadata: %+v", metadata)
			response.Metadata = metadata

			imageID, buildErrorMessage, err := r.Controller.BuildImage(ctx, info, pulledInfo.Path)
			if err != nil {
				r.l.Error("r.Controller.BuildImage(): %w:", err)
				r.l.Error(buildErrorMessage)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}

				if buildErrorMessage != "" {
					buildErr.Message = buildErrorMessage
					buildErr.Fault = model.ResponseErrorFaultUser
				} else {
					buildErr.Message = err.Error()
					buildErr.Fault = model.ResponseErrorFaultService
				}
				r.SendResponse(response)
				continue
			}

			response.ImageID = imageID

			imageName := r.Controller.GenerateImageName(info.UserID, pulledInfo)
			response.ImageName = imageName

			if err := r.Controller.PushImage(ctx, imageID, imageName); err != nil {
				r.l.Error("r.Controller.PushImage(): %w:", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Error("r.Consume.Nack(): %w:", err)
				}

				buildErr.Message = err.Error()
				buildErr.Fault = model.ResponseErrorFaultService
				r.SendResponse(response)
				continue
			}

			if err := d.Ack(false); err != nil {
				r.l.Error("r.Consume.Ack(): %w:", err)
			}

			response.Status = model.ResponseStatusSuccess
			response.Error = nil
			r.SendResponse(response)

			r.l.Infof("image %s built and pushed successfully", imageName)
		}
	}
}

func (r *RabbitMQ) SendResponse(response model.BuildResponse) {
	r.l.Info("sending response to rabbitmq")
	r.l.Debug(response)

	body, err := json.Marshal(response)
	if err != nil {
		r.l.Error("r.SendResponse.json.Marshal(): %w:", err)
		return
	}

	if err := r.Channel.Publish(
		"",                  // exchange
		r.responseQueueName, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
		r.l.Error("r.SendResponse.Channel.Publish(): %w:", err)
	}

	r.l.Info("response sent to rabbitmq")
}
