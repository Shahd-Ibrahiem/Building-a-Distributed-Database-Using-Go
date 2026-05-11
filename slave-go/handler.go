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

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		respond(w, 200, map[string]string{"status": "ok", "role": "slave-go"})
	})

	mux.HandleFunc("/sync", func(w http.ResponseWriter, r *http.Request) {
		var snapshot map[string]interface{}
		json.NewDecoder(r.Body).Decode(&snapshot)
		applyFullSync(snapshot)
		respond(w, 200, map[string]string{"status": "synced"})
	})

	// Replication from master only
	mux.HandleFunc("/replicate", func(w http.ResponseWriter, r *http.Request) {
		var p ReplicationPayload
		json.NewDecoder(r.Body).Decode(&p)
		switch p.Action {
		case "create_db":    applyCreateDB(p.DB)
		case "drop_db":      applyDropDB(p.DB)
		case "create_table": applyCreateTable(p.DB, p.Table, p.Columns)
		case "delete_table": applyDeleteTable(p.DB, p.Table)
		case "insert":       applyInsert(p.DB, p.Table, p.ID, p.Data)
		case "update":       applyUpdate(p.DB, p.Table, p.ID, p.Data)
		case "delete":       applyDelete(p.DB, p.Table, p.ID)
		}
		respond(w, 200, map[string]string{"status": "applied"})
	})

	// --- Select ---
	mux.HandleFunc("/record/select", func(w http.ResponseWriter, r *http.Request) {
		rows, err := selectRecords(r.URL.Query().Get("db"), r.URL.Query().Get("table"))
		if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
		respond(w, 200, rows)
	})

	// --- Search ---
	mux.HandleFunc("/record/search", func(w http.ResponseWriter, r *http.Request) {
		rows, err := searchRecords(r.URL.Query().Get("db"), r.URL.Query().Get("table"), r.URL.Query().Get("field"), r.URL.Query().Get("value"))
		if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
		respond(w, 200, rows)
	})

	// --- Insert ---
	mux.HandleFunc("/record/insert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB     string            `json:"db"`
			Table  string            `json:"table"`
			Record map[string]string `json:"record"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		id, err := insertRecord(body.DB, body.Table, body.Record)
		if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
		respond(w, 200, map[string]string{"status": "inserted", "id": id})
	})

	// --- Update ---
	mux.HandleFunc("/record/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB      string            `json:"db"`
			Table   string            `json:"table"`
			ID      string            `json:"id"`
			Updates map[string]string `json:"updates"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if err := updateRecord(body.DB, body.Table, body.ID, body.Updates); err != nil {
			respond(w, 400, map[string]string{"error": err.Error()}); return
		}
		respond(w, 200, map[string]string{"status": "updated"})
	})

	// --- Delete Record ---
	mux.HandleFunc("/record/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB    string `json:"db"`
			Table string `json:"table"`
			ID    string `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if err := deleteRecord(body.DB, body.Table, body.ID); err != nil {
			respond(w, 400, map[string]string{"error": err.Error()}); return
		}
		respond(w, 200, map[string]string{"status": "deleted"})
	})

	// --- Delete Table ---
	mux.HandleFunc("/table/delete", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB    string `json:"db"`
			Table string `json:"table"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if err := deleteTable(body.DB, body.Table); err != nil {
			respond(w, 400, map[string]string{"error": err.Error()}); return
		}
		respond(w, 200, map[string]string{"status": "table deleted"})
	})

	// --- Databases ---
	mux.HandleFunc("/databases", func(w http.ResponseWriter, r *http.Request) {
		dbs, _ := listDBs()
		respond(w, 200, dbs)
	})

	// --- Tables ---
	mux.HandleFunc("/tables", func(w http.ResponseWriter, r *http.Request) {
		tables, err := listTables(r.URL.Query().Get("db"))
		if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
		respond(w, 200, tables)
	})

	// --- Columns ---
	mux.HandleFunc("/columns", func(w http.ResponseWriter, r *http.Request) {
		cols, err := getColumns(r.URL.Query().Get("db"), r.URL.Query().Get("table"))
		if err != nil { respond(w, 400, map[string]string{"error": err.Error()}); return }
		respond(w, 200, cols)
	})
}