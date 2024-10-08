package controller

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipaas-org/image-builder/model"
	dockerBuilder "github.com/ipaas-org/image-builder/providers/builders/docker"
	nixBuilder "github.com/ipaas-org/image-builder/providers/builders/nixpacks"
)

func (c *Controller) AnalyzeRepositoryContent(ctx context.Context, path, root, repo, branch string) (*model.RepoAnalisys, error) {
	c.l.Infof("Analyzing repository content at root %s on repo %s <%s>", root, repo, branch)

	toAnalyzePath := filepath.Join(path, root)
	if _, err := os.Stat(toAnalyzePath); err != nil {
		if os.IsNotExist(err) {
			return nil, ErrInexistingRootDir
		}
		return nil, err
	}

	// analyze repo content
	repoInfo, err := c.Analyzer.DetectBuilders(ctx, toAnalyzePath)
	if err != nil {
		c.l.Errorf("error analyzing %s: %v", repo, err)
		return nil, err
	}
	c.l.Infof("analyzed %s successfully", repo)
	c.l.Debugf("repo info: %+v", repoInfo)
	isBuildable := true
	reason := ""
	if len(repoInfo.Builders) == 0 {
		isBuildable = false
		if repoInfo.Docker == nil {
			reason = fmt.Sprintf("no Dockerfile found and in %s there are not enough information to automatically detect a build plan", root)
		} else if repoInfo.Docker.DockerIgnoreFound {
			reason = "no Dockerfile found and .dockerignore found, the dockerignore prevents our autobuilder from building the repo"
		}
	}
	analisys := &model.RepoAnalisys{
		IsBuildable: isBuildable,
		Reason:      reason,
		RepoInfo:    repoInfo,
	}

	return analisys, nil
}

func (c *Controller) GenerateBuildConfig(ctx context.Context, repoAnalysis *model.RepoAnalisys) (*model.BuildConfig, error) {
	if !repoAnalysis.IsBuildable || repoAnalysis.RepoInfo == nil {
		return nil, ErrNotBuildable
	}

	buildConfig := new(model.BuildConfig)

	// always default to docker
	if repoAnalysis.RepoInfo.Docker == nil {
		// if there is a nixpacks.[json|toml] file it will defaults to that
		// otherwise use the just generated plan
		nixpacks := repoAnalysis.RepoInfo.NixPacks
		if nixpacks.NixPacksConfigPath != "" {
			buildConfig.NixpacksPath = nixpacks.NixPacksConfigPath
		} else {
			buildConfig.Builder = nixBuilder.NixPackBuilderKind
			buildConfig.Envs = convertNixpacksVariablesToModelKeyValue(nixpacks.Variables)
			buildConfig.NixPkgs = nixpacks.NixPackages
			buildConfig.AptPkgs = nixpacks.AptPackages
			buildConfig.NixLibs = nixpacks.NixLibraries
			buildConfig.InstallCommand = strings.Join(nixpacks.InstallCommands, " ")
			buildConfig.BuildCommand = strings.Join(nixpacks.BuildCommands, " ") //todo: check if we should add "; " as join
			buildConfig.StartCommand = nixpacks.StartCommand
		}
	} else {
		// defaults to Dockerfile, if not found use the first dockerfile found
		buildConfig.Builder = dockerBuilder.DockerBuilderKind
		for _, dockerfile := range repoAnalysis.RepoInfo.Docker.Dockerfiles {
			if dockerfile == "Dockerfile" {
				buildConfig.DockerfilePath = dockerfile
				break
			}
		}
		if buildConfig.DockerfilePath == "" {
			buildConfig.DockerfilePath = repoAnalysis.RepoInfo.Docker.Dockerfiles[0]
		}
	}

	return buildConfig, nil
}

func convertNixpacksVariablesToModelKeyValue(vars map[string]string) []model.KeyValue {
	var kvs []model.KeyValue
	for k, v := range vars {
		kvs = append(kvs, model.KeyValue{
			Key:   k,
			Value: v,
		})
	}
	return kvs
}
