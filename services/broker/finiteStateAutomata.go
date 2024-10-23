package broker

import (
	"broker/internals/models"
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	"io"
	"maps"
	"sync"
)

type FsmMachine struct {
	Messages map[string][]models.Message
	rwMutex  sync.RWMutex
}

type MessageBrokerSnapshot struct {
	messages map[string][]models.Message
}

func (fsm *FsmMachine) Apply(log *raft.Log) interface{} {
	var msg models.Message
	if err := json.Unmarshal(log.Data, &msg); err != nil {
		fmt.Println("Error unmarshalling message:", err)
	}

	fsm.rwMutex.Lock()
	defer fsm.rwMutex.Unlock()
	fsm.Messages[msg.Subject] = append(fsm.Messages[msg.Subject], msg)
	return fsm
}

func (fsm *FsmMachine) Snapshot() (raft.FSMSnapshot, error) {
	fsm.rwMutex.Lock()
	defer fsm.rwMutex.Unlock()
	var fsmSnapshot map[string][]models.Message
	maps.Copy(fsmSnapshot, fsm.Messages)
	return &MessageBrokerSnapshot{
		messages: fsmSnapshot,
	}, nil
}

func (fsm *FsmMachine) Restore(reader io.ReadCloser) error {
	var snapshot struct {
		Messages map[string][]models.Message
	}
	if err := json.NewDecoder(reader).Decode(&snapshot); err != nil {
		return err
	}
	fsm.rwMutex.Lock()
	defer fsm.rwMutex.Unlock()
	fsm.Messages = snapshot.Messages
	return nil
}

func (snapshot *MessageBrokerSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := json.NewEncoder(sink).Encode(struct {
		Messages map[string][]models.Message
	}{
		Messages: snapshot.messages,
	}); err != nil {
		err := sink.Cancel()
		if err != nil {
			return err
		}
		return nil
	}
	return sink.Close()
}

func (snapshot *MessageBrokerSnapshot) Release() {}
