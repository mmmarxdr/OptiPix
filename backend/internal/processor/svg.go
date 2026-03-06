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
	// Usamos "-" que en Unix significa "Leer de Stdin / Escribir a Stdout"
	args := []string{"-", "-o", "-", fmt.Sprintf("--precision=%d", opts.Precision)}
	if opts.Multipass {
		args = append(args, "--multipass")
	}

	// Usamos CommandContext para que el proceso de Node.js se mate automáticamente
	// si el cliente web cancela la petición (ctx.Done())
	cmd := exec.CommandContext(ctx, svgoPath, args...)

	// El input (bytes) se conecta a la "entrada" estándar (stdin)
	cmd.Stdin = bytes.NewReader(input)

	// Preparamos un buffer en memoria para recolectar la salida
	var out bytes.Buffer
	cmd.Stdout = &out

	// Y otro para recolectar errores en caso de que falle
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("svgo failed: %w, stderr: %s", err, stderr.String())
	}

	// Extraemos los bytes optimizados directamente desde la memoria
	outputData := out.Bytes()

	return &SVGResult{
		Data:         outputData,
		OriginalSize: len(input),
		OutputSize:   len(outputData),
	}, nil
}
