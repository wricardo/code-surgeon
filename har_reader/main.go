package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/toon-format/toon-go"
)

// Define the structures for request and response
type HarEntry struct {
	Request  HarRequest  `json:"request" toon:"request"`
	Response HarResponse `json:"response" toon:"response"`
}

type HarRequest struct {
	Method   string    `json:"method" toon:"method"`
	URL      string    `json:"url" toon:"url"`
	Headers  []Header  `json:"headers" toon:"headers"`
	BodySize int       `json:"bodySize" toon:"bodySize"`
	PostData *PostData `json:"postData" toon:"postData"`
}

type Header struct {
	Name  string `json:"name" toon:"name"`
	Value string `json:"value" toon:"value"`
}

type PostData struct {
	MimeType string `json:"mimeType" toon:"mimeType"`
	Text     string `json:"text" toon:"text"`
}

type HarResponse struct {
	Status   int             `json:"status" toon:"status"`
	Headers  []Header        `json:"headers" toon:"headers"`
	BodySize int             `json:"bodySize" toon:"bodySize"`
	Content  ResponseContent `json:"content" toon:"content"`
}

type ResponseContent struct {
	Size     int    `json:"size" toon:"size"`
	MimeType string `json:"mimeType" toon:"mimeType"`
	Text     string `json:"text" toon:"text"`
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
	format := flag.String("format", "text", "Output format: text, json, json-small")

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

	// Filter entries based on prefix
	var filteredEntries []HarEntry
	for _, entry := range harLog.Log.Entries {
		if *prefixMatch == "" || strings.HasPrefix(entry.Request.URL, *prefixMatch) {
			filteredEntries = append(filteredEntries, entry)
		}
	}

	// Output based on format
	switch *format {
	case "json":
		outputJSON(filteredEntries, true)
	case "json-small":
		outputJSON(filteredEntries, false)
	case "text":
		outputText(filteredEntries)
	case "toon":
		outputToon(filteredEntries)
	default:
		log.Fatalf("Unknown format: %s (supported: text, json, json-small, toon)", *format)
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

func outputJSON(entries []HarEntry, pretty bool) {
	var output []byte
	var err error

	if pretty {
		output, err = json.MarshalIndent(entries, "", "  ")
	} else {
		output, err = json.Marshal(entries)
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(output))
}

// outputToon outputs in TOON format using the toon-go library
func outputToon(entries []HarEntry) {
	// Clean up entries to reduce size: remove empty fields and null values
	cleanedEntries := make([]HarEntry, 0, len(entries))
	for _, entry := range entries {
		cleaned := entry
		// Remove empty headers
		if len(cleaned.Request.Headers) == 0 {
			cleaned.Request.Headers = nil
		}
		if len(cleaned.Response.Headers) == 0 {
			cleaned.Response.Headers = nil
		}
		cleanedEntries = append(cleanedEntries, cleaned)
	}

	// Convert entries to TOON format with length markers, 0 indent for compactness
	toonStr, err := toon.MarshalString(cleanedEntries,
		toon.WithLengthMarkers(true),
		toon.WithIndent(0),
	)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(toonStr)
}
