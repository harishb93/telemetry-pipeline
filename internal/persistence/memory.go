package persistence

import (
	"sync"
	"time"
)

// MemoryStore handles basic key-value operations
type MemoryStore struct {
	data map[string]interface{}
	lock sync.RWMutex
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]interface{}),
	}
}

func (ms *MemoryStore) Save(key string, value interface{}) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	ms.data[key] = value
}

func (ms *MemoryStore) Load(key string) (interface{}, bool) {
	ms.lock.RLock()
	defer ms.lock.RUnlock()
	value, exists := ms.data[key]
	return value, exists
}

func (ms *MemoryStore) Delete(key string) {
	ms.lock.Lock()
	defer ms.lock.Unlock()
	delete(ms.data, key)
}

// MemoryStorage handles telemetry-specific memory persistence
type MemoryStorage struct {
	data       map[string][]Telemetry // GPU ID -> telemetry entries
	maxEntries int
	mu         sync.RWMutex
}

// NewMemoryStorage creates a new memory storage instance
func NewMemoryStorage(maxEntriesPerGPU int) *MemoryStorage {
	return &MemoryStorage{
		data:       make(map[string][]Telemetry),
		maxEntries: maxEntriesPerGPU,
	}
}

// StoreTelemetry stores telemetry data in memory with LRU eviction
func (ms *MemoryStorage) StoreTelemetry(telemetry Telemetry) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	gpuID := telemetry.GPUId
	entries := ms.data[gpuID]

	// Add new entry
	entries = append(entries, telemetry)

	// Implement LRU eviction if needed
	if len(entries) > ms.maxEntries {
		// Remove oldest entries
		entries = entries[len(entries)-ms.maxEntries:]
	}

	ms.data[gpuID] = entries
}

// GetTelemetryForGPU returns all telemetry data for a specific GPU
func (ms *MemoryStorage) GetTelemetryForGPU(gpuID string) []Telemetry {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entries, exists := ms.data[gpuID]
	if !exists {
		return []Telemetry{}
	}

	// Return a copy to avoid concurrent modification
	result := make([]Telemetry, len(entries))
	copy(result, entries)
	return result
}

// GetAllGPUIDs returns all GPU IDs that have data
func (ms *MemoryStorage) GetAllGPUIDs() []string {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	gpuIDs := make([]string, 0, len(ms.data))
	for gpuID := range ms.data {
		gpuIDs = append(gpuIDs, gpuID)
	}
	return gpuIDs
}

// GetLatestTelemetryForGPU returns the most recent telemetry entry for a GPU
func (ms *MemoryStorage) GetLatestTelemetryForGPU(gpuID string) (*Telemetry, bool) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	entries, exists := ms.data[gpuID]
	if !exists || len(entries) == 0 {
		return nil, false
	}

	latest := entries[len(entries)-1]
	return &latest, true
}

// GetStats returns memory storage statistics
func (ms *MemoryStorage) GetStats() map[string]interface{} {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	totalEntries := 0
	gpuCounts := make(map[string]int)

	for gpuID, entries := range ms.data {
		count := len(entries)
		totalEntries += count
		gpuCounts[gpuID] = count
	}

	return map[string]interface{}{
		"total_entries":       totalEntries,
		"total_gpus":          len(ms.data),
		"max_entries_per_gpu": ms.maxEntries,
		"gpu_entry_counts":    gpuCounts,
	}
}

// ClearGPUData removes all data for a specific GPU
func (ms *MemoryStorage) ClearGPUData(gpuID string) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	delete(ms.data, gpuID)
}

// ClearOldEntries removes entries older than the specified duration
func (ms *MemoryStorage) ClearOldEntries(olderThan time.Duration) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)

	for gpuID, entries := range ms.data {
		var newEntries []Telemetry
		for _, entry := range entries {
			if entry.Timestamp.After(cutoff) {
				newEntries = append(newEntries, entry)
			}
		}
		ms.data[gpuID] = newEntries
	}
}
