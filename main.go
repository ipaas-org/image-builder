package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/ipaas-org/image-builder/config"
	"github.com/ipaas-org/image-builder/controller"
	"github.com/ipaas-org/image-builder/handlers/rabbitmq"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/pkg/logger"
	"github.com/ipaas-org/image-builder/providers/builders/nixpacks"
	"github.com/ipaas-org/image-builder/providers/connectors/github"
	"github.com/ipaas-org/image-builder/providers/registry/registry"
)

func main() {
	conf, err := config.NewConfig()
	if err != nil {
		log.Fatalln(err)
	}

	l := logger.NewLogger(conf.Log.Level, conf.Log.Type)
	l.Debug("initizalized logger")

	c := controller.NewBuilderController(l)

	l.Info(conf)
	for _, providerInfo := range conf.Services.Connectors {
		switch providerInfo.Name {
		case model.ConnectorGithub:
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

	if conf.Services.Registries[0].Name != model.RegistryDocker {
		log.Fatal("only docker registry is supported in the app version:", conf.App.Version)
	}
	r, err := registry.NewRegistry(conf.Services.Registries[0].ServerAddress, os.Getenv("REGISTRY_DOCKER_USERNAME"), os.Getenv("REGISTRY_DOCKER_PASSWORD"))
	if err != nil {
		log.Fatalf("error building docker registry: %v\n", err)
	}

	c.AddRegistry(r)

	c.AddBuilder(model.DownloaderNixpacks, nix)
	l.Info("succesfully added nixpacks as builder")

	rmq := rabbitmq.NewRabbitMQ(conf.RMQ.URI, conf.RMQ.ExchangeQueue, c, l)

	if err := rmq.Connect(); err != nil {
		l.Fatalf("error connecting to rabbitmq: %s", err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	go func() {
		rmq.Consume(ctx)
	}()

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
		cancel()
	case err = <-rmq.Error:
		l.Error(fmt.Errorf("rabbitmq: %w", err))
	}
}
