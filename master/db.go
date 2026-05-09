package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Database structure
type Record map[string]string

type Table struct {
	Columns []string            `json:"columns"`
	Rows    map[string]Record   `json:"rows"`
	NextID  int                 `json:"next_id"`
}

type Database struct {
	Name   string             `json:"name"`
	Tables map[string]*Table  `json:"tables"`
}

// In-memory store
var (
	databases = map[string]*Database{}
	dbMu      sync.RWMutex
)

// ---------- DB Operations ----------

func createDB(name string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, exists := databases[name]; exists {
		return fmt.Errorf("database '%s' already exists", name)
	}
	databases[name] = &Database{Name: name, Tables: map[string]*Table{}}
	saveDB(name)
	return nil
}

func dropDB(name string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, exists := databases[name]; !exists {
		return fmt.Errorf("database '%s' not found", name)
	}
	delete(databases, name)
	os.Remove("data_" + name + ".json")
	return nil
}

// ---------- Table Operations ----------

func createTable(dbName, tableName string, columns []string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists {
		return fmt.Errorf("database '%s' not found", dbName)
	}
	if _, exists := db.Tables[tableName]; exists {
		return fmt.Errorf("table '%s' already exists", tableName)
	}
	db.Tables[tableName] = &Table{
		Columns: columns,
		Rows:    map[string]Record{},
		NextID:  1,
	}
	saveDB(dbName)
	return nil
}

func deleteTable(dbName, tableName string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists {
		return fmt.Errorf("database '%s' not found", dbName)
	}
	if _, exists := db.Tables[tableName]; !exists {
		return fmt.Errorf("table '%s' not found", tableName)
	}
	delete(db.Tables, tableName)
	saveDB(dbName)
	return nil
}

// ---------- Record Operations ----------

func insertRecord(dbName, tableName string, record Record) (string, error) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists {
		return "", fmt.Errorf("database '%s' not found", dbName)
	}
	table, exists := db.Tables[tableName]
	if !exists {
		return "", fmt.Errorf("table '%s' not found", tableName)
	}
	id := fmt.Sprintf("%d", table.NextID)
	table.NextID++
	table.Rows[id] = record
	saveDB(dbName)
	return id, nil
}

func selectRecords(dbName, tableName string) ([]map[string]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	db, exists := databases[dbName]
	if !exists {
		return nil, fmt.Errorf("database '%s' not found", dbName)
	}
	table, exists := db.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}
	var results []map[string]string
	for id, row := range table.Rows {
		r := map[string]string{"id": id}
		for k, v := range row {
			r[k] = v
		}
		results = append(results, r)
	}
	return results, nil
}

func searchRecords(dbName, tableName, field, value string) ([]map[string]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	db, exists := databases[dbName]
	if !exists {
		return nil, fmt.Errorf("database '%s' not found", dbName)
	}
	table, exists := db.Tables[tableName]
	if !exists {
		return nil, fmt.Errorf("table '%s' not found", tableName)
	}
	var results []map[string]string
	for id, row := range table.Rows {
		if v, ok := row[field]; ok && strings.Contains(strings.ToLower(v), strings.ToLower(value)) {
			r := map[string]string{"id": id}
			for k, val := range row {
				r[k] = val
			}
			results = append(results, r)
		}
	}
	return results, nil
}

func updateRecord(dbName, tableName, id string, updates Record) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists {
		return fmt.Errorf("database '%s' not found", dbName)
	}
	table, exists := db.Tables[tableName]
	if !exists {
		return fmt.Errorf("table '%s' not found", tableName)
	}
	row, exists := table.Rows[id]
	if !exists {
		return fmt.Errorf("record '%s' not found", id)
	}
	for k, v := range updates {
		row[k] = v
	}
	saveDB(dbName)
	return nil
}

func deleteRecord(dbName, tableName, id string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists {
		return fmt.Errorf("database '%s' not found", dbName)
	}
	table, exists := db.Tables[tableName]
	if !exists {
		return fmt.Errorf("table '%s' not found", tableName)
	}
	if _, exists := table.Rows[id]; !exists {
		return fmt.Errorf("record '%s' not found", id)
	}
	delete(table.Rows, id)
	saveDB(dbName)
	return nil
}

// ---------- Persistence (JSON) ----------

func saveDB(name string) {
	db, exists := databases[name]
	if !exists {
		return
	}
	data, _ := json.MarshalIndent(db, "", "  ")
	os.WriteFile("data_"+name+".json", data, 0644)
}

func loadAllDBs() {
	files, _ := os.ReadDir(".")
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "data_") && strings.HasSuffix(f.Name(), ".json") {
			data, err := os.ReadFile(f.Name())
			if err != nil {
				continue
			}
			var db Database
			if err := json.Unmarshal(data, &db); err == nil {
				databases[db.Name] = &db
			}
		}
	}
}

func getFullSnapshot() map[string]*Database {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return databases
}
