package strategy_test

import (
	"testing"

	"github.com/optipix/backend/internal/processor"
	"github.com/optipix/backend/internal/strategy"
)

func TestResolve(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		input          string
		format         string
		wantFmt        processor.OutputFormat
		wantSkip       bool
		wantIsSVG      bool
		wantExtLen     bool // just check OutputExt is non-empty
		wantAlphaCheck bool // NeedsAlphaCheck expected value
	}{
		{
			name:  "jpeg auto → webp",
			input: "photo.jpg", format: "auto",
			wantFmt: processor.FormatWebP, wantExtLen: true,
		},
		{
			name:  "JPEG uppercase .JPG auto → webp",
			input: "photo.JPG", format: "auto",
			wantFmt: processor.FormatWebP, wantExtLen: true,
		},
		{
			name:  "jpeg extension auto → webp",
			input: "photo.jpeg", format: "auto",
			wantFmt: processor.FormatWebP, wantExtLen: true,
		},
		{
			name:  "png auto → webp with alpha check",
			input: "icon.png", format: "auto",
			wantFmt: processor.FormatWebP, wantExtLen: true, wantAlphaCheck: true,
		},
		{
			name:  "PNG uppercase .PNG auto → webp with alpha check",
			input: "icon.PNG", format: "auto",
			wantFmt: processor.FormatWebP, wantExtLen: true, wantAlphaCheck: true,
		},
		{
			name:  "webp auto → skip already optimal",
			input: "logo.webp", format: "auto",
			wantSkip: true,
		},
		{
			name:  "avif auto → skip already optimal",
			input: "hero.avif", format: "auto",
			wantSkip: true,
		},
		{
			name:  "svg auto → isSVG",
			input: "icon.svg", format: "auto",
			wantIsSVG: true,
		},
		{
			name:  "SVG uppercase .SVG auto → isSVG",
			input: "icon.SVG", format: "auto",
			wantIsSVG: true,
		},
		{
			name:  "gif auto → skip unsupported",
			input: "anim.gif", format: "auto",
			wantSkip: true,
		},
		{
			name:  "explicit avif on jpg",
			input: "photo.jpg", format: "avif",
			wantFmt: processor.FormatAVIF, wantExtLen: true,
		},
		{
			name:  "explicit png on jpg",
			input: "photo.jpg", format: "png",
			wantFmt: processor.FormatPNG, wantExtLen: true,
		},
		{
			name:  "explicit webp on jpg",
			input: "photo.jpg", format: "webp",
			wantFmt: processor.FormatWebP, wantExtLen: true,
		},
		{
			name:  "explicit jpeg on png — no alpha check (explicit format)",
			input: "banner.png", format: "jpeg",
			wantFmt: processor.FormatJPEG, wantExtLen: true,
		},
		{
			name:  "invalid format bmp → skip",
			input: "photo.jpg", format: "bmp",
			wantSkip: true,
		},
		{
			name:  "tiff auto → skip unsupported",
			input: "image.tiff", format: "auto",
			wantSkip: true,
		},
		{
			name:  "bmp auto → skip unsupported",
			input: "image.bmp", format: "auto",
			wantSkip: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := strategy.Resolve(tc.input, tc.format)

			if got.Skip != tc.wantSkip {
				t.Errorf("Skip: got %v, want %v (Reason: %s)", got.Skip, tc.wantSkip, got.Reason)
			}
			if got.IsSVG != tc.wantIsSVG {
				t.Errorf("IsSVG: got %v, want %v", got.IsSVG, tc.wantIsSVG)
			}
			if !tc.wantSkip && !tc.wantIsSVG {
				if got.OutputFormat != tc.wantFmt {
					t.Errorf("OutputFormat: got %q, want %q", got.OutputFormat, tc.wantFmt)
				}
			}
			if tc.wantExtLen && got.OutputExt == "" {
				t.Error("OutputExt should be non-empty for a processed file")
			}
			if tc.wantIsSVG && got.OutputExt != ".svg" {
				t.Errorf("OutputExt for SVG: got %q, want .svg", got.OutputExt)
			}
			if got.NeedsAlphaCheck != tc.wantAlphaCheck {
				t.Errorf("NeedsAlphaCheck: got %v, want %v", got.NeedsAlphaCheck, tc.wantAlphaCheck)
			}
		})
	}
}

// TestResolve_ExtensionCaseInsensitive verifies that extension matching is
// case-insensitive for all common input types.
func TestResolve_ExtensionCaseInsensitive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		input     string
		format    string
		wantSkip  bool
		wantIsSVG bool
		wantFmt   processor.OutputFormat
	}{
		{".JPG", "auto", false, false, processor.FormatWebP},
		{".PNG", "auto", false, false, processor.FormatWebP},
		{".SVG", "auto", false, true, ""},
		{".WEBP", "auto", true, false, ""},
		{".AVIF", "auto", true, false, ""},
	}

	for _, tc := range cases {
		tc := tc
		t.Run("uppercase"+tc.input, func(t *testing.T) {
			t.Parallel()
			// Create a fake filename with uppercase extension.
			got := strategy.Resolve("image"+tc.input, tc.format)
			if got.Skip != tc.wantSkip {
				t.Errorf("input %s: Skip got %v want %v (Reason: %s)", tc.input, got.Skip, tc.wantSkip, got.Reason)
			}
			if got.IsSVG != tc.wantIsSVG {
				t.Errorf("input %s: IsSVG got %v want %v", tc.input, got.IsSVG, tc.wantIsSVG)
			}
			if !tc.wantSkip && !tc.wantIsSVG && got.OutputFormat != tc.wantFmt {
				t.Errorf("input %s: OutputFormat got %q want %q", tc.input, got.OutputFormat, tc.wantFmt)
			}
		})
	}
}
