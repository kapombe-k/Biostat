package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"path/filepath"
	"runtime/debug"
	"time"

	"github.com/kapombe-k/Biostat/internal/config"
	"github.com/kapombe-k/Biostat/internal/services"
	"github.com/kapombe-k/Biostat/internal/templates"
)

// Server wires HTTP routes, HTML/HTMX templates, static assets, and domain services.
type Server struct {
	cfg       *config.Config
	explorer  *services.ExplorerService
	blast     *services.BlastManager
	renderer  *templates.Renderer
	logger    *slog.Logger
	startedAt time.Time
	staticDir string
	mux       *http.ServeMux
}

// NewServer constructs and configures all BioScope routes using Go 1.22+ standard library pattern routing.
func NewServer(
	cfg *config.Config,
	explorer *services.ExplorerService,
	blast *services.BlastManager,
	renderer *templates.Renderer,
	staticDir string,
	logger *slog.Logger,
) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	if staticDir == "" {
		staticDir = "static"
	}

	s := &Server{
		cfg:       cfg,
		explorer:  explorer,
		blast:     blast,
		renderer:  renderer,
		logger:    logger,
		startedAt: time.Now().UTC(),
		staticDir: staticDir,
		mux:       http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	// Health & observability
	s.mux.HandleFunc("GET /healthz", s.handleHealthz)

	// Pinned local static assets (HTMX, Mol*, 3Dmol.js, CSS, JS)
	fs := http.FileServer(http.Dir(filepath.Clean(s.staticDir)))
	s.mux.Handle("GET /static/", http.StripPrefix("/static/", cacheStaticMiddleware(fs)))

	// Primary pages & HTMX workspace partials
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /partials/search", s.handleSearchPartial)
	s.mux.HandleFunc("GET /explore", s.handleExplore)
	s.mux.HandleFunc("GET /partials/sequence", s.handleSequencePartial)

	// Structure coordinates & MolViewSpec JSON color/tooltip annotations
	s.mux.HandleFunc("GET /api/structures/{id}/coordinates", s.handleStructureCoordinates)
	s.mux.HandleFunc("GET /api/structures/{id}/annotations", s.handleStructureAnnotations)

	// Asynchronous NCBI BLAST & classroom similarity search
	s.mux.HandleFunc("POST /partials/blast/submit", s.handleBlastSubmit)
	s.mux.HandleFunc("GET /partials/blast/status/{jobID}", s.handleBlastStatus)

	// Guided educational curriculum & interactive quizzes
	s.mux.HandleFunc("GET /lessons", s.handleLessonsIndex)
	s.mux.HandleFunc("GET /lessons/{slug}", s.handleLessonView)
	s.mux.HandleFunc("GET /lessons/{slug}/step/{step}", s.handleLessonStep)
	s.mux.HandleFunc("POST /lessons/{slug}/quiz/{questionID}", s.handleLessonQuiz)

	// Reproducible exports
	s.mux.HandleFunc("GET /api/export/json", s.handleExportJSON)
	s.mux.HandleFunc("GET /api/export/fasta", s.handleExportFASTA)
}

// Handler returns the root http.Handler wrapped with panic recovery, security headers, and structured logging.
func (s *Server) Handler() http.Handler {
	return s.recoverMiddleware(s.securityHeadersMiddleware(s.loggingMiddleware(s.mux)))
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	resp := map[string]any{
		"status":     "ok",
		"service":    "bioscope",
		"version":    "1.0.0",
		"uptime_sec": int(time.Since(s.startedAt).Seconds()),
		"cache":      s.explorer.CacheStats(),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Permissive iframe embedding for Hugging Face Spaces & Arena Live Preview, while restricting scripts/connections strictly to same-origin + WebGL blobs.
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' blob:; worker-src 'self' blob:; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; connect-src 'self' data: blob:; font-src 'self' data:;")
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (sw *statusWriter) WriteHeader(code int) {
	sw.status = code
	sw.ResponseWriter.WriteHeader(code)
}

func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		s.logger.Info("http_request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", sw.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				s.logger.Error("panic recovered", "err", rec, "stack", string(debug.Stack()))
				http.Error(w, "Internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func cacheStaticMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		next.ServeHTTP(w, r)
	})
}
