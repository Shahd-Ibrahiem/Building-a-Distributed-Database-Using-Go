const express = require("express");
const {
  loadAllDBs, applyCreateDB, applyDropDB,
  applyCreateTable, applyDeleteTable,
  applyInsert, applyUpdate, applyDelete,
  applyFullSync, selectRecords, searchRecords,
  getDatabases, getTables, exportCSV
} = require("./db");

const app = express();
const PORT = process.env.PORT || 8083;

app.use(express.json());
app.use((req, res, next) => {
  res.header("Access-Control-Allow-Origin", "*");
  res.header("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS");
  res.header("Access-Control-Allow-Headers", "Content-Type");
  if (req.method === "OPTIONS") return res.sendStatus(200);
  next();
});

app.get("/health", (req, res) => res.json({ status: "ok", role: "slave-node" }));

// Full sync from master
app.post("/sync", (req, res) => {
  applyFullSync(req.body);
  res.json({ status: "synced" });
});

// Replicate
app.post("/replicate", (req, res) => {
  const { action, db, table, id, data, columns } = req.body;
  switch (action) {
    case "create_db":     applyCreateDB(db); break;
    case "drop_db":       applyDropDB(db); break;
    case "create_table":  applyCreateTable(db, table, columns); break;
    case "delete_table":  applyDeleteTable(db, table); break;
    case "insert":        applyInsert(db, table, id, data); break;
    case "update":        applyUpdate(db, table, id, data); break;
    case "delete":        applyDelete(db, table, id); break;
  }
  res.json({ status: "applied" });
});

// Select
app.get("/record/select", (req, res) => {
  const result = selectRecords(req.query.db, req.query.table);
  if (result?.error) return res.status(400).json(result);
  res.json(result);
});

// Search
app.get("/record/search", (req, res) => {
  const result = searchRecords(req.query.db, req.query.table, req.query.field, req.query.value);
  if (result?.error) return res.status(400).json(result);
  res.json(result);
});

// Databases
app.get("/databases", (req, res) => res.json(getDatabases()));

// Tables
app.get("/tables", (req, res) => {
  const t = getTables(req.query.db);
  if (!t) return res.status(404).json({ error: "db not found" });
  res.json(t);
});

// Special Feature: Export table as CSV (Node.js unique contribution)
app.get("/export/csv", (req, res) => {
  const result = exportCSV(req.query.db, req.query.table);
  if (typeof result === "object" && result.error) return res.status(400).json(result);
  res.header("Content-Type", "text/csv");
  res.send(result);
});

loadAllDBs();
app.listen(PORT, () => console.log(`[SLAVE-NODE] Running on port ${PORT}`));
