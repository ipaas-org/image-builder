package harbor

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/docker/docker/errdefs"
	"github.com/ipaas-org/image-builder/providers/registry/registry"
	goharbor "github.com/x893675/go-harbor"
	"github.com/x893675/go-harbor/schema"
)

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

type HarborClient struct {
	serverAddress    string
	username         string
	password         string
	userPullUsername string
	userPullPassword string

	registry     *registry.Registry
	harborClient *goharbor.Client
}

// if no authentication is required, leave username and password empty
func NewHarborRegistry(registryUri, username, password, userPullUsername, userPullPassword string) (*HarborClient, error) {
	if username == "" || password == "" {
		return nil, fmt.Errorf("username and password are required")
	}
	if userPullUsername == "" || userPullPassword == "" {
		return nil, fmt.Errorf("userPullUsername and userPullPassword are required")
	}

	c := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	harborClient, err := goharbor.NewClientWithOpts(goharbor.WithHost("https://"+registryUri),
		goharbor.WithHTTPClient(c),
		goharbor.WithBasicAuth(username, password))
	if err != nil {
		return nil, err
	}

	r, err := registry.NewDefaultRegistry(registryUri, username, password)
	if err != nil {
		return nil, err
	}
	return &HarborClient{
		serverAddress:    registryUri,
		username:         username,
		password:         password,
		userPullUsername: userPullUsername,
		// userPullPassword: userPullPassword,
		registry:     r,
		harborClient: harborClient,
	}, nil
}

func (r *HarborClient) TagImage(ctx context.Context, localImageID, userCode, appName string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute*3)
	defer cancel()

	_, err := r.harborClient.GetProject(ctx, userCode)
	if err != nil {
		log.Println("project not found, creating project of name:", userCode, "error is", err)
		if errdefs.IsNotFound(err) {
			log.Println("creating project")
			err = r.harborClient.CreateProject(ctx, schema.CreateProjectOptions{
				Name: userCode,
				Metadata: &schema.ProjectMetadata{
					Public: "false",
				},
			})
			if err != nil {
				log.Println("error creating project", err)
				return "", err
			}
			// log.Println("sleeping cause it takes time to create project")
			// time.Sleep(2 * time.Second)
			log.Println("getting project after create")
			project, err := r.harborClient.GetProject(ctx, userCode)
			if err != nil {
				log.Println("error getting project after create", err)
				return "", err
			}

			log.Println("adding project member")
			err = r.harborClient.AddProjectMember(ctx, project.ProjectID, schema.ProjectMember{
				RoleID:      schema.Guest,
				MemberGroup: schema.UserGroup{},
				MemberUser: schema.UserEntity{
					Username: r.userPullUsername,
				},
			})
			if err != nil {
				log.Println("error adding project member", err)
				return "", err
			}
		} else {
			return "", err
		}
	}

	return r.registry.TagImage(ctx, localImageID, userCode, appName)
}

func (r *HarborClient) PushImage(ctx context.Context, imageID string) error {
	return r.registry.PushImage(ctx, imageID)
}
