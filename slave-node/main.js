const express = require("express");
const {
  connect, applyCreateDB, applyDropDB, applyCreateTable, applyDeleteTable,
  applyInsert, applyUpdate, applyDelete, applyFullSync,
  selectRecords, searchRecords, listDBs, listTables, getColumns, exportCSV
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

app.post("/sync", async (req, res) => {
  await applyFullSync(req.body);
  res.json({ status: "synced" });
});

app.post("/replicate", async (req, res) => {
  const { action, db, table, id, data, columns } = req.body;
  try {
    switch (action) {
      case "create_db":    await applyCreateDB(db); break;
      case "drop_db":      await applyDropDB(db); break;
      case "create_table": await applyCreateTable(db, table, columns); break;
      case "delete_table": await applyDeleteTable(db, table); break;
      case "insert":       await applyInsert(db, table, id, data); break;
      case "update":       await applyUpdate(db, table, id, data); break;
      case "delete":       await applyDelete(db, table, id); break;
    }
    res.json({ status: "applied" });
  } catch (e) {
    res.status(500).json({ error: e.message });
  }
});

app.get("/record/select", async (req, res) => {
  try {
    const rows = await selectRecords(req.query.db, req.query.table);
    res.json(rows);
  } catch (e) { res.status(400).json({ error: e.message }); }
});

app.get("/record/search", async (req, res) => {
  try {
    const rows = await searchRecords(req.query.db, req.query.table, req.query.field, req.query.value);
    res.json(rows);
  } catch (e) { res.status(400).json({ error: e.message }); }
});

app.get("/databases", async (req, res) => {
  res.json(await listDBs());
});

app.get("/tables", async (req, res) => {
  try { res.json(await listTables(req.query.db)); }
  catch (e) { res.status(400).json({ error: e.message }); }
});

app.get("/columns", async (req, res) => {
  try { res.json(await getColumns(req.query.db, req.query.table)); }
  catch (e) { res.status(400).json({ error: e.message }); }
});

// Special Feature: CSV Export
app.get("/export/csv", async (req, res) => {
  try {
    const csv = await exportCSV(req.query.db, req.query.table);
    res.header("Content-Type", "text/csv");
    res.send(csv);
  } catch (e) { res.status(400).json({ error: e.message }); }
});

// Start server after connecting to MySQL
connect()
  .then(() => {
    console.log("[SLAVE-NODE] Connected to MySQL");
    app.listen(PORT, () => console.log(`[SLAVE-NODE] Running on port ${PORT}`));
  })
  .catch(err => {
    console.error("[SLAVE-NODE] MySQL connection failed:", err.message);
    process.exit(1);
  });
