// Package rewriter scans source files for references to image files and
// rewrites those references when an image has been renamed (e.g. logo.jpg →
// logo.webp after optimization).  It supports ES imports, CommonJS require(),
// CSS url(), HTML src/href attributes, and Markdown image links.
package rewriter

import (
	"os"
	"path/filepath"
	"strings"
)

// Rename describes one file basename change that occurred during optimization.
type Rename struct {
	// OldName is the original basename, e.g. "logo.jpg".
	OldName string
	// NewName is the new basename, e.g. "logo.webp".
	NewName string
}

// Patch represents a single line substitution within a source file.
type Patch struct {
	// File is the absolute or relative path of the patched source file.
	File string
	// Line is the 1-based line number of the substitution.
	Line int
	// OldLine is the original line content (before substitution).
	OldLine string
	// NewLine is the replacement line content (after substitution).
	NewLine string
}

// DiffReport is returned by Scan and contains all planned or applied patches.
type DiffReport struct {
	// Patches lists every individual line substitution.
	Patches []Patch
	// Files is the number of unique source files that contained at least one patch.
	Files int
}

// sourceExts is the set of file extensions whose contents are scanned for
// image references.
var sourceExts = map[string]bool{
	".js":     true,
	".jsx":    true,
	".ts":     true,
	".tsx":    true,
	".html":   true,
	".css":    true,
	".scss":   true,
	".vue":    true,
	".svelte": true,
	".md":     true,
}

// isSourceFile reports whether path has an extension that should be scanned.
func isSourceFile(path string) bool {
	return sourceExts[strings.ToLower(filepath.Ext(path))]
}

// Scan walks sourceRoot recursively, finds all source files, and applies the
// renames list to each file.  When dryRun is false the files are rewritten on
// disk; when dryRun is true no files are modified but all patches are returned.
//
// Returns an error if sourceRoot does not exist or cannot be walked.  Per-file
// read/write errors are treated as non-fatal and those files are simply skipped.
func Scan(sourceRoot string, renames []Rename, dryRun bool) (*DiffReport, error) {
	// Validate that sourceRoot exists before starting the walk.
	if _, err := os.Stat(sourceRoot); err != nil {
		return nil, err
	}

	report := &DiffReport{}
	seen := map[string]bool{}

	err := filepath.WalkDir(sourceRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !isSourceFile(path) {
			return nil
		}
		patches, patchErr := patchFile(path, renames, dryRun)
		if patchErr != nil {
			// Non-fatal: skip files that cannot be read/written.
			return nil
		}
		if len(patches) > 0 {
			report.Patches = append(report.Patches, patches...)
			if !seen[path] {
				seen[path] = true
				report.Files++
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}
