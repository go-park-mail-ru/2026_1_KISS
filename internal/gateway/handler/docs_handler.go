package handler

import (
	"io/fs"
	"net/http"

	"github.com/go-park-mail-ru/2026_1_KISS/docs"
)

type DocsHandler struct {
	fileServer http.Handler
}

func NewDocsHandler() *DocsHandler {
	uiFS, _ := fs.Sub(docs.Assets, "swagger-ui")

	return &DocsHandler{
		fileServer: http.FileServer(http.FS(uiFS)),
	}
}

func (h *DocsHandler) RegisterRoutes(mux *http.ServeMux) {
	docsCSP := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; script-src 'self'")
			next.ServeHTTP(w, r)
		})
	}

	mux.Handle("GET /api/v1/docs/swagger.json", docsCSP(http.HandlerFunc(h.serveSpec)))
	mux.Handle("GET /api/v1/docs/", docsCSP(http.StripPrefix("/api/v1/docs/", h.fileServer)))
	mux.Handle("GET /api/v1/docs", docsCSP(http.HandlerFunc(h.redirectToDocs)))
}

func (h *DocsHandler) redirectToDocs(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/api/v1/docs/", http.StatusMovedPermanently)
}

func (h *DocsHandler) serveSpec(w http.ResponseWriter, r *http.Request) {
	data, _ := docs.Assets.ReadFile("swagger.json")
	w.Header().Set("Content-Type", "application/json")
	w.Write(data)
}
