package controller

import (
	"github.com/ipaas-org/image-builder/providers/builders"
	"github.com/ipaas-org/image-builder/providers/connectors"
	"github.com/ipaas-org/image-builder/providers/registry"
	"github.com/ipaas-org/image-builder/repo"
	"github.com/sirupsen/logrus"
)

// var _ BuilderController = new(Builder)

type Controller struct {
	connectors      map[string]connectors.Connector
	Builder         builders.Builder
	Registry        registry.Registryer
	ApplicationRepo repo.ApplicationRepoer
	l               *logrus.Logger
}

func NewController(log *logrus.Logger) *Controller {
	return &Controller{
		connectors: make(map[string]connectors.Connector),
		l:          log,
	}
}

func (b *Controller) AddConnector(name string, conn connectors.Connector) {
	b.connectors[name] = conn
}
