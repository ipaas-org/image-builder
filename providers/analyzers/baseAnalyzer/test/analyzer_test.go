package baseAnalyzer

import (
	"context"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/ipaas-org/image-builder/model"
	"github.com/ipaas-org/image-builder/providers/analyzers/baseAnalyzer"
	"github.com/ipaas-org/image-builder/providers/builders/nixpacks"
)

// just a simple pull repo function, all repos must be public
func pullRepo(ctx context.Context, repo, branch, to string) {
	config := &git.CloneOptions{
		URL: repo,
		// Progress: os.Stdout,
		Depth: 1,
		// ReferenceName: plumbing.NewBranchReferenceName(branch),
	}
	if branch != "" {
		config.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}

	_, err := git.PlainCloneContext(ctx, to, false, config)
	if err != nil {
		panic(err)
	}
}

func CreateBaseAnalyzer() *baseAnalyzer.BaseAnalyzer {
	base, err := baseAnalyzer.NewBaseAnalyzer()
	if err != nil {
		panic(err)
	}
	return base
}

func defaultAction(t *testing.T, repo, branch, to string) *model.DetectedInfo {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// test case
	pullRepo(ctx, repo, branch, to)
	base := CreateBaseAnalyzer()
	// test
	detectedInfos, err := base.DetectBuilders(ctx, to)
	if err != nil {
		t.Fatal(err)
	}
	return detectedInfos
}

func TestDetectBuilders(t *testing.T) {
	// repo := "https://github.com/vano2903/testing"
	t.Run("detect builders with no dockerfile", func(t *testing.T) {
		repo := "https://github.com/vano2903/testing"
		to := "./tmp/testing"
		info := defaultAction(t, repo, "", to)

		if len(info.Builders) == 0 {
			t.Fatal("no builders detected")
		}

		if info.Builders[0] != nixpacks.NixPackBuilderKind {
			t.Fatal("expected nixpacks builder")
		}
	})

	t.Run("detect builders with dockerfile", func(t *testing.T) {
		repo := "https://github.com/vano2903/testing"
		branch := "dockerfile"
		to := "./tmp/testing-dockerfile"
		info := defaultAction(t, repo, branch, to)

		if len(info.Builders) != 2 {
			t.Fatal("expected 2 builders, got", info.Builders)
		}
	})

	t.Run("detect builders with multiple dockerfiles", func(t *testing.T) {
		repo := "https://github.com/vano2903/testing"
		branch := "multiple-dockerfiles"
		to := "./tmp/testing-multiple-dockerfiles"
		info := defaultAction(t, repo, branch, to)

		if len(info.Builders) != 2 {
			t.Fatal("expected 2 builders, got", info.Builders)
		}

		if len(info.Docker.Dockerfiles) <= 1 {
			t.Fatal("expected multiple dockerfiles, got", info.Docker.Dockerfiles)
		}
	})

	t.Run("detect builders with dockerignore", func(t *testing.T) {
		repo := "https://github.com/vano2903/testing"
		branch := "dockerignore"
		to := "./tmp/testing-dockerignore"
		info := defaultAction(t, repo, branch, to)

		if info.Docker == nil {
			t.Fatal("expected docker info to be found")
		}

		if !info.Docker.DockerIgnoreFound {
			t.Fatal("expected dockerignore to be found")
		}

		if len(info.Builders) != 1 {
			t.Fatal("expected 1 builder, got", info.Builders)
		}
	})

	t.Run("detect builders with dockerignore but no dockerfile", func(t *testing.T) {
		repo := "https://github.com/vano2903/testing"
		branch := "dockerignore"
		to := "./tmp/testing-no-builders-available"
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		pullRepo(ctx, repo, branch, to)
		if err := os.Remove(to + "/dockerfile"); err != nil {
			t.Fatal(err)
		}

		base := CreateBaseAnalyzer()
		info, err := base.DetectBuilders(ctx, to)
		if err != nil {
			t.Fatal(err)
		}

		if len(info.Builders) != 0 {
			t.Fatal("expected 0 builders, got", info.Builders)
		}

		if info.Docker == nil {
			t.Fatal("expected docker info to be found")
		}

		if !info.Docker.DockerIgnoreFound {
			t.Fatal("expected dockerignore to be found")
		}
	})

	t.Run("detect builders in unable to detect repo", func(t *testing.T) {
		// this repo does not provide any information about the language or
		// any dockerfiles (in the root directory) so we are unable to detect
		// the builders
		repo := "https://github.com/vano2903/abalancer"
		to := "./tmp/abalancer"

		info := defaultAction(t, repo, "", to)

		if len(info.Builders) != 0 {
			t.Fatal("expected 0 builders, got", info.Builders)
		}

		t.Log(info)
	})

	t.Run("whut", func(t *testing.T) {
		repo := "https://github.com/ale-rinaldi/marquee.today"
		to := "./tmp/marquee.today"
		info := defaultAction(t, repo, "", to)

		if len(info.Builders) == 0 {
			t.Fatal("no builders detected")
		}

		t.Logf("%+v", info.NixPacks)
	})

	t.Cleanup(func() {
		if err := os.RemoveAll("./tmp/"); err != nil {
			panic(err)
		}
	})
}
