package persistence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore handles basic file operations
type FileStore struct {
	filePath string
	lock     sync.Mutex
}

func NewFileStore(filePath string) *FileStore {
	return &FileStore{filePath: filePath}
}

func (fs *FileStore) Save(data interface{}) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	file, err := os.OpenFile(fs.filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			// Note: Can't return error from defer, just log
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	encoder := json.NewEncoder(file)
	return encoder.Encode(data)
}

func (fs *FileStore) Load(target interface{}) error {
	fs.lock.Lock()
	defer fs.lock.Unlock()

	file, err := os.Open(fs.filePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	decoder := json.NewDecoder(file)
	return decoder.Decode(target)
}

// FileStorage handles telemetry-specific file persistence
type FileStorage struct {
	dataDir string
	mu      sync.Mutex
}

// NewFileStorage creates a new file storage instance
func NewFileStorage(dataDir string) *FileStorage {
	return &FileStorage{
		dataDir: dataDir,
	}
}

// WriteTelemetry writes telemetry data to per-GPU JSONL files
func (fs *FileStorage) WriteTelemetry(telemetry interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Extract GPU ID from telemetry data
	var gpuID string

	// Handle different telemetry types
	switch t := telemetry.(type) {
	case *Telemetry:
		gpuID = t.GPUId
	case Telemetry:
		gpuID = t.GPUId
	default:
		// Try to extract from a map structure
		if telMap, ok := telemetry.(map[string]interface{}); ok {
			if id, exists := telMap["gpu_id"]; exists {
				if idStr, ok := id.(string); ok {
					gpuID = idStr
				}
			}
		}
	}

	if gpuID == "" {
		return fmt.Errorf("cannot determine GPU ID from telemetry data")
	}

	// Ensure data directory exists
	if err := os.MkdirAll(fs.dataDir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create file path
	filePath := filepath.Join(fs.dataDir, fmt.Sprintf("%s.jsonl", gpuID))

	// Open file for appending
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	// Marshal telemetry data to JSON
	jsonData, err := json.Marshal(telemetry)
	if err != nil {
		return fmt.Errorf("failed to marshal telemetry data: %w", err)
	}

	// Write JSON line
	if _, err := file.Write(append(jsonData, '\n')); err != nil {
		return fmt.Errorf("failed to write to file %s: %w", filePath, err)
	}

	return nil
}

// ReadTelemetryFile reads all telemetry data from a specific GPU file
func (fs *FileStorage) ReadTelemetryFile(gpuID string) ([]json.RawMessage, error) {
	filePath := filepath.Join(fs.dataDir, fmt.Sprintf("%s.jsonl", gpuID))

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []json.RawMessage{}, nil // Return empty slice if file doesn't exist
		}
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	var messages []json.RawMessage
	decoder := json.NewDecoder(file)

	for decoder.More() {
		var msg json.RawMessage
		if err := decoder.Decode(&msg); err != nil {
			// Log error but continue reading
			continue
		}
		messages = append(messages, msg)
	}

	return messages, nil
}

// ListGPUFiles returns a list of all GPU IDs that have data files
func (fs *FileStorage) ListGPUFiles() ([]string, error) {
	entries, err := os.ReadDir(fs.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var gpuIDs []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			gpuID := entry.Name()[:len(entry.Name())-6] // Remove .jsonl extension
			gpuIDs = append(gpuIDs, gpuID)
		}
	}

	return gpuIDs, nil
}

// ListGPUFilesWithExtension returns a list of all GPU IDs specific data file name
func (fs *FileStorage) ListGPUFilesWithExtension() ([]string, error) {
	entries, err := os.ReadDir(fs.dataDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read data directory: %w", err)
	}

	var gpuIDFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".jsonl" {
			gpuID := entry.Name()
			gpuIDFiles = append(gpuIDFiles, gpuID)
		}
	}

	return gpuIDFiles, nil
}

// GetAllHosts returns all unique hostnames that have telemetry data
func (fs *FileStorage) GetAllHosts() ([]string, error) {
	gpuIDFiles, err := fs.ListGPUFilesWithExtension()
	if err != nil {
		return nil, err
	}
	hostsMap := make(map[string]bool)
	for _, gpuIDFile := range gpuIDFiles {
		entriesRaw, err := fs.ReadTelemetryFile(gpuIDFile)
		if err != nil {
			return nil, err
		}
		// Convert []json.RawMessage to []Telemetry
		var entries []Telemetry
		for _, entry := range entriesRaw {
			var tel Telemetry
			if err := json.Unmarshal(entry, &tel); err != nil {
				continue
			}
			entries = append(entries, tel)
		}
		for _, entry := range entries {
			if entry.Hostname != "" {
				hostsMap[entry.Hostname] = true
			}
		}
	}

	hosts := make([]string, 0, len(hostsMap))
	for host := range hostsMap {
		hosts = append(hosts, host)
	}
	return hosts, nil
}

// GetGPUsForHost returns all GPU IDs associated with a specific hostname
func (fs *FileStorage) GetGPUsForHost(hostname string) ([]string, error) {
	gpuIDFiles, err := fs.ListGPUFilesWithExtension()
	if err != nil {
		return nil, err
	}
	gpusMap := make(map[string]bool)
	for _, gpuIDFile := range gpuIDFiles {
		entriesRaw, err := fs.ReadTelemetryFile(gpuIDFile)
		if err != nil {
			return nil, err
		}
		// Convert []json.RawMessage to []Telemetry
		var entries []Telemetry
		for _, entry := range entriesRaw {
			var tel Telemetry
			if err := json.Unmarshal(entry, &tel); err != nil {
				continue
			}
			entries = append(entries, tel)
		}
		for _, entry := range entries {
			if entry.Hostname == hostname {
				gpusMap[gpuIDFile[:len(gpuIDFile)-6]] = true
				break // Found this GPU on the host, no need to check more entries
			}
		}
	}

	gpus := make([]string, 0, len(gpusMap))
	for gpu := range gpusMap {
		gpus = append(gpus, gpu)
	}
	return gpus, nil
}

// GetFileStats returns statistics about stored files
func (fs *FileStorage) GetFileStats() (map[string]interface{}, error) {
	gpuIDs, err := fs.ListGPUFiles()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_gpu_files": len(gpuIDs),
		"gpu_ids":         gpuIDs,
		"data_directory":  fs.dataDir,
	}

	// Add file sizes
	fileSizes := make(map[string]int64)
	for _, gpuID := range gpuIDs {
		filePath := filepath.Join(fs.dataDir, fmt.Sprintf("%s.jsonl", gpuID))
		if info, err := os.Stat(filePath); err == nil {
			fileSizes[gpuID] = info.Size()
		}
	}
	stats["file_sizes"] = fileSizes

	return stats, nil
}

// Telemetry represents a typed telemetry data point
type Telemetry struct {
	GPUId     string             `json:"gpu_id"`
	Hostname  string             `json:"hostname"`
	Metrics   map[string]float64 `json:"metrics"`
	Timestamp time.Time          `json:"timestamp"`
}
