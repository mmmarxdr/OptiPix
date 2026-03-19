// Package walker provides an asynchronous filesystem walker that emits
// discovered file entries on a channel. It supports recursive traversal,
// extension filtering, glob-based exclusion, and graceful context cancellation.
package walker

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
)

// FileEntry describes a single discovered file.
type FileEntry struct {
	// InputPath is the absolute or relative path to the file as discovered.
	InputPath string
	// RelPath is the path relative to the walk root, used for output-dir mapping.
	RelPath string
}

// Options controls walk behaviour.
type Options struct {
	// Recursive enables descending into subdirectories. When false, only
	// depth-1 files (direct children of root) are emitted.
	Recursive bool
	// Extensions is a list of lowercase file extensions to include (e.g. ".jpg", ".png").
	// An empty slice means all extensions are accepted.
	Extensions []string
	// Exclude is a list of glob patterns matched against the base name of each
	// directory or file. Matching entries are skipped entirely.
	Exclude []string
}

// Walk starts an asynchronous directory walk rooted at root and returns a
// read-only channel of FileEntry values. The channel is closed when the walk
// completes or when ctx is cancelled. The returned channel is buffered (cap 64)
// to decouple the walker goroutine from the consumer.
//
// If root is a single file (not a directory), Walk emits exactly one FileEntry
// for that file (provided it passes extension and exclusion filters) and closes
// the channel.
//
// Errors encountered during traversal (e.g. unreadable entries) are silently
// skipped — they do not abort the walk. If root itself does not exist,
// filepath.WalkDir returns an error on the first call; the goroutine exits and
// closes the channel immediately.
func Walk(ctx context.Context, root string, opts Options) <-chan FileEntry {
	ch := make(chan FileEntry, 64)
	go func() {
		defer close(ch)
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// Skip unreadable entries without aborting the walk.
				return nil
			}

			// Check for cancellation before doing any work on this entry.
			select {
			case <-ctx.Done():
				return filepath.SkipAll
			default:
			}

			if d.IsDir() {
				// Never skip the root itself.
				if path == root {
					return nil
				}
				// Non-recursive: skip any subdirectory.
				if !opts.Recursive {
					return filepath.SkipDir
				}
				// Excluded directory: skip it and all its contents.
				if matchesExclude(filepath.Base(path), opts.Exclude) {
					return filepath.SkipDir
				}
				return nil
			}

			// File: apply extension filter.
			ext := strings.ToLower(filepath.Ext(path))
			if !matchesExtensions(ext, opts.Extensions) {
				return nil
			}

			// File: apply exclusion filter on the basename.
			if matchesExclude(filepath.Base(path), opts.Exclude) {
				return nil
			}

			rel, _ := filepath.Rel(root, path)

			// Emit the entry; honour context cancellation while blocked.
			select {
			case ch <- FileEntry{InputPath: path, RelPath: rel}:
			case <-ctx.Done():
				return filepath.SkipAll
			}
			return nil
		})
	}()
	return ch
}

// matchesExtensions reports whether ext is in the allowed list.
// An empty allowed list matches everything.
func matchesExtensions(ext string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, a := range allowed {
		if ext == a {
			return true
		}
	}
	return false
}

// matchesExclude reports whether name matches any of the glob patterns.
func matchesExclude(name string, patterns []string) bool {
	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, name); matched {
			return true
		}
	}
	return false
}
