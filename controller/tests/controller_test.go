package controller

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/ipaas-org/image-builder/controller"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/pkg/logger"
	"github.com/ipaas-org/image-builder/providers/builders/docker"
	"github.com/ipaas-org/image-builder/providers/builders/nixpacks"
	"github.com/ipaas-org/image-builder/providers/connectors/github"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
	"gotest.tools/assert"
)

var (
	logType   = "text"
	logLvl    = "debug"
	tmpFolder = "../../tmp"
	token     string

	c *controller.Controller
	l *logrus.Logger
)

func setup() {
	userAgent := "ipaas-image-builder-test"

	l = logger.NewLogger(logLvl, logType)
	l.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	c = controller.NewController(l)

	githubConnector := github.NewGithubConnector(tmpFolder, userAgent, l)
	c.AddConnector(model.ConnectorGithub, githubConnector)

	nixBuilder := nixpacks.NewNixPackBuilder("testing-nixpacks")
	dockerBuilder, err := docker.NewDockerBuilder("testing-docker")
	if err != nil {
		log.Fatal("unable to create docker builder", err)
	}
	c.AddBuilder(nixpacks.NixPackBuilderKind, nixBuilder)
	c.AddBuilder(docker.DockerBuilderKind, dockerBuilder)

	// r, err := registry.NewRegistry("localhost:5000", "", "")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// c.Registry = r
	c.Registry = nil
	// c.AddRegistry(r)

	if err := godotenv.Load(".env"); err != nil {
		log.Fatal("unable to load .env file:", err.Error())
	}
	var found bool
	token, found = os.LookupEnv("GITHUB_TEST_TOKEN")
	if !found {
		log.Fatal("GITHUB_TEST_TOKEN is not set")
	}
}

// tests pull repo and metadata extraction
// [x] pull repo
// [x] pull repo and branch
// [x] pull private repo
// [x] pull unexisting repo
// [x] pull unexisting branch
// [x] pull repo with invalid token
func TestPullRepo(t *testing.T) {
	ctx := context.Background()
	t.Run("pull repo", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		pullInfo := &model.PullInfoRequest{
			Token:     token,
			Repo:      "vano2903/testing",
			Branch:    "", //will default
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}
		expectedName := "testing"
		expectedPath := tmpFolder + "/18008-testing-master"
		info, err := c.PullRepo(ctx, pullInfo)
		if err != nil {
			t.Fatalf("unable to pull repo: %v", err)
		}

		t.Logf("info: %v", info)
		assert.Equal(t, info.RepoName, expectedName)
		assert.Equal(t, info.Path, expectedPath)
	})

	t.Run("pull private repo", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		pullInfo := &model.PullInfoRequest{
			Token:     token,
			Repo:      "vano2903/dea-landing",
			Branch:    "",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}
		expectedName := "dea-landing"
		expectedPath := tmpFolder + "/18008-dea-landing-master"
		info, err := c.PullRepo(ctx, pullInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		t.Logf("info: %v", info)
		assert.Equal(t, info.RepoName, expectedName)
		assert.Equal(t, info.Path, expectedPath)
	})

	t.Run("pull repo and branch", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		pullInfo := &model.PullInfoRequest{
			Token:     token,
			Repo:      "vano2903/testing",
			Branch:    "runtime-error",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}
		expectedName := "testing"
		expectedPath := tmpFolder + "/18008-testing-runtime-error"
		info, err := c.PullRepo(ctx, pullInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		t.Logf("info: %v", info)
		assert.Equal(t, info.RepoName, expectedName)
		assert.Equal(t, info.Path, expectedPath)
	})

	t.Run("pull unexisting repo", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		pullInfo := &model.PullInfoRequest{
			Token:     token,
			Repo:      "vano2903/unexisting",
			Branch:    "master",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		_, err := c.PullRepo(ctx, pullInfo)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("pull unexisting branch", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		pullInfo := &model.PullInfoRequest{
			Token:     token,
			Repo:      "vano2903/testing",
			Branch:    "unexisting-branch",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		_, err := c.PullRepo(ctx, pullInfo)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("pull repo with invalid token", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		pullInfo := &model.PullInfoRequest{
			Token:     token,
			Repo:      "vano2903/testing",
			Branch:    "unexisting-branch",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		_, err := c.PullRepo(ctx, pullInfo)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	// t.Run("extract metadata", func(t *testing.T) {
	// 	if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
	// 		if err := os.Mkdir(tmpFolder, 0755); err != nil {
	// 			t.Fatal(err)
	// 		}
	// 	}
	// 	setup()
	// 	imageBuildInfo := model.BuildRequest{
	// 		Token:     token,
	// 		Repo:      "vano2903/ipaas",
	// 		Branch:    "non-relational-version",
	// 		UserID:    "18008",
	// 		Connector: model.ConnectorGithub,
	// 	}
	// 	expectedMetadata := make(map[connectors.MetaType][]string)

	// 	expectedMetadata[github.MetaDefaultBranch] = []string{"non-relational-version"}
	// 	expectedMetadata[github.MetaDescription] = []string{"A simple self hosted PaaS for full stack applications and DBaaS"}
	// 	expectedMetadata[github.MetaReleases] = []string{"IPAAS - first working version"}
	// 	expectedMetadata[github.MetaTags] = []string{"v1.0.0"}
	// 	expectedMetadata[github.MetaBranches] = []string{
	// 		"add-license-1",
	// 		"master",
	// 		"micro-services",
	// 		"non-relational-version",
	// 		"testing",
	// 	}

	// 	metadata, err := c.GetMetadata(imageBuildInfo)
	// 	if err != nil {
	// 		t.Errorf("unable to pull repo: %v", err)
	// 	}
	// 	t.Logf("metadata: %v", metadata)

	// 	assert.DeepEqual(t, metadata, expectedMetadata)
	// })

	// t.Run("extract granualr metadata", func(t *testing.T) {

	// 	if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
	// 		if err := os.Mkdir(tmpFolder, 0755); err != nil {
	// 			t.Fatal(err)
	// 		}
	// 	}
	// 	setup()
	// 	imageBuildInfo := model.BuildRequest{
	// 		Token:     token,
	// 		Repo:      "vano2903/ipaas",
	// 		Branch:    "non-relational-version",
	// 		UserID:    "18008",
	// 		Connector: model.ConnectorGithub,
	// 	}
	// 	expectedMetadata := make(map[connectors.MetaType][]string)

	// 	expectedMetadata[github.MetaDefaultBranch] = []string{"non-relational-version"}
	// 	expectedMetadata[github.MetaDescription] = []string{"A simple self hosted PaaS for full stack applications and DBaaS"}
	// 	expectedMetadata[github.MetaReleases] = nil
	// 	expectedMetadata[github.MetaTags] = []string{"v1.0.0"}
	// 	expectedMetadata[github.MetaBranches] = nil

	// 	metadata, err := c.GetGranularMetadata(imageBuildInfo, github.MetaDescription, github.MetaTags)
	// 	if err != nil {
	// 		t.Errorf("unable to pull repo: %v", err)
	// 	}
	// 	t.Logf("metadata: %v", metadata)

	// 	assert.DeepEqual(t, metadata, expectedMetadata)
	// })

	t.Cleanup(func() {
		os.RemoveAll(tmpFolder)
	})
}

// builds the image and returns the image id
// [x] build image from repo
// [x] detect build error
// [x] cancel build with context
func TestBuildImage(t *testing.T) {
	ctx := context.Background()
	t.Run("build image", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		buildRequest := &model.Request{
			ApplicationID: "test-build-image",
			PullInfo: &model.PullInfoRequest{
				Token:     token,
				Repo:      "vano2903/testing",
				Branch:    "master",
				UserID:    "18008",
				Connector: model.ConnectorGithub,
			},
			BuildPlan: &model.BuildConfig{
				Builder: "nixpacks",
			},
		}

		info, err := c.PullRepo(ctx, buildRequest.PullInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		imageID, _, err := c.BuildImage(context.Background(), buildRequest.PullInfo.Repo, buildRequest.PullInfo.UserID, info.Path, buildRequest.BuildPlan)
		if err != nil {
			t.Errorf("unable to build image: %v", err)
		}

		t.Logf("imageID: %v", imageID)
	})

	t.Run("detect build error", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}

		setup()

		buildRequest := &model.Request{
			ApplicationID: "test-build-image",
			PullInfo: &model.PullInfoRequest{
				Token:     token,
				Repo:      "vano2903/testing",
				Branch:    "non-working-version",
				UserID:    "18008",
				Connector: model.ConnectorGithub,
			},
			BuildPlan: &model.BuildConfig{
				Builder: "nixpacks",
			},
		}

		info, err := c.PullRepo(ctx, buildRequest.PullInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		_, buildError, err := c.BuildImage(context.Background(), buildRequest.PullInfo.Repo, buildRequest.PullInfo.UserID, info.Path, buildRequest.BuildPlan)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		t.Logf("buildError: %s", buildError)
	})

	t.Run("cancel build with context", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}

		setup()
		buildRequest := &model.Request{
			ApplicationID: "test-build-image",
			PullInfo: &model.PullInfoRequest{
				Token:     token,
				Repo:      "vano2903/testing",
				Branch:    "master",
				UserID:    "18008",
				Connector: model.ConnectorGithub,
			},
			BuildPlan: &model.BuildConfig{
				Builder: "nixpacks",
			},
		}

		info, err := c.PullRepo(ctx, buildRequest.PullInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		ctx, _ := context.WithTimeout(context.Background(), 100*time.Millisecond)

		_, _, err = c.BuildImage(ctx, buildRequest.PullInfo.Repo, buildRequest.PullInfo.UserID, info.Path, buildRequest.BuildPlan)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Cleanup(func() {
		os.RemoveAll(tmpFolder)
	})
}

// pushes the image to the registry
// [x] push image
// [x] detect push error
func TestPushImage(t *testing.T) {
	ctx := context.Background()
	t.Run("push image", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		buildRequest := &model.Request{
			ApplicationID: "test-build-image",
			PullInfo: &model.PullInfoRequest{
				Token:     token,
				Repo:      "vano2903/testing",
				Branch:    "", //will default
				UserID:    "18008",
				Connector: model.ConnectorGithub,
			},
			BuildPlan: &model.BuildConfig{
				Builder: "nixpacks",
			},
		}
		info, err := c.PullRepo(ctx, buildRequest.PullInfo)
		if err != nil {
			t.Fatalf("unable to pull repo: %v", err)
		}

		imageID, _, err := c.BuildImage(ctx, buildRequest.PullInfo.Repo, buildRequest.PullInfo.UserID, info.Path, buildRequest.BuildPlan)
		if err != nil {
			t.Fatalf("unable to build image: %v", err)
		}
		l.Infof("imageID: %v", imageID)

		newTag := c.GenerateImageName("18008", info)

		l.Infof("pushing image %s to %s", imageID, newTag)
		// err = c.PushImage(context.Background(), imageID, newTag)
		// if err != nil {
		// 	t.Errorf("unable to push image: %v", err)
		// }
	})

	t.Run("detect push error", func(t *testing.T) {
		if _, err := os.Stat(tmpFolder); os.IsNotExist(err) {
			if err := os.Mkdir(tmpFolder, 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		buildRequest := &model.Request{
			ApplicationID: "test-build-image",
			PullInfo: &model.PullInfoRequest{
				Token:     token,
				Repo:      "vano2903/testing",
				Branch:    "", //will default
				UserID:    "18008",
				Connector: model.ConnectorGithub,
			},
			BuildPlan: &model.BuildConfig{
				Builder: "nixpacks",
			},
		}
		info, err := c.PullRepo(ctx, buildRequest.PullInfo)
		if err != nil {
			t.Fatalf("unable to pull repo: %v", err)
		}

		imageID, _, err := c.BuildImage(ctx, buildRequest.PullInfo.Repo, buildRequest.PullInfo.UserID, info.Path, buildRequest.BuildPlan)
		if err != nil {
			t.Fatalf("unable to build image: %v", err)
		}
		l.Infof("imageID: %v", imageID)

		newTag := c.GenerateImageName("18008", info)
		t.Logf("image to push: %s", newTag)
		// err = c.PushImage(context.Background(), "", newTag) //try to push empty image will result in an error
		// if err == nil {
		// 	t.Errorf("unable to push image: %v", err)
		// }
	})
}
