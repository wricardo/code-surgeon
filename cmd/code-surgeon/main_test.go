package main

import (
	"encoding/json"
	"testing"

	"github.com/Jeffail/gabs"
)

// Sample JSON data
var jsonData = []byte(`{
	"openapi": "3.0.0",
	"info": {
		"title": "codesurgeon",
		"version": "1.0.0"
	},
	"paths": {
		"/codesurgeon.GptService/SearchForFunction": {
			"post": {
				"summary": "SearchForFunction",
				"operationId": "codesurgeon.GptService.SearchForFunction",
				"requestBody": {
					"content": {
						"application/json": {
							"schema": {
								"$ref": "#/components/schemas/codesurgeon.SearchForFunctionRequest"
							}
						}
					}
				},
				"responses": {
					"200": {
						"description": "A successful response",
						"content": {}
					}
				}
			}
		}
	}
}`)

// Benchmark using encoding/json with map[string]interface{}
func BenchmarkJSONMapManipulation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var result map[string]interface{}
		if err := json.Unmarshal(jsonData, &result); err != nil {
			b.Fatalf("Error unmarshaling JSON: %v", err)
		}

		// Add "servers" key
		result["servers"] = []map[string]string{
			{"url": "http://localhost"},
		}

		// Marshal back to JSON (to simulate full round-trip manipulation)
		_, err := json.Marshal(result)
		if err != nil {
			b.Fatalf("Error marshaling JSON: %v", err)
		}
	}
}

// Benchmark using github.com/Jeffail/gabs
func BenchmarkGabsManipulation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		jsonParsed, err := gabs.ParseJSON(jsonData)
		if err != nil {
			b.Fatalf("Error parsing JSON with gabs: %v", err)
		}

		// Add "servers" field using SetP
		jsonParsed.SetP("http://localhost", "servers.0.url")

		// Stringify the modified JSON (to simulate full round-trip manipulation)
		_ = jsonParsed.String()
	}
}
