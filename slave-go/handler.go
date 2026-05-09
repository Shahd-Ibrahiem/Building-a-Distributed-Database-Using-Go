package main

import (
	"encoding/json"
	"net/http"
)

type ReplicationPayload struct {
	Action  string            `json:"action"`
	DB      string            `json:"db"`
	Table   string            `json:"table,omitempty"`
	ID      string            `json:"id,omitempty"`
	Data    map[string]string `json:"data,omitempty"`
	Columns []string          `json:"columns,omitempty"`
}

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func setupRoutes(mux *http.ServeMux) {

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{"status": "ok", "role": "slave-go"})
	})

	// Full sync from master
	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		var snapshot map[string]*Database
		json.NewDecoder(r.Body).Decode(&snapshot)
		applyFullSync(snapshot)
		respond(w, 200, map[string]string{"status": "synced"})
	})

	// Replicate individual actions from master
	mux.HandleFunc("/replicate", func(w http.ResponseWriter, r *http.Request) {
		var payload ReplicationPayload
		json.NewDecoder(r.Body).Decode(&payload)

		switch payload.Action {
		case "create_db":
			applyCreateDB(payload.DB)
		case "drop_db":
			applyDropDB(payload.DB)
		case "create_table":
			applyCreateTable(payload.DB, payload.Table, payload.Columns)
		case "delete_table":
			applyDeleteTable(payload.DB, payload.Table)
		case "insert":
			applyInsert(payload.DB, payload.Table, payload.ID, payload.Data)
		case "update":
			applyUpdate(payload.DB, payload.Table, payload.ID, payload.Data)
		case "delete":
			applyDelete(payload.DB, payload.Table, payload.ID)
		}
		respond(w, 200, map[string]string{"status": "applied"})
	})

	// Select
	mux.HandleFunc("/record/select", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		db := r.URL.Query().Get("db")
		table := r.URL.Query().Get("table")
		rows, err := selectRecords(db, table)
		if err != nil {
			respond(w, 400, map[string]string{"error": err.Error()}); return
		}
		respond(w, 200, rows)
	})

	// Search
	mux.HandleFunc("/record/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		db := r.URL.Query().Get("db")
		table := r.URL.Query().Get("table")
		field := r.URL.Query().Get("field")
		value := r.URL.Query().Get("value")
		rows, err := searchRecords(db, table, field, value)
		if err != nil {
			respond(w, 400, map[string]string{"error": err.Error()}); return
		}
		respond(w, 200, rows)
	})

	// List databases
	mux.HandleFunc("/databases", func(w http.ResponseWriter, r *http.Request) {
		dbMu.RLock()
		defer dbMu.RUnlock()
		names := []string{}
		for name := range databases { names = append(names, name) }
		respond(w, 200, names)
	})

	// List tables
	mux.HandleFunc("/tables", func(w http.ResponseWriter, r *http.Request) {
		dbName := r.URL.Query().Get("db")
		dbMu.RLock()
		defer dbMu.RUnlock()
		db, exists := databases[dbName]
		if !exists { respond(w, 404, map[string]string{"error": "db not found"}); return }
		tables := []string{}
		for t := range db.Tables { tables = append(tables, t) }
		respond(w, 200, tables)
	})
}
