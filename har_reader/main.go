package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

// Define the structures for request and response
type HarEntry struct {
	Request  HarRequest  `json:"request"`
	Response HarResponse `json:"response"`
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

	// Parse command-line flags
	flag.Parse()

	// Open the JSON file specified by the --file flag
	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	// Read the JSON file into byte array
	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	// Unmarshal the JSON data
	var harLog HarLog
	err = json.Unmarshal(data, &harLog)
	if err != nil {
		log.Fatal(err)
	}

	// Loop through the entries and print important request and response attributes
	for i, entry := range harLog.Log.Entries {
		if strings.HasPrefix(entry.Request.URL, *prefixMatch) {
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
}
