package providers

import (
	"context"
	"errors"
	"sync"
)

// MockProvider is an in-memory implementation of the Provider interface for testing.
type MockProvider struct {
	instances map[string]*Instance
	mu        sync.Mutex
	idCounter int
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		instances: make(map[string]*Instance),
		idCounter: 1,
	}
}

func (m *MockProvider) CreateVPS(ctx context.Context, spec *VPSSpec) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	id := m.nextID()
	inst := &Instance{
		ID:     id,
		IP:     "192.168.0." + id,
		Region: spec.Region,
		Status: StatusRunning,
	}
	m.instances[id] = inst
	return inst, nil
}

func (m *MockProvider) DeleteVPS(ctx context.Context, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instances[instanceID]; !ok {
		return errors.New("instance not found")
	}
	delete(m.instances, instanceID)
	return nil
}

func (m *MockProvider) GetInstanceStatus(ctx context.Context, instanceID string) (Status, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	inst, ok := m.instances[instanceID]
	if !ok {
		return StatusError, errors.New("instance not found")
	}
	return inst.Status, nil
}

func (m *MockProvider) AttachVolume(ctx context.Context, instanceID, volumeID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.instances[instanceID]; !ok {
		return errors.New("instance not found")
	}
	// No-op for mock
	return nil
}

func (m *MockProvider) nextID() string {
	id := m.idCounter
	m.idCounter++
	return string(rune(48 + id)) // simple string id
}
