package downloader

import (
	"log"
	"os"
	"testing"

	"github.com/ipaas-org/image-builder/pkg/logger"
	"github.com/ipaas-org/image-builder/providers/connectors/github"
	"github.com/joho/godotenv"
)

const (
	userAgent   = "ipaas-image-builder-test"
	downloadTmp = "./tmpConnector"
)

var token string

func NewGithubConnector() *github.GithubConnector {
	if err := godotenv.Load(); err != nil {
		log.Fatal("err loading .env:", err.Error())
	}
	var found bool
	token, found = os.LookupEnv("GITHUB_TEST_TOKEN")
	if !found {
		log.Fatal("GITHUB_TEST_TOKEN is not set")
	}
	return github.NewGithubConnector(downloadTmp, userAgent, logger.NewLogger("debug", "text"))
}

func TestPullRepo(t *testing.T) {
	t.Run("pull repo", func(t *testing.T) {
		//create downloadTmp if not exists
		if _, err := os.Stat(downloadTmp); os.IsNotExist(err) {
			if err := os.Mkdir(downloadTmp, 0755); err != nil {
				t.Fatal(err)
			}
		}
		g := NewGithubConnector()
		path, _, _, err := g.Pull("18008", "", "vano2903/testing", token)
		if err != nil {
			t.Fatal(err)
		}
		t.Log(path)
	})

	t.Run("pull repo with non default branch", func(t *testing.T) {
		if _, err := os.Stat(downloadTmp); os.IsNotExist(err) {
			if err := os.Mkdir(downloadTmp, 0755); err != nil {
				t.Fatal(err)
			}
		}
		g := NewGithubConnector()
		_, _, _, err := g.Pull("18008", "env-with-db-connection", "vano2903/testing", token)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("pull unexisting repo", func(t *testing.T) {
		if _, err := os.Stat(downloadTmp); os.IsNotExist(err) {
			if err := os.Mkdir(downloadTmp, 0755); err != nil {
				t.Fatal(err)
			}
		}

		g := NewGithubConnector()
		_, _, _, err := g.Pull("18008", "", "vano2903/unexisting", token)
		if err == nil {
			t.Fatal("should have returned an error")
		}
	})

	t.Run("pull repo with unexisting branch", func(t *testing.T) {
		if _, err := os.Stat(downloadTmp); os.IsNotExist(err) {
			if err := os.Mkdir(downloadTmp, 0755); err != nil {
				t.Fatal(err)
			}
		}
		g := NewGithubConnector()
		_, _, _, err := g.Pull("18008", "unexisting-branch", "vano2903/testing", token)
		if err == nil {
			t.Fatal(err)
		}
	})

	t.Run("pull with invalid token", func(t *testing.T) {
		if _, err := os.Stat(downloadTmp); os.IsNotExist(err) {
			if err := os.Mkdir(downloadTmp, 0755); err != nil {
				t.Fatal(err)
			}
		}
		g := NewGithubConnector()
		_, _, _, err := g.Pull("18008", "", "vano2903/testing", "invalid-token")
		if err == nil {
			t.Fatal("should have returned an error")
		}
	})

	t.Cleanup(func() {
		//delete the tmp directory
		os.RemoveAll(downloadTmp)
	})
}

func TestValidUrl(t *testing.T) {
	t.Run("valid url", func(t *testing.T) {
		validUrls := []string{
			"vano2903/ipaas",
		}

		invalidUrls := []string{
			"/vano2903/ipaas",
			"github.com//vano2903/ipaas",
			"https://github.com/vano2903/ipaas",
			"http://github.com/vano2903/ipaas",
			"github.com/vano2903/ipaas",
			"https://github.it/vano2903/ipaas",
			"http://github.it/vano2903/ipaas",
			"github/vano2903/ipaas",
			"github.com/",
			"github.com",
			"https://github.com/",
			"https://github.com",
			"/github.com/vano2903/ipaas",
			"/ipaas",
			"vano2903/",
			"https://github.com/user-name",
			"https://github.com/user.name",
			"https://github.com//repo-name",
			"https://github.com/user-name/",
			"https://github.com/vano2903/inexistent-repo",
			"https://github.com/user-name/repo_name/subdir",
		}

		g := NewGithubConnector()

		// wg := sync.WaitGroup{}
		for _, url := range validUrls {
			// wg.Add(1)
			// go func(wg *sync.WaitGroup, url string) {
			// 	defer wg.Done()
			_, err := g.ValidateAndLintUrl(url, token)
			if err != nil {
				t.Errorf("url %s should be valid but was recognized as invalid: %s", url, err.Error())
			}
			// }(&wg, url)
		}

		for _, url := range invalidUrls {
			// wg.Add(1)
			// go func(wg *sync.WaitGroup, url string) {
			// defer wg.Done()
			_, err := g.ValidateAndLintUrl(url, token)
			if err == nil {
				t.Errorf("url %s should be invalid but was recognized as valid", url)
			}
			// }(&wg, url)
		}
		// wg.Wait()
	})
}
