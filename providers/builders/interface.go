package builders

import (
	"context"
	"fmt"

	"github.com/ipaas-org/image-builder/model"
)

type Plan string

type Builder interface {
	Plan(ctx context.Context, config *model.BuildConfig, path string) (plan Plan, err error)
	Build(ctx context.Context, userID, repo, path string, plan Plan) (imageName string, imageOutput []byte, err error)
}

var (
	ErrMissingConfig    = fmt.Errorf("missing config")
	ErrInvalidConfig    = fmt.Errorf("invalid config")
	ErrInvalidPlan      = fmt.Errorf("invalid plan")
	ErrImageNotCompiled = fmt.Errorf("image not compiled")
)
