package controller

import (
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/analyzers"
	"github.com/ipaas-org/image-builder/providers/builders"
	"github.com/ipaas-org/image-builder/providers/connectors"
	"github.com/ipaas-org/image-builder/providers/registry"
	"github.com/ipaas-org/image-builder/repo"
	"github.com/sirupsen/logrus"
)

// var _ BuilderController = new(Builder)

type Controller struct {
	connectors      map[string]connectors.Connector
	Builders        map[model.BuilderKind]builders.Builder
	Analyzer        analyzers.Analyzer
	Registry        registry.Registryer
	ApplicationRepo repo.ApplicationRepoer
	l               *logrus.Logger
}

func NewController(log *logrus.Logger) *Controller {
	return &Controller{
		connectors: make(map[string]connectors.Connector),
		Builders:   make(map[model.BuilderKind]builders.Builder),
		l:          log,
	}
}

func (c *Controller) AddConnector(name string, conn connectors.Connector) {
	c.connectors[name] = conn
}

func (c *Controller) AddBuilder(name model.BuilderKind, builder builders.Builder) {
	c.Builders[name] = builder
}
