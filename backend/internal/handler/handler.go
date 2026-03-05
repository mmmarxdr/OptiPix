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
	cfg *config.Config
}

func New(cfg *config.Config) *Handler {
	return &Handler{cfg: cfg}
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

func (h *Handler) Optimize(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	opts := parseImageOptions(r)

	result, err := processor.OptimizeImage(data, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	savings := "0"
	if result.OriginalSize > 0 {
		pct := float64(result.OriginalSize-result.OutputSize) / float64(result.OriginalSize) * 100
		savings = fmt.Sprintf("%.1f", pct)
	}

	filename := replaceExt(header.Filename, "."+string(opts.Format))

	if r.URL.Query().Get("response") == "json" {
		writeJSON(w, http.StatusOK, OptimizeResponse{
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

	w.Header().Set("Content-Type", result.MimeType)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("X-Original-Size", strconv.Itoa(result.OriginalSize))
	w.Header().Set("X-Output-Size", strconv.Itoa(result.OutputSize))
	w.Header().Set("X-Savings-Percent", savings)
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Original-Size, X-Output-Size, X-Savings-Percent")

	w.WriteHeader(http.StatusOK)
	w.Write(result.Data)
}

func (h *Handler) OptimizeSVG(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(h.cfg.MaxUploadSize); err != nil {
		writeError(w, http.StatusBadRequest, "failed to parse multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	opts := parseSVGOptions(r)

	result, err := processor.OptimizeSVG(data, opts, h.cfg.TempDir, h.cfg.SVGOPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	savings := "0"
	if result.OriginalSize > 0 {
		pct := float64(result.OriginalSize-result.OutputSize) / float64(result.OriginalSize) * 100
		savings = fmt.Sprintf("%.1f", pct)
	}

	if r.URL.Query().Get("response") == "json" {
		writeJSON(w, http.StatusOK, OptimizeResponse{
			Filename:     header.Filename,
			OriginalSize: result.OriginalSize,
			OutputSize:   result.OutputSize,
			Savings:      savings,
			Format:       "svg",
			MimeType:     "image/svg+xml",
		})
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", header.Filename))
	w.Header().Set("X-Original-Size", strconv.Itoa(result.OriginalSize))
	w.Header().Set("X-Output-Size", strconv.Itoa(result.OutputSize))
	w.Header().Set("X-Savings-Percent", savings)
	w.Header().Set("Access-Control-Expose-Headers", "Content-Disposition, X-Original-Size, X-Output-Size, X-Savings-Percent")

	w.WriteHeader(http.StatusOK)
	w.Write(result.Data)
}

func (h *Handler) BatchOptimize(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusNotImplemented, ErrorResponse{Error: "Not implemented"})
}

func parseImageOptions(r *http.Request) processor.ImageOptions {
	opts := processor.DefaultOptions()

	if f := r.FormValue("format"); f != "" {
		if parsed, err := processor.ParseFormat(f); err == nil {
			opts.Format = parsed
		}
	}
	if q := r.FormValue("quality"); q != "" {
		if val, err := strconv.Atoi(q); err == nil {
			opts.Quality = val
		}
	}
	if w := r.FormValue("width"); w != "" {
		if val, err := strconv.Atoi(w); err == nil {
			opts.Width = val
		}
	}
	if h := r.FormValue("height"); h != "" {
		if val, err := strconv.Atoi(h); err == nil {
			opts.Height = val
		}
	}
	if sm := r.FormValue("strip_metadata"); sm != "" {
		opts.StripMetadata = (sm == "true" || sm == "1")
	}
	if l := r.FormValue("lossless"); l != "" {
		opts.Lossless = (l == "true" || l == "1")
	}
	
	if e := r.FormValue("effort"); e != "" {
		if val, err := strconv.Atoi(e); err == nil {
			opts.Effort = val
		}
	}
	return opts
}

func parseSVGOptions(r *http.Request) processor.SVGOptions {
	opts := processor.DefaultSVGOptions()

	if mp := r.FormValue("multipass"); mp != "" {
		opts.Multipass = (mp == "true" || mp == "1")
	}
	if p := r.FormValue("precision"); p != "" {
		if val, err := strconv.Atoi(p); err == nil {
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
