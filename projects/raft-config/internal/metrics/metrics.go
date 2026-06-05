package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"raft-config/internal/cluster"
)

var (
	isLeader = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "raft_config_is_leader",
			Help: "1 if this node is the Raft leader, else 0",
		},
		[]string{"node_id"},
	)

	raftTerm = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "raft_config_term",
			Help: "Current Raft term",
		},
		[]string{"node_id"},
	)

	commitIndex = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "raft_config_commit_index",
			Help: "Raft commit index on this node",
		},
		[]string{"node_id"},
	)

	configWrites = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "raft_config_writes_total",
			Help: "Successful replicated config writes",
		},
	)

	configReads = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "raft_config_reads_total",
			Help: "Local config reads",
		},
	)

	applyErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "raft_config_apply_errors_total",
			Help: "Failed Raft apply operations",
		},
	)

	notLeaderWrites = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "raft_config_not_leader_total",
			Help: "Write attempts rejected because this node is not leader",
		},
	)
)

func init() {
	prometheus.MustRegister(isLeader, raftTerm, commitIndex, configWrites, configReads, applyErrors, notLeaderWrites)
}

func Init() {}

func Handler(c *gin.Context) {
	promhttp.Handler().ServeHTTP(c.Writer, c.Request)
}

func RecordWrite()       { configWrites.Inc() }
func RecordRead()        { configReads.Inc() }
func RecordApplyError()  { applyErrors.Inc() }
func RecordNotLeader()   { notLeaderWrites.Inc() }

func UpdateRaftState(node *cluster.Node) {
	id := node.ID()
	if node.IsLeader() {
		isLeader.WithLabelValues(id).Set(1)
	} else {
		isLeader.WithLabelValues(id).Set(0)
	}
	raftTerm.WithLabelValues(id).Set(float64(node.Term()))
	commitIndex.WithLabelValues(id).Set(float64(node.CommitIndex()))
}

// PollRaftState updates gauges on an interval.
func PollRaftState(node *cluster.Node, stop <-chan struct{}) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			UpdateRaftState(node)
		}
	}
}
