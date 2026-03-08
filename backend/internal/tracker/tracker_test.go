package tracker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTracker(t *testing.T) {
	tmpDir := t.TempDir()
	stateFile := filepath.Join(tmpDir, "state.json")

	trk, err := New(stateFile)
	if err != nil {
		t.Fatalf("Expected no error creating tracker, got %v", err)
	}

	imgFile := filepath.Join(tmpDir, "test.jpg")
	content := []byte("dummy image content")
	if err := os.WriteFile(imgFile, content, 0644); err != nil {
		t.Fatalf("Failed to write dummy image: %v", err)
	}

	hash, err := trk.ComputeHash(imgFile)
	if err != nil {
		t.Fatalf("ComputeHash failed: %v", err)
	}

	if trk.IsProcessed("test.jpg", hash) {
		t.Errorf("Expected IsProcessed to be false for new file")
	}

	trk.MarkAsProcessed("test.jpg", hash)

	if !trk.IsProcessed("test.jpg", hash) {
		t.Errorf("Expected IsProcessed to be true after MarkAsProcessed")
	}

	if err := trk.Save(); err != nil {
		t.Fatalf("Failed to save state: %v", err)
	}

	trk2, err := New(stateFile)
	if err != nil {
		t.Fatalf("Expected no error reloading tracker, got %v", err)
	}

	if !trk2.IsProcessed("test.jpg", hash) {
		t.Errorf("Expected IsProcessed to be true in reloaded tracker")
	}

	if err := os.WriteFile(imgFile, []byte("changed dummy image context"), 0644); err != nil {
		t.Fatalf("Failed to modify dummy image: %v", err)
	}

	hash2, err := trk2.ComputeHash(imgFile)
	if err != nil {
		t.Fatalf("ComputeHash failed on modified file: %v", err)
	}

	if hash == hash2 {
		t.Errorf("Expected hash to change for modified file content")
	}

	if trk2.IsProcessed("test.jpg", hash2) {
		t.Errorf("Expected IsProcessed to be false for modified file")
	}
}
