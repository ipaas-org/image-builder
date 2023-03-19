package controller

import (
	"github.com/vano2903/image-builder/model"
	"github.com/vano2903/image-builder/providers/builders"
	"github.com/vano2903/image-builder/providers/connectors"
)

type (
	BuilderController interface {
		AddBuilder(name string, builder builders.Builder)
		AddConnector(name string, conn connectors.Connector)
		PullRepo(model.ImageBuildInfo) (model.PulledRepoInfo, error)
		BuildImage(path string) (imageID string, buildError string, err error)
		PushImage(imageID string) error
		CallContainerManager(imageID string) error
		Clean(path string) error
	}
)
