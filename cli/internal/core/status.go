package core

type InfrastructureState string

const (
	InfrastructureStateUnknown  InfrastructureState = "unknown"
	InfrastructureStateCreating InfrastructureState = "creating"
	InfrastructureStateRunning  InfrastructureState = "running"
	InfrastructureStateStopped  InfrastructureState = "stopped"
	InfrastructureStateDeleting InfrastructureState = "deleting"
	InfrastructureStateNotFound InfrastructureState = "not_found"
	InfrastructureStateError    InfrastructureState = "error"
)

type InfrastructureStatus struct {
	ID       string
	Name     string
	PublicIP string
	State    InfrastructureState
	Detail   string
}

func (s InfrastructureStatus) HasError() bool {
	return s.State == InfrastructureStateError
}
