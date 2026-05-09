package main

import (
	"encoding/json"
	"net/http"
)

func respond(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, msg string) {
	respond(w, status, map[string]string{"error": msg})
}

func setupRoutes(mux *http.ServeMux) {

	// OPTIONS (CORS preflight)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusOK)
		}
	})

	// --- Register slave ---
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct{ Address string `json:"address"` }
		json.NewDecoder(r.Body).Decode(&body)
		slavesMu.Lock()
		slaves = append(slaves, body.Address)
		slavesMu.Unlock()
		go syncSlave(body.Address)
		respond(w, 200, map[string]string{"status": "registered"})
	})

	// --- List slaves ---
	mux.HandleFunc("/slaves", func(w http.ResponseWriter, r *http.Request) {
		slavesMu.RLock()
		defer slavesMu.RUnlock()
		respond(w, 200, slaves)
	})

	// --- List all databases ---
	mux.HandleFunc("/databases", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		dbMu.RLock()
		defer dbMu.RUnlock()
		names := []string{}
		for name := range databases {
			names = append(names, name)
		}
		respond(w, 200, names)
	})

	// --- Create DB ---
	mux.HandleFunc("/db/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct{ Name string `json:"name"` }
		json.NewDecoder(r.Body).Decode(&body)
		if err := createDB(body.Name); err != nil {
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "create_db", DB: body.Name})
		respond(w, 200, map[string]string{"status": "created"})
	})

	// --- Drop DB (Master only) ---
	mux.HandleFunc("/db/drop", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct{ Name string `json:"name"` }
		json.NewDecoder(r.Body).Decode(&body)
		if err := dropDB(body.Name); err != nil {
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "drop_db", DB: body.Name})
		respond(w, 200, map[string]string{"status": "dropped"})
	})

	// --- Create Table ---
	mux.HandleFunc("/table/create", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB      string   `json:"db"`
			Table   string   `json:"table"`
			Columns []string `json:"columns"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if err := createTable(body.DB, body.Table, body.Columns); err != nil {
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "create_table", DB: body.DB, Table: body.Table, Columns: body.Columns})
		respond(w, 200, map[string]string{"status": "table created"})
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
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "delete_table", DB: body.DB, Table: body.Table})
		respond(w, 200, map[string]string{"status": "table deleted"})
	})

	// --- List tables ---
	mux.HandleFunc("/tables", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		dbName := r.URL.Query().Get("db")
		dbMu.RLock()
		defer dbMu.RUnlock()
		db, exists := databases[dbName]
		if !exists {
			respondError(w, 404, "database not found"); return
		}
		tables := []string{}
		for t := range db.Tables {
			tables = append(tables, t)
		}
		respond(w, 200, tables)
	})

	// --- Insert ---
	mux.HandleFunc("/record/insert", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB     string `json:"db"`
			Table  string `json:"table"`
			Record Record `json:"record"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		id, err := insertRecord(body.DB, body.Table, body.Record)
		if err != nil {
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "insert", DB: body.DB, Table: body.Table, ID: id, Data: body.Record})
		respond(w, 200, map[string]string{"status": "inserted", "id": id})
	})

	// --- Select ---
	mux.HandleFunc("/record/select", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		db := r.URL.Query().Get("db")
		table := r.URL.Query().Get("table")
		rows, err := selectRecords(db, table)
		if err != nil {
			respondError(w, 400, err.Error()); return
		}
		respond(w, 200, rows)
	})

	// --- Search ---
	mux.HandleFunc("/record/search", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		db := r.URL.Query().Get("db")
		table := r.URL.Query().Get("table")
		field := r.URL.Query().Get("field")
		value := r.URL.Query().Get("value")
		rows, err := searchRecords(db, table, field, value)
		if err != nil {
			respondError(w, 400, err.Error()); return
		}
		respond(w, 200, rows)
	})

	// --- Update ---
	mux.HandleFunc("/record/update", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions { respond(w, 200, nil); return }
		var body struct {
			DB      string `json:"db"`
			Table   string `json:"table"`
			ID      string `json:"id"`
			Updates Record `json:"updates"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		if err := updateRecord(body.DB, body.Table, body.ID, body.Updates); err != nil {
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "update", DB: body.DB, Table: body.Table, ID: body.ID, Data: body.Updates})
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
			respondError(w, 400, err.Error()); return
		}
		broadcast(ReplicationPayload{Action: "delete", DB: body.DB, Table: body.Table, ID: body.ID})
		respond(w, 200, map[string]string{"status": "deleted"})
	})
}
