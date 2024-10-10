package chatcli

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"sync"

	"github.com/instructor-ai/instructor-go/pkg/instructor"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type ChatRepository interface {
	GetChat(chatID string) (*ChatImpl, error)
	SaveChat(chatID string, chat *ChatImpl) error
	SaveToDisk() error
}

type InMemoryChatRepository struct {
	driver           *neo4j.DriverWithContext
	instructorClient *instructor.InstructorOpenAI
	chatMap          map[string]*ChatImpl
	rwmu             sync.RWMutex
	filePath         string
}

// NewInMemoryChatRepository creates a new InMemoryChatRepository.
// It attempts to load existing chat data from the specified file.
// If the file does not exist, it starts with an empty repository.
func NewInMemoryChatRepository(filePath string, driver *neo4j.DriverWithContext, instructorClient *instructor.InstructorOpenAI) (*InMemoryChatRepository, error) {
	if instructorClient == nil {
		panic("instructorClient is required")
	}
	log.Warn().Msg("Initializing InMemoryChatRepository")
	repo := &InMemoryChatRepository{
		chatMap:          make(map[string]*ChatImpl),
		filePath:         filePath,
		driver:           driver,
		instructorClient: instructorClient,
	}

	return repo, repo.LoadFromDisk(driver, instructorClient)
}

// GetChat retrieves a chat by its ID.
// Returns an error if the chat is not found.
func (r *InMemoryChatRepository) GetChat(chatID string) (*ChatImpl, error) {
	r.rwmu.RLock()
	defer r.rwmu.RUnlock()
	x := log.Debug()
	x.Str("chatID", chatID)
	if log.Logger.GetLevel() == zerolog.TraceLevel {
		x.Interface("chatMap", r.chatMap)
	}
	defer x.Msg("InMemoryChatRepository.GetChat")
	chat, ok := r.chatMap[chatID]
	if !ok {
		return nil, nil
	}
	if chat.instructor == nil {
		panic("chat.instructor is nil")
	}
	return chat, nil
}

// SaveChat saves or updates a chat in the repository and persists it to disk.
// Returns an error if the chat is nil or if disk operations fail.
func (r *InMemoryChatRepository) SaveChat(chatID string, chat *ChatImpl) error {
	if chat == nil {
		return errors.New("chat is nil")
	}
	log.Debug().Str("chatID", chatID).Interface("chat", chat).Msg("InMemoryChatRepository.SaveChat")
	r.rwmu.RLock()
	r.chatMap[chatID] = chat
	r.rwmu.RUnlock()

	// Persist to disk
	if err := r.SaveToDisk(); err != nil {
		return err
	}

	return nil
}

// SaveToDisk serializes the chatMap and writes it to the specified file.
// It uses a temporary file and atomic rename to ensure data integrity.
func (r *InMemoryChatRepository) SaveToDisk() error {
	r.rwmu.RLock()
	defer r.rwmu.RUnlock()

	// Serialize chatMap to JSON
	data, err := json.MarshalIndent(r.chatMap, "", "  ")
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal chatMap to JSON")
		return err
	}

	f, err := os.Create(r.filePath)
	if err != nil {
		log.Error().Err(err).Str("file", r.filePath).Msg("Failed to create file")
		return err
	}

	if _, err = f.Write(data); err != nil {
		return err
	}

	log.Info().Str("file", r.filePath).Msg("Chat data saved to disk successfully")
	return nil
}

// LoadFromDisk reads the chat data from the specified file and deserializes it into chatMap.
// Returns an error if the file does not exist or if deserialization fails.
func (r *InMemoryChatRepository) LoadFromDisk(driver *neo4j.DriverWithContext, instructorClient *instructor.InstructorOpenAI) error {

	// Read the file
	data, err := ioutil.ReadFile(r.filePath)
	if err != nil {
		return err // Caller will handle os.ErrNotExist
	}

	// Deserialize JSON into chatMap
	var loadedChatMap map[string]*ChatImpl
	if err := json.Unmarshal(data, &loadedChatMap); err != nil {
		log.Error().Err(err).Str("file", r.filePath).Msg("Failed to unmarshal JSON data")
		return err
	}

	r.rwmu.Lock()
	defer r.rwmu.Unlock()
	r.chatMap = loadedChatMap
	for k := range r.chatMap {
		r.chatMap[k].Setup(driver, instructorClient, r)
	}
	log.Info().Str("file", r.filePath).Msg("Chat data loaded from disk successfully")
	return nil
}
