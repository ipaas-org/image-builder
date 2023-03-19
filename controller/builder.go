package controller

import (
	"context"
	"errors"

	"github.com/sirupsen/logrus"
	"github.com/vano2903/image-builder/model"
	"github.com/vano2903/image-builder/providers/builders"
	"github.com/vano2903/image-builder/providers/connectors"
)

// var _ BuilderController = new(Builder)

var (
	ErrConnectorNotFound = errors.New("connector not found")
	ErrBuilderNotFound   = errors.New("builder not found")
)

type Builder struct {
	connectors map[string]connectors.Connector
	builders   map[string]builders.Builder
	l          *logrus.Logger
}

func NewBuilderController(log *logrus.Logger) *Builder {
	return &Builder{
		connectors: make(map[string]connectors.Connector),
		builders:   make(map[string]builders.Builder),
		l:          log,
	}
}

func (c *Builder) GetConnector(name string) (connectors.Connector, error) {
	connector, ok := c.connectors[name]
	if !ok {
		return nil, ErrConnectorNotFound
	}
	return connector, nil
}

func (c *Builder) GetBuilder(name string) (builders.Builder, error) {
	builder, ok := c.builders[name]
	if !ok {
		return nil, ErrConnectorNotFound
	}
	return builder, nil
}

func (c *Builder) AddConnector(name string, conn connectors.Connector) {
	c.connectors[name] = conn
}

func (c *Builder) AddBuilder(name string, builder builders.Builder) {
	c.builders[name] = builder
}

func (c *Builder) PullRepo(info model.ImageBuildInfo) (model.PulledRepoInfo, error) {
	connector, ok := c.connectors[info.Connector]
	if !ok {
		return model.PulledRepoInfo{}, ErrConnectorNotFound
	}

	c.l.Infof("%s is pulling %s at branch %s", info.UserID, info.Repo, info.Branch)
	c.l.Debugf("info received: %+v", info)

	var pullInfo model.PulledRepoInfo
	var err error
	pullInfo.Path, pullInfo.RepoName, pullInfo.LastCommit, err = connector.Pull(info.UserID, info.Branch, info.Repo)
	if err != nil {
		c.l.Errorf("error pulling %s: %v", info.Repo, err)
		return model.PulledRepoInfo{}, err
	}

	pullInfo.Metadata, err = connector.GetMetadata(info.Repo)
	if err != nil {
		c.l.Errorf("error getting metadata for %s: %v", info.Repo, err)
		return model.PulledRepoInfo{}, err
	}

	return pullInfo, nil
}

func (c *Builder) BuildImage(path string) (string, string, error) {
	c.l.Infof("building image from %s", path)
	builder, ok := c.builders[model.DownloaderNixpacks]
	if !ok {
		return "", "", ErrBuilderNotFound
	}

	buildPlan, err := builder.Plan(context.Background(), path)
	if err != nil {
		c.l.Errorf("error planning build: %v", err)
		return "", "", err
	}

	c.l.Debugf("build plan: %+v", buildPlan)

	return builder.Build(context.Background(), buildPlan, path)
}
