package codesurgeon

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

func EnsureGoFileExists(filename string, packageName string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if f, err := os.Create(filename); err != nil {
			return fmt.Errorf("Failed to create file: %v", err)
		} else {
			f.Write([]byte("package " + packageName + "\n"))
			defer f.Close()
		}
	}
	return nil
}

func FormatWithGoImports(filename string) error {
	// Check if the file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", filename)
	}

	// Prepare the command to run `goimports`
	cmd := exec.Command("goimports", "-w", filename) // "-w" flag to write result to the file

	// Capture the output and error
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run goimports: %v, stderr: %s", err, stderr.String())
	}

	return nil
}
