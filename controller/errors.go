package controller

import "errors"

var (
	ErrConnectorNotFound = errors.New("connector not found")
	ErrBuilderNotFound   = errors.New("builder not found")
	ErrEmptyToken        = errors.New("empty token")
	ErrInvalidToken      = errors.New("invalid token")
	ErrMissingRegistry   = errors.New("missing registry")
	ErrInexistingRootDir = errors.New("inexisting root directory")
	ErrNotBuildable      = errors.New("not buildable")
)
