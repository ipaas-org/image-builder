package builders

import "context"

type Builder interface {
	Publish(string) error
	Plan(context.Context, string) (string, error)
	Build(context.Context, string, string) (imageName string, errorMessage string, err error)
}
