package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/builders"
	"github.com/ipaas-org/image-builder/providers/connectors"
	"github.com/ipaas-org/image-builder/providers/registry"
	"github.com/sirupsen/logrus"
)

// var _ BuilderController = new(Builder)

var (
	ErrConnectorNotFound = errors.New("connector not found")
	ErrBuilderNotFound   = errors.New("builder not found")
	ErrEmptyToken        = errors.New("empty token")
	ErrInvalidToken      = errors.New("invalid token")
)

type Builder struct {
	connectors map[string]connectors.Connector
	builders   map[string]builders.Builder
	registry   registry.Registryer
	l          *logrus.Logger
}

func NewBuilderController(log *logrus.Logger) *Builder {
	return &Builder{
		connectors: make(map[string]connectors.Connector),
		builders:   make(map[string]builders.Builder),
		l:          log,
	}
}

func (b *Builder) GetConnector(name string) (connectors.Connector, error) {
	connector, ok := b.connectors[name]
	if !ok {
		return nil, ErrConnectorNotFound
	}
	return connector, nil
}

func (b *Builder) GetBuilder(name string) (builders.Builder, error) {
	builder, ok := b.builders[name]
	if !ok {
		return nil, ErrConnectorNotFound
	}
	return builder, nil
}

func (b *Builder) AddConnector(name string, conn connectors.Connector) {
	b.connectors[name] = conn
}

func (b *Builder) AddBuilder(name string, builder builders.Builder) {
	b.builders[name] = builder
}

func (b *Builder) AddRegistry(reg registry.Registryer) {
	b.registry = reg
}

func (b *Builder) PullRepo(info model.BuildRequest) (model.PulledRepoInfo, error) {
	if info.Token == "" {
		return model.PulledRepoInfo{}, ErrEmptyToken
	}

	connector, ok := b.connectors[info.Connector]
	if !ok {
		return model.PulledRepoInfo{}, ErrConnectorNotFound
	}

	b.l.Infof("%s is pulling %s at branch %s", info.UserID, info.Repo, info.Branch)
	b.l.Debugf("info received: %+v", info)

	var pullInfo model.PulledRepoInfo
	var err error
	pullInfo.Path, pullInfo.RepoName, pullInfo.LastCommit, err = connector.Pull(info.UserID, info.Branch, info.Repo, info.Token)
	if err != nil {
		b.l.Errorf("error pulling %s: %v", info.Repo, err)
		return model.PulledRepoInfo{}, err
	}

	return pullInfo, nil
}

func (b *Builder) GetMetadata(info model.BuildRequest) (map[connectors.MetaType][]string, error) {
	if info.Token == "" {
		return nil, ErrEmptyToken
	}

	connector, ok := b.connectors[info.Connector]
	if !ok {
		return nil, ErrConnectorNotFound
	}

	b.l.Infof("getting metadata for %s-%s", info.Repo, info.Branch)
	b.l.Debugf("info received: %+v", info)

	metadata, err := connector.GetMetadata(info.Repo, info.Token)
	if err != nil {
		b.l.Errorf("error getting metadata for %s: %v", info.Repo, err)
		return nil, err
	}
	return metadata, nil
}

func (b *Builder) GetGranularMetadata(info model.BuildRequest, meta ...connectors.MetaType) (map[connectors.MetaType][]string, error) {
	if info.Token == "" {
		return nil, ErrEmptyToken
	}

	connector, ok := b.connectors[info.Connector]
	if !ok {
		return nil, ErrConnectorNotFound
	}

	b.l.Infof("getting %v metadata for %s-%s", meta, info.Repo, info.Branch)
	b.l.Debugf("info received: %+v", info)

	metadata, err := connector.GetMetadata(info.Repo, info.Token, meta...)
	if err != nil {
		b.l.Errorf("error getting metadata for %s: %v", info.Repo, err)
		return nil, err
	}
	return metadata, nil
}

func (b *Builder) BuildImage(ctx context.Context, info model.BuildRequest, path string) (string, string, error) {
	//clean up the path
	defer func() {
		b.l.Infof("cleaning up %s", path)
		os.RemoveAll(path)
	}()

	b.l.Infof("building image from %s", path)
	builder, ok := b.builders[model.DownloaderNixpacks]
	if !ok {
		return "", "", ErrBuilderNotFound
	}

	b.l.Debug("planning build")
	buildPlan, err := builder.Plan(ctx, path)
	if err != nil {
		b.l.Errorf("error planning build: %v", err)
		return "", "", err
	}

	b.l.Debugf("build plan: %+v", buildPlan)
	b.l.Debug("building image")
	imageID, errorMessage, err := builder.Build(ctx, info.UserID, info.Repo, buildPlan, path)
	if err != nil {
		b.l.Errorf("error building image: %v", err)
		b.l.Errorf("error message: %s", errorMessage)
		return "", errorMessage, err
	}
	return imageID, "", nil
}

func (b *Builder) GenerateImageName(userID string, info model.PulledRepoInfo) string {
	repo := strings.Split(info.Path, "/")[len(strings.Split(info.Path, "/"))-1]
	return fmt.Sprintf("%s/%s:%s", userID, repo, info.LastCommit)
}

func (b *Builder) PushImage(ctx context.Context, imageID, newImageName string) error {
	b.l.Infof("pushing image %s as %s", imageID, newImageName)
	toPush, err := b.registry.TagImage(imageID, newImageName)
	if err != nil {
		b.l.Errorf("error tagging image %s as %s: %v", imageID, newImageName, err)
	}

	if err := b.registry.PushImage(ctx, toPush); err != nil {
		b.l.Errorf("error pushing image %s: %v", toPush, err)
		return err
	}
	return nil
}
