package processor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/google/uuid"
)

type SVGOptions struct {
	Multipass     bool
	RemoveTitle   bool
	RemoveDesc    bool
	RemoveViewBox bool
	Precision     int
}

type SVGResult struct {
	Data         []byte
	OriginalSize int
	OutputSize   int
}

func DefaultSVGOptions() SVGOptions {
	return SVGOptions{
		Multipass: true,
		Precision: 3,
	}
}

func OptimizeSVG(input []byte, opts SVGOptions, tempDir, svgoPath string) (*SVGResult, error) {
	id := uuid.New().String()
	inPath := filepath.Join(tempDir, fmt.Sprintf("optipix-svg-in-%s.svg", id))
	outPath := filepath.Join(tempDir, fmt.Sprintf("optipix-svg-out-%s.svg", id))

	defer os.Remove(inPath)
	defer os.Remove(outPath)

	if err := os.WriteFile(inPath, input, 0600); err != nil {
		return nil, fmt.Errorf("failed to write input svg: %w", err)
	}

	args := []string{inPath, "-o", outPath, fmt.Sprintf("--precision=%d", opts.Precision)}
	if opts.Multipass {
		args = append(args, "--multipass")
	}

	cmd := exec.Command(svgoPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("svgo failed: %w, stderr: %s", err, stderr.String())
	}

	outputData, err := os.ReadFile(outPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read output svg: %w", err)
	}

	return &SVGResult{
		Data:         outputData,
		OriginalSize: len(input),
		OutputSize:   len(outputData),
	}, nil
}
