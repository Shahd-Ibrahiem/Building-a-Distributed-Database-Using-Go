const fs = require("fs");
const path = require("path");

let databases = {};

function saveDB(name) {
  const db = databases[name];
  if (!db) return;
  fs.writeFileSync(`data_${name}.json`, JSON.stringify(db, null, 2));
}

function loadAllDBs() {
  const files = fs.readdirSync(".");
  for (const f of files) {
    if (f.startsWith("data_") && f.endsWith(".json")) {
      try {
        const db = JSON.parse(fs.readFileSync(f, "utf8"));
        databases[db.name] = db;
      } catch {}
    }
  }
}

function applyCreateDB(name) {
  if (!databases[name]) {
    databases[name] = { name, tables: {} };
    saveDB(name);
  }
}

function applyDropDB(name) {
  delete databases[name];
  try { fs.unlinkSync(`data_${name}.json`); } catch {}
}

function applyCreateTable(dbName, tableName, columns) {
  const db = databases[dbName];
  if (db && !db.tables[tableName]) {
    db.tables[tableName] = { columns, rows: {}, next_id: 1 };
    saveDB(dbName);
  }
}

function applyDeleteTable(dbName, tableName) {
  const db = databases[dbName];
  if (db) {
    delete db.tables[tableName];
    saveDB(dbName);
  }
}

function applyInsert(dbName, tableName, id, record) {
  const db = databases[dbName];
  if (db && db.tables[tableName]) {
    db.tables[tableName].rows[id] = record;
    saveDB(dbName);
  }
}

function applyUpdate(dbName, tableName, id, updates) {
  const db = databases[dbName];
  if (db && db.tables[tableName] && db.tables[tableName].rows[id]) {
    Object.assign(db.tables[tableName].rows[id], updates);
    saveDB(dbName);
  }
}

function applyDelete(dbName, tableName, id) {
  const db = databases[dbName];
  if (db && db.tables[tableName]) {
    delete db.tables[tableName].rows[id];
    saveDB(dbName);
  }
}

function applyFullSync(snapshot) {
  databases = snapshot;
  for (const name of Object.keys(databases)) {
    saveDB(name);
  }
}

function selectRecords(dbName, tableName) {
  const db = databases[dbName];
  if (!db) return { error: "database not found" };
  const table = db.tables[tableName];
  if (!table) return { error: "table not found" };
  return Object.entries(table.rows).map(([id, row]) => ({ id, ...row }));
}

function searchRecords(dbName, tableName, field, value) {
  const db = databases[dbName];
  if (!db) return { error: "database not found" };
  const table = db.tables[tableName];
  if (!table) return { error: "table not found" };
  return Object.entries(table.rows)
    .filter(([, row]) => row[field] && row[field].toLowerCase().includes(value.toLowerCase()))
    .map(([id, row]) => ({ id, ...row }));
}

function getDatabases() {
  return Object.keys(databases);
}

function getTables(dbName) {
  const db = databases[dbName];
  if (!db) return null;
  return Object.keys(db.tables);
}

// Special feature: export table as CSV (Node.js unique contribution)
function exportCSV(dbName, tableName) {
  const db = databases[dbName];
  if (!db) return { error: "database not found" };
  const table = db.tables[tableName];
  if (!table) return { error: "table not found" };

  const cols = ["id", ...table.columns];
  const rows = Object.entries(table.rows).map(([id, row]) =>
    cols.map(c => (c === "id" ? id : (row[c] || ""))).join(",")
  );
  return [cols.join(","), ...rows].join("\n");
}

module.exports = {
  loadAllDBs, applyCreateDB, applyDropDB,
  applyCreateTable, applyDeleteTable,
  applyInsert, applyUpdate, applyDelete,
  applyFullSync, selectRecords, searchRecords,
  getDatabases, getTables, exportCSV
};
