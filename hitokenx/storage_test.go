package hitokenx

import (
	"testing"

	"github.com/xiehqing/hitoken/storage/memory"
)

func TestNewStorageDefaultsToMemory(t *testing.T) {
	storage, err := newStorage("", "", "")
	if err != nil {
		t.Fatalf("newStorage returned error: %v", err)
	}
	if _, ok := storage.(*memory.Storage); !ok {
		t.Fatalf("expected memory storage, got %T", storage)
	}
}

func TestNewStorageUsesLegacyTypeWhenSplitTypeIsEmpty(t *testing.T) {
	storage, err := newStorage("", "", `{"type":"memory"}`)
	if err != nil {
		t.Fatalf("newStorage returned error: %v", err)
	}
	if _, ok := storage.(*memory.Storage); !ok {
		t.Fatalf("expected memory storage, got %T", storage)
	}
}
