package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

type Record map[string]string

type Table struct {
	Columns []string          `json:"columns"`
	Rows    map[string]Record `json:"rows"`
	NextID  int               `json:"next_id"`
}

type Database struct {
	Name   string            `json:"name"`
	Tables map[string]*Table `json:"tables"`
}

var (
	databases = map[string]*Database{}
	dbMu      sync.RWMutex
)

func applyCreateDB(name string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	if _, exists := databases[name]; !exists {
		databases[name] = &Database{Name: name, Tables: map[string]*Table{}}
		saveDB(name)
	}
}

func applyDropDB(name string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	delete(databases, name)
	os.Remove("data_" + name + ".json")
}

func applyCreateTable(dbName, tableName string, columns []string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists { return }
	if _, exists := db.Tables[tableName]; !exists {
		db.Tables[tableName] = &Table{Columns: columns, Rows: map[string]Record{}, NextID: 1}
		saveDB(dbName)
	}
}

func applyDeleteTable(dbName, tableName string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists { return }
	delete(db.Tables, tableName)
	saveDB(dbName)
}

func applyInsert(dbName, tableName, id string, record Record) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists { return }
	table, exists := db.Tables[tableName]
	if !exists { return }
	table.Rows[id] = record
	saveDB(dbName)
}

func applyUpdate(dbName, tableName, id string, updates Record) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists { return }
	table, exists := db.Tables[tableName]
	if !exists { return }
	row, exists := table.Rows[id]
	if !exists { return }
	for k, v := range updates {
		row[k] = v
	}
	saveDB(dbName)
}

func applyDelete(dbName, tableName, id string) {
	dbMu.Lock()
	defer dbMu.Unlock()
	db, exists := databases[dbName]
	if !exists { return }
	table, exists := db.Tables[tableName]
	if !exists { return }
	delete(table.Rows, id)
	saveDB(dbName)
}

func selectRecords(dbName, tableName string) ([]map[string]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	db, exists := databases[dbName]
	if !exists { return nil, fmt.Errorf("database not found") }
	table, exists := db.Tables[tableName]
	if !exists { return nil, fmt.Errorf("table not found") }
	var results []map[string]string
	for id, row := range table.Rows {
		r := map[string]string{"id": id}
		for k, v := range row { r[k] = v }
		results = append(results, r)
	}
	return results, nil
}

func searchRecords(dbName, tableName, field, value string) ([]map[string]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	db, exists := databases[dbName]
	if !exists { return nil, fmt.Errorf("database not found") }
	table, exists := db.Tables[tableName]
	if !exists { return nil, fmt.Errorf("table not found") }
	var results []map[string]string
	for id, row := range table.Rows {
		if v, ok := row[field]; ok && strings.Contains(strings.ToLower(v), strings.ToLower(value)) {
			r := map[string]string{"id": id}
			for k, val := range row { r[k] = val }
			results = append(results, r)
		}
	}
	return results, nil
}

func applyFullSync(snapshot map[string]*Database) {
	dbMu.Lock()
	defer dbMu.Unlock()
	databases = snapshot
	for name := range databases {
		saveDB(name)
	}
}

func saveDB(name string) {
	db, exists := databases[name]
	if !exists { return }
	data, _ := json.MarshalIndent(db, "", "  ")
	os.WriteFile("data_"+name+".json", data, 0644)
}

func loadAllDBs() {
	files, _ := os.ReadDir(".")
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "data_") && strings.HasSuffix(f.Name(), ".json") {
			data, err := os.ReadFile(f.Name())
			if err != nil { continue }
			var db Database
			if err := json.Unmarshal(data, &db); err == nil {
				databases[db.Name] = &db
			}
		}
	}
}
