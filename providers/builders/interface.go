package builders

import "context"

type Builder interface {
	Plan(ctx context.Context, path string) (plan string, err error)
	Build(ctx context.Context, userID, repo, config, path string) (imageName string, errorMessage string, err error)
}
