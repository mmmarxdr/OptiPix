// Package strategy implements the per-file output-format decision engine.
// Given an input file path and the value of the --format flag, Resolve returns
// a Decision that describes whether the file should be processed, skipped, or
// routed through SVGO (for SVG files), and what the output extension will be.
package strategy

import (
	"path/filepath"
	"strings"

	"github.com/optipix/backend/internal/processor"
)

// Decision is the result of format resolution for a single input file.
type Decision struct {
	// IsSVG is true when the input is an SVG file; in that case OutputFormat
	// is zero and the file should be routed through processor.OptimizeSVG.
	IsSVG bool
	// OutputFormat is the target image format. Zero value when IsSVG or Skip.
	OutputFormat processor.OutputFormat
	// OutputExt is the file extension for the output file, including the leading
	// dot (e.g. ".webp"). Empty when Skip is true and IsSVG is false.
	OutputExt string
	// Skip is true when the file should not be processed (already optimal, or
	// unsupported extension, or invalid --format value).
	Skip bool
	// NeedsAlphaCheck is true when the caller must inspect the image data to
	// determine whether an alpha channel is present. When the check confirms
	// alpha, ImageOptions.Lossless must be set to true (WebP lossless).
	// Currently set only for PNG → WebP conversions under "auto" mode.
	NeedsAlphaCheck bool
	// Reason is a human-readable explanation used for verbose/dry-run output.
	Reason string
}

// autoFormatEntry describes an entry in autoFormatMap.
type autoFormatEntry struct {
	format          processor.OutputFormat
	skip            bool
	needsAlphaCheck bool
}

// autoFormatMap maps lower-cased input extensions to their preferred output
// format under the "auto" heuristic (spec Domain 4).
//
// Extensions that map to themselves (e.g. .webp → WebP) are listed so that
// Resolve can return Skip=true for already-optimal files.
//
// PNG entries have needsAlphaCheck=true so the caller can select lossless
// WebP when an alpha channel is detected.
var autoFormatMap = map[string]autoFormatEntry{
	".jpg":  {format: processor.FormatWebP},
	".jpeg": {format: processor.FormatWebP},
	".png":  {format: processor.FormatWebP, needsAlphaCheck: true},
	// Already-optimal formats → skip.
	".webp": {skip: true},
	".avif": {skip: true},
}

// Resolve determines the output format and extension for inputPath given the
// value of the --format flag. formatFlag must be one of:
//
//	"auto" | "webp" | "avif" | "jpeg" | "png"
//
// For SVG inputs, the Decision always has IsSVG=true regardless of formatFlag.
// For "auto", the auto-format heuristic table is applied. For an explicit
// format, processor.ParseFormat is called; an unrecognised value returns
// Decision{Skip: true}.
func Resolve(inputPath string, formatFlag string) Decision {
	ext := strings.ToLower(filepath.Ext(inputPath))

	// SVG files are always routed through SVGO, regardless of --format.
	if ext == ".svg" {
		return Decision{
			IsSVG:     true,
			OutputExt: ".svg",
			Reason:    "SVG: optimize with SVGO in-place",
		}
	}

	if formatFlag == "auto" {
		entry, ok := autoFormatMap[ext]
		if !ok {
			return Decision{
				Skip:   true,
				Reason: "unsupported input extension: " + ext,
			}
		}
		if entry.skip {
			return Decision{
				Skip:   true,
				Reason: "already optimal format, skipping: " + ext,
			}
		}
		return Decision{
			OutputFormat:    entry.format,
			OutputExt:       "." + string(entry.format),
			NeedsAlphaCheck: entry.needsAlphaCheck,
			Reason:          ext + " → ." + string(entry.format) + " (auto)",
		}
	}

	// Explicit format flag.
	outFmt, err := processor.ParseFormat(formatFlag)
	if err != nil {
		return Decision{
			Skip:   true,
			Reason: "invalid --format value: " + formatFlag,
		}
	}
	return Decision{
		OutputFormat: outFmt,
		OutputExt:    "." + string(outFmt),
		Reason:       ext + " → ." + string(outFmt) + " (explicit)",
	}
}
