package processor

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/davidbyttow/govips/v2/vips"
)

type OutputFormat string

const (
	FormatWebP OutputFormat = "webp"
	FormatAVIF OutputFormat = "avif"
	FormatJPEG OutputFormat = "jpeg"
	FormatPNG  OutputFormat = "png"
)

type ImageOptions struct {
	Format        OutputFormat
	Quality       int
	Width         int
	Height        int
	StripMetadata bool
	Lossless      bool
	Effort        int
}

type OptimizeResult struct {
	Data         []byte
	Format       OutputFormat
	OriginalSize int
	OutputSize   int
	Width        int
	Height       int
	MimeType     string
}

func InitVips() {
	vips.Startup(nil)
}

func ShutdownVips() {
	vips.Shutdown()
}

func DefaultOptions() ImageOptions {
	return ImageOptions{
		Format:        FormatWebP,
		Quality:       80,
		StripMetadata: true,
		Effort:        4,
	}
}

func OptimizeImage(ctx context.Context, input []byte, opts ImageOptions) (*OptimizeResult, error) {
	// Verificar si el cliente ya cerró la conexión antes de arrancar a procesar
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("request cancelled before image processing: %w", err)
	}

	img, err := vips.NewImageFromBuffer(input)
	if err != nil {
		return nil, fmt.Errorf("failed to load image: %w", err)
	}
	defer img.Close()

	if err := img.AutoRotate(); err != nil {
		return nil, fmt.Errorf("failed to auto-rotate: %w", err)
	}

	origW := img.Width()
	origH := img.Height()
	var scale float64 = 1.0

	if opts.Width > 0 && opts.Height > 0 {
		scaleW := float64(opts.Width) / float64(origW)
		scaleH := float64(opts.Height) / float64(origH)
		if scaleW < scaleH {
			scale = scaleW
		} else {
			scale = scaleH
		}
	} else if opts.Width > 0 {
		scale = float64(opts.Width) / float64(origW)
	} else if opts.Height > 0 {
		scale = float64(opts.Height) / float64(origH)
	}

	if scale < 1.0 {
		if err := img.Resize(scale, vips.KernelLanczos3); err != nil {
			return nil, fmt.Errorf("failed to resize: %w", err)
		}
	}

	// For govips, removing metadata is usually done via a method or during export strips
	if opts.StripMetadata {
		// govips might not export Get/RemoveMetadata correctly in all versions,
		// but typically it is handled correctly in the Export params.
		// Still, we can attempt to remove metadata buffers if govips exposes it.
		// Actually, setting it in ExportParams is usually enough.
	}

	var outputData []byte
	var outputError error

	switch opts.Format {
	case FormatWebP:
		p := vips.NewWebpExportParams()
		p.Quality = opts.Quality
		p.Lossless = opts.Lossless
		p.StripMetadata = opts.StripMetadata
		p.ReductionEffort = opts.Effort
		outputData, _, outputError = img.ExportWebp(p)
	case FormatAVIF:
		p := vips.NewAvifExportParams()
		p.Quality = opts.Quality
		p.Lossless = opts.Lossless
		// Govips defines explicit exported attributes, we configure what's standard.
		p.Speed = 9 - opts.Effort
		if p.Speed < 0 {
			p.Speed = 0
		}
		if p.Speed > 9 {
			p.Speed = 9
		}
		outputData, _, outputError = img.ExportAvif(p)
	case FormatJPEG:
		p := vips.NewJpegExportParams()
		p.Quality = opts.Quality
		p.StripMetadata = opts.StripMetadata
		p.OptimizeCoding = true
		p.Interlace = true
		p.TrellisQuant = true
		p.OvershootDeringing = true
		p.OptimizeScans = true
		outputData, _, outputError = img.ExportJpeg(p)
	case FormatPNG:
		p := vips.NewPngExportParams()
		p.Compression = 9
		// p.StripMetadata = opts.StripMetadata // StripMetadata might be on jpeg/webp
		p.Interlace = true
		outputData, _, outputError = img.ExportPng(p)
	default:
		return nil, errors.New("unsupported output format")
	}

	if outputError != nil {
		return nil, fmt.Errorf("failed to export: %w", outputError)
	}

	return &OptimizeResult{
		Data:         outputData,
		Format:       opts.Format,
		OriginalSize: len(input),
		OutputSize:   len(outputData),
		Width:        img.Width(),
		Height:       img.Height(),
		MimeType:     mimeForFormat(opts.Format),
	}, nil
}

func DetectFormat(data []byte) string {
	if len(data) > 12 && bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
		return "png"
	}
	if len(data) > 3 && bytes.HasPrefix(data, []byte("\xff\xd8\xff")) {
		return "jpeg"
	}
	return "unknown"
}

func ParseFormat(s string) (OutputFormat, error) {
	switch s {
	case "webp":
		return FormatWebP, nil
	case "avif":
		return FormatAVIF, nil
	case "jpeg", "jpg":
		return FormatJPEG, nil
	case "png":
		return FormatPNG, nil
	default:
		return FormatWebP, errors.New("unsupported format")
	}
}

func SupportedFormats() []string {
	return []string{"webp", "avif", "jpeg", "png"}
}

func SupportedInputFormats() []string {
	return []string{"jpeg", "jpg", "png", "webp", "gif", "tiff", "bmp", "heif", "avif", "svg"}
}

func mimeForFormat(f OutputFormat) string {
	switch f {
	case FormatWebP:
		return "image/webp"
	case FormatAVIF:
		return "image/avif"
	case FormatJPEG:
		return "image/jpeg"
	case FormatPNG:
		return "image/png"
	default:
		return "application/octet-stream"
	}
}
