package broker

import (
	"broker/internals/models"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	"io"
	"sync"
)

type FsmMachine struct {
	Messages map[string][]models.Message
	rwMutex  sync.RWMutex
}
type Command struct {
	Op      string         `json:"op"`      // Operation type (e.g., "publish", "delete")
	Subject string         `json:"subject"` // Message subject/topic
	Message models.Message `json:"message"` // The actual message
}

type MessageBrokerSnapshot struct {
	messages map[string][]models.Message
}

func (f *FsmMachine) Apply(logEntry *raft.Log) interface{} {
	var command Command
	if err := json.Unmarshal(logEntry.Data, &command); err != nil {
		return err
	}

	f.rwMutex.Lock()
	defer f.rwMutex.Unlock()

	switch command.Op {
	case "publish":
		// Initialize slice if it doesn't exist
		if _, exists := f.Messages[command.Subject]; !exists {
			f.Messages[command.Subject] = make([]models.Message, 0)
		}

		// Generate message ID based on the current length
		command.Message.Id = int(int64(len(f.Messages[command.Subject]) + 1))

		// Append message to the subject's message list
		f.Messages[command.Subject] = append(f.Messages[command.Subject], command.Message)

		return command.Message.Id

	// You can add more operations here like "delete", "update", etc.
	default:
		return fmt.Errorf("unknown command operation: %s", command.Op)
	}
}

// Snapshot implements the raft.FSM interface
func (f *FsmMachine) Snapshot() (raft.FSMSnapshot, error) {
	f.rwMutex.RLock()
	defer f.rwMutex.RUnlock()

	// Create a copy of the messages map
	messages := make(map[string][]models.Message)
	for k, v := range f.Messages {
		messages[k] = make([]models.Message, len(v))
		copy(messages[k], v)
	}

	return &FsmSnapshot{
		Messages: messages,
	}, nil
}

// Restore implements the raft.FSM interface
func (f *FsmMachine) Restore(rc io.ReadCloser) error {
	f.rwMutex.Lock()
	defer f.rwMutex.Unlock()
	defer rc.Close()

	var messages map[string][]models.Message
	if err := json.NewDecoder(rc).Decode(&messages); err != nil {
		return err
	}

	f.Messages = messages
	return nil
}

// FsmSnapshot represents a snapshot of the FSM's state
type FsmSnapshot struct {
	Messages map[string][]models.Message
}

// Persist implements the raft.FSMSnapshot interface
func (f *FsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := json.NewEncoder(sink).Encode(f.Messages)
	if err != nil {
		sink.Cancel()
		return err
	}
	return sink.Close()
}

// Release implements the raft.FSMSnapshot interface
func (f *FsmSnapshot) Release() {}
