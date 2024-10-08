package registry

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"time"

	"github.com/docker/docker/api/types/image"
	registryType "github.com/docker/docker/api/types/registry"
	"github.com/ipaas-org/image-builder/providers/registry"

	"github.com/docker/docker/client"
)

var _ registry.Registryer = new(Registry)

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

type Registry struct {
	serverAddress string
	username      string
	password      string

	dockerClient *client.Client
}

// if no authentication is required, leave username and password empty
func NewDefaultRegistry(registryUri, username, password string) (*Registry, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}

	return &Registry{
		serverAddress: registryUri,
		username:      username,
		password:      password,
		dockerClient:  cli,
	}, nil
}

func (r *Registry) TagImage(ctx context.Context, localImageID, userCode, appName string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*3)
	defer cancel()

	new := r.serverAddress + "/" + userCode + "/" + appName //
	if err := r.dockerClient.ImageTag(ctx, localImageID, new); err != nil {
		return "", err
	}
	return new, nil
}

func (r *Registry) PushImage(ctx context.Context, imageID string) error {
	var authConfig = registryType.AuthConfig{
		Username:      r.username,
		Password:      r.password,
		ServerAddress: r.serverAddress,
	}
	authConfigBytes, _ := json.Marshal(authConfig)
	authConfigEncoded := base64.URLEncoding.EncodeToString(authConfigBytes)

	opts := image.PushOptions{RegistryAuth: authConfigEncoded}
	rd, err := r.dockerClient.ImagePush(ctx, imageID, opts)
	if err != nil {
		return err
	}

	defer rd.Close()

	return checkErr(rd)
}

func checkErr(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		// fmt.Println(scanner.Text())
	}

	errLine := &ErrorLine{}
	if err := json.Unmarshal([]byte(lastLine), errLine); err != nil {
		return err
	}

	if errLine.Error != "" {
		//TODO: we are ignoring the error detail here, we should consider them
		return errors.New(errLine.Error)
	}

	return scanner.Err()
}
