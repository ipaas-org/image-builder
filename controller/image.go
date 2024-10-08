package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/builders"
	"github.com/ipaas-org/image-builder/providers/builders/docker"
	"github.com/ipaas-org/image-builder/providers/builders/nixpacks"
)

func (b *Controller) BuildImage(ctx context.Context, repo, userID, repoPath string, config *model.BuildConfig) (imageID string, imageOutput []byte, err error) {
	//clean up the path
	defer func() {
		b.l.Infof("cleaning up %s", repoPath)
		os.RemoveAll(repoPath)
	}()
	path := filepath.Join(repoPath, config.RootDirectory)

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return "", nil, ErrInexistingRootDir
		}
		return "", nil, err
	}

	b.l.Infof("building image from %s", repo)
	b.l.Infof("location: %s", path)
	if b.Builders == nil {
		return "", nil, ErrBuilderNotFound
	}

	var builder builders.Builder
	switch config.Builder {
	case "docker":
		b.l.Info("using docker as builder")
		builder = b.Builders[docker.DockerBuilderKind]
	case "nixpacks":
		b.l.Info("using nixpacks as builder")
		builder = b.Builders[nixpacks.NixPackBuilderKind]
	default:
		return "", nil, ErrBuilderNotFound
	}

	b.l.Debug("planning build")
	buildPlan, err := builder.Plan(ctx, config, path)
	if err != nil {
		b.l.Errorf("error planning build: %v", err)
		return "", nil, err
	}

	b.l.Debugf("build plan: %+v", buildPlan)
	b.l.Info("plan created successfully")
	b.l.Debug("building image")
	imageID, imageOutput, err = builder.Build(ctx, userID, repo, path, buildPlan)
	if err != nil {
		b.l.Errorf("error building image: %v", err)
		b.l.Errorf("build output: %s", imageOutput)
		return "", imageOutput, err
	}
	b.l.Infof("image built successfully: id=%s", imageID)
	return imageID, imageOutput, nil

}

func (b *Controller) GenerateImageName(userID string, info *model.PulledRepoInfo) string {
	repo := strings.Split(info.Path, "/")[len(strings.Split(info.Path, "/"))-1]
	return fmt.Sprintf("%s/%s:%s", "applications", repo, info.PulledCommit)
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
