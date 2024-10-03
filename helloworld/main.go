package main

import (
    "fmt"
    "log"
    "net/http"
    "os"
)

// helloHandler responds with a simple "Hello, World!" message
func helloHandler(w http.ResponseWriter, r *http.Request) {
    if r.Method == http.MethodGet {
        fmt.Fprintln(w, "Hello, World!")
    } else {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
    }
}

// main is the entry point of the application
func main() {
    port := "8080" // default port
    if len(os.Args) > 1 {
        port = os.Args[1]
    }

    http.HandleFunc("/hello", helloHandler)

    fmt.Printf("Starting server on port %s...\n", port)
    log.Fatal(http.ListenAndServe(":"+port, nil))
}
