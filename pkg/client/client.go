package client

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
)

// StreamClient defines the standard lifecycle interface across all stream protocols.
type StreamClient interface {
	Connect(ctx context.Context) error
	Send(payload []byte, opcode OpCode) error
	Close() error
	Stats() ConnectionStats
}

// Manager maintains active stream sessions.
type Manager struct {
	mu          sync.RWMutex
	connections map[string]StreamClient
	configs     map[string]ConnectionConfig
}

// NewManager creates a connection manager.
func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]StreamClient),
		configs:     make(map[string]ConnectionConfig),
	}
}

// DetectProtocol infers protocol type from URL scheme and explicit config.
func DetectProtocol(rawURL string, explicit Protocol) Protocol {
	if explicit != "" {
		return explicit
	}

	lower := strings.ToLower(rawURL)
	if strings.HasPrefix(lower, "ws://") {
		return ProtocolWS
	}
	if strings.HasPrefix(lower, "wss://") {
		return ProtocolWSS
	}
	if strings.Contains(lower, "socket.io") {
		return ProtocolSocketIO
	}
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		return ProtocolSSE
	}

	return ProtocolWS
}

// CreateClient initializes the appropriate protocol client for a configuration.
func CreateClient(cfg ConnectionConfig, onFrame FrameHandler, onState StateHandler) (StreamClient, error) {
	proto := DetectProtocol(cfg.URL, cfg.Protocol)
	cfg.Protocol = proto

	switch proto {
	case ProtocolWS, ProtocolWSS:
		return NewWSClient(cfg, onFrame, onState), nil
	case ProtocolSSE:
		return NewSSEClient(cfg, onFrame, onState), nil
	case ProtocolSocketIO:
		return NewSocketIOClient(cfg, "/", onFrame, onState), nil
	default:
		return nil, fmt.Errorf("unsupported streaming protocol: %s", proto)
	}
}

// Register registers and stores an active client.
func (m *Manager) Register(id string, client StreamClient, cfg ConnectionConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[id] = client
	m.configs[id] = cfg
}

// Get retrieves an active client by ID.
func (m *Manager) Get(id string) (StreamClient, ConnectionConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	client, ok := m.connections[id]
	cfg := m.configs[id]
	return client, cfg, ok
}

// List returns all active connection IDs and configurations.
func (m *Manager) List() map[string]ConnectionConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	list := make(map[string]ConnectionConfig, len(m.configs))
	for k, v := range m.configs {
		list[k] = v
	}
	return list
}

// Close terminates and unregisters a specific client.
func (m *Manager) Close(id string) error {
	m.mu.Lock()
	client, ok := m.connections[id]
	if ok {
		delete(m.connections, id)
		delete(m.configs, id)
	}
	m.mu.Unlock()

	if !ok {
		return errors.New("connection not found")
	}

	return client.Close()
}

// CloseAll terminates all managed connections.
func (m *Manager) CloseAll() {
	m.mu.Lock()
	clients := make([]StreamClient, 0, len(m.connections))
	for _, c := range m.connections {
		clients = append(clients, c)
	}
	m.connections = make(map[string]StreamClient)
	m.configs = make(map[string]ConnectionConfig)
	m.mu.Unlock()

	for _, c := range clients {
		_ = c.Close()
	}
}
