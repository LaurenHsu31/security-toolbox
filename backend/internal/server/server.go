package server

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"net/http"
	"strings"

	"security-toolbox/internal/tools"
)

const maxBody = 8 << 20 // 8 MiB is plenty for certs/tokens/JSON

// New builds the HTTP handler. staticFS is the embedded, built frontend.
func New(staticFS fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("/api/v1/tools", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, tools.All())
	})

	// POST /api/v1/run/{tool}
	mux.HandleFunc("/api/v1/run/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeErr(w, http.StatusMethodNotAllowed, "use POST")
			return
		}
		name := strings.TrimPrefix(r.URL.Path, "/api/v1/run/")
		body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
		if err != nil {
			writeErr(w, http.StatusBadRequest, "cannot read body")
			return
		}
		out, err := tools.Run(name, body)
		if err != nil {
			if errors.Is(err, tools.ErrUnknownTool) {
				writeErr(w, http.StatusNotFound, err.Error())
				return
			}
			// User-facing parse errors are 422, not 500.
			writeErr(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": out})
	})

	// SPA static files with index.html fallback for client-side routing.
	fileServer := http.FileServer(http.FS(staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := fs.Stat(staticFS, strings.TrimPrefix(r.URL.Path, "/")); err != nil && r.URL.Path != "/" {
			// Not a real file -> serve index.html (SPA route).
			r2 := r.Clone(r.Context())
			r2.URL.Path = "/"
			fileServer.ServeHTTP(w, r2)
			return
		}
		fileServer.ServeHTTP(w, r)
	})

	return securityHeaders(mux)
}

// securityHeaders enforces a Content-Security-Policy that blocks any outbound
// connection. This is the technical guarantee behind "your keys never leave
// this machine": the page can only talk to its own origin.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; connect-src 'self'; img-src 'self' data:; "+
				"style-src 'self' 'unsafe-inline'; script-src 'self'; "+
				"object-src 'none'; base-uri 'none'; frame-ancestors 'none'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("Cross-Origin-Opener-Policy", "same-origin")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]any{"ok": false, "error": msg})
}
