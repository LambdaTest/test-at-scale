package core

import (
	"context"
	"sync"
)

// SynapseManager denfines operations for synapse client
type SynapseManager interface {
	// InitiateConnection initiates the connection with LT cloud
	InitiateConnection(ctx context.Context, wg *sync.WaitGroup, connectionFailed chan struct{})
}
