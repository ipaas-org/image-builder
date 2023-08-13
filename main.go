package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/ipaas-org/image-builder/config"
	"github.com/ipaas-org/image-builder/controller"
	"github.com/ipaas-org/image-builder/handlers/rabbitmq"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/pkg/logger"
	"github.com/ipaas-org/image-builder/providers/builders/nixpacks"
	"github.com/ipaas-org/image-builder/providers/connectors/github"
	"github.com/ipaas-org/image-builder/providers/registry/registry"
	"github.com/sirupsen/logrus"
)

const (
	gracefulShutdownTimeout = 15 * time.Second
)

const (
	StartRMQRoutine int = iota + 1
)

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatalln(err)
	}

	l := logger.NewLogger(conf.Log.Level, conf.Log.Type)
	l.Debug("initizalized logger")

	defer func(l *logrus.Logger) {
		if r := recover(); r != nil {
			l.Errorf("panic: recover: %v", r)
			l.Errorf("stacktrace from panic: \n%s", string(debug.Stack()))
		}
	}(l)

	c := controller.NewBuilderController(l)

	l.Info(conf)

	if len(conf.Services.Connectors) == 0 {
		log.Fatal("no connectors specified")
	}

	for _, providerInfo := range conf.Services.Connectors {
		switch providerInfo.Name {
		case model.ConnectorGithub:
			if _, err := os.Stat(providerInfo.DownloadDirectory); os.IsNotExist(err) {
				if err := os.MkdirAll(providerInfo.DownloadDirectory, os.ModePerm); err != nil {
					l.Fatalf("failed to create directory %s: %s", providerInfo.DownloadDirectory, err)
				}
			}
			g := github.NewGithubConnector(providerInfo.DownloadDirectory, fmt.Sprintf("ipaas-%s-%s", conf.App.Name, conf.App.Version), l)
			c.AddConnector(model.ConnectorGithub, g)
			l.Infof("succesfully added %s as downloader", providerInfo.Name)

		default:
			l.Errorf("provider %s not supported", providerInfo.Name)
		}
	}

	if len(conf.Services.Builders) == 0 {
		log.Fatal("no builders specified")
	}

	if conf.Services.Builders[0].Name != model.DownloaderNixpacks {
		log.Fatal("only nixpacks builder is supported in the app version:", conf.App.Version)
	}

	nix := nixpacks.NewNixPackBuilder(conf.App.Version)

	c.AddBuilder(model.DownloaderNixpacks, nix)
	l.Info("succesfully added nixpacks as builder")

	if conf.Services.Registries[0].Name != model.RegistryDocker {
		log.Fatal("only docker registry is supported in the app version:", conf.App.Version)
	}
	r, err := registry.NewRegistry(conf.Services.Registries[0].ServerAddress, os.Getenv("REGISTRY_DOCKER_USERNAME"), os.Getenv("REGISTRY_DOCKER_PASSWORD"))
	if err != nil {
		log.Fatalf("error building docker registry: %v\n", err)
	}

	c.AddRegistry(r)
	l.Info("succesfully added docker registry")

	rmq := rabbitmq.NewRabbitMQ(conf.RMQ.URI, conf.RMQ.RequestQueue, conf.RMQ.ResponseQueue, c, l)

	ctx, cancel := context.WithCancel(context.Background())
	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt,
		syscall.SIGINT,
		syscall.SIGQUIT,
		syscall.SIGABRT,
		syscall.SIGTERM)

	var RoutineMonitor = make(chan int, 100)
	RoutineMonitor <- StartRMQRoutine

	for {
		select {
		case i := <-interrupt:
			l.Info("main - signal: " + i.String())
			l.Info("main - canceling context")
			cancel()
			gracefulTimer := time.Tick(gracefulShutdownTimeout)
			select {
			case <-gracefulTimer:
				l.Info("main - graceful shutdown timeout reached")
				os.Exit(1)
			case <-rmq.Done:
				l.Info("main - rabbitmq finished")
			}

			os.Exit(1)
		case err = <-rmq.Error:
			l.Error(fmt.Errorf("rabbitmq: %w", err))
		default:
		}

		select {
		case ID := <-RoutineMonitor:
			l.Infof("Starting Routine: %d", ID)
			switch ID {
			case StartRMQRoutine:
				go rmq.Start(ctx, StartRMQRoutine, RoutineMonitor)
			default:
			}
		default:
		}

		time.Sleep(10 * time.Millisecond)
	}

}
