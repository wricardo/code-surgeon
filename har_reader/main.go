package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// Define the structures for request and response
type HarEntry struct {
	Request      HarRequest  `json:"request"`
	Response     HarResponse `json:"response"`
	ResourceType string      `json:"_resourceType"`
}

type HarRequest struct {
	Method   string    `json:"method"`
	URL      string    `json:"url"`
	Headers  []Header  `json:"headers"`
	BodySize int       `json:"bodySize"`
	PostData *PostData `json:"postData"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type PostData struct {
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type HarResponse struct {
	Status   int             `json:"status"`
	Headers  []Header        `json:"headers"`
	BodySize int             `json:"bodySize"`
	Content  ResponseContent `json:"content"`
}

type ResponseContent struct {
	Size     int    `json:"size"`
	MimeType string `json:"mimeType"`
	Text     string `json:"text"`
}

type HarLog struct {
	Log HarLogEntries `json:"log"`
}

type HarLogEntries struct {
	Entries []HarEntry `json:"entries"`
}

func main() {
	// Define flags for file path and prefix match
	filePath := flag.String("file", "", "Path to the HAR file")
	prefixMatch := flag.String("prefix", "", "URL prefix to match")
	format := flag.String("format", "text", "Output format: text, json, json-small, jsonl, jsonl-small")
	excludeType := flag.String("exclude-type", "", "Comma-separated resource types to exclude (e.g., script,stylesheet,image,fetch,document,other)")

	// Parse command-line flags
	flag.Parse()

	// Open the JSON file specified by the --file flag
	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the JSON file into byte array
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the JSON data
	var harLog HarLog
	err = json.Unmarshal(data, &harLog)
	if err != nil {
		log.Fatal(err)
	}

	// Parse excluded types
	var excludedTypes map[string]bool
	if *excludeType != "" {
		excludedTypes = make(map[string]bool)
		for _, t := range strings.Split(*excludeType, ",") {
			excludedTypes[strings.TrimSpace(t)] = true
		}
	}

	// Filter entries based on prefix and excluded types
	var filteredEntries []HarEntry
	for _, entry := range harLog.Log.Entries {
		// Check prefix match
		if *prefixMatch != "" && !strings.HasPrefix(entry.Request.URL, *prefixMatch) {
			continue
		}
		// Check resource type exclusion
		if excludedTypes != nil && excludedTypes[entry.ResourceType] {
			continue
		}
		filteredEntries = append(filteredEntries, entry)
	}

	// Output based on format
	switch *format {
	case "json":
		outputJSON(filteredEntries, true)
	case "json-small":
		outputJSON(filteredEntries, false)
	case "jsonl":
		outputJSONL(filteredEntries)
	case "jsonl-small":
		outputJSONLSmall(filteredEntries)
	case "text":
		outputText(filteredEntries)
	default:
		log.Fatalf("Unknown format: %s (supported: text, json, json-small, jsonl, jsonl-small)", *format)
	}
}

func outputText(entries []HarEntry) {
	for i, entry := range entries {
		fmt.Printf("Entry %d:\n", i+1)
		fmt.Printf("Request Method: %s\n", entry.Request.Method)
		fmt.Printf("Request URL: %s\n", entry.Request.URL)

		if entry.Request.PostData != nil {
			if strings.HasPrefix(entry.Request.PostData.Text, "{") || strings.HasPrefix(entry.Request.PostData.Text, "[") {
				fmt.Printf("Request Body (PostData): %s\n", entry.Request.PostData.Text)
			} else if strings.HasPrefix(entry.Request.PostData.MimeType, "application/json") {
				log.Println("Request Body (PostData) is not a JSON object")
			}
		}

		fmt.Printf("Response Status: %d\n", entry.Response.Status)

		if entry.Response.Content.Text != "" {
			if strings.HasPrefix(entry.Response.Content.Text, "{") || strings.HasPrefix(entry.Response.Content.Text, "[") {
				fmt.Printf("Response Content: %s\n", entry.Response.Content.Text)
			} else if strings.HasPrefix(entry.Response.Content.MimeType, "application/json") {
				log.Println("Response Content is not a JSON object")
			}
		}

		fmt.Println("-------")
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func outputJSON(entries []HarEntry, pretty bool) {
	var output []byte
	var err error

	if pretty {
		output, err = json.MarshalIndent(entries, "", "  ")
	} else {
		// For json-small, use simplified format with only essential fields
		simplified := simplifyEntries(entries)
		output, err = json.Marshal(simplified)
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(output))
}

func simplifyEntries(entries []HarEntry) []map[string]interface{} {
	simplified := make([]map[string]interface{}, 0, len(entries))

	for _, entry := range entries {
		item := map[string]interface{}{
			"method":              entry.Request.Method,
			"url":                 entry.Request.URL,
			"status":              entry.Response.Status,
			"request_bytes":       entry.Request.BodySize,
			"response_bytes":      entry.Response.BodySize,
		}

		// Add request body if present, truncated to 100 chars
		if entry.Request.PostData != nil && entry.Request.PostData.Text != "" {
			item["request_body"] = truncateString(entry.Request.PostData.Text, 100)
		}

		// Add response content if present, truncated to 100 chars
		if entry.Response.Content.Text != "" {
			item["response_body"] = truncateString(entry.Response.Content.Text, 100)
		}

		simplified = append(simplified, item)
	}

	return simplified
}

func outputJSONL(entries []HarEntry) {
	for _, entry := range entries {
		output, err := json.Marshal(entry)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(output))
	}
}

func outputJSONLSmall(entries []HarEntry) {
	simplified := simplifyEntries(entries)
	for _, item := range simplified {
		output, err := json.Marshal(item)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(string(output))
	}
}
