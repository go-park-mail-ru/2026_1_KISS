package main

import (
	"log"
	"net/http"
	"time"

	"github.com/go-park-mail-ru/2026_1_KISS/internal/handlers"
	"github.com/go-park-mail-ru/2026_1_KISS/internal/middleware"
)

func main() {

	userHandler := handlers.NewUserHandler()
	fileHandler := handlers.NewFileHandler()

	mux := http.NewServeMux()

	mux.HandleFunc("POST /api/register", userHandler.Register)
	mux.HandleFunc("POST /api/login", userHandler.Login)
	mux.HandleFunc("POST /api/logout", userHandler.Logout)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, r *http.Request) {
		handlers.RespondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/user/me", middleware.Auth(userHandler.GetMe))

	mux.HandleFunc("GET /api/files", middleware.Auth(fileHandler.ListFiles))
	mux.HandleFunc("POST /api/files", middleware.Auth(fileHandler.CreateFile))
	mux.HandleFunc("GET /api/files/{id}", middleware.Auth(fileHandler.GetFile))
	mux.HandleFunc("DELETE /api/files/{id}", middleware.Auth(fileHandler.DeleteFile))

	mux.HandleFunc("POST /api/files/{id}/cells", middleware.Auth(fileHandler.CreateCell))
	mux.HandleFunc("PUT /api/files/{id}/cells/{cellId}", middleware.Auth(fileHandler.UpdateCell))
	mux.HandleFunc("DELETE /api/files/{id}/cells/{cellId}", middleware.Auth(fileHandler.DeleteCell))

	handlerWithCORS := middleware.CORS(mux)

	// http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	fmt.Fprintf(w, "OK")
	// })

	srv := &http.Server{
		Addr:         ":8080",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		// Handler:      http.DefaultServeMux,
		Handler: handlerWithCORS,
	}

	log.Println("Server starting on http://localhost:8080")
	log.Println("Available endpoints:")
	log.Println("  POST   /api/register")
	log.Println("  POST   /api/login")
	log.Println("  POST   /api/logout")
	log.Println("  GET    /api/user/me")
	log.Println("  GET    /api/health")
	log.Println("  GET    /api/files")
	log.Println("  POST   /api/files")
	log.Println("  GET    /api/files/{id}")
	log.Println("  DELETE /api/files/{id}")
	log.Println("  POST   /api/files/{id}/cells")
	log.Println("  PUT    /api/files/{id}/cells/{cellId}")
	log.Println("  DELETE /api/files/{id}/cells/{cellId}")

	log.Fatal(srv.ListenAndServe())
}
