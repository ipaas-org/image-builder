package baseAnalyzer

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/builders/docker"
	nixBuilder "github.com/ipaas-org/image-builder/providers/builders/nixpacks"
	"github.com/vano2903/nixpacks-go"
)

type BaseAnalyzer struct {
	nixpacks *nixpacks.Nixpacks
}

func NewBaseAnalyzer() (*BaseAnalyzer, error) {
	nix, err := nixpacks.NewNixpacks()
	return &BaseAnalyzer{
		nixpacks: nix,
	}, err
}

func (b *BaseAnalyzer) DetectBuilders(ctx context.Context, path string) (*model.DetectedInfo, error) {
	files, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	info := new(model.DetectedInfo)

	nixInfo := new(model.NixPacksInfo)

	out, err := b.nixpacks.Detect(ctx, nixpacks.DetectOptions{
		Path: path,
	}).Result()
	if err != nil {
		return nil, err
	}
	plan, err := b.nixpacks.Plan(ctx, nixpacks.PlanOptions{
		Path: path,
	}).Result()
	if err != nil {
		return nil, err
	}

	fmt.Printf("plan >>>>> %+v\n", plan)

	if len(out.Providers) > 0 { //nil pointer dereference
		nixInfo.NixPacksProviders = out.Providers
		nixInfo.StartCommand = plan.Start.Cmd
		nixInfo.BuildCommands = plan.Phases.Build.Cmds
		nixInfo.InstallCommands = plan.Phases.Install.Cmds
		nixInfo.Variables = plan.Variables

		//* it should not return those packages because those packages
		//* are used by nixpacks and can not be modified
		// nixInfo.NixPackages = plan.Phases.Setup.NixPkgs
		// nixInfo.AptPackages = plan.Phases.Setup.AptPkgs
	}

	fmt.Printf("nixInfo >>>>> %+v\n", nixInfo)

	dockerInfo := new(model.DockerInfo)
	for _, f := range files {
		name := strings.ToLower(f.Name())
		if name == "nixpacks.json" || name == "nixpacks.toml" {
			nixInfo.NixPacksConfigPath = f.Name()
		}

		if strings.HasSuffix(name, "dockerfile") || strings.HasPrefix(name, "dockerfile") {
			dockerInfo.Dockerfiles = append(dockerInfo.Dockerfiles, f.Name())
		}
		if strings.HasSuffix(name, ".dockerignore") {
			dockerInfo.DockerIgnoreFound = true
		}
	}

	if len(dockerInfo.Dockerfiles) > 0 {
		info.Builders = append(info.Builders, docker.DockerBuilderKind)
		info.Docker = dockerInfo
	}
	if dockerInfo.DockerIgnoreFound {
		info.Docker = dockerInfo
	}

	if !dockerInfo.DockerIgnoreFound {
		if len(nixInfo.NixPacksProviders) > 0 || nixInfo.NixPacksConfigPath != "" {
			info.Builders = append(info.Builders, nixBuilder.NixPackBuilderKind)
			info.NixPacks = nixInfo
		}
	}

	return info, nil
}
