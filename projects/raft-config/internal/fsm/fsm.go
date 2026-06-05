package fsm

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

// Op names replicated through the Raft log.
const (
	OpSet    = "set"
	OpDelete = "delete"
)

// Command is serialized and appended to the Raft log.
type Command struct {
	Op    string `json:"op"`
	Key   string `json:"key"`
	Value string `json:"value,omitempty"`
}

// KVFSM is the replicated state machine: a string key-value store.
type KVFSM struct {
	mu   sync.RWMutex
	data map[string]string
}

func New() *KVFSM {
	return &KVFSM{data: make(map[string]string)}
}

func (f *KVFSM) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	switch cmd.Op {
	case OpSet:
		f.data[cmd.Key] = cmd.Value
	case OpDelete:
		delete(f.data, cmd.Key)
	}

	return nil
}

func (f *KVFSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	copyData := make(map[string]string, len(f.data))
	for k, v := range f.data {
		copyData[k] = v
	}

	return &fsmSnapshot{data: copyData}, nil
}

func (f *KVFSM) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	var data map[string]string
	if err := json.NewDecoder(rc).Decode(&data); err != nil {
		return err
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.data = data
	return nil
}

// Get returns a key's value and whether it exists (local FSM read; may be stale on followers).
func (f *KVFSM) Get(key string) (string, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	v, ok := f.data[key]
	return v, ok
}

// List returns a snapshot of all keys and values.
func (f *KVFSM) List() map[string]string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	out := make(map[string]string, len(f.data))
	for k, v := range f.data {
		out[k] = v
	}
	return out
}

type fsmSnapshot struct {
	data map[string]string
}

func (s *fsmSnapshot) Release() {}

func (s *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	if err := json.NewEncoder(sink).Encode(s.data); err != nil {
		_ = sink.Cancel()
		return err
	}
	return sink.Close()
}
