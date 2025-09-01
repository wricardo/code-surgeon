package grpc

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"connectrpc.com/connect"
	"connectrpc.com/grpcreflect"
	"github.com/joho/godotenv"
	codesurgeon "github.com/wricardo/code-surgeon"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/api/apiconnect"
	"github.com/wricardo/code-surgeon/neo4j2"
	"golang.ngrok.com/ngrok"
	"golang.ngrok.com/ngrok/config"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

type Server struct {
	options      ServerOptions
	ln           ngrok.Tunnel
	ctx          context.Context
	neo4jCloseFn func()
}

func NewServer(options ServerOptions) *Server {
	return &Server{
		options: options,
		ctx:     context.Background(),
	}
}

func (s *Server) Start() error {
	log.Debug().Interface("options", s.options).Msg("Starting server")

	ctx := context.Background()
	driver, closeFn, err := neo4j2.Connect(ctx, s.options.Neo4jDbUri, s.options.Neo4jDbUser, s.options.Neo4jDbPassword)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to Neo4j: %v", err)
	}
	s.neo4jCloseFn = closeFn
	instructorClient := ai.GetInstructor()

	// Initialize ngrok listener if needed
	if s.options.UseNgrok {
		var err error
		s.ln, err = ngrok.Listen(s.ctx,
			config.HTTPEndpoint(config.WithDomain(s.options.NgrokDomain)),
			ngrok.WithAuthtokenFromEnv(),
		)
		if err != nil {
			return fmt.Errorf("failed to start ngrok listener: %w", err)
		}
	}

	// Set up gRPC and HTTP handlers
	grpcHandler := NewHandler(s.options.NgrokDomain, driver, instructorClient)
	mux := setupHTTPHandlers(grpcHandler, s.options.SlackToken)

	// Start HTTP server
	go func() {
		err := http.ListenAndServe(
			fmt.Sprintf(":%d", s.options.Port),
			h2c.NewHandler(mux, &http2.Server{}),
		)
		if err != nil {
			log.Error().Msgf("server failed to start: %s\n", err)
			panic(err)
		}
	}()

	// Start ngrok listener server if ngrok is enabled
	if s.options.UseNgrok {
		log.Print("Starting ngrok server")
		go func() {
			log.Info().Msgf("ngrok tunnel established at: %s", s.ln.URL())
			err := http.Serve(s.ln, h2c.NewHandler(mux, &http2.Server{}))
			if err != nil {
				log.Error().Msgf("ngrok server failed to start: %s\n", err)
				panic(err)
			}
		}()
	}

	// Graceful shutdown
	return s.gracefulShutdown()
}

func (s *Server) Stop() {
	log.Info().Msg("Stopping server")

	if s.neo4jCloseFn != nil {
		s.neo4jCloseFn()
	}
	// defer func() {
	// 	if err := chatcli.GLOBAL_CHAT.SaveState("chat_state.json"); err != nil {
	// 		log.Error().Msgf("Failed to save CLI chat state: %v", err)
	// 	} else {
	// 		fmt.Println("\nBye saved", err)
	// 	}
	// }()
	// Perform any cleanup or resource freeing here if needed
	if s != nil && s.ctx != nil {
		s.ctx.Done()
	}
}

func setupHTTPHandlers(grpcHandler apiconnect.GptServiceHandler, slackToken string) *http.ServeMux {
	mux := http.NewServeMux()
	// TODO: Re-implement SlackBot without chatcli
	// bot := chatcli.NewSlackBot(slackToken, grpcHandler)
	// mux.HandleFunc("/slack_message", bot.SlackMessageHandler())

	// Add static file route
	mux.Handle("/api/", http.StripPrefix("/api/", http.FileServer(http.FS(codesurgeon.STATICFS))))

	// Add gRPC handlers
	path, handler := apiconnect.NewGptServiceHandler(grpcHandler, connect.WithInterceptors(LoggerInterceptor()))
	mux.Handle(path, handler)

	reflector := grpcreflect.NewStaticReflector(apiconnect.GptServiceName)
	mux.Handle(grpcreflect.NewHandlerV1(reflector))
	mux.Handle(grpcreflect.NewHandlerV1Alpha(reflector))

	return mux
}

func (s *Server) gracefulShutdown() error {
	log.Print("on graceful shutdown")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Info().Msgf("server received signal:%s", sig.String())

	// Perform any cleanup or resource freeing here if needed
	return nil
}

// ServerOptions struct to hold server configuration
type ServerOptions struct {
	Port            int
	UseNgrok        bool
	NgrokDomain     string
	Neo4jDbUri      string
	Neo4jDbUser     string
	Neo4jDbPassword string
	SlackToken      string
}

func NewServerOptions() ServerOptions {
	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Fatal().Msg("Error loading .env file")
	}

	ngrokDomain, useNgrok := myEnv["NGROK_DOMAIN"]
	neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
	neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
	neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
	ngrokEnabled, _ := myEnv["NGROK_ENABLED"]
	if ngrokDomain == "" {
		useNgrok = false
	}
	if ngrokEnabled != "true" {
		useNgrok = false
	}
	slackToken, _ := myEnv["SLACK_BOT_TOKEN"]

	return ServerOptions{
		Port:            8010,
		UseNgrok:        useNgrok,
		NgrokDomain:     ngrokDomain,
		Neo4jDbUri:      neo4jDbUri,
		Neo4jDbUser:     neo4jDbUser,
		Neo4jDbPassword: neo4jDbPassword,
		SlackToken:      slackToken,
	}
}
