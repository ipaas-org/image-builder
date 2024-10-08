package connectors

import (
	"context"

	"github.com/ipaas-org/image-builder/model"
)

type MetaType string

type Connector interface {
	// commit specify the commit hash you want to pull, if you want to pull the latest set the value to "latest"|""
	Pull(ctx context.Context, userID, branch, url, commitHash, token string) (*model.PulledRepoInfo, error)
	GetUserAndRepo(ctx context.Context, url, token string) (username string, repoName string, err error)
	ValidateAndLintUrl(ctx context.Context, url, token string) (linted string, err error)
	// GetMetadata(url, token string, meta ...MetaType) (metadata map[MetaType][]string, err error)
}
