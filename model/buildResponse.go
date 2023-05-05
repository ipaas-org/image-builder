package model

import "github.com/ipaas-org/image-builder/providers/connectors"

type (
	BuildResponse struct {
		UUID         string      `json:"uuid"` // same uuid from the request
		Repo         string      `json:"repo"`
		Status       string      `json:"status"` // success | failed
		ImageID      string      `json:"imageID"`
		ImageName    string      `json:"imageName"`
		LatestCommit string      `json:"latestCommit"`
		Error        *BuildError `json:"error"`
		Metadata     map[connectors.MetaType][]string
	}

	BuildError struct {
		Fault string `json:"fault"` // service | user
		// if user's fault this message will be the reason why the image didnt compile
		//otherwise it will be the service error
		Message string `json:"message"`
	}
)

const (
	ResponseStatusSuccess = "success"
	ResponseStatusFailed  = "failed"

	ResponseErrorFaultService = "service"
	ResponseErrorFaultUser    = "user"
)
