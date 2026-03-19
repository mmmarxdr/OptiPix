// Command optipix is a standalone CLI tool for batch image optimization.
// It supports JPEG, PNG, WebP, AVIF, and SVG files with configurable output
// formats, quality, dimensions, concurrency, and import reference rewriting.
//
// Usage:
//
//	optipix optimize <path> [flags]
//	optipix help | --help | -h
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"github.com/optipix/backend/internal/processor"
	"github.com/optipix/backend/internal/rewriter"
	"github.com/optipix/backend/internal/strategy"
	"github.com/optipix/backend/internal/walker"
)

// CLIConfig holds all parsed flags and the positional path argument for the
// optimize subcommand.
type CLIConfig struct {
	// Path is the positional argument: a file or directory to process.
	Path string
	// Format is the output format: "auto" | "webp" | "avif" | "jpeg" | "png".
	Format string
	// Quality is the compression quality (1–100).
	Quality int
	// Width is the optional output width in pixels (0 = preserve).
	Width int
	// Height is the optional output height in pixels (0 = preserve).
	Height int
	// Recursive enables recursive directory walking.
	Recursive bool
	// Write enables in-place writes (replaces originals with new extension).
	Write bool
	// OutputDir specifies a separate directory for output files.
	OutputDir string
	// RewriteImports is the root dir to scan for import references. Empty = disabled.
	RewriteImports string
	// StripMetadata removes EXIF/ICC metadata from output images.
	StripMetadata bool
	// Concurrency limits the number of parallel processing goroutines.
	Concurrency int
	// SVGOPath is the path to the svgo executable.
	SVGOPath string
	// Verbose enables per-file output lines.
	Verbose bool
}

// Result captures the outcome of processing a single file.
type Result struct {
	Entry      walker.FileEntry
	Decision   strategy.Decision
	BytesIn    int64
	BytesOut   int64
	OutputPath string
	Err        error
	Skipped    bool
	SkipReason string
}

// Stats accumulates summary counters across all results.
type Stats struct {
	Processed        int
	BytesSaved       int64
	Skipped          int
	SkipReasons      map[string]int
	Failed           int
	ImportsRewritten int
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(3)
	}

	subcommand := os.Args[1]
	args := os.Args[2:]

	processor.InitVips()
	defer processor.ShutdownVips()

	var exitCode int
	switch subcommand {
	case "optimize":
		exitCode = runOptimize(ctx, args)
	case "help", "--help", "-h":
		printUsage()
		exitCode = 0
	default:
		fmt.Fprintf(os.Stderr, "error: unknown subcommand %q\n\n", subcommand)
		printUsage()
		exitCode = 3
	}

	os.Exit(exitCode)
}

// splitArgsAndFlags separates positional arguments from flag arguments so that
// flags appearing after the positional path are still honoured. For example:
//
//	optipix optimize ./assets --write   →  positionals=[./assets]  flags=[--write]
//	optipix optimize --write ./assets   →  positionals=[./assets]  flags=[--write]
//
// For --flag value pairs (value does not start with '-'), both tokens are
// collected into flags. --flag=value single-token forms are handled naturally.
func splitArgsAndFlags(args []string) (positionals []string, flags []string) {
	i := 0
	for i < len(args) {
		if strings.HasPrefix(args[i], "-") {
			flags = append(flags, args[i])
			// Consume the next token as the flag value when the flag has no
			// embedded '=' and the following token is not itself a flag.
			if !strings.Contains(args[i], "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Only consume if the flag likely takes a value argument.
				// We conservatively peek: if parsing would succeed we grab it;
				// otherwise we leave it as a positional. Since we can't know
				// ahead of time without the FlagSet, we grab it only when the
				// flag is one of the known value-taking flags.
				i++
				flags = append(flags, args[i])
			}
		} else {
			positionals = append(positionals, args[i])
		}
		i++
	}
	return
}

// runOptimize parses flags for the optimize subcommand, validates them, and
// delegates to the optimize pipeline.
func runOptimize(ctx context.Context, args []string) int {
	fs := flag.NewFlagSet("optimize", flag.ContinueOnError)

	cfg := &CLIConfig{}
	fs.StringVar(&cfg.Format, "format", "auto", "output format: auto|webp|avif|jpeg|png")
	fs.IntVar(&cfg.Quality, "quality", 80, "compression quality 1–100")
	fs.IntVar(&cfg.Width, "width", 0, "resize width in pixels (0 = preserve aspect ratio)")
	fs.IntVar(&cfg.Height, "height", 0, "resize height in pixels (0 = preserve aspect ratio)")
	fs.BoolVar(&cfg.Recursive, "recursive", false, "walk subdirectories recursively")
	fs.BoolVar(&cfg.Recursive, "r", false, "alias for --recursive")
	fs.BoolVar(&cfg.Write, "write", false, "write output files to disk (default: dry-run preview)")
	fs.StringVar(&cfg.OutputDir, "output-dir", "", "directory to write processed files (mutually exclusive with --write)")
	fs.StringVar(&cfg.RewriteImports, "rewrite-imports", "", "root dir to scan and patch import references after optimization")
	fs.BoolVar(&cfg.StripMetadata, "strip-metadata", true, "strip EXIF/ICC metadata from output images")
	fs.IntVar(&cfg.Concurrency, "concurrency", 4, "max parallel processing goroutines (≥1)")
	fs.StringVar(&cfg.SVGOPath, "svgo-path", "svgo", "path to svgo executable")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "print per-file decisions and rewriter patches")

	// Separate positional arguments from flag arguments to handle both orderings:
	//   optipix optimize --write ./assets  (standard)
	//   optipix optimize ./assets --write  (flags after path)
	positionals, flagArgs := splitArgsAndFlags(args)
	if err := fs.Parse(flagArgs); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		return 3
	}

	if len(positionals) < 1 {
		fmt.Fprintln(os.Stderr, "error: path argument required")
		fmt.Fprintln(os.Stderr)
		fs.Usage()
		return 3
	}
	cfg.Path = positionals[0]

	// --- Validation ---

	// Mutually exclusive flags.
	if cfg.Write && cfg.OutputDir != "" {
		fmt.Fprintln(os.Stderr, "error: --write and --output-dir are mutually exclusive")
		return 3
	}
	// Quality range.
	if cfg.Quality < 1 || cfg.Quality > 100 {
		fmt.Fprintf(os.Stderr, "error: --quality must be between 1 and 100 (got %d)\n", cfg.Quality)
		return 3
	}
	// Concurrency.
	if cfg.Concurrency < 1 {
		fmt.Fprintf(os.Stderr, "error: --concurrency must be at least 1 (got %d)\n", cfg.Concurrency)
		return 3
	}
	// Format validity.
	if cfg.Format != "auto" {
		if _, err := processor.ParseFormat(cfg.Format); err != nil {
			fmt.Fprintf(os.Stderr, "error: invalid --format %q; valid values: auto, webp, avif, jpeg, png\n", cfg.Format)
			return 3
		}
	}
	// Path existence.
	if _, err := os.Stat(cfg.Path); err != nil {
		fmt.Fprintf(os.Stderr, "error: path not found: %s\n", cfg.Path)
		return 3
	}

	return optimize(ctx, cfg)
}

// optimize runs the walker → strategy → processor pipeline for cfg.Path and
// returns an exit code (0 = success, 1 = optimization error, 2 = rewrite error).
func optimize(ctx context.Context, cfg *CLIConfig) int {
	// All supported input extensions.
	supportedExts := []string{
		".jpg", ".jpeg", ".png", ".webp", ".avif", ".svg",
		".gif", ".bmp", ".tiff", ".heif",
	}

	walkerOpts := walker.Options{
		Recursive:  cfg.Recursive,
		Extensions: supportedExts,
	}

	entries := walker.Walk(ctx, cfg.Path, walkerOpts)

	semaphore := make(chan struct{}, cfg.Concurrency)
	results := make(chan Result, cfg.Concurrency*2)
	var wg sync.WaitGroup

	// Worker loop: consume entries, skip or process each file.
loop:
	for {
		select {
		case entry, ok := <-entries:
			if !ok {
				break loop
			}
			dec := strategy.Resolve(entry.InputPath, cfg.Format)
			if dec.Skip {
				results <- Result{
					Entry:      entry,
					Decision:   dec,
					Skipped:    true,
					SkipReason: dec.Reason,
				}
				continue
			}
			// Acquire concurrency slot.
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				break loop
			}
			wg.Add(1)
			go func(e walker.FileEntry, d strategy.Decision) {
				defer wg.Done()
				defer func() { <-semaphore }()
				results <- processFile(ctx, e, d, cfg)
			}(entry, dec)
		case <-ctx.Done():
			break loop
		}
	}

	// Close results after all goroutines finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Drain results and collect renames for the import rewriter.
	var allResults []Result
	for r := range results {
		allResults = append(allResults, r)
		if cfg.Verbose {
			printFileResult(r, cfg.Write)
		}
	}

	// Build the rename list for the import rewriter.
	var renames []rewriter.Rename
	for _, r := range allResults {
		if r.Skipped || r.Err != nil {
			continue
		}
		oldBase := filepath.Base(r.Entry.InputPath)
		newBase := filepath.Base(r.OutputPath)
		if oldBase != newBase {
			renames = append(renames, rewriter.Rename{OldName: oldBase, NewName: newBase})
		}
	}

	// Run import rewriter if requested.
	var rewriteReport *rewriter.DiffReport
	var rewriteErr error
	if cfg.RewriteImports != "" && len(renames) > 0 {
		if _, statErr := os.Stat(cfg.RewriteImports); statErr != nil {
			fmt.Fprintf(os.Stderr, "error: --rewrite-imports path not found: %s\n", cfg.RewriteImports)
			return 2
		}
		// In dry-run mode (no --write), do not write source file changes.
		isDryRun := !cfg.Write
		rewriteReport, rewriteErr = rewriter.Scan(cfg.RewriteImports, renames, isDryRun)
		if rewriteErr != nil {
			fmt.Fprintf(os.Stderr, "error: import rewrite failed: %v\n", rewriteErr)
			return 2
		}
		// Print dry-run diff preview.
		if isDryRun && rewriteReport != nil {
			printRewriteDiff(rewriteReport)
		}
	} else if cfg.RewriteImports != "" {
		// --rewrite-imports set but no renames (same extension after compression) — validate path.
		if _, statErr := os.Stat(cfg.RewriteImports); statErr != nil {
			fmt.Fprintf(os.Stderr, "error: --rewrite-imports path not found: %s\n", cfg.RewriteImports)
			return 2
		}
	}

	// Build stats and print summary.
	stats := buildStats(allResults, rewriteReport)
	printSummary(stats, !cfg.Write, rewriteReport, cfg.Verbose)

	return determineExitCode(allResults, rewriteErr)
}

// processFile reads an input file, optimizes it through the processor, and
// optionally writes the result to disk. It returns a Result capturing all
// relevant metrics and any error.
func processFile(ctx context.Context, entry walker.FileEntry, dec strategy.Decision, cfg *CLIConfig) Result {
	data, err := os.ReadFile(entry.InputPath)
	if err != nil {
		return Result{Entry: entry, Decision: dec, Err: fmt.Errorf("read: %w", err)}
	}

	outputPath := computeOutputPath(entry, dec, cfg)

	if dec.IsSVG {
		// Check svgo availability first (graceful degradation).
		if _, lookErr := exec.LookPath(cfg.SVGOPath); lookErr != nil {
			return Result{
				Entry:      entry,
				Decision:   dec,
				Skipped:    true,
				SkipReason: "svgo not found in PATH",
				BytesIn:    int64(len(data)),
			}
		}

		svgOpts := processor.DefaultSVGOptions()
		result, svgErr := processor.OptimizeSVG(ctx, data, svgOpts, cfg.SVGOPath)
		if svgErr != nil {
			return Result{Entry: entry, Decision: dec, BytesIn: int64(len(data)), Err: svgErr}
		}

		if cfg.Write || cfg.OutputDir != "" {
			if mkErr := os.MkdirAll(filepath.Dir(outputPath), 0755); mkErr != nil {
				return Result{Entry: entry, Decision: dec, BytesIn: int64(len(data)), Err: mkErr}
			}
			if writeErr := os.WriteFile(outputPath, result.Data, 0644); writeErr != nil {
				return Result{Entry: entry, Decision: dec, BytesIn: int64(len(data)), Err: writeErr}
			}
		}
		return Result{
			Entry:      entry,
			Decision:   dec,
			BytesIn:    int64(result.OriginalSize),
			BytesOut:   int64(result.OutputSize),
			OutputPath: outputPath,
		}
	}

	// Raster image.
	lossless := false
	if dec.NeedsAlphaCheck {
		lossless = imageHasAlpha(data)
	}
	imgOpts := processor.ImageOptions{
		Format:        dec.OutputFormat,
		Quality:       cfg.Quality,
		Width:         cfg.Width,
		Height:        cfg.Height,
		StripMetadata: cfg.StripMetadata,
		Lossless:      lossless,
	}

	result, imgErr := processor.OptimizeImage(ctx, data, imgOpts)
	if imgErr != nil {
		return Result{Entry: entry, Decision: dec, BytesIn: int64(len(data)), Err: imgErr}
	}

	if cfg.Write || cfg.OutputDir != "" {
		if mkErr := os.MkdirAll(filepath.Dir(outputPath), 0755); mkErr != nil {
			return Result{Entry: entry, Decision: dec, BytesIn: int64(len(data)), Err: mkErr}
		}
		if writeErr := os.WriteFile(outputPath, result.Data, 0644); writeErr != nil {
			return Result{Entry: entry, Decision: dec, BytesIn: int64(len(data)), Err: writeErr}
		}
	}

	return Result{
		Entry:      entry,
		Decision:   dec,
		BytesIn:    int64(result.OriginalSize),
		BytesOut:   int64(result.OutputSize),
		OutputPath: outputPath,
	}
}

// imageHasAlpha reports whether the image encoded in data contains a channel
// with non-opaque pixels. It decodes the image using the standard library and
// samples every pixel; a non-fully-opaque alpha value is taken as proof of
// transparency. Falls back to false on decode errors.
func imageHasAlpha(data []byte) bool {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return false
	}
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			// RGBA() returns values in [0, 65535]; fully opaque == 65535.
			if a < 65535 {
				return true
			}
		}
	}
	// Also check whether the color model itself supports alpha.
	switch img.ColorModel() {
	case color.RGBAModel, color.RGBA64Model, color.NRGBAModel, color.NRGBA64Model,
		color.AlphaModel, color.Alpha16Model:
		// Model supports alpha; the pixel scan already checked for non-opaque.
	}
	return false
}

// computeOutputPath determines the destination path for a processed file.
// Priority: --output-dir > --write (in-place with new ext) > dry-run planned path.
func computeOutputPath(entry walker.FileEntry, dec strategy.Decision, cfg *CLIConfig) string {
	base := filepath.Base(entry.InputPath)
	// Replace extension.
	newBase := strings.TrimSuffix(base, filepath.Ext(base)) + dec.OutputExt

	if cfg.OutputDir != "" {
		// Mirror relative path under output-dir.
		return filepath.Join(cfg.OutputDir, filepath.Dir(entry.RelPath), newBase)
	}

	// In-place: replace the file alongside its source (new extension).
	dir := filepath.Dir(entry.InputPath)
	return filepath.Join(dir, newBase)
}

// buildStats aggregates processing results into a Stats summary.
func buildStats(results []Result, report *rewriter.DiffReport) Stats {
	s := Stats{SkipReasons: make(map[string]int)}
	for _, r := range results {
		if r.Err != nil {
			s.Failed++
			continue
		}
		if r.Skipped {
			s.Skipped++
			if r.SkipReason != "" {
				s.SkipReasons[r.SkipReason]++
			}
			continue
		}
		s.Processed++
		if r.BytesOut > 0 {
			saved := r.BytesIn - r.BytesOut
			if saved > 0 {
				s.BytesSaved += saved
			}
		}
	}
	if report != nil {
		s.ImportsRewritten = report.Files
	}
	return s
}

// printSummary writes the human-readable processing summary to stdout.
func printSummary(stats Stats, dryRun bool, report *rewriter.DiffReport, verbose bool) {
	if dryRun {
		fmt.Println("\nOptiPix CLI — Summary [DRY-RUN]")
	} else {
		fmt.Println("\nOptiPix CLI — Summary")
	}
	fmt.Println(strings.Repeat("─", 40))

	total := stats.Processed + stats.Skipped + stats.Failed
	if total == 0 {
		fmt.Println("No image files found.")
		return
	}

	fmt.Printf("Files processed : %d\n", stats.Processed)

	savedKB := float64(stats.BytesSaved) / 1024
	if dryRun {
		fmt.Printf("Est. bytes saved: %.1f KB\n", savedKB)
	} else {
		fmt.Printf("Bytes saved     : %.1f KB\n", savedKB)
	}

	if stats.Skipped > 0 {
		fmt.Printf("Files skipped   : %d\n", stats.Skipped)
		for reason, count := range stats.SkipReasons {
			fmt.Printf("  - %s: %d\n", reason, count)
		}
	}

	if stats.Failed > 0 {
		fmt.Printf("Errors          : %d\n", stats.Failed)
	}

	if report != nil && report.Files > 0 {
		fmt.Printf("Imports rewritten: %d patches across %d file(s)\n",
			len(report.Patches), report.Files)
	}
}

// printFileResult emits a per-file line for verbose mode.
func printFileResult(r Result, write bool) {
	if r.Skipped {
		fmt.Printf("  SKIP  %s — %s\n", r.Entry.InputPath, r.SkipReason)
		return
	}
	if r.Err != nil {
		fmt.Printf("  ERR   %s — %v\n", r.Entry.InputPath, r.Err)
		return
	}
	saved := r.BytesIn - r.BytesOut
	prefix := "[DRY-RUN] "
	if write {
		prefix = ""
	}
	fmt.Printf("  OK    %s%s → %s (~%.1f KB saved)\n",
		prefix, r.Entry.InputPath, r.OutputPath, float64(saved)/1024)
}

// printRewriteDiff prints a diff-style preview of planned import patches.
func printRewriteDiff(report *rewriter.DiffReport) {
	if report == nil || len(report.Patches) == 0 {
		return
	}
	fmt.Printf("\n[DRY-RUN] Import rewrite preview (%d patch(es) across %d file(s)):\n",
		len(report.Patches), report.Files)
	for _, p := range report.Patches {
		fmt.Printf("  %s:%d\n", p.File, p.Line)
		fmt.Printf("  - %s\n", p.OldLine)
		fmt.Printf("  + %s\n", p.NewLine)
	}
}

// determineExitCode inspects results and a rewrite error to compute the
// correct exit code.  Import rewrite errors take highest priority (2), then
// optimization errors (1), then success (0).
func determineExitCode(results []Result, rewriteErr error) int {
	if rewriteErr != nil {
		return 2
	}
	for _, r := range results {
		if r.Err != nil {
			return 1
		}
	}
	return 0
}

// printUsage prints the top-level usage message to stdout.
func printUsage() {
	fmt.Print(`OptiPix CLI — standalone image optimization tool

Usage:
  optipix <subcommand> [flags]

Subcommands:
  optimize <path>   Optimize images in a file or directory

  optimize flags:
    --format string          Output format: auto|webp|avif|jpeg|png (default "auto")
    --quality int            Compression quality 1-100 (default 80)
    --width int              Resize width in pixels, 0 = preserve (default 0)
    --height int             Resize height in pixels, 0 = preserve (default 0)
    --recursive, -r          Walk subdirectories recursively
    --write                  Write output files to disk (default: dry-run)
    --output-dir string      Directory to write processed files
    --rewrite-imports string Root dir to scan and patch import references
    --strip-metadata         Strip EXIF/ICC metadata (default true)
    --concurrency int        Max parallel goroutines (default 4)
    --svgo-path string       Path to svgo executable (default "svgo")
    --verbose                Print per-file decisions

  help | --help | -h        Show this help message

Exit codes:
  0  Success (or dry-run preview complete)
  1  One or more optimization errors
  2  Import rewrite error
  3  Config / invocation error
`)
}
