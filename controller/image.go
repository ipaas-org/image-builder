package controller

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ipaas-org/image-builder/model"
)

func (b *Controller) BuildImage(ctx context.Context, info *model.BuildRequest, path string) (imageID string, errorMessage string, err error) {
	//clean up the path
	defer func() {
		b.l.Infof("cleaning up %s", path)
		os.RemoveAll(path)
	}()

	b.l.Infof("building image from %s", info.Repo)
	if b.Builder == nil {
		return "", "", ErrBuilderNotFound
	}
	b.l.Info("using nixpacks as builder")

	b.l.Debug("planning build")
	buildPlan, err := b.Builder.Plan(ctx, path)
	if err != nil {
		b.l.Errorf("error planning build: %v", err)
		return "", "", err
	}

	b.l.Debugf("build plan: %+v", buildPlan)
	b.l.Info("plan created successfully")
	b.l.Debug("building image")
	imageID, errorMessage, err = b.Builder.Build(ctx, info.UserID, info.Repo, buildPlan, path)
	if err != nil {
		b.l.Errorf("error building image: %v", err)
		b.l.Errorf("error message: %s", errorMessage)
		return "", errorMessage, err
	}
	b.l.Infof("image built successfully: id=%s", imageID)
	return imageID, "", nil
}

func (b *Controller) GenerateImageName(userID string, info *model.PulledRepoInfo) string {
	repo := strings.Split(info.Path, "/")[len(strings.Split(info.Path, "/"))-1]
	return fmt.Sprintf("%s/%s:%s", "applications", repo, info.LastCommit)
}

func (b *Controller) PushImage(ctx context.Context, imageID, username, appName string) (string, error) {
	if b.Registry == nil {
		return "", ErrMissingRegistry
	}
	newImageName := fmt.Sprintf("%s/%s", username, appName)
	b.l.Infof("pushing image %s as %s", imageID, newImageName)
	toPush, err := b.Registry.TagImage(ctx, imageID, username, appName)
	if err != nil {
		b.l.Errorf("error tagging image %s as %s: %v", imageID, newImageName, err)
		return "", err
	}

	if err := b.Registry.PushImage(ctx, toPush); err != nil {
		b.l.Errorf("error pushing image %s: %v", toPush, err)
		return "", err
	}
	return toPush, nil
}

func (b *Controller) IsPushRequired() bool {
	return b.Registry != nil
}
