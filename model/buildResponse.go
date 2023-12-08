package model

type (
	BuildResponse struct {
		ApplicationID string             `json:"applicationID"`
		Repo          string             `json:"repo"`
		Status        ResponseStatus     `json:"status"` // success | failed
		ImageID       string             `json:"imageID"`
		ImageName     string             `json:"imageName"`
		BuiltCommit   string             `json:"buildCommit"`
		IsError       bool               `json:"isError"`
		Fault         ResponseErrorFault `json:"fault"` // service | user
		Message       string             `json:"message"`
		// Metadata     map[connectors.MetaType][]string
	}
)

type ResponseStatus string
type ResponseErrorFault string

const (
	ResponseStatusSuccess ResponseStatus = "success"
	ResponseStatusFailed  ResponseStatus = "failed"

	ResponseErrorFaultService ResponseErrorFault = "service"
	ResponseErrorFaultUser    ResponseErrorFault = "user"
)
