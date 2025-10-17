package persistence

import (
	"os"
	"reflect"
	"testing"
)

func TestFileStore(t *testing.T) {
	filePath := "test_data.json"
	defer os.Remove(filePath)

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
