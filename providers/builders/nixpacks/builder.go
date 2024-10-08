package nixpacks

import (
	"context"
	"time"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/builders"
	nixpacks "github.com/vano2903/nixpacks-go"
)

const NixPackBuilderKind model.BuilderKind = "nixpacks"

var _ builders.Builder = new(NixPackBuilder)

type NixPackBuilder struct {
	builderVersion string
}

func NewNixPackBuilder(builderVersion string) *NixPackBuilder {
	return &NixPackBuilder{
		builderVersion: builderVersion,
	}
}

func convertBuildConfigEnvsToNixpacksEnvs(envs []model.KeyValue) []nixpacks.Env {
	nixpacksEnvs := make([]nixpacks.Env, 0)
	for _, env := range envs {
		nixpacksEnvs = append(nixpacksEnvs, nixpacks.Env{
			Key:   env.Key,
			Value: env.Value,
		})
	}
	return nixpacksEnvs
}

func (b NixPackBuilder) Plan(ctx context.Context, config *model.BuildConfig, path string) (builders.Plan, error) {
	n, err := nixpacks.NewNixpacks()
	if err != nil {
		return "", err
	}

	opt := nixpacks.PlanOptions{
		Path: path,
		Envs: convertBuildConfigEnvsToNixpacksEnvs(config.Envs),

		Config: config.NixpacksPath,

		NixPackages: config.NixPkgs,
		// NixLibraries: config.NixLibs,
		AptPackages: config.AptPkgs,

		InstallCommand: config.InstallCommand,
		BuildCommand:   config.BuildCommand,
		StartCommand:   config.StartCommand,
	}

	planCmd := n.Plan(ctx, opt)
	if planCmd.Error() != nil {
		return "", planCmd.Error()
	}

	plan, err := planCmd.Result()
	if err != nil {
		return "", err
	}

	return builders.Plan(plan.Response), nil
}

// first string is the image name, second is build output
func (b NixPackBuilder) Build(ctx context.Context, userID, repo, path string, plan builders.Plan) (string, []byte, error) {
	n, err := nixpacks.NewNixpacks()
	if err != nil {
		return "", nil, err
	}

	buildCmd, err := n.Build(ctx, nixpacks.BuildOptions{
		Labels: []nixpacks.Label{
			{
				Key:   "org.ipaas.image-builder.version",
				Value: b.builderVersion,
			},
			{
				Key:   "org.ipaas.image-builder.builder",
				Value: string(NixPackBuilderKind),
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
		JsonPlan: string(plan),
	})
	if err != nil {
		return "", nil, err
	}

	build, err := buildCmd.Result()
	return build.ImageName, build.Response, err
}
