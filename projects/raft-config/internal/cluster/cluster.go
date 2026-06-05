package cluster

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"raft-config/internal/fsm"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb/v2"
)

const (
	defaultRaftBind  = "0.0.0.0:7000"
	defaultDataDir   = "/data/raft"
	raftApplyTimeout = 10 * time.Second
	raftTCPMaxPool   = 3
	raftTCPTimeout   = 10 * time.Second
)

// Node wraps a HashiCorp Raft instance and the KV state machine.
type Node struct {
	raft *raft.Raft
	fsm  *fsm.KVFSM
	id   raft.ServerID
}

// Config holds node startup parameters.
type Config struct {
	NodeID        string
	RaftBind      string
	RaftAdvertise string
	DataDir       string
	Bootstrap     bool
	Peers         []string // "node-id=host:port" entries for initial cluster
}

func New(cfg Config) (*Node, error) {
	if cfg.NodeID == "" {
		return nil, fmt.Errorf("NODE_ID is required")
	}
	if cfg.RaftBind == "" {
		cfg.RaftBind = defaultRaftBind
	}
	if cfg.RaftAdvertise == "" {
		cfg.RaftAdvertise = cfg.RaftBind
	}
	if cfg.DataDir == "" {
		cfg.DataDir = defaultDataDir
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	raftCfg := raft.DefaultConfig()
	raftCfg.LocalID = raft.ServerID(cfg.NodeID)

	advertiseTCP, err := net.ResolveTCPAddr("tcp", cfg.RaftAdvertise)
	if err != nil {
		return nil, fmt.Errorf("resolve RAFT_ADVERTISE: %w", err)
	}

	transport, err := raft.NewTCPTransport(cfg.RaftBind, advertiseTCP, raftTCPMaxPool, raftTCPTimeout, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("raft transport: %w", err)
	}

	logPath := filepath.Join(cfg.DataDir, "logs.db")
	stablePath := filepath.Join(cfg.DataDir, "stable.db")
	logStore, err := raftboltdb.NewBoltStore(logPath)
	if err != nil {
		return nil, fmt.Errorf("log store: %w", err)
	}

	stableStore, err := raftboltdb.NewBoltStore(stablePath)
	if err != nil {
		return nil, fmt.Errorf("stable store: %w", err)
	}

	snapPath := filepath.Join(cfg.DataDir, "snapshots")
	snapStore, err := raft.NewFileSnapshotStore(snapPath, 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("snapshot store: %w", err)
	}

	kvFSM := fsm.New()
	r, err := raft.NewRaft(raftCfg, kvFSM, logStore, stableStore, snapStore, transport)
	if err != nil {
		return nil, fmt.Errorf("raft instance: %w", err)
	}

	if cfg.Bootstrap {
		existing, err := raft.HasExistingState(logStore, stableStore, snapStore)
		if err != nil {
			return nil, fmt.Errorf("check existing state: %w", err)
		}
		if !existing {
			servers, err := parsePeerConfig(cfg.Peers, cfg.NodeID, cfg.RaftAdvertise)
			if err != nil {
				return nil, err
			}
			future := r.BootstrapCluster(raft.Configuration{Servers: servers})
			if err := future.Error(); err != nil {
				return nil, fmt.Errorf("bootstrap cluster: %w", err)
			}
		}
	}

	return &Node{raft: r, fsm: kvFSM, id: raft.ServerID(cfg.NodeID)}, nil
}

// parsePeerConfig builds the initial Raft configuration from PEERS env.
// Format per peer: "node-id=host:port". The local node is included if missing.
func parsePeerConfig(peers []string, localID, localAddr string) ([]raft.Server, error) {
	seen := make(map[raft.ServerID]bool)
	var servers []raft.Server

	add := func(id, addr string) {
		sid := raft.ServerID(id)
		if seen[sid] {
			return
		}
		seen[sid] = true
		servers = append(servers, raft.Server{
			ID:      sid,
			Address: raft.ServerAddress(addr),
		})
	}

	for _, p := range peers {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid peer %q: expected node-id=host:port", p)
		}
		add(parts[0], parts[1])
	}

	if !seen[raft.ServerID(localID)] {
		add(localID, localAddr)
	}

	if len(servers) == 0 {
		return nil, fmt.Errorf("no peers configured for bootstrap")
	}

	return servers, nil
}

func (n *Node) FSM() *fsm.KVFSM {
	return n.fsm
}

func (n *Node) Raft() *raft.Raft {
	return n.raft
}

func (n *Node) ID() string {
	return string(n.id)
}

func (n *Node) IsLeader() bool {
	return n.raft.State() == raft.Leader
}

func (n *Node) LeaderAddress() string {
	addr, _ := n.raft.LeaderWithID()
	return string(addr)
}

func (n *Node) LeaderID() string {
	_, id := n.raft.LeaderWithID()
	return string(id)
}

func (n *Node) State() string {
	switch n.raft.State() {
	case raft.Leader:
		return "leader"
	case raft.Candidate:
		return "candidate"
	case raft.Follower:
		return "follower"
	default:
		return "unknown"
	}
}

func (n *Node) Term() uint64 {
	term, _ := strconv.ParseUint(n.raft.Stats()["term"], 10, 64)
	return term
}

func (n *Node) CommitIndex() uint64 {
	return n.raft.CommitIndex()
}

func (n *Node) LastIndex() uint64 {
	return n.raft.LastIndex()
}

func (n *Node) Apply(cmd fsm.Command) error {
	b, err := json.Marshal(cmd)
	if err != nil {
		return err
	}
	future := n.raft.Apply(b, raftApplyTimeout)
	if err := future.Error(); err != nil {
		return err
	}
	if resp := future.Response(); resp != nil {
		if err, ok := resp.(error); ok {
			return err
		}
	}
	return nil
}

func (n *Node) Shutdown() error {
	return n.raft.Shutdown().Error()
}
