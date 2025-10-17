package persistence

import (
	"fmt"
	"sync"
	"time"
)

// Checkpoint represents a processing checkpoint
type Checkpoint struct {
	LastProcessedTime time.Time         `json:"last_processed_time"`
	ProcessedCount    int64             `json:"processed_count"`
	Metadata          map[string]string `json:"metadata,omitempty"`
}

// CheckpointManager manages checkpoint persistence
type CheckpointManager struct {
	fileStore   *FileStore
	memoryStore *MemoryStore
	mu          sync.RWMutex
}

// NewCheckpointManager creates a new checkpoint manager
func NewCheckpointManager(checkpointFile string) *CheckpointManager {
	return &CheckpointManager{
		fileStore:   NewFileStore(checkpointFile),
		memoryStore: NewMemoryStore(),
	}
}

// SaveCheckpoint saves a checkpoint to both memory and file
func (cm *CheckpointManager) SaveCheckpoint(name string, checkpoint *Checkpoint) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Save to memory for fast access
	cm.memoryStore.Save(name, checkpoint)

	// Save to file for persistence
	allCheckpoints := make(map[string]*Checkpoint)

	// Load existing checkpoints
	if err := cm.fileStore.Load(&allCheckpoints); err != nil {
		// If file doesn't exist or is empty, start with empty map
		allCheckpoints = make(map[string]*Checkpoint)
	}

	// Update with new checkpoint
	allCheckpoints[name] = checkpoint

	// Save back to file
	return cm.fileStore.Save(allCheckpoints)
}

// LoadCheckpoint loads a checkpoint, preferring memory cache
func (cm *CheckpointManager) LoadCheckpoint(name string) (*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	// Try memory first
	if data, exists := cm.memoryStore.Load(name); exists {
		if checkpoint, ok := data.(*Checkpoint); ok {
			return checkpoint, nil
		}
	}

	// Load from file
	allCheckpoints := make(map[string]*Checkpoint)
	if err := cm.fileStore.Load(&allCheckpoints); err != nil {
		return nil, fmt.Errorf("failed to load checkpoints from file: %w", err)
	}

	checkpoint, exists := allCheckpoints[name]
	if !exists {
		return nil, fmt.Errorf("checkpoint %s not found", name)
	}

	// Cache in memory for next time
	cm.memoryStore.Save(name, checkpoint)

	return checkpoint, nil
}

// GetAllCheckpoints returns all checkpoints
func (cm *CheckpointManager) GetAllCheckpoints() (map[string]*Checkpoint, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	allCheckpoints := make(map[string]*Checkpoint)
	if err := cm.fileStore.Load(&allCheckpoints); err != nil {
		return make(map[string]*Checkpoint), nil // Return empty map if no file
	}

	return allCheckpoints, nil
}

// DeleteCheckpoint removes a checkpoint
func (cm *CheckpointManager) DeleteCheckpoint(name string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	// Remove from memory
	cm.memoryStore.Delete(name)

	// Remove from file
	allCheckpoints := make(map[string]*Checkpoint)
	if err := cm.fileStore.Load(&allCheckpoints); err != nil {
		return nil // Nothing to delete if file doesn't exist
	}

	delete(allCheckpoints, name)
	return cm.fileStore.Save(allCheckpoints)
}

// UpdateProcessedCount increments the processed count for a checkpoint
func (cm *CheckpointManager) UpdateProcessedCount(name string, increment int64) error {
	checkpoint, err := cm.LoadCheckpoint(name)
	if err != nil {
		// Create new checkpoint if it doesn't exist
		checkpoint = &Checkpoint{
			LastProcessedTime: time.Now(),
			ProcessedCount:    0,
			Metadata:          make(map[string]string),
		}
	}

	checkpoint.ProcessedCount += increment
	checkpoint.LastProcessedTime = time.Now()

	return cm.SaveCheckpoint(name, checkpoint)
}
