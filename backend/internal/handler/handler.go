package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/optipix/backend/internal/config"
	"github.com/optipix/backend/internal/processor"
)

type Handler struct {
	cfg       *config.Config
	semaphore chan struct{}
}

func New(cfg *config.Config) *Handler {
	return &Handler{
		cfg:       cfg,
		semaphore: make(chan struct{}, cfg.MaxConcurrency),
	}
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}

type OptimizeResponse struct {
	Filename     string `json:"filename"`
	OriginalSize int    `json:"original_size"`
	OutputSize   int    `json:"output_size"`
	Savings      string `json:"savings_percent"`
	Format       string `json:"format"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	MimeType     string `json:"mime_type"`
}

type HealthResponse struct {
	Status           string   `json:"status"`
	Version          string   `json:"version"`
	SupportedFormats []string `json:"supported_formats"`
}

type FormatsResponse struct {
	Input  []string `json:"input"`
	Output []string `json:"output"`
}

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:           "ok",
		Version:          "1.0.0",
		SupportedFormats: processor.SupportedFormats(),
	})
}

func (h *Handler) Formats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, FormatsResponse{
		Input:  processor.SupportedInputFormats(),
		Output: processor.SupportedFormats(),
	})
}

func (handler *Handler) Optimize(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseMultipartForm(handler.cfg.MaxUploadSize); err != nil {
		writeError(writer, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Adquirimos el "ticket" del semáforo. Si se cancela la petición antes de que
	// haya un hueco libre, salimos e interrumpimos el proceso.
	select {
	case handler.semaphore <- struct{}{}:
		defer func() { <-handler.semaphore }()
	case <-request.Context().Done():
		writeError(writer, http.StatusRequestTimeout, "request cancelled")
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "failed to read file")
		return
	}

	opts := parseImageOptions(request)

	result, err := processor.OptimizeImage(request.Context(), data, opts)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	savings := "0"
	if result.OriginalSize > 0 {
		pct := float64(result.OriginalSize-result.OutputSize) / float64(result.OriginalSize) * 100
		savings = fmt.Sprintf("%.1f", pct)
	}

	filename := replaceExt(header.Filename, "."+string(opts.Format))

	if request.URL.Query().Get("response") == "json" {
		writeJSON(writer, http.StatusOK, OptimizeResponse{
			Filename:     filename,
			OriginalSize: result.OriginalSize,
			OutputSize:   result.OutputSize,
			Savings:      savings,
			Format:       string(opts.Format),
			Width:        result.Width,
			Height:       result.Height,
			MimeType:     result.MimeType,
		})
		return
	}

	writer.Header().Set("Content-Type", result.MimeType)
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	writer.Header().Set("X-Original-Size", strconv.Itoa(result.OriginalSize))
	writer.Header().Set("X-Output-Size", strconv.Itoa(result.OutputSize))
	writer.Header().Set("X-Savings-Percent", savings)
	writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Original-Size, X-Output-Size, X-Savings-Percent")
	writer.WriteHeader(http.StatusOK)
	writer.Write(result.Data)
}

func (h *Handler) OptimizeSVG(writer http.ResponseWriter, request *http.Request) {
	if err := request.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
		writeError(writer, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	// Hacemos exactamente lo mismo para SVGO: control concurrente con contexto.
	select {
	case h.semaphore <- struct{}{}:
		defer func() { <-h.semaphore }()
	case <-request.Context().Done():
		writeError(writer, http.StatusRequestTimeout, "request cancelled")
		return
	}

	file, header, err := request.FormFile("file")
	if err != nil {
		writeError(writer, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, "failed to read file")
		return
	}

	opts := parseSVGOptions(request)

	result, err := processor.OptimizeSVG(request.Context(), data, opts, h.cfg.SVGOPath)
	if err != nil {
		writeError(writer, http.StatusInternalServerError, err.Error())
		return
	}

	savings := "0"
	if result.OriginalSize > 0 {
		pct := float64(result.OriginalSize-result.OutputSize) / float64(result.OriginalSize) * 100
		savings = fmt.Sprintf("%.1f", pct)
	}

	if request.URL.Query().Get("response") == "json" {
		writeJSON(writer, http.StatusOK, OptimizeResponse{
			Filename:     header.Filename,
			OriginalSize: result.OriginalSize,
			OutputSize:   result.OutputSize,
			Savings:      savings,
			Format:       "svg",
			MimeType:     "image/svg+xml",
		})
		return
	}

	writer.Header().Set("Content-Type", "image/svg+xml")
	writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", header.Filename))
	writer.Header().Set("X-Original-Size", strconv.Itoa(result.OriginalSize))
	writer.Header().Set("X-Output-Size", strconv.Itoa(result.OutputSize))
	writer.Header().Set("X-Savings-Percent", savings)
	writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Original-Size, X-Output-Size, X-Savings-Percent")

	writer.WriteHeader(http.StatusOK)
	writer.Write(result.Data)
}

func (h *Handler) BatchOptimize(writer http.ResponseWriter, request *http.Request) {
	writeJSON(writer, http.StatusNotImplemented, ErrorResponse{Error: "Not implemented"})
}

func parseImageOptions(r *http.Request) processor.ImageOptions {
	opts := processor.DefaultOptions()

	if format := r.FormValue("format"); format != "" {
		if parsed, err := processor.ParseFormat(format); err == nil {
			opts.Format = parsed
		}
	}
	if quality := r.FormValue("quality"); quality != "" {
		if val, err := strconv.Atoi(quality); err == nil {
			opts.Quality = val
		}
	}
	if width := r.FormValue("width"); width != "" {
		if val, err := strconv.Atoi(width); err == nil {
			opts.Width = val
		}
	}
	if height := r.FormValue("height"); height != "" {
		if val, err := strconv.Atoi(height); err == nil {
			opts.Height = val
		}
	}
	if stripMetadata := r.FormValue("strip_metadata"); stripMetadata != "" {
		opts.StripMetadata = (stripMetadata == "true" || stripMetadata == "1")
	}
	if lossless := r.FormValue("lossless"); lossless != "" {
		opts.Lossless = (lossless == "true" || lossless == "1")
	}

	if effort := r.FormValue("effort"); effort != "" {
		if val, err := strconv.Atoi(effort); err == nil {
			opts.Effort = val
		}
	}
	return opts
}

func parseSVGOptions(r *http.Request) processor.SVGOptions {
	opts := processor.DefaultSVGOptions()

	if multipass := r.FormValue("multipass"); multipass != "" {
		opts.Multipass = (multipass == "true" || multipass == "1")
	}
	if precision := r.FormValue("precision"); precision != "" {
		if val, err := strconv.Atoi(precision); err == nil {
			opts.Precision = val
		}
	}
	return opts
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, ErrorResponse{Error: msg})
}

func replaceExt(name, newExt string) string {
	ext := filepath.Ext(name)
	if ext == "" {
		return name + newExt
	}
	return strings.TrimSuffix(name, ext) + newExt
}
