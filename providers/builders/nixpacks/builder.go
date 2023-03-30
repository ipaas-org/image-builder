package nixpacks

import (
	"context"

	"github.com/ipaas-org/image-builder/providers/builders"
	nixpacks "github.com/vano2903/nixpacks-go"
)

var _ builders.Builder = new(NixPackBuilder)

type NixPackBuilder struct {
	RegistryUri string
}

func NewNixPackBuilder(registryUri string) *NixPackBuilder {
	return &NixPackBuilder{
		RegistryUri: registryUri,
	}
}

func (b NixPackBuilder) Publish(image string) error {
	return nil
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
func (b NixPackBuilder) Build(ctx context.Context, config, path string) (string, string, error) {
	n, err := nixpacks.NewNixpacks()
	if err != nil {
		return "", "", err
	}

	buildCmd, err := n.Build(ctx, nixpacks.BuildOptions{
		Path:     path,
		JsonPlan: config,
	})
	if err != nil {
		return "", "", err
	}

	build, err := buildCmd.Result()

	return build.ImageName, build.BuildError, err
}
