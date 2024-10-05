package chatcli

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"github.com/sashabaranov/go-openai"
	. "github.com/wricardo/code-surgeon/api"
)

type PostgresMode struct {
	chat *ChatImpl
	db   *sql.DB

	form *Form
	host StringPromise
	port StringPromise
	user StringPromise
	pass StringPromise
	dbnm StringPromise
}

func NewPostgresMode(chat *ChatImpl) *PostgresMode {
	m := &PostgresMode{
		chat: chat,
	}
	connectionForm := NewForm()

	m.host = connectionForm.AddQuestion("Enter the PostgreSQL host:", true, fnValidateNotEmpty, "localhost")
	m.port = connectionForm.AddQuestion("Enter the PostgreSQL port:", true, fnValidateNotEmpty, "5432")
	m.user = connectionForm.AddQuestion("Enter the PostgreSQL user:", true, fnValidateNotEmpty, "")
	m.pass = connectionForm.AddQuestion("Enter the PostgreSQL password:", true, fnValidateNotEmpty, "")
	m.dbnm = connectionForm.AddQuestion("Enter the PostgreSQL database name:", true, fnValidateNotEmpty, "test_kosmos")

	m.form = connectionForm
	return m
}

func (pm *PostgresMode) Start() (*Message, *Command, error) {
	if !pm.form.IsFilled() {
		return &Message{Form: pm.form.MakeFormMessage()}, MODE_START, nil
	}
	return &Message{}, MODE_START, nil
}

func (pm *PostgresMode) BestShot(msg *Message) (*Message, *Command, error) {
	message, _, err := pm.HandleResponse(msg)
	return message, NOOP, err
}

func (pm *PostgresMode) HandleIntent(msg *Message, intent Intent) (*Message, *Command, error) {
	// For simplicity, you can delegate to HandleResponse
	return pm.HandleResponse(msg)
}

func (pm *PostgresMode) HandleResponse(msg *Message) (*Message, *Command, error) {
	if msg.Form != nil || !pm.form.IsFilled() {
		if msg.Form != nil {
			for _, qa := range msg.Form.Questions {
				pm.form.Answer(qa.Question, qa.Answer)
			}
		}
		if !pm.form.IsFilled() {
			return &Message{Form: pm.form.MakeFormMessage()}, NOOP, nil
		}
	}

	// Connect to the database
	err := pm.connectToDatabase()
	if err != nil {
		return TextMessage("Failed to connect to the database: " + err.Error()), NOOP, nil
	}

	// improve this
	// Retrieve schema and formulate query
	schema, err := pm.retrieveSchema()
	if err != nil {
		return TextMessage("Failed to retrieve schema: " + err.Error()), NOOP, nil
	}

	if msg.Text == "" {
		tables, err := pm.retrieveTables()
		if err != nil {
			return TextMessage("Failed to retrieve tables: " + err.Error()), NOOP, nil
		}

		return TextMessage("Please provide a query.\nTables in the database:\n" + tables), NOOP, nil
	}

	// Ask the user for a query
	query, err := pm.formulateQuery(schema, msg.Text)
	if err != nil {
		return TextMessage("Failed to formulate query: " + err.Error()), NOOP, nil
	}

	// Execute the query
	result, err := pm.executeQuery(query)
	if err != nil {
		return TextMessage("Failed to execute query: " + err.Error()), NOOP, nil
	}

	return &Message{Text: query + "\n" + result}, NOOP, nil
}

func (pm *PostgresMode) Stop() error {
	if pm.db != nil {
		return pm.db.Close()
	}
	return nil
}

func (pm *PostgresMode) connectToDatabase() error {
	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		pm.host(),
		pm.port(),
		pm.user(),
		pm.pass(),
		pm.dbnm(),
	)

	// pm.questionAnswerMap["Enter the PostgreSQL database name:"])
	log.Debug().
		Str("connStr", connStr).
		Msg("Connecting to database")
	log.Printf("Connecting to database with connection string: %s", connStr)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return err
	}

	pm.db = db
	return nil
}

func (pm *PostgresMode) retrieveTables() (string, error) {
	rows, err := pm.db.Query("SELECT table_name FROM information_schema.columns WHERE table_schema = 'public' group by table_schema, table_name")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var schema string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return "", err
		}
		schema += fmt.Sprintf("%s\n", tableName)
	}
	return schema, nil
}

func (pm *PostgresMode) retrieveSchema() (string, error) {
	rows, err := pm.db.Query("SELECT table_schema, table_name, column_name, data_type FROM information_schema.columns WHERE table_schema = 'public'")
	if err != nil {
		return "", err
	}
	defer rows.Close()

	var schema string
	for rows.Next() {
		var tableSchema, tableName, columnName, dataType string
		if err := rows.Scan(&tableSchema, &tableName, &columnName, &dataType); err != nil {
			return "", err
		}
		schema += fmt.Sprintf("Schema: %s, Table: %s, Column: %s, Data Type: %s\n", tableSchema, tableName, columnName, dataType)
	}
	return schema, nil
}

func (pm *PostgresMode) formulateQuery(schema string, ask string) (string, error) {
	type AiOutput struct {
		Query string `json:"query" jsonschema:"title=query,description=the SQL query to be executed."`
	}

	var aiOut AiOutput
	err := pm.chat.Chat(&aiOut, []openai.ChatCompletionMessage{
		{
			Role:    "system",
			Content: "You are a helpful assistant that can generate POSTGRES SQL queries based on the database schema and user requests. Wait for the use to ask a sql query it wants.",
		},
		{
			Role:    "user",
			Content: "Given the following schema:\n" + schema,
		},
		{
			Role:    "user",
			Content: ask,
		},
	})
	if err != nil {
		return "", err
	}

	return aiOut.Query, nil
}

func (pm *PostgresMode) executeQuery(query string) (string, error) {
	rows, err := pm.db.Query(query)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}

	results := ""
	for rows.Next() {
		columnsData := make([]interface{}, len(columns))
		columnsPointers := make([]interface{}, len(columns))
		for i := range columnsData {
			columnsPointers[i] = &columnsData[i]
		}

		if err := rows.Scan(columnsPointers...); err != nil {
			return "", err
		}

		for i, colName := range columns {
			// if columnsData[i]  is a []byte, convert it to a string
			if v, ok := columnsData[i].([]byte); ok {
				columnsData[i] = string(v)
			}

			results += fmt.Sprintf("%s: %v\n", colName, columnsData[i])
		}
		results += "\n"
	}

	return results, nil
}

func init() {
	RegisterMode("postgres", NewPostgresMode)
}
