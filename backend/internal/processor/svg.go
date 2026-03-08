package processor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
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

func OptimizeSVG(ctx context.Context, input []byte, opts SVGOptions, svgoPath string) (*SVGResult, error) {
	args := []string{"-", "-o", "-", fmt.Sprintf("--precision=%d", opts.Precision)}
	if opts.Multipass {
		args = append(args, "--multipass")
	}

	cmd := exec.CommandContext(ctx, svgoPath, args...)

	cmd.Stdin = bytes.NewReader(input)

	var out bytes.Buffer
	cmd.Stdout = &out

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("svgo failed: %w, stderr: %s", err, stderr.String())
	}

	outputData := out.Bytes()

	return &SVGResult{
		Data:         outputData,
		OriginalSize: len(input),
		OutputSize:   len(outputData),
	}, nil
}
