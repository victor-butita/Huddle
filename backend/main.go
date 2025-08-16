package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

type spaHandler struct{ staticPath, indexPath string }

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(h.staticPath, r.URL.Path)
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, path)
}

func main() {
	log.Println("-----------------------------------------")
	log.Println("Initializing Huddle Server...")
	log.Println("-----------------------------------------")
	godotenv.Load()
	initDB("./huddle.db")
	hub := newHub()
	go hub.run()
	r := mux.NewRouter()
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/ai", handleAIRequest).Methods("POST")
	r.HandleFunc("/ws/{boardId}", func(w http.ResponseWriter, r *http.Request) { serveWs(hub, w, r) })
	spa := spaHandler{staticPath: "../frontend", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("[INFO] Server starting on http://localhost:%s", port)
	srv := &http.Server{Handler: r, Addr: ":" + port, WriteTimeout: 15 * time.Second, ReadTimeout: 15 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("[FATAL] ListenAndServe error: %v", err)
	}
}
