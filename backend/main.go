package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
)

// spaHandler correctly serves the Single Page Application.
type spaHandler struct {
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	path = filepath.Join(h.staticPath, path)

	_, err = os.Stat(path)
	if os.IsNotExist(err) || strings.HasSuffix(path, "/") {
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
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
	r.HandleFunc("/ws/{boardId}", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	spa := spaHandler{staticPath: "../frontend", indexPath: "index.html"}
	r.PathPrefix("/").Handler(spa)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("[INFO] Server starting on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("[FATAL] ListenAndServe error: %v", err)
	}
}
