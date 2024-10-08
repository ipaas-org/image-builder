package nixpacks

import (
	"context"
	"os"
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/ipaas-org/image-builder/model"
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

func TestPlan(t *testing.T) {
	t.Run("test plan", func(t *testing.T) {
		repo := "https://github.com/ale-rinaldi/marquee.today"
		to := "./tmp/marquee.today"
		pullRepo(context.Background(), repo, "", to)
		config := &model.BuildConfig{
			RootDirectory: "/",
			NixPkgs:       []string{"npm-9_x"},
		}
		nixbuilder := nixpacks.NewNixPackBuilder("0.0.1")
		plan, err := nixbuilder.Plan(context.Background(), config, to)
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("plan: %+v", plan)

	})

	t.Cleanup(func() {
		os.RemoveAll("./tmp")
	})
}
