package server

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/grpc"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func Start(
	port int,
	useNgrok bool,
	ngrokDomain string,
	neo4jDbUri string,
	neo4jDbUser string,
	neo4jDbPassword string,
) error {
	log.Printf("Starting server on port %d\n", port)
	// Initialize ngrok listener
	ctx := context.Background()
	var ln ngrok.Tunnel
	if useNgrok {
		var err error
		//frog-able-inherently.ngrok-free.app
		ln, err = ngrok.Listen(ctx,
			config.HTTPEndpoint(config.WithDomain(ngrokDomain)),
			ngrok.WithAuthtokenFromEnv(), // Make sure to set your ngrok authtoken in environment variables
		)
		if err != nil {
			log.Fatalf("failed to start ngrok listener: %s\n", err)
			panic(err)
		}
	}

	// graceful shutdown
	grpcHandler := grpc.NewHandler(ngrokDomain)

	mux := http.NewServeMux()

	// add static file route

	fs.WalkDir(codesurgeon.STATICFS, ".", func(path string, d fs.DirEntry, err error) error {
		return nil
	})

	mux.Handle("/api/", http.StripPrefix("/api/",
		http.FileServerFS(codesurgeon.STATICFS),
	))

	path, handler := apiconnect.NewGptServiceHandler(grpcHandler, connect.WithInterceptors(grpc.LoggerInterceptor()))
	mux.Handle(path, handler)

	reflector := grpcreflect.NewStaticReflector(
		apiconnect.GptServiceName,
	)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	go func() {
		err := http.ListenAndServe(
			fmt.Sprintf(":%d", port),
			// Use h2c so we can serve HTTP/2 without TLS.
			h2c.NewHandler(mux, &http2.Server{}),
		)

		if err != nil {
			log.Fatalf("server failed to start: %s\n", err)
		}
	}()

	go func() {
		if !useNgrok {
			return
		}
		log.Println("ngrok tunnel established at:", ln.URL())
		err := http.Serve(
			ln, // Use ngrok listener instead of a standard port listener
			h2c.NewHandler(mux, &http2.Server{}),
		)
		if err != nil {
			log.Fatalf("server failed to start: %s\n", err)
		}

		if err != nil {
			log.Fatalf("server failed to start: %s\n", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	i := <-quit
	log.Println("server receive a signal: ", i.String())

	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// if err := server.Shutdown(ctx); err != nil {
	// 	logger.Fatalf("server shutdown error: %s\n", err)
	// }

	return nil
}
