package analyzers

import (
	"context"

	"github.com/ipaas-org/image-builder/model"
)

type Analyzer interface {
	// returns builders that can be used on the specified path
	DetectBuilders(ctx context.Context, path string) (*model.DetectedInfo, error)
}
