package gotools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// runGoplsSymbols runs `gopls symbols` on a given Go file and returns the output.
func runGoplsSymbols(filePath string) (string, error) {
	cmd := exec.Command("gopls", "symbols", filePath)
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run gopls symbols on %s: %v", filePath, err)
	}

	return cleanSymbolsOutput(out.String()), nil
}

// getSymbolsByFile finds Go files and runs `gopls symbols` on each, organized by filename.
func getSymbolsByFile(root string, maxDepth int) (map[string]string, error) {
	goFiles, err := findGoFiles(root, maxDepth)
	if err != nil {
		return nil, err
	}

	symbolsByFile := make(map[string]string)

	for _, filePath := range goFiles {
		symbols, err := runGoplsSymbols(filePath)
		if err != nil {
			return nil, err
		}
		symbolsByFile[filePath] = symbols
	}

	return symbolsByFile, nil
}

// cleanSymbolsOutput uses regex to clean up the gopls symbols output by removing line and column numbers.
func cleanSymbolsOutput(symbols string) string {
	// Define the regex pattern to match line and column numbers like ":507:6-507:13"
	pattern := `\s+\d+:\d+-\d+:\d+`

	// Compile the regex
	re := regexp.MustCompile(pattern)

	// Replace the matched patterns with an empty string
	cleanedSymbols := re.ReplaceAllString(symbols, "")

	// Remove extra spaces and newlines
	cleanedSymbols = strings.TrimSpace(cleanedSymbols)
	cleanedSymbols = strings.ReplaceAll(cleanedSymbols, "\n\t", "\n")

	return cleanedSymbols
}

// findGoFiles recursively finds all Go files up to the given depth.
func findGoFiles(root string, maxDepth int) ([]string, error) {
	var goFiles []string

	// Walk the directory tree up to the specified depth
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate the depth of the current path
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		depth := strings.Count(relPath, string(os.PathSeparator))

		// If the depth exceeds maxDepth, skip this directory
		if info.IsDir() && depth > maxDepth {
			return filepath.SkipDir
		}

		// If it's a Go file, add it to the list
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".go") {
			goFiles = append(goFiles, path)
		}

		return nil
	})

	return goFiles, err
}

// I don't think this is used anymore
// Hacky way to store the symbols by file cache
var symbolsByFileCache string

func cacheSymbols() string {
	log.Println("Caching symbols")
	m, err := getSymbolsByFile(".", 3)
	if err != nil {
		log.Printf("Failed to cache symbols: %v", err)
		return ""
	}
	encoded, err := json.Marshal(m)
	if err != nil {
		log.Printf("Failed to cache symbols: %v", err)
		return ""
	}

	symbolsByFileCache = string(encoded)
	log.Println("Symbols cached")
	return symbolsByFileCache
}
