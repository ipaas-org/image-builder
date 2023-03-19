package controller

import (
	"fmt"
	"log"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/vano2903/image-builder/config"
	"github.com/vano2903/image-builder/controller"
	"github.com/vano2903/image-builder/model"
	"github.com/vano2903/image-builder/pkg/logger"
	"github.com/vano2903/image-builder/providers/builders/nixpacks"
	"github.com/vano2903/image-builder/providers/connectors/github"
	"gotest.tools/assert"
)

var (
	c *controller.Builder
	l *logrus.Logger
)

func setup() {
	conf := config.Config{}
	conf.Log.Level = "debug"
	conf.Log.Type = "text"
	conf.Database.Driver = "mock"
	userAgent := "ipaas-image-builder-test"
	l = logger.NewLogger(conf.Log.Level, conf.Log.Type)

	if conf.Database.Driver != "mock" {
		log.Fatal("only mock database is supported in this example")
	}

	c = controller.NewBuilderController(l)

	githubConnector := github.NewGithubConnector("./tmp", userAgent, "token", l)
	c.AddConnector(model.ConnectorGithub, githubConnector)

	fmt.Printf("githubConnector: %+v", *githubConnector)
	nix := nixpacks.NewNixPackBuilder("unused")
	c.AddBuilder(model.DownloaderNixpacks, nix)
}

// tests pull repo and metadata extraction
// [x] pull repo and branch
// [x] extract metadata
func TestPullRepo(t *testing.T) {
	t.Run("pull repo", func(t *testing.T) {
		if _, err := os.Stat("./tmp"); os.IsNotExist(err) {
			if err := os.Mkdir("./tmp", 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		imageBuildInfo := model.ImageBuildInfo{
			Repo:      "https://github.com/vano2903/testing",
			Branch:    "master",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}
		expectedName := "testing"
		expectedPath := "./tmp/18008-testing-master"
		info, err := c.PullRepo(imageBuildInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		t.Logf("info: %v", info)
		assert.Equal(t, info.RepoName, expectedName)
		assert.Equal(t, info.Path, expectedPath)
	})

	t.Run("pull unexisting repo", func(t *testing.T) {
		if _, err := os.Stat("./tmp"); os.IsNotExist(err) {
			if err := os.Mkdir("./tmp", 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		imageBuildInfo := model.ImageBuildInfo{
			Repo:      "https://github.com/vano2903/unexisting",
			Branch:    "master",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		_, err := c.PullRepo(imageBuildInfo)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("pull unexisting branch", func(t *testing.T) {
		if _, err := os.Stat("./tmp"); os.IsNotExist(err) {
			if err := os.Mkdir("./tmp", 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		imageBuildInfo := model.ImageBuildInfo{
			Repo:      "https://github.com/vano2903/testing",
			Branch:    "unexisting-branch",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		_, err := c.PullRepo(imageBuildInfo)
		if err == nil {
			t.Errorf("expected error, got nil")
		}
	})

	t.Run("extract metadata", func(t *testing.T) {
		if _, err := os.Stat("./tmp"); os.IsNotExist(err) {
			if err := os.Mkdir("./tmp", 0755); err != nil {
				t.Fatal(err)
			}
		}
		setup()
		imageBuildInfo := model.ImageBuildInfo{
			Repo:      "https://github.com/vano2903/ipaas",
			Branch:    "non-relational-version",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}
		expectedMetadata := make(map[string][]string)

		expectedMetadata[github.DefaultBranchMeta] = []string{"non-relational-version"}
		expectedMetadata[github.DescriptionMeta] = []string{"A simple self hosted PaaS for full stack applications and DBaaS"}
		expectedMetadata[github.ReleasesMeta] = []string{"IPAAS - first working version"}
		expectedMetadata[github.TagsMeta] = []string{"v1.0.0"}
		expectedMetadata[github.BranchesMeta] = []string{
			"add-license-1",
			"master",
			"micro-services",
			"non-relational-version",
			"testing",
		}

		info, err := c.PullRepo(imageBuildInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}
		t.Logf("info: %v", info)

		assert.DeepEqual(t, info.Metadata, expectedMetadata)
	})

	t.Cleanup(func() {
		os.RemoveAll("./tmp")
	})
}

// builds the image and returns the image id
// [ ] build image from repo
// [ ] detect build error
func TestBuildImage(t *testing.T) {

	t.Run("build image", func(t *testing.T) {
		if _, err := os.Stat("./tmp"); os.IsNotExist(err) {
			if err := os.Mkdir("./tmp", 0755); err != nil {
				t.Fatal(err)
			}
		}

		setup()
		imageBuildInfo := model.ImageBuildInfo{
			Repo:      "https://github.com/vano2903/testing",
			Branch:    "master",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		info, err := c.PullRepo(imageBuildInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		// info := model.PulledRepoInfo{
		// 	Path: "./tmp/18008-testing-master",
		// }

		imageID, _, err := c.BuildImage(info.Path)
		if err != nil {
			t.Errorf("unable to build image: %v", err)
		}

		t.Logf("imageID: %v", imageID)
	})

	t.Run("detect build error", func(t *testing.T) {
		if _, err := os.Stat("./tmp"); os.IsNotExist(err) {
			if err := os.Mkdir("./tmp", 0755); err != nil {
				t.Fatal(err)
			}
		}

		setup()
		imageBuildInfo := model.ImageBuildInfo{
			Repo:      "https://github.com/vano2903/testing",
			Branch:    "non-working-version",
			UserID:    "18008",
			Connector: model.ConnectorGithub,
		}

		info, err := c.PullRepo(imageBuildInfo)
		if err != nil {
			t.Errorf("unable to pull repo: %v", err)
		}

		// info := model.PulledRepoInfo{
		// 	Path: "./tmp/18008-testing-broken",
		// }

		_, buildError, err := c.BuildImage(info.Path)
		if err == nil {
			t.Errorf("expected error, got nil")
		}

		t.Logf("buildError: %v", buildError)
	})

	t.Cleanup(func() {
		os.RemoveAll("./tmp")
	})
}
