package registry

import "context"

type Registryer interface {
	TagImage(ctx context.Context, localImageID, userCode, appName string) (string, error)
	PushImage(ctx context.Context, localImageID string) error
}
