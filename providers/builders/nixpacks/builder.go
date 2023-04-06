package nixpacks

import (
	"context"
	"time"

	"github.com/ipaas-org/image-builder/providers/builders"
	nixpacks "github.com/vano2903/nixpacks-go"
)

var _ builders.Builder = new(NixPackBuilder)

type NixPackBuilder struct {
	builderVersion string
}

func NewNixPackBuilder(builderVersion string) *NixPackBuilder {
	return &NixPackBuilder{
		builderVersion: builderVersion,
	}
}

func (b NixPackBuilder) Plan(ctx context.Context, path string) (string, error) {
	n, err := nixpacks.NewNixpacks()
	if err != nil {
		return "", err
	}
	planCmd, err := n.Plan(ctx, nixpacks.PlanOptions{Path: path})
	if err != nil {
		return "", err
	}

	plan, err := planCmd.Result()
	if err != nil {
		return "", err
	}

	return string(plan.Response), nil
}

// first string is the image name, second is an error message if there was an error building the image
func (b NixPackBuilder) Build(ctx context.Context, userID, repo, config, path string) (string, string, error) {
	n, err := nixpacks.NewNixpacks()
	if err != nil {
		return "", "", err
	}

	buildCmd, err := n.Build(ctx, nixpacks.BuildOptions{
		Labels: []nixpacks.Label{
			{
				Key:   "org.ipaas.image-builder.version",
				Value: b.builderVersion,
			},
			{
				Key:   "org.ipaas.image-builder.builder",
				Value: "nixpacks",
			},
			{
				Key:   "application.repo",
				Value: repo,
			}, {
				Key:   "application.userID",
				Value: userID,
			},
			{
				Key:   "application.builtAt",
				Value: time.Now().Format("02/01/2006 15:04:05"),
			},
		},
		Path:     path,
		JsonPlan: config,
	})
	if err != nil {
		return "", "", err
	}

	build, err := buildCmd.Result()
	return build.ImageName, build.BuildError, err
}
