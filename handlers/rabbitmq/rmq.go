package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/ipaas-org/image-builder/controller"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/connectors/github"
	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type RabbitMQ struct {
	Connection        *amqp.Connection
	Channel           *amqp.Channel
	ResponseQueue     amqp.Queue
	Delivery          <-chan amqp.Delivery
	uri               string
	requestQueueName  string
	responseQueueName string

	Controller *controller.Controller
	l          *logrus.Logger

	Done  chan struct{}
	Error <-chan error
}

func NewRabbitMQ(uri, requestQueue, reponseQueue string, controller *controller.Controller, logger *logrus.Logger) *RabbitMQ {
	doneChan := make(chan struct{})
	return &RabbitMQ{
		uri:               uri,
		l:                 logger,
		requestQueueName:  requestQueue,
		responseQueueName: reponseQueue,
		Controller:        controller,
		Done:              doneChan,
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

	requestQueue, err := r.Channel.QueueDeclare(
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
		requestQueue.Name, // queue
		"",                // consumer
		false,             // auto-ack
		false,             // exclusive
		false,             // no-local
		false,             // no-wait
		nil,               // args
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
	defer func() {
		if err := r.Close(); err != nil {
			r.l.Errorf("error closing connection with rmq: %v:", err)
		}
		r.l.Info("rabbitmq connection closed")
		rec := recover()
		r.l.Debug("recover:", rec)
		if rec != nil {
			r.l.Error("rabbitmq routine panic:", rec)
			r.l.Error(string(debug.Stack()))
		}
		if ctx.Err() == nil {
			routineMonitor <- ID
		} else {
			r.l.Infof("rabbitmq routine [ID=%d] not restarting", ID)
			r.Done <- struct{}{}
		}
	}()

	r.l.Infof("starting rabbitmq routine [ID=%d]", ID)
	if err := r.Connect(); err != nil {
		r.l.Error("r.Connect():", err)
		return
	}

	r.l.Infof("rabbitmq routine [ID=%d] connected", ID)
	r.consume(ctx)
	r.l.Info("rabbitmq done consuming")
}

func (r *RabbitMQ) consume(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			r.l.Info("stopping rabbitmq consumer, context cancelled")
			return
		case d := <-r.Delivery:
			r.l.Info("received message from rabbitmq")
			r.l.Debugf("received: %q", string(d.Body))
			// r.l.Debugf("delivery: %+v", d)
			if d.Body == nil {
				if err := d.Ack(false); err != nil {
					r.l.Errorf("r.Consume.Ack(): %v:", err)
					return
				}
				continue
			}

			info := new(model.BuildRequest)
			response := new(model.BuildResponse)

			response.Status = model.ResponseStatusFailed
			response.IsError = true

			if err := json.Unmarshal(d.Body, info); err != nil {
				r.l.Errorf("r.Consume.json.Unmarshal(): %v:", err)
				r.l.Debug(string(d.Body))
				if err := d.Ack(false); err != nil {
					r.l.Errorf("r.Consume.Ack(): %v:", err)
					return
				}

				response.Message = err.Error()
				response.Fault = model.ResponseErrorFaultUser
				if err := r.SendResponse(response); err != nil {
					r.l.Errorf("r.SendResponse(): %v:", err)
					r.l.Errorf("response: %v", response)
					return
				}
				continue
			}

			r.l.Debug(info)

			shouldBuild, err := r.Controller.ShouldBuild(ctx, info.ApplicationID)
			if err != nil {
				r.l.Errorf("r.Controller.ShouldBuild(): %v:", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Errorf("r.Consume.Nack(): %v:", err)
					return
				}

				response.Message = err.Error()
				response.Fault = model.ResponseErrorFaultService
				if err := r.SendResponse(response); err != nil {
					r.l.Errorf("r.SendResponse(): %v:", err)
					r.l.Errorf("response: %v", response)
					return
				}
				continue
			}
			if !shouldBuild {
				r.l.Infof("application should not be built, skipping")
				if err := d.Ack(false); err != nil {
					r.l.Errorf("r.Consume.Ack(): %v:", err)
					return
				}
				continue
			}

			if err := r.Controller.UpdateApplicationStateToBuilding(ctx, info.ApplicationID); err != nil {
				r.l.Errorf("r.Controller.UpdateApplicationStateToBuilding(): %v:", err)
				if err := d.Nack(false, true); err != nil {
					r.l.Errorf("r.Consume.Nack(): %v:", err)
					return
				}

				response.Message = err.Error()
				response.Fault = model.ResponseErrorFaultService
				if err := r.SendResponse(response); err != nil {
					r.l.Errorf("r.SendResponse(): %v:", err)
					r.l.Errorf("response: %v", response)
					return
				}
				continue
			}
			response.ApplicationID = info.ApplicationID
			response.Repo = info.Repo
			pulledInfo, err := r.Controller.PullRepo(info)
			if err != nil {
				var fault model.ResponseErrorFault
				switch err {
				case github.ErrMissingRepoName,
					github.ErrMissingUsername,
					github.ErrInvalidUrl,
					controller.ErrConnectorNotFound,
					controller.ErrEmptyToken:
					if err := d.Ack(false); err != nil {
						return
					}
					fault = model.ResponseErrorFaultUser
					if err := r.Controller.UpdateApplicationStateToFailed(ctx, info.ApplicationID); err != nil {
						r.l.Errorf("r.Controller.UpdateApplicationStateToFailed(): %v:", err)
						if err := d.Nack(false, true); err != nil {
							r.l.Errorf("r.Consume.Nack(): %v:", err)
							return
						}

						response.Message = err.Error()
						response.Fault = model.ResponseErrorFaultService
						if err := r.SendResponse(response); err != nil {
							r.l.Errorf("r.SendResponse(): %v:", err)
							r.l.Errorf("response: %v", response)
							return
						}
						continue
					}
				default: //in case of rate limit it should be put in a queue that retries after a while, or return an error
					if err := d.Nack(false, true); err != nil {
						return
					}
					fault = model.ResponseErrorFaultService

				}
				r.l.Errorf("r.Controller.PullRepo(): %v", err)

				response.Message = err.Error()
				response.Fault = fault
				if err := r.SendResponse(response); err != nil {
					r.l.Errorf("r.SendResponse(): %v:", err)
					r.l.Errorf("response: %v", response)
					return
				}
				continue
			}
			response.BuiltCommit = pulledInfo.LastCommit

			imageID, buildErrorMessage, err := r.Controller.BuildImage(ctx, info, pulledInfo.Path)
			if err != nil {
				r.l.Errorf("r.Controller.BuildImage(): %v:", err)
				r.l.Error(buildErrorMessage)

				if buildErrorMessage != "" {
					response.Message = buildErrorMessage
					response.Fault = model.ResponseErrorFaultUser
					if err := d.Ack(false); err != nil {
						r.l.Errorf("r.Consume.Nack(): %v:", err)
						return
					}
					if err := r.Controller.UpdateApplicationStateToFailed(ctx, info.ApplicationID); err != nil {
						r.l.Errorf("r.Controller.UpdateApplicationStateToFailed(): %v:", err)
						if err := d.Nack(false, true); err != nil {
							r.l.Errorf("r.Consume.Nack(): %v:", err)
							return
						}

						response.Message = err.Error()
						response.Fault = model.ResponseErrorFaultService
						if err := r.SendResponse(response); err != nil {
							r.l.Errorf("r.SendResponse(): %v:", err)
							r.l.Errorf("response: %v", response)
							return
						}
						continue
					}
				} else {
					response.Message = err.Error()
					response.Fault = model.ResponseErrorFaultService
					if err := d.Nack(false, true); err != nil {
						r.l.Errorf("r.Consume.Nack(): %v:", err)
						return
					}
				}
				if err := r.SendResponse(response); err != nil {
					r.l.Errorf("r.SendResponse(): %v:", err)
					r.l.Errorf("response: %v", response)
					return
				}
				continue
			}

			response.ImageID = imageID

			if r.Controller.IsPushRequired() {
				imageName := r.Controller.GenerateImageName(info.UserID, pulledInfo)
				response.ImageName = imageName

				if err := r.Controller.PushImage(ctx, imageID, imageName); err != nil {
					r.l.Errorf("r.Controller.PushImage(): %v:", err)
					if err := d.Nack(false, true); err != nil {
						r.l.Errorf("r.Consume.Nack(): %v:", err)
						return
					}

					response.Message = err.Error()
					response.Fault = model.ResponseErrorFaultService
					if err := r.SendResponse(response); err != nil {
						r.l.Errorf("r.SendResponse(): %v:", err)
						r.l.Errorf("response: %v", response)
						return
					}
					continue
				}
				r.l.Info("image pushed to regsitry correctly")
			} else {
				r.l.Info("pushing image to registry is not required")
			}

			if err := d.Ack(false); err != nil {
				r.l.Errorf("r.Consume.Ack(): %v:", err)
				return
			}

			r.l.Infof("image %s built successfully", response.ImageID)
			response.Status = model.ResponseStatusSuccess
			response.IsError = false
			if err := r.SendResponse(response); err != nil {
				r.l.Errorf("r.SendResponse(): %v:", err)
				r.l.Errorf("response: %v", response)
				return
			}
		}
	}
}

func (r *RabbitMQ) SendResponse(response *model.BuildResponse) error {
	r.l.Info("sending response to rabbitmq")
	r.l.Debug(response)

	body, err := json.Marshal(response)
	if err != nil {
		r.l.Errorf("r.SendResponse.json.Marshal(): %v:", err)
		return err
	}

	r.l.Debugf("sending response: %q", string(body))

	if err := r.Channel.Publish(
		"",                  // exchange
		r.responseQueueName, // routing key
		false,               // mandatory
		false,               // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		}); err != nil {
		r.l.Errorf("r.SendResponse.Channel.Publish(): %v:", err)
		return err
	}

	r.l.Info("response sent to rabbitmq")
	return nil
}
