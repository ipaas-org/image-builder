package model

type ApplicationState string

const (
	ApplicationStateBuilding ApplicationState = "building"
	ApplicationStateFailed   ApplicationState = "failed"
)
