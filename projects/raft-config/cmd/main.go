package main

import (
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"raft-config/internal/api"
	"raft-config/internal/cluster"
	"raft-config/internal/metrics"

	"github.com/gin-gonic/gin"
)

func main() {
	metrics.Init()

	nodeID := os.Getenv("NODE_ID")
	raftBind := envOr("RAFT_BIND", "0.0.0.0:7000")
	raftAdvertise := envOr("RAFT_ADVERTISE", raftBind) 
	dataDir := envOr("RAFT_DATA_DIR", "/data/raft")
	bootstrap := os.Getenv("RAFT_BOOTSTRAP") == "true"
	peers := parsePeers(os.Getenv("RAFT_PEERS"))

	node, err := cluster.New(cluster.Config{
		NodeID:        nodeID,
		RaftBind:      raftBind,
		RaftAdvertise: raftAdvertise,
		DataDir:       dataDir,
		Bootstrap:     bootstrap,
		Peers:         peers,
	})
	if err != nil {
		log.Fatalf("raft node: %v", err)
	}

	stopMetrics := make(chan struct{})
	go metrics.PollRaftState(node, stopMetrics)

	h := api.NewHandler(node)

	r := gin.Default()
	r.GET("/health", h.Health)
	r.GET("/api/cluster", h.ClusterStatus)
	r.GET("/api/config", h.ListConfig)
	r.GET("/api/config/:key", h.GetConfig)
	r.PUT("/api/config/:key", h.SetConfig)
	r.DELETE("/api/config/:key", h.DeleteConfig)
	r.GET("/metrics", metrics.Handler)

	port := envOr("HTTP_PORT", ":8080")
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	go func() {
		log.Printf("node=%s state=%s http=%s raft=%s bootstrap=%v",
			node.ID(), node.State(), port, raftAdvertise, bootstrap)
		if err := r.Run(port); err != nil {
			log.Fatalf("http server: %v", err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	close(stopMetrics)
	if err := node.Shutdown(); err != nil {
		log.Printf("raft shutdown: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parsePeers(raw string) []string {
	if raw == "" {
		return nil
	}
	return strings.Split(raw, ",")
}
