package main

import (
	"encoding/json"
	"net/http"
	"os"
)

type AIRequest struct{ Action, Code, Lang string }

func handleAIRequest(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		http.Error(w, `{"error": "AI features are disabled. No API key configured on the server."}`, http.StatusNotImplemented)
		return
	}
	var req AIRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var result string
	var err error
	switch req.Action {
	case "analyze":
		result, err = analyzeCode(apiKey, req.Code, req.Lang)
	case "refactor":
		result, err = refactorCode(apiKey, req.Code, req.Lang)
	case "add_comments":
		result, err = addCommentsToCode(apiKey, req.Code, req.Lang)
	default:
		http.Error(w, "Invalid AI action", http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"result": result})
}
