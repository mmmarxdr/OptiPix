package rewriter_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/optipix/backend/internal/rewriter"
)

// --- applyRename unit tests (table-driven) ---
// We export applyRename via the package-internal test alias below by using
// the internal package test approach: test is in the same package.

// applyRenameForTest is a package-level shim that lets us call the unexported
// applyRename from outside the package.  Since Go only allows internal tests in
// the same package, we use a black-box approach here via Scan + temp files.
// Instead, we test applyRename indirectly through patchFile/Scan.

// TestApplyRename uses Scan with a single-file temp dir to test each pattern.
func TestApplyRename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		line     string
		oldName  string
		newName  string
		wantLine string
		ext      string // file extension to determine which scanner runs
	}{
		{
			name:     "ES import single quote",
			line:     "import logo from './assets/logo.jpg'",
			oldName:  "logo.jpg",
			newName:  "logo.webp",
			wantLine: "import logo from './assets/logo.webp'",
			ext:      ".js",
		},
		{
			name:     "ES import double quote",
			line:     `import x from "./img/banner.jpeg"`,
			oldName:  "banner.jpeg",
			newName:  "banner.webp",
			wantLine: `import x from "./img/banner.webp"`,
			ext:      ".ts",
		},
		{
			name:     "require call",
			line:     "const img = require('./images/hero.png')",
			oldName:  "hero.png",
			newName:  "hero.webp",
			wantLine: "const img = require('./images/hero.webp')",
			ext:      ".js",
		},
		{
			name:     "CSS url with quotes",
			line:     "background: url('./images/bg.jpg')",
			oldName:  "bg.jpg",
			newName:  "bg.webp",
			wantLine: "background: url('./images/bg.webp')",
			ext:      ".css",
		},
		{
			name:     "CSS url no quotes",
			line:     "background: url(bg.jpg)",
			oldName:  "bg.jpg",
			newName:  "bg.webp",
			wantLine: "background: url(bg.webp)",
			ext:      ".css",
		},
		{
			name:     "HTML src attribute",
			line:     `<img src="./assets/banner.png">`,
			oldName:  "banner.png",
			newName:  "banner.webp",
			wantLine: `<img src="./assets/banner.webp">`,
			ext:      ".html",
		},
		{
			name:     "HTML data-src attribute",
			line:     `<img data-src="./img/lazy.jpg">`,
			oldName:  "lazy.jpg",
			newName:  "lazy.webp",
			wantLine: `<img data-src="./img/lazy.webp">`,
			ext:      ".html",
		},
		{
			name:     "Markdown image reference",
			line:     "![logo](./assets/logo.jpg)",
			oldName:  "logo.jpg",
			newName:  "logo.webp",
			wantLine: "![logo](./assets/logo.webp)",
			ext:      ".md",
		},
		{
			name:     "No match different file",
			line:     "import x from './other.jpg'",
			oldName:  "logo.jpg",
			newName:  "logo.webp",
			wantLine: "import x from './other.jpg'",
			ext:      ".js",
		},
		{
			name:     "Same extension no-op",
			line:     "import x from './assets/photo.jpg'",
			oldName:  "photo.jpg",
			newName:  "photo.jpg",
			wantLine: "import x from './assets/photo.jpg'",
			ext:      ".js",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			file := filepath.Join(root, "test"+tc.ext)

			if err := os.WriteFile(file, []byte(tc.line), 0644); err != nil {
				t.Fatal(err)
			}

			renames := []rewriter.Rename{{OldName: tc.oldName, NewName: tc.newName}}

			// dryRun=true so we get patches without writing.
			report, err := rewriter.Scan(root, renames, true)
			if err != nil {
				t.Fatalf("Scan error: %v", err)
			}

			if tc.wantLine == tc.line {
				// Expect no patch.
				if len(report.Patches) != 0 {
					t.Errorf("expected no patches, got %d: %+v", len(report.Patches), report.Patches)
				}
				return
			}

			if len(report.Patches) == 0 {
				t.Fatalf("expected at least 1 patch, got 0")
			}
			if got := report.Patches[0].NewLine; got != tc.wantLine {
				t.Errorf("NewLine:\n  got  %q\n  want %q", got, tc.wantLine)
			}
		})
	}
}

func TestPatchFile_DryRun(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "app.js")
	original := "import logo from './assets/logo.jpg'"
	if err := os.WriteFile(file, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	renames := []rewriter.Rename{{OldName: "logo.jpg", NewName: "logo.webp"}}
	report, err := rewriter.Scan(root, renames, true /* dryRun */)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Patches should be non-empty.
	if len(report.Patches) == 0 {
		t.Error("expected patches in dry-run mode")
	}

	// File on disk must be unchanged.
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != original {
		t.Errorf("dry-run modified file: got %q, want %q", string(data), original)
	}
}

func TestPatchFile_WriteMode(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	file := filepath.Join(root, "app.js")
	original := "import logo from './assets/logo.jpg'"
	if err := os.WriteFile(file, []byte(original), 0644); err != nil {
		t.Fatal(err)
	}

	renames := []rewriter.Rename{{OldName: "logo.jpg", NewName: "logo.webp"}}
	report, err := rewriter.Scan(root, renames, false /* dryRun=false, write */)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(report.Patches) == 0 {
		t.Error("expected at least 1 patch")
	}

	// File on disk must be updated.
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	expected := "import logo from './assets/logo.webp'"
	if string(data) != expected {
		t.Errorf("write mode: file content got %q, want %q", string(data), expected)
	}
}

func TestScan_MultipleFileTypes(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	exts := []string{".js", ".ts", ".jsx", ".tsx", ".html", ".css", ".scss", ".vue", ".svelte", ".md"}
	for _, ext := range exts {
		content := "import logo from './assets/logo.jpg'"
		if ext == ".html" {
			content = `<img src="./assets/logo.jpg">`
		} else if ext == ".css" || ext == ".scss" {
			content = "background: url('./assets/logo.jpg')"
		} else if ext == ".md" {
			content = "![logo](./assets/logo.jpg)"
		}
		if err := os.WriteFile(filepath.Join(root, "file"+ext), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	renames := []rewriter.Rename{{OldName: "logo.jpg", NewName: "logo.webp"}}
	report, err := rewriter.Scan(root, renames, false)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if report.Files != len(exts) {
		t.Errorf("Files patched: got %d, want %d", report.Files, len(exts))
	}
}

func TestScan_NonExistentRoot(t *testing.T) {
	t.Parallel()

	renames := []rewriter.Rename{{OldName: "logo.jpg", NewName: "logo.webp"}}
	_, err := rewriter.Scan("/no/such/path/exists", renames, true)
	if err == nil {
		t.Error("expected error for non-existent root, got nil")
	}
}

func TestScan_DryRun(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	files := []string{"a.js", "b.ts", "c.html"}
	for _, f := range files {
		content := "import logo from './assets/logo.jpg'"
		if f == "c.html" {
			content = `<img src="./assets/logo.jpg">`
		}
		if err := os.WriteFile(filepath.Join(root, f), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	renames := []rewriter.Rename{{OldName: "logo.jpg", NewName: "logo.webp"}}
	report, err := rewriter.Scan(root, renames, true /* dryRun */)
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(report.Patches) == 0 {
		t.Error("expected patches in dry-run mode")
	}

	// All files on disk must be unchanged.
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(root, f))
		if err != nil {
			t.Fatal(err)
		}
		if string(data) == "" {
			t.Errorf("file %s is unexpectedly empty", f)
		}
		// The file should still contain the old name.
		content := string(data)
		if len(renames) > 0 && contains(content, renames[0].NewName) {
			t.Errorf("dry-run modified %s — found new name %q", f, renames[0].NewName)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
