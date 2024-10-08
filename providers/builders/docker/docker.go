package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
	"github.com/google/uuid"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/builders"
)

const DockerBuilderKind model.BuilderKind = "docker"

var _ builders.Builder = new(DockerBuilder)

type DockerBuilder struct {
	builderVersion string
	cli            *client.Client
}

type DockerBuilderConfig struct {
	DockerFilePath string            `json:"dockerfilePath"`
	Envs           map[string]string `json:"envs"`
	// Args           map[string]string `json:"args"`
	// StartCommand   string `json:"startCommand"`
	// DockerIngorePath string `json:"dockerignorePath"`
}

func NewDockerBuilder(builderVersion string) (*DockerBuilder, error) {
	// creating docker client from env
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &DockerBuilder{
		builderVersion: builderVersion,
		cli:            cli,
	}, nil
}

func convertModelKeyValueToDockerEnvs(envs []model.KeyValue) map[string]string {
	envMap := make(map[string]string)
	for _, env := range envs {
		envMap[env.Key] = env.Value
	}
	return envMap
}

func (b DockerBuilder) Plan(ctx context.Context, config *model.BuildConfig, path string) (builders.Plan, error) {
	plan := new(DockerBuilderConfig)

	// dockerfilePath := filepath.Join(config.RootDirectory, config.DockerfilePath)
	plan.DockerFilePath = config.DockerfilePath
	plan.Envs = convertModelKeyValueToDockerEnvs(config.Envs)
	jsonPlan, err := json.Marshal(plan)
	if err != nil {
		return "", err
	}
	return builders.Plan(jsonPlan), nil
}

// first string is the image name, second is an error message if there was an error building the image
func (b DockerBuilder) Build(ctx context.Context, userID, repo, path string, plan builders.Plan) (string, []byte, error) {
	config := new(DockerBuilderConfig)
	if err := json.Unmarshal([]byte(plan), config); err != nil {
		return "", nil, builders.ErrInvalidPlan
	}

	//create a build context, is a tar with the temp repo,
	//needed since we are not using the filesystem as a context
	buildContext, err := archive.TarWithOptions(path, &archive.TarOptions{
		NoLchown: true,
	})
	if err != nil {
		return "", nil, err
	}
	defer buildContext.Close()

	imageName := uuid.New().String()

	resp, err := b.cli.ImageBuild(ctx, buildContext, types.ImageBuildOptions{
		//Squash: true,
		// Version: types.BuilderBuildKit,
		Dockerfile: config.DockerFilePath,
		Tags:       []string{imageName},
		Labels: map[string]string{
			"org.ipaas.image-builder.version": b.builderVersion,
			"org.ipaas.image-builder.builder": string(DockerBuilderKind),
			"application.repo":                repo,
			"application.userID":              userID,
			"application.builtAt":             time.Now().Format("02/01/2006 15:04:05"),
		},
		Remove:      true,
		ForceRemove: true,
	})

	if err != nil {
		if strings.Contains(err.Error(), "Cannot locate specified Dockerfile") {
			return "", nil, builders.ErrMissingConfig
		} else if strings.Contains(err.Error(), "dockerfile parse error") {
			return "", nil, builders.ErrInvalidConfig
		}
		return "", nil, err
	}

	imageBuildOutput, err := ConvertOutput(resp.Body)
	if err != nil {
		return "", nil, err
	}

	imageID, err := getImageId(ctx, imageName)
	if err != nil {
		return "", nil, err
	}

	//find the id of the image just created
	if !checkIfImageCompiled(imageBuildOutput) {
		return imageID, imageBuildOutput, builders.ErrImageNotCompiled
	}

	return imageID, imageBuildOutput, nil
}

func getImageId(ctx context.Context, imageName string) (string, error) {
	var out bytes.Buffer
	cmd := exec.CommandContext(ctx, "docker", "images", "-q", imageName)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	imageID := strings.Replace(out.String(), "\n", "", -1)
	return imageID, nil
}

func checkIfImageCompiled(imageBuildOutput []byte) bool {
	lines := bytes.Split(imageBuildOutput, []byte{'\n'})
	return strings.Contains(string(lines[len(lines)-2]), "Successfully tagged")
}
