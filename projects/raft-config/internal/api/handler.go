package api

import (
	"net/http"

	"raft-config/internal/cluster"
	"raft-config/internal/fsm"
	"raft-config/internal/metrics"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	node *cluster.Node
}

func NewHandler(node *cluster.Node) *Handler {
	return &Handler{node: node}
}

type setRequest struct {
	Value string `json:"value" binding:"required"`
}

func (h *Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"node_id": h.node.ID(),
		"state":   h.node.State(),
	})
}

func (h *Handler) ClusterStatus(c *gin.Context) {
	metrics.UpdateRaftState(h.node)
	c.JSON(http.StatusOK, gin.H{
		"node_id":      h.node.ID(),
		"state":        h.node.State(),
		"is_leader":    h.node.IsLeader(),
		"leader_id":    h.node.LeaderID(),
		"leader_addr":  h.node.LeaderAddress(),
		"term":         h.node.Term(),
		"commit_index": h.node.CommitIndex(),
		"last_index":   h.node.LastIndex(),
	})
}

func (h *Handler) GetConfig(c *gin.Context) {
	key := c.Param("key")
	metrics.RecordRead()

	value, ok := h.node.FSM().Get(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found", "key": key})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": value,
		"node":  h.node.ID(),
	})
}

func (h *Handler) ListConfig(c *gin.Context) {
	metrics.RecordRead()
	c.JSON(http.StatusOK, gin.H{
		"items": h.node.FSM().List(),
		"node":  h.node.ID(),
	})
}

func (h *Handler) SetConfig(c *gin.Context) {
	if !h.node.IsLeader() {
		h.respondNotLeader(c)
		return
	}

	key := c.Param("key")
	var req setRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.node.Apply(fsm.Command{Op: fsm.OpSet, Key: key, Value: req.Value})
	if err != nil {
		metrics.RecordApplyError()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.RecordWrite()
	c.JSON(http.StatusOK, gin.H{
		"key":   key,
		"value": req.Value,
		"node":  h.node.ID(),
	})
}

func (h *Handler) DeleteConfig(c *gin.Context) {
	if !h.node.IsLeader() {
		h.respondNotLeader(c)
		return
	}

	key := c.Param("key")
	_, ok := h.node.FSM().Get(key)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found", "key": key})
		return
	}

	err := h.node.Apply(fsm.Command{Op: fsm.OpDelete, Key: key})
	if err != nil {
		metrics.RecordApplyError()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	metrics.RecordWrite()
	c.JSON(http.StatusOK, gin.H{"deleted": key, "node": h.node.ID()})
}

func (h *Handler) respondNotLeader(c *gin.Context) {
	metrics.RecordNotLeader()
	c.Header("X-Raft-Leader-Id", h.node.LeaderID())
	c.Header("X-Raft-Leader-Addr", h.node.LeaderAddress())
	c.JSON(http.StatusServiceUnavailable, gin.H{
		"error":       "not the Raft leader; retry write against the leader",
		"leader_id":   h.node.LeaderID(),
		"leader_addr": h.node.LeaderAddress(),
		"node_id":     h.node.ID(),
	})
}
