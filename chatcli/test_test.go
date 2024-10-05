package chatcli

import (
	"context"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
	"github.com/wricardo/code-surgeon/ai"
	"github.com/wricardo/code-surgeon/neo4j2"
)

func newTestChat() *ChatImpl {
	var myEnv map[string]string
	myEnv, err := godotenv.Read()
	if err != nil {
		log.Fatal().Msg("Error loading .env file")
	}
	neo4jDbUri, _ := myEnv["NEO4J_DB_URI"]
	neo4jDbUser, _ := myEnv["NEO4J_DB_USER"]
	neo4jDbPassword, _ := myEnv["NEO4J_DB_PASSWORD"]
	ctx := context.Background()
	driver, _, err := neo4j2.Connect(ctx, neo4jDbUri, neo4jDbUser, neo4jDbPassword)
	if err != nil {
		log.Fatal().Msgf("Failed to connect to Neo4j: %v", err)
	}
	// defer closeFn()

	instructorClient := ai.GetInstructor()

	// Instantiate chat service
	chat := NewChat(&driver, instructorClient)
	chat.test = true
	return chat
}
