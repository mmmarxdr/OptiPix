package tracker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type State struct {
	Files map[string]string `json:"files"`
}

type Tracker struct {
	stateFile string
	state     *State
	mu        sync.RWMutex
}

func New(stateFile string) (*Tracker, error) {
	t := &Tracker{
		stateFile: stateFile,
		state: &State{
			Files: make(map[string]string),
		},
	}

	if err := t.load(); err != nil {
		return nil, err
	}
	return t, nil
}

func (t *Tracker) load() error {
	data, err := os.ReadFile(t.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if err := json.Unmarshal(data, t.state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	if t.state.Files == nil {
		t.state.Files = make(map[string]string)
	}

	return nil
}

func (t *Tracker) Save() error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	data, err := json.MarshalIndent(t.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode state: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(t.stateFile), 0755); err != nil {
		return fmt.Errorf("failed to create state dir: %w", err)
	}

	if err := os.WriteFile(t.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (t *Tracker) ComputeHash(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

func (t *Tracker) IsProcessed(filePath string, currentHash string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()

	lastHash, exists := t.state.Files[filePath]
	if !exists {
		return false
	}

	return lastHash == currentHash
}

func (t *Tracker) MarkAsProcessed(filePath, hash string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.state.Files[filePath] = hash
}
