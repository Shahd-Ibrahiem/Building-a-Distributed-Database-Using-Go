package main

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"
)

// Record is a map of column -> value
type Record map[string]string

var (
	db   *sql.DB
	dbMu sync.RWMutex
)

// ---------- Connect ----------

func connectMySQL() error {
	var err error
	// Connect to MySQL server (no specific database yet)
	db, err = sql.Open("mysql", "root:root123@tcp(127.0.0.1:3306)/")
	if err != nil {
		return fmt.Errorf("failed to open MySQL: %v", err)
	}
	if err = db.Ping(); err != nil {
		return fmt.Errorf("failed to ping MySQL: %v", err)
	}
	return nil
}

// ---------- DB Operations ----------

func createDB(name string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err := db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", name))
	if err != nil {
		return fmt.Errorf("failed to create database: %v", err)
	}
	return nil
}

func dropDB(name string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", name))
	if err != nil {
		return fmt.Errorf("failed to drop database: %v", err)
	}
	return nil
}

func listDBs() ([]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	rows, err := db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	// Skip system databases
	system := map[string]bool{
		"information_schema": true,
		"mysql":              true,
		"performance_schema": true,
		"sys":                true,
	}
	var dbs []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		if !system[name] {
			dbs = append(dbs, name)
		}
	}
	return dbs, nil
}

// ---------- Table Operations ----------

func createTable(dbName, tableName string, columns []string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	// Build: id INT AUTO_INCREMENT PRIMARY KEY, col1 VARCHAR(255), col2 VARCHAR(255), ...
	cols := "id INT AUTO_INCREMENT PRIMARY KEY"
	for _, c := range columns {
		cols += fmt.Sprintf(", `%s` VARCHAR(255)", c)
	}
	query := fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s`.`%s` (%s)", dbName, tableName, cols)
	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}
	return nil
}

func deleteTable(dbName, tableName string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err := db.Exec(fmt.Sprintf("DROP TABLE IF EXISTS `%s`.`%s`", dbName, tableName))
	if err != nil {
		return fmt.Errorf("failed to delete table: %v", err)
	}
	return nil
}

func listTables(dbName string) ([]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	rows, err := db.Query(fmt.Sprintf("SHOW TABLES IN `%s`", dbName))
	if err != nil {
		return nil, fmt.Errorf("failed to list tables: %v", err)
	}
	defer rows.Close()
	var tables []string
	for rows.Next() {
		var t string
		rows.Scan(&t)
		tables = append(tables, t)
	}
	return tables, nil
}

func getColumns(dbName, tableName string) ([]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	rows, err := db.Query(fmt.Sprintf("SHOW COLUMNS FROM `%s`.`%s`", dbName, tableName))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []string
	for rows.Next() {
		var field, typ, null, key, def, extra sql.NullString
		rows.Scan(&field, &typ, &null, &key, &def, &extra)
		if field.String != "id" {
			cols = append(cols, field.String)
		}
	}
	return cols, nil
}

// ---------- Record Operations ----------

func insertRecord(dbName, tableName string, record Record) (string, error) {
	dbMu.Lock()
	defer dbMu.Unlock()

	cols := []string{}
	vals := []interface{}{}
	placeholders := []string{}
	for k, v := range record {
		cols = append(cols, fmt.Sprintf("`%s`", k))
		vals = append(vals, v)
		placeholders = append(placeholders, "?")
	}

	query := fmt.Sprintf(
		"INSERT INTO `%s`.`%s` (%s) VALUES (%s)",
		dbName, tableName,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
	)
	result, err := db.Exec(query, vals...)
	if err != nil {
		return "", fmt.Errorf("failed to insert: %v", err)
	}
	id, _ := result.LastInsertId()
	return fmt.Sprintf("%d", id), nil
}

func selectRecords(dbName, tableName string) ([]map[string]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return queryRows(fmt.Sprintf("SELECT * FROM `%s`.`%s`", dbName, tableName))
}

func searchRecords(dbName, tableName, field, value string) ([]map[string]string, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	query := fmt.Sprintf("SELECT * FROM `%s`.`%s` WHERE `%s` LIKE ?", dbName, tableName, field)
	return queryRows(query, "%"+value+"%")
}

func updateRecord(dbName, tableName, id string, updates Record) error {
	dbMu.Lock()
	defer dbMu.Unlock()

	sets := []string{}
	vals := []interface{}{}
	for k, v := range updates {
		sets = append(sets, fmt.Sprintf("`%s` = ?", k))
		vals = append(vals, v)
	}
	vals = append(vals, id)

	query := fmt.Sprintf(
		"UPDATE `%s`.`%s` SET %s WHERE id = ?",
		dbName, tableName, strings.Join(sets, ", "),
	)
	_, err := db.Exec(query, vals...)
	if err != nil {
		return fmt.Errorf("failed to update: %v", err)
	}
	return nil
}

func deleteRecord(dbName, tableName, id string) error {
	dbMu.Lock()
	defer dbMu.Unlock()
	_, err := db.Exec(
		fmt.Sprintf("DELETE FROM `%s`.`%s` WHERE id = ?", dbName, tableName), id,
	)
	if err != nil {
		return fmt.Errorf("failed to delete: %v", err)
	}
	return nil
}

// ---------- Helper ----------

func queryRows(query string, args ...interface{}) ([]map[string]string, error) {
	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var results []map[string]string
	for rows.Next() {
		vals := make([]interface{}, len(cols))
		ptrs := make([]interface{}, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		rows.Scan(ptrs...)
		row := map[string]string{}
		for i, col := range cols {
			if vals[i] != nil {
				row[col] = fmt.Sprintf("%s", vals[i])
			} else {
				row[col] = ""
			}
		}
		results = append(results, row)
	}
	return results, nil
}

// getFullSnapshot returns all data for replication to new slaves
func getFullSnapshot() map[string]interface{} {
	dbs, _ := listDBs()
	snapshot := map[string]interface{}{}
	for _, dbName := range dbs {
		tables, _ := listTables(dbName)
		dbData := map[string]interface{}{}
		for _, tableName := range tables {
			cols, _ := getColumns(dbName, tableName)
			rows, _ := selectRecords(dbName, tableName)
			dbData[tableName] = map[string]interface{}{
				"columns": cols,
				"rows":    rows,
			}
		}
		snapshot[dbName] = dbData
	}
	return snapshot
}
