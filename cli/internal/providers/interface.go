package providers

import "context"

type Status string

const (
	StatusRunning      Status = "Running"
	StatusStopped      Status = "Stopped"
	StatusProvisioning Status = "Provisioning"
	StatusError        Status = "Error"
)

type VPSSpec struct {
	Name            string
	Region          string
	Plan            string
	CloudInitScript string
}

type Instance struct {
	ID     string
	IP     string
	Region string
	Status Status
}

type Provider interface {
	CreateVPS(ctx context.Context, spec *VPSSpec) (*Instance, error)
	DeleteVPS(ctx context.Context, instanceID string) error
	GetInstanceStatus(ctx context.Context, instanceID string) (Status, error)
	AttachVolume(ctx context.Context, instanceID, volumeID string) error
}

// NewProvider returns a Provider implementation based on the name.
// This is a stub for orchestration integration.
func NewProvider(name string) Provider {
	if name == "mock" {
		return NewMockProvider()
	}
	// TODO: Add real providers (Hetzner, AWS, etc.)
	return nil
}
