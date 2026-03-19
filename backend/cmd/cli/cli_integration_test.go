//go:build integration

// Package main integration tests require libvips to be installed.
// Run with: go test -race -tags=integration ./cmd/cli/...
package main

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

// makeMinimalJPEG generates a valid 1×1 red pixel JPEG in memory using the
// standard library. This avoids hard-coded byte slices that may be rejected
// by stricter decoders such as libvips.
func makeMinimalJPEG() []byte {
	img := image.NewRGBA(image.Rect(0, 0, 1, 1))
	img.Set(0, 0, color.RGBA{R: 255, G: 0, B: 0, A: 255})
	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	return buf.Bytes()
}

// minimalJPEG is a valid 1×1 JPEG generated at package-init time.
var minimalJPEG = makeMinimalJPEG()

func TestCLI_OptimizeSingleJPEG(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "photo.jpg")
	if err := os.WriteFile(src, minimalJPEG, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:          src,
		Format:        "auto",
		Quality:       80,
		Write:         true,
		Concurrency:   1,
		SVGOPath:      "svgo",
		StripMetadata: true,
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	out := filepath.Join(root, "photo.webp")
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected output file %s to exist: %v", out, err)
	}
}

func TestCLI_DryRun_NoFiles(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "photo.jpg")
	if err := os.WriteFile(src, minimalJPEG, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:        src,
		Format:      "auto",
		Quality:     80,
		Write:       false, // dry-run
		Concurrency: 1,
		SVGOPath:    "svgo",
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// No .webp file should have been written.
	out := filepath.Join(root, "photo.webp")
	if _, err := os.Stat(out); err == nil {
		t.Error("dry-run should NOT have written photo.webp")
	}
}

func TestCLI_RecursiveDirectory(t *testing.T) {
	root := t.TempDir()

	// 2 JPEGs at each of 3 levels.
	for _, dir := range []string{".", "sub", filepath.Join("sub", "deep")} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0755); err != nil {
			t.Fatal(err)
		}
		for _, name := range []string{"a.jpg", "b.jpg"} {
			p := filepath.Join(root, dir, name)
			if err := os.WriteFile(p, minimalJPEG, 0644); err != nil {
				t.Fatal(err)
			}
		}
	}

	cfg := &CLIConfig{
		Path:        root,
		Format:      "auto",
		Quality:     80,
		Write:       true,
		Recursive:   true,
		Concurrency: 2,
		SVGOPath:    "svgo",
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	// All 6 WebP files should exist.
	for _, dir := range []string{".", "sub", filepath.Join("sub", "deep")} {
		for _, name := range []string{"a.webp", "b.webp"} {
			p := filepath.Join(root, dir, name)
			if _, err := os.Stat(p); err != nil {
				t.Errorf("expected %s to exist: %v", p, err)
			}
		}
	}
}

func TestCLI_ExplicitFormat(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "photo.jpg")
	if err := os.WriteFile(src, minimalJPEG, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:        src,
		Format:      "avif",
		Quality:     80,
		Write:       true,
		Concurrency: 1,
		SVGOPath:    "svgo",
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	out := filepath.Join(root, "photo.avif")
	if _, err := os.Stat(out); err != nil {
		t.Errorf("expected output file %s to exist: %v", out, err)
	}
}

func TestCLI_RewriteImports(t *testing.T) {
	root := t.TempDir()
	// Create source image.
	src := filepath.Join(root, "logo.jpg")
	if err := os.WriteFile(src, minimalJPEG, 0644); err != nil {
		t.Fatal(err)
	}

	// Create a JS file referencing the original.
	srcDir := filepath.Join(root, "src")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	jsFile := filepath.Join(srcDir, "app.js")
	if err := os.WriteFile(jsFile, []byte("import logo from '../logo.jpg'"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:           root,
		Format:         "auto",
		Quality:        80,
		Write:          true,
		Concurrency:    1,
		SVGOPath:       "svgo",
		RewriteImports: srcDir,
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}

	data, err := os.ReadFile(jsFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "import logo from '../logo.webp'" {
		t.Errorf("JS file not patched: got %q", string(data))
	}
}

func TestCLI_SVGOAbsent(t *testing.T) {
	root := t.TempDir()
	svgFile := filepath.Join(root, "icon.svg")
	if err := os.WriteFile(svgFile, []byte("<svg><rect/></svg>"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:        root,
		Format:      "auto",
		Quality:     80,
		Write:       false,
		Concurrency: 1,
		SVGOPath:    "/nonexistent/svgo",
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 0 {
		t.Errorf("expected exit code 0 when svgo absent (graceful degradation), got %d", exitCode)
	}
}

func TestCLI_ExitCode_CorruptFile(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "bad.jpg")
	// Write an empty file with .jpg extension (invalid JPEG).
	if err := os.WriteFile(src, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:        src,
		Format:      "auto",
		Quality:     80,
		Write:       true,
		Concurrency: 1,
		SVGOPath:    "svgo",
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 1 {
		t.Errorf("expected exit code 1 for corrupt file, got %d", exitCode)
	}
}

func TestCLI_ExitCode_RewriteImportsMissing(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "photo.jpg")
	if err := os.WriteFile(src, minimalJPEG, 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &CLIConfig{
		Path:           src,
		Format:         "auto",
		Quality:        80,
		Write:          true,
		Concurrency:    1,
		SVGOPath:       "svgo",
		RewriteImports: filepath.Join(root, "nonexistent"),
	}
	exitCode := optimize(context.Background(), cfg)
	if exitCode != 2 {
		t.Errorf("expected exit code 2 for missing rewrite-imports path, got %d", exitCode)
	}
}
