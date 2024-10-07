package grpc

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	StartServer()
	time.Sleep(1 * time.Second)
	code := m.Run()
	StopServer()
	os.Exit(code)
}

var srv *Server
var mu sync.Mutex

// StartServer initializes and starts the server
func StartServer() {
	mu.Lock()
	defer mu.Unlock()
	srv = NewServer(NewServerOptions())
	go srv.Start()
}

// StopServer stops the server
func StopServer() {
	mu.Lock()
	defer mu.Unlock()
	if srv != nil {
		srv.Stop()
		srv = nil
	}
}
