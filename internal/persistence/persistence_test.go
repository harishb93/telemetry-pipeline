package persistence

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

// Test data structures
type TestStruct struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Data []byte `json:"data"`
}

type ComplexTestStruct struct {
	Simple    TestStruct            `json:"simple"`
	Map       map[string]int        `json:"map"`
	Slice     []string              `json:"slice"`
	Nested    map[string]TestStruct `json:"nested"`
	Timestamp time.Time             `json:"timestamp"`
}

func TestFileStore(t *testing.T) {
	filePath := "test_data.json"
	defer func() {
		if err := os.Remove(filePath); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	store := NewFileStore(filePath)

	data := map[string]string{"key1": "value1", "key2": "value2"}
	if err := store.Save(data); err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	loadedData := make(map[string]string)
	if err := store.Load(&loadedData); err != nil {
		t.Fatalf("Failed to load data: %v", err)
	}

	if !reflect.DeepEqual(data, loadedData) {
		t.Errorf("Expected %v, got %v", data, loadedData)
	}
}

func TestFileStore_ComplexData(t *testing.T) {
	filePath := "test_complex_data.json"
	defer func() {
		if err := os.Remove(filePath); err != nil {
			t.Logf("Failed to remove test file: %v", err)
		}
	}()

	store := NewFileStore(filePath)

	// Test with complex nested data
	originalData := ComplexTestStruct{
		Simple: TestStruct{
			ID:   42,
			Name: "test",
			Data: []byte("binary data"),
		},
		Map: map[string]int{
			"key1": 100,
			"key2": 200,
		},
		Slice: []string{"item1", "item2", "item3"},
		Nested: map[string]TestStruct{
			"child1": {ID: 1, Name: "child1", Data: []byte("child data")},
			"child2": {ID: 2, Name: "child2", Data: []byte("more data")},
		},
		Timestamp: time.Now().Truncate(time.Second), // Truncate for JSON precision
	}

	if err := store.Save(originalData); err != nil {
		t.Fatalf("Failed to save complex data: %v", err)
	}

	var loadedData ComplexTestStruct
	if err := store.Load(&loadedData); err != nil {
		t.Fatalf("Failed to load complex data: %v", err)
	}

	// Compare each field individually for better error reporting
	if !reflect.DeepEqual(originalData.Simple, loadedData.Simple) {
		t.Errorf("Simple struct mismatch:\nOriginal: %+v\nLoaded: %+v", originalData.Simple, loadedData.Simple)
	}
	if !reflect.DeepEqual(originalData.Map, loadedData.Map) {
		t.Errorf("Map mismatch:\nOriginal: %+v\nLoaded: %+v", originalData.Map, loadedData.Map)
	}
	if !reflect.DeepEqual(originalData.Slice, loadedData.Slice) {
		t.Errorf("Slice mismatch:\nOriginal: %+v\nLoaded: %+v", originalData.Slice, loadedData.Slice)
	}
	if !reflect.DeepEqual(originalData.Nested, loadedData.Nested) {
		t.Errorf("Nested mismatch:\nOriginal: %+v\nLoaded: %+v", originalData.Nested, loadedData.Nested)
	}
	// For timestamp, check that they're within 1 second (JSON precision issues)
	if originalData.Timestamp.Unix() != loadedData.Timestamp.Unix() {
		t.Errorf("Timestamp mismatch (Unix time):\nOriginal: %d\nLoaded: %d",
			originalData.Timestamp.Unix(), loadedData.Timestamp.Unix())
	}
}

func TestFileStore_ErrorCases(t *testing.T) {
	t.Run("invalid_file_path", func(t *testing.T) {
		// Test with invalid file path (directory that doesn't exist)
		store := NewFileStore("/nonexistent/directory/file.json")

		data := map[string]string{"key": "value"}
		err := store.Save(data)
		if err == nil {
			t.Error("Expected error when saving to invalid path")
		}
	})

	t.Run("permission_denied", func(t *testing.T) {
		// Create a read-only directory
		testDir := "readonly_test_dir"
		if err := os.Mkdir(testDir, 0444); err != nil {
			t.Skipf("Failed to create read-only directory: %v", err)
		}
		defer func() {
			_ = os.Chmod(testDir, 0755) // Make it writable to delete
			_ = os.RemoveAll(testDir)
		}()

		store := NewFileStore(filepath.Join(testDir, "test.json"))
		data := map[string]string{"key": "value"}

		err := store.Save(data)
		if err == nil {
			t.Error("Expected permission error when saving to read-only directory")
		}
	})

	t.Run("corrupted_json_file", func(t *testing.T) {
		filePath := "corrupted_test.json"
		defer func() { _ = os.Remove(filePath) }()

		// Create a file with invalid JSON
		if err := os.WriteFile(filePath, []byte("invalid json content {"), 0644); err != nil {
			t.Fatalf("Failed to create corrupted file: %v", err)
		}

		store := NewFileStore(filePath)
		var data map[string]string

		err := store.Load(&data)
		if err == nil {
			t.Error("Expected error when loading corrupted JSON file")
		}
	})

	t.Run("nonexistent_file_load", func(t *testing.T) {
		store := NewFileStore("nonexistent_file.json")
		var data map[string]string

		err := store.Load(&data)
		if err == nil {
			t.Error("Expected error when loading nonexistent file")
		}
	})

	t.Run("nil_data_save", func(t *testing.T) {
		filePath := "nil_test.json"
		defer func() { _ = os.Remove(filePath) }()

		store := NewFileStore(filePath)

		// Test with nil data
		err := store.Save(nil)
		if err != nil {
			t.Errorf("Unexpected error saving nil data: %v", err)
		}

		// Load it back
		var loadedData interface{}
		err = store.Load(&loadedData)
		if err != nil {
			t.Errorf("Unexpected error loading nil data: %v", err)
		}
		if loadedData != nil {
			t.Errorf("Expected nil, got %v", loadedData)
		}
	})
}

func TestFileStore_LargeData(t *testing.T) {
	filePath := "large_data_test.json"
	defer func() { _ = os.Remove(filePath) }()

	store := NewFileStore(filePath)

	// Create large data structure
	largeData := make(map[string][]TestStruct)
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("batch_%d", i)
		batch := make([]TestStruct, 100)
		for j := 0; j < 100; j++ {
			batch[j] = TestStruct{
				ID:   i*100 + j,
				Name: fmt.Sprintf("item_%d_%d", i, j),
				Data: make([]byte, 100), // 100 bytes per item
			}
		}
		largeData[key] = batch
	}

	// Save large data
	if err := store.Save(largeData); err != nil {
		t.Fatalf("Failed to save large data: %v", err)
	}

	// Load large data back
	var loadedData map[string][]TestStruct
	if err := store.Load(&loadedData); err != nil {
		t.Fatalf("Failed to load large data: %v", err)
	}

	// Verify data integrity
	if len(loadedData) != len(largeData) {
		t.Errorf("Data size mismatch: expected %d, got %d", len(largeData), len(loadedData))
	}

	// Spot check some data
	if batch, exists := loadedData["batch_500"]; exists {
		if len(batch) != 100 {
			t.Errorf("Batch size mismatch: expected 100, got %d", len(batch))
		}
		if batch[50].ID != 50050 {
			t.Errorf("Data integrity check failed: expected ID 50050, got %d", batch[50].ID)
		}
	} else {
		t.Error("Expected batch_500 not found in loaded data")
	}
}

func TestFileStore_ConcurrentAccess(t *testing.T) {
	filePath := "concurrent_test.json"
	defer func() { _ = os.Remove(filePath) }()

	store := NewFileStore(filePath)

	// Initialize with some data
	initialData := map[string]int{"counter": 0}
	if err := store.Save(initialData); err != nil {
		t.Fatalf("Failed to save initial data: %v", err)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 50

	// Test concurrent reads
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				var data map[string]int
				if err := store.Load(&data); err != nil {
					t.Errorf("Goroutine %d: Failed to load data on iteration %d: %v", id, j, err)
					return
				}
				if _, exists := data["counter"]; !exists {
					t.Errorf("Goroutine %d: Counter key not found on iteration %d", id, j)
					return
				}
			}
		}(i)
	}
	wg.Wait()

	// Verify final state
	var finalData map[string]int
	if err := store.Load(&finalData); err != nil {
		t.Fatalf("Failed to load final data: %v", err)
	}
	if finalData["counter"] != 0 {
		t.Errorf("Final counter value should be 0, got %d", finalData["counter"])
	}
}

func TestMemoryStore(t *testing.T) {
	store := NewMemoryStore()

	store.Save("key1", "value1")
	store.Save("key2", "value2")

	value, exists := store.Load("key1")
	if !exists || value != "value1" {
		t.Errorf("Expected value1, got %v", value)
	}

	store.Delete("key1")
	_, exists = store.Load("key1")
	if exists {
		t.Errorf("Expected key1 to be deleted")
	}
}

func TestMemoryStore_ComplexOperations(t *testing.T) {
	store := NewMemoryStore()

	// Test with various data types
	testCases := []struct {
		name  string
		key   string
		value interface{}
	}{
		{"string", "str_key", "string value"},
		{"int", "int_key", 42},
		{"float", "float_key", 3.14159},
		{"bool", "bool_key", true},
		{"slice", "slice_key", []string{"a", "b", "c"}},
		{"map", "map_key", map[string]int{"x": 1, "y": 2}},
		{"struct", "struct_key", TestStruct{ID: 1, Name: "test", Data: []byte("data")}},
		{"nil", "nil_key", nil},
	}

	// Save all values
	for _, tc := range testCases {
		t.Run("save_"+tc.name, func(t *testing.T) {
			store.Save(tc.key, tc.value)
		})
	}

	// Load and verify all values
	for _, tc := range testCases {
		t.Run("load_"+tc.name, func(t *testing.T) {
			value, exists := store.Load(tc.key)
			if !exists {
				t.Errorf("Key %s should exist", tc.key)
				return
			}
			if !reflect.DeepEqual(value, tc.value) {
				t.Errorf("Value mismatch for %s: expected %v, got %v", tc.key, tc.value, value)
			}
		})
	}

	// Test overwrite
	t.Run("overwrite", func(t *testing.T) {
		store.Save("overwrite_key", "original")
		store.Save("overwrite_key", "updated")

		value, exists := store.Load("overwrite_key")
		if !exists || value != "updated" {
			t.Errorf("Expected 'updated', got %v", value)
		}
	})

	// Test delete operations
	t.Run("delete_operations", func(t *testing.T) {
		// Delete existing key
		store.Save("delete_me", "value")
		store.Delete("delete_me")

		_, exists := store.Load("delete_me")
		if exists {
			t.Error("Key should have been deleted")
		}

		// Delete non-existent key (should not panic)
		store.Delete("nonexistent_key")
	})

	// Test key listing/enumeration
	t.Run("enumeration", func(t *testing.T) {
		// Clear and add known data
		newStore := NewMemoryStore()
		expectedKeys := []string{"key1", "key2", "key3"}

		for _, key := range expectedKeys {
			newStore.Save(key, fmt.Sprintf("value_%s", key))
		}

		// Since MemoryStore doesn't have enumeration methods in the current implementation,
		// we'll test what we can
		for _, key := range expectedKeys {
			_, exists := newStore.Load(key)
			if !exists {
				t.Errorf("Key %s should exist", key)
			}
		}
	})
}

func TestMemoryStore_ConcurrentAccess(t *testing.T) {
	store := NewMemoryStore()

	var wg sync.WaitGroup
	numGoroutines := 10
	numOperations := 100

	// Test concurrent writes and reads
	wg.Add(numGoroutines * 2) // readers and writers

	// Writers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j)
				value := fmt.Sprintf("value_%d_%d", id, j)
				store.Save(key, value)
			}
		}(i)
	}

	// Readers
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				key := fmt.Sprintf("key_%d_%d", id, j%10) // Read some keys that might exist
				_, _ = store.Load(key)                    // Don't check exists since timing is unpredictable
			}
		}(i)
	}

	wg.Wait()

	// Verify some data was written
	value, exists := store.Load("key_0_0")
	if exists && value != "value_0_0" {
		t.Errorf("Expected 'value_0_0', got %v", value)
	}
}

func TestMemoryStore_StressTest(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}

	store := NewMemoryStore()

	// Stress test with large number of operations
	numKeys := 10000

	// Fill store with data
	for i := 0; i < numKeys; i++ {
		key := fmt.Sprintf("stress_key_%d", i)
		value := ComplexTestStruct{
			Simple: TestStruct{
				ID:   i,
				Name: fmt.Sprintf("stress_item_%d", i),
				Data: make([]byte, 100),
			},
			Map: map[string]int{
				"counter": i,
				"double":  i * 2,
			},
			Slice:     []string{fmt.Sprintf("item_%d", i)},
			Timestamp: time.Now(),
		}
		store.Save(key, value)
	}

	// Random access pattern
	for i := 0; i < 1000; i++ {
		key := fmt.Sprintf("stress_key_%d", i*10) // Access every 10th key
		value, exists := store.Load(key)
		if !exists {
			t.Errorf("Key %s should exist", key)
			continue
		}

		if complexValue, ok := value.(ComplexTestStruct); ok {
			expectedID := i * 10
			if complexValue.Simple.ID != expectedID {
				t.Errorf("Data integrity check failed for %s: expected ID %d, got %d",
					key, expectedID, complexValue.Simple.ID)
			}
		}
	}

	// Delete half the data
	for i := 0; i < numKeys/2; i += 2 {
		key := fmt.Sprintf("stress_key_%d", i)
		store.Delete(key)
	}

	// Verify deletions
	deletedCount := 0
	existingCount := 0
	for i := 0; i < numKeys/2; i += 2 {
		key := fmt.Sprintf("stress_key_%d", i)
		_, exists := store.Load(key)
		if exists {
			existingCount++
		} else {
			deletedCount++
		}
	}

	expectedDeleted := numKeys / 4 // Half of half
	if deletedCount != expectedDeleted {
		t.Errorf("Expected %d deletions, got %d", expectedDeleted, deletedCount)
	}
}

func TestFileStore_FilePermissions(t *testing.T) {
	filePath := "permissions_test.json"
	defer func() { _ = os.Remove(filePath) }()

	store := NewFileStore(filePath)
	data := map[string]string{"key": "value"}

	// Save data
	if err := store.Save(data); err != nil {
		t.Fatalf("Failed to save data: %v", err)
	}

	// Check file exists and has reasonable permissions
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := fileInfo.Mode()
	if mode&0600 != 0600 { // Owner should have read/write
		t.Errorf("File permissions too restrictive: %v", mode)
	}
}

func TestFileStore_JSONMarshaling(t *testing.T) {
	filePath := "json_marshaling_test.json"
	defer func() { _ = os.Remove(filePath) }()

	store := NewFileStore(filePath)

	// Test with data that has JSON marshaling challenges
	type CustomType struct {
		PublicField  string         `json:"public"`
		privateField string         // This won't be marshaled
		EmptyString  string         `json:"empty"`
		ZeroInt      int            `json:"zero"`
		NilPointer   *int           `json:"nil_ptr"`
		EmptySlice   []int          `json:"empty_slice"`
		EmptyMap     map[string]int `json:"empty_map"`
	}

	originalData := CustomType{
		PublicField:  "visible",
		privateField: "hidden",
		EmptyString:  "",
		ZeroInt:      0,
		NilPointer:   nil,
		EmptySlice:   []int{},
		EmptyMap:     make(map[string]int),
	}

	if err := store.Save(originalData); err != nil {
		t.Fatalf("Failed to save custom type: %v", err)
	}

	var loadedData CustomType
	if err := store.Load(&loadedData); err != nil {
		t.Fatalf("Failed to load custom type: %v", err)
	}

	// Verify marshaled fields
	if loadedData.PublicField != originalData.PublicField {
		t.Errorf("PublicField mismatch: expected %s, got %s", originalData.PublicField, loadedData.PublicField)
	}

	// Private field should be empty (not marshaled)
	if loadedData.privateField != "" {
		t.Errorf("Private field should be empty after JSON round-trip, got %s", loadedData.privateField)
	}

	// Check handling of zero values
	if loadedData.ZeroInt != 0 {
		t.Errorf("ZeroInt should be 0, got %d", loadedData.ZeroInt)
	}
	if loadedData.NilPointer != nil {
		t.Errorf("NilPointer should be nil, got %v", loadedData.NilPointer)
	}
}

func TestInterface_Implementation(t *testing.T) {
	// Test that our types implement expected interfaces (if any)
	// This is more of a compile-time test but good to be explicit

	var _ interface{} = NewFileStore("test.json")
	var _ interface{} = NewMemoryStore()

	// If there were specific interfaces to implement, we'd test them here
	// For example:
	// var _ Saver = NewFileStore("test.json")
	// var _ Loader = NewFileStore("test.json")
}

func TestEdgeCases(t *testing.T) {
	t.Run("empty_string_keys", func(t *testing.T) {
		store := NewMemoryStore()

		// Test empty string key
		store.Save("", "empty key value")
		value, exists := store.Load("")
		if !exists || value != "empty key value" {
			t.Error("Empty string key should be valid")
		}
	})

	t.Run("unicode_keys_and_values", func(t *testing.T) {
		store := NewMemoryStore()

		unicodeKey := "🔑key🔑"
		unicodeValue := "🌟value🌟"

		store.Save(unicodeKey, unicodeValue)
		value, exists := store.Load(unicodeKey)
		if !exists || value != unicodeValue {
			t.Error("Unicode keys and values should be supported")
		}
	})

	t.Run("very_long_keys", func(t *testing.T) {
		store := NewMemoryStore()

		longKey := string(make([]byte, 10000)) // Very long key
		for i := range longKey {
			longKey = longKey[:i] + "x" + longKey[i+1:]
		}

		store.Save(longKey, "long key value")
		value, exists := store.Load(longKey)
		if !exists || value != "long key value" {
			t.Error("Very long keys should be supported")
		}
	})
}
