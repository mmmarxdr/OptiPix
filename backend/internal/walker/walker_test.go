package walker_test

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/optipix/backend/internal/walker"
)

// collectEntries drains the channel and returns all InputPaths.
func collectEntries(ch <-chan walker.FileEntry) []string {
	var paths []string
	for e := range ch {
		paths = append(paths, e.InputPath)
	}
	sort.Strings(paths)
	return paths
}

// buildTree creates a directory structure in a temp directory and returns the root.
//
//	root/
//	  a.jpg
//	  b.png
//	  sub/
//	    c.jpg
func buildTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	writeFile(t, filepath.Join(root, "a.jpg"), "jpg")
	writeFile(t, filepath.Join(root, "b.png"), "png")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "sub", "c.jpg"), "jpg")
	return root
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestWalk_NonRecursive(t *testing.T) {
	t.Parallel()
	root := buildTree(t)

	ch := walker.Walk(context.Background(), root, walker.Options{Recursive: false})
	got := collectEntries(ch)

	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d: %v", len(got), got)
	}
	for _, p := range got {
		if filepath.Dir(p) != root {
			t.Errorf("expected only root-level files, got %s", p)
		}
	}
}

func TestWalk_Recursive(t *testing.T) {
	t.Parallel()
	root := buildTree(t)

	ch := walker.Walk(context.Background(), root, walker.Options{Recursive: true})
	got := collectEntries(ch)

	if len(got) != 3 {
		t.Fatalf("expected 3 entries, got %d: %v", len(got), got)
	}
}

func TestWalk_ExtensionFilter(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "a.jpg"), "jpg")
	writeFile(t, filepath.Join(root, "b.txt"), "txt")
	writeFile(t, filepath.Join(root, "c.png"), "png")

	ch := walker.Walk(context.Background(), root, walker.Options{
		Recursive:  false,
		Extensions: []string{".jpg"},
	})
	got := collectEntries(ch)

	if len(got) != 1 {
		t.Fatalf("expected 1 entry (.jpg only), got %d: %v", len(got), got)
	}
	if filepath.Ext(got[0]) != ".jpg" {
		t.Errorf("expected .jpg extension, got %s", got[0])
	}
}

func TestWalk_ExcludePattern(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, "node_modules"), 0755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "node_modules", "vendor.png"), "png")
	writeFile(t, filepath.Join(root, "img.png"), "png")

	ch := walker.Walk(context.Background(), root, walker.Options{
		Recursive: true,
		Exclude:   []string{"node_modules"},
	})
	got := collectEntries(ch)

	for _, p := range got {
		if filepath.Base(p) == "vendor.png" {
			t.Errorf("vendor.png should have been excluded, got %s", p)
		}
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry (img.png), got %d: %v", len(got), got)
	}
}

func TestWalk_ContextCancellation(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	// Create 100 files.
	for i := 0; i < 100; i++ {
		writeFile(t, filepath.Join(root, filepath.FromSlash(
			// use a simple numeric name
			string([]byte{byte('0' + i/10), byte('0' + i%10)}))+".jpg"), "jpg")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := walker.Walk(ctx, root, walker.Options{Recursive: false})

	// Receive a few entries, then cancel.
	received := 0
	for range ch {
		received++
		if received >= 5 {
			cancel()
			break
		}
	}

	// Drain remaining entries after cancel; channel should close within 500 ms.
	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()

	select {
	case <-done:
		// Good — channel closed promptly.
	case <-time.After(500 * time.Millisecond):
		t.Error("channel did not close within 500ms after context cancellation")
	}
}

func TestWalk_SingleFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "photo.jpg")
	writeFile(t, file, "jpg")

	ch := walker.Walk(context.Background(), file, walker.Options{Recursive: false})
	got := collectEntries(ch)

	if len(got) != 1 {
		t.Fatalf("expected exactly 1 FileEntry for a single file, got %d: %v", len(got), got)
	}
	if got[0] != file {
		t.Errorf("expected %s, got %s", file, got[0])
	}
}

func TestWalk_NonExistentRoot(t *testing.T) {
	t.Parallel()

	ch := walker.Walk(context.Background(), "/this/path/does/not/exist", walker.Options{})

	done := make(chan struct{})
	go func() {
		for range ch {
		}
		close(done)
	}()

	select {
	case <-done:
		// Good — channel closed without deadlock.
	case <-time.After(500 * time.Millisecond):
		t.Error("channel did not close within 500ms for non-existent root")
	}
}
