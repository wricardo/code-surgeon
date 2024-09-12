package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// SearchForFunctionRequest represents the expected request structure for searching a function.
type SearchForFunctionRequest struct {
	FunctionName string `json:"function_name"`
	Path         string `json:"path"`
	Receiver     string `json:"receiver"`
}

// SearchForFunctionResponse represents the response structure for a function search.
type SearchForFunctionResponse struct {
	FilePath string `json:"filepath"`
}

// UpsertDocumentationToFunctionRequest represents the expected request structure for upserting documentation.
type UpsertDocumentationToFunctionRequest struct {
	Documentation string `json:"documentation"`
	FilePath      string `json:"filepath"`
	FunctionName  string `json:"function_name"`
	Receiver      string `json:"receiver"`
}

// UpsertDocumentationToFunctionResponse represents the response structure for upserting documentation.
type UpsertDocumentationToFunctionResponse struct {
	OK bool `json:"ok"`
}

// searchForFunctionHandler handles the SearchForFunction endpoint.
func searchForFunctionHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("searchForFunctionHandler")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	respBody := map[string]string{
		"filepath": "path/to/file.go",
	}
	if err := json.NewEncoder(w).Encode(respBody); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}

}

func main() {
	http.HandleFunc("/", searchForFunctionHandler)

	if err := http.ListenAndServe(":8002", nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
