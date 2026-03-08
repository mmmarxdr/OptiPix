package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/optipix/backend/internal/processor"
	"github.com/optipix/backend/internal/tracker"
)

func main() {
	var inputDir string
	var outputDir string
	var stateFile string
	var format string
	var quality int

	flag.StringVar(&inputDir, "input", "./images", "Directory with source images")
	flag.StringVar(&outputDir, "output", "./optimized", "Directory for optimized images")
	flag.StringVar(&stateFile, "state", ".optipix-state.json", "Tracker state file")
	flag.StringVar(&format, "format", "auto", "Output format (webp, avif, jpeg, png, auto)")
	flag.IntVar(&quality, "quality", 80, "Compression quality (0-100)")
	flag.Parse()

	processor.InitVips()
	defer processor.ShutdownVips()

	trk, err := tracker.New(stateFile)
	if err != nil {
		log.Fatalf("Failed to initialize tracker: %v", err)
	}

	var parsedFormat processor.OutputFormat
	if format != "auto" {
		var err error
		parsedFormat, err = processor.ParseFormat(format)
		if err != nil {
			log.Fatalf("Invalid format %q: %v", format, err)
		}
	}

	baseOpts := processor.DefaultOptions()
	baseOpts.Quality = quality

	var processedCount int
	var skippedCount int
	var failedCount int

	log.Printf("Scanning directory: %s", inputDir)
	err = filepath.WalkDir(inputDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".jpg" && ext != ".jpeg" && ext != ".png" && ext != ".webp" && ext != ".avif" {
			log.Printf("Skipping non-image file: %s", path)
			return nil
		}

		relPath, err := filepath.Rel(inputDir, path)
		if err != nil {
			return err
		}

		hash, err := trk.ComputeHash(path)
		if err != nil {
			log.Printf("Failed to hash %s: %v", relPath, err)
			failedCount++
			return nil
		}

		if trk.IsProcessed(relPath, hash) {
			skippedCount++
			return nil
		}

		opts := baseOpts
		var outExt string

		if format == "auto" {
			cleanExt := strings.TrimPrefix(ext, ".")
			if cleanExt == "jpg" {
				cleanExt = "jpeg"
			}
			f, err := processor.ParseFormat(cleanExt)
			if err != nil {
				log.Printf("Warning: unsupported format for auto %s, falling back to webp", ext)
				opts.Format = processor.FormatWebP
				outExt = ".webp"
			} else {
				opts.Format = f
				outExt = ext
			}
		} else {
			opts.Format = parsedFormat
			outExt = fmt.Sprintf(".%s", format)
		}

		log.Printf("Optimizing %s (format: %s)...", relPath, opts.Format)

		data, err := os.ReadFile(path)
		if err != nil {
			log.Printf("Error reading %s: %v", relPath, err)
			failedCount++
			return nil
		}

		ctx := context.Background()
		res, err := processor.OptimizeImage(ctx, data, opts)
		if err != nil {
			log.Printf("Error processing %s: %v", relPath, err)
			failedCount++
			return nil
		}

		outName := strings.TrimSuffix(filepath.Base(relPath), ext) + outExt
		outPath := filepath.Join(outputDir, filepath.Dir(relPath), outName)

		if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
			log.Printf("Error creating output dir for %s: %v", outPath, err)
			failedCount++
			return nil
		}

		if res.OutputSize >= res.OriginalSize {
			log.Printf("⚠ %s (Savings: %d%%) - Discarded, original is smaller and will be kept.", outName, ((res.OriginalSize-res.OutputSize)*100)/res.OriginalSize)

			outPathOriginalExt := filepath.Join(outputDir, filepath.Dir(relPath), filepath.Base(relPath))
			if err := os.MkdirAll(filepath.Dir(outPathOriginalExt), 0755); err != nil {
				log.Printf("Error creating output dir for %s: %v", outPathOriginalExt, err)
				failedCount++
				return nil
			}

			if err := os.WriteFile(outPathOriginalExt, data, 0644); err != nil {
				log.Printf("Error saving original file fallback %s: %v", outPathOriginalExt, err)
				failedCount++
				return nil
			}
		} else {
			if err := os.WriteFile(outPath, res.Data, 0644); err != nil {
				log.Printf("Error saving %s: %v", outPath, err)
				failedCount++
				return nil
			}
			log.Printf("✓ %s (Savings: %d%%)", outName, ((res.OriginalSize-res.OutputSize)*100)/res.OriginalSize)
		}

		trk.MarkAsProcessed(relPath, hash)
		if err := trk.Save(); err != nil {
			log.Printf("Error saving tracker state: %v", err)
		}

		processedCount++
		return nil
	})

	if err != nil {
		log.Fatalf("Directory walk aborted: %v", err)
	}

	log.Printf("\nDone! Processed: %d, Skipped: %d, Failed: %d\n", processedCount, skippedCount, failedCount)
}
