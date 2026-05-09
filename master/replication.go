package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
)

// Slave registry
var (
	slaves   []string
	slavesMu sync.RWMutex
)

type ReplicationPayload struct {
	Action  string      `json:"action"`
	DB      string      `json:"db"`
	Table   string      `json:"table,omitempty"`
	ID      string      `json:"id,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Columns []string    `json:"columns,omitempty"`
}

// Broadcast to all slaves
func broadcast(payload ReplicationPayload) {
	slavesMu.RLock()
	defer slavesMu.RUnlock()

	body, _ := json.Marshal(payload)
	for _, addr := range slaves {
		go func(addr string) {
			resp, err := http.Post("http://"+addr+"/replicate", "application/json", bytes.NewReader(body))
			if err != nil {
				log.Printf("[REPLICATION] Slave %s unreachable: %v", addr, err)
				return
			}
			defer resp.Body.Close()
			log.Printf("[REPLICATION] Slave %s responded: %d", addr, resp.StatusCode)
		}(addr)
	}
}

// Send full snapshot to a newly registered slave
func syncSlave(addr string) error {
	snapshot := getFullSnapshot()
	body, _ := json.Marshal(snapshot)
	resp, err := http.Post("http://"+addr+"/sync", "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("sync failed: %v", err)
	}
	defer resp.Body.Close()
	return nil
}
