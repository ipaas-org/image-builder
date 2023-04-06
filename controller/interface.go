package controller

import (
	"context"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/builders"
	"github.com/ipaas-org/image-builder/providers/connectors"
	"github.com/ipaas-org/image-builder/providers/registry"
)

type (
	BuilderController interface {
		AddBuilder(name string, builder builders.Builder)
		AddConnector(name string, conn connectors.Connector)
		AddRegistry(reg registry.Registryer)
		PullRepo(model.ImageBuildInfo) (model.PulledRepoInfo, error)
		BuildImage(ctx context.Context, info model.ImageBuildInfo, path string) (imageID string, buildError string, err error)
		PushImage(ctx context.Context, imageID, newImageName string) error
		// CallContainerManager(imageID string) error
	}
)
