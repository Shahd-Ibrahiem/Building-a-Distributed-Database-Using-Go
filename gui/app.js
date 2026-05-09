// ---- Config ----
const MASTER = "http://localhost:8080";
const SLAVE_GO = "http://localhost:8081";
const SLAVE_PY = "http://localhost:8082";
const SLAVE_NODE = "http://localhost:8083";

const NODES = [
  { name: "Master (Go)",    addr: MASTER,     role: "master" },
  { name: "Slave 1 (Go)",   addr: SLAVE_GO,   role: "slave-go" },
  { name: "Slave 2 (Py)",   addr: SLAVE_PY,   role: "slave-python" },
  { name: "Slave 3 (Node)", addr: SLAVE_NODE, role: "slave-node" },
];

let currentDB = null;
let currentTable = null;

// ---- Init ----
window.onload = async () => {
  await refreshNodes();
  await refreshDBs();
  setInterval(refreshNodes, 5000);
};

// ---- API helper ----
async function api(url, method = "GET", body = null) {
  const opts = { method, headers: { "Content-Type": "application/json" } };
  if (body) opts.body = JSON.stringify(body);
  const res = await fetch(url, opts);
  const ct = res.headers.get("content-type") || "";
  if (ct.includes("application/json")) return res.json();
  return res.text();
}

// ---- Nodes ----
async function refreshNodes() {
  const list = document.getElementById("nodes-list");
  const results = await Promise.all(
    NODES.map(n => fetch(n.addr + "/health").then(r => r.json()).catch(() => null))
  );
  list.innerHTML = NODES.map((n, i) => {
    const up = results[i] !== null;
    return `<div class="node-item"><div class="dot ${up ? "" : "down"}"></div>${n.name}</div>`;
  }).join("");
}

// ---- Databases ----
async function refreshDBs() {
  const dbs = await api(MASTER + "/databases").catch(() => []);
  const list = document.getElementById("db-list");
  list.innerHTML = (dbs || []).map(db =>
    `<div class="db-item ${db === currentDB ? "active" : ""}" onclick="selectDB('${db}')">${db}</div>`
  ).join("");
}

async function selectDB(name) {
  currentDB = name;
  currentTable = null;
  document.getElementById("current-context").textContent = `📁 ${name}`;
  document.getElementById("btn-drop-db").style.display = "";
  document.getElementById("btn-create-table").style.display = "";
  document.getElementById("toolbar").style.display = "none";
  document.getElementById("results").innerHTML = "";
  await refreshDBs();
  await refreshTables();
}

// ---- Tables ----
async function refreshTables() {
  const tables = await api(MASTER + `/tables?db=${currentDB}`).catch(() => []);
  const bar = document.getElementById("tables-bar");
  bar.innerHTML = (tables || []).map(t =>
    `<div class="table-tab ${t === currentTable ? "active" : ""}" onclick="selectTable('${t}')">${t}</div>`
  ).join("");
}

async function selectTable(name) {
  currentTable = name;
  document.getElementById("toolbar").style.display = "";
  await refreshTables();
  await loadRecords();
  // Show special features based on node
  document.getElementById("special-btn").innerHTML = `
    <button onclick="fetchStats()" title="Python slave feature">📊 Stats (Py)</button>
    <button onclick="exportCSV()" title="Node.js slave feature">⬇ CSV (Node)</button>
  `;
}

// ---- Records ----
async function loadRecords() {
  if (!currentDB || !currentTable) return;
  const rows = await api(MASTER + `/record/select?db=${currentDB}&table=${currentTable}`);
  renderTable(rows);
}

function renderTable(rows) {
  const div = document.getElementById("results");
  if (!rows || rows.length === 0) {
    div.innerHTML = `<div class="empty">No records found</div>`; return;
  }
  const cols = Object.keys(rows[0]);
  div.innerHTML = `<table>
    <thead><tr>${cols.map(c => `<th>${c}</th>`).join("")}<th>Actions</th></tr></thead>
    <tbody>
      ${rows.map(r => `<tr>
        ${cols.map(c => `<td>${r[c] ?? ""}</td>`).join("")}
        <td>
          <button class="action-btn" onclick='openUpdate(${JSON.stringify(r)})'>Edit</button>
          <button class="action-btn del" onclick="deleteRecord('${r.id}')">Del</button>
        </td>
      </tr>`).join("")}
    </tbody>
  </table>`;
}

// ---- Special Features ----
async function fetchStats() {
  const data = await api(SLAVE_PY + `/stats?db=${currentDB}&table=${currentTable}`);
  if (data.error) { toast(data.error, true); return; }
  document.getElementById("results").innerHTML = `
    <div class="stats-box">
      <div class="stat"><span class="stat-val">${data.row_count}</span><span class="stat-label">Rows</span></div>
      <div class="stat"><span class="stat-val">${data.column_count}</span><span class="stat-label">Columns</span></div>
      <div class="stat"><span class="stat-val">${data.columns.join(", ")}</span><span class="stat-label">Column Names</span></div>
    </div>
  ` + document.getElementById("results").innerHTML;
}

async function exportCSV() {
  const csv = await fetch(SLAVE_NODE + `/export/csv?db=${currentDB}&table=${currentTable}`).then(r => r.text());
  const blob = new Blob([csv], { type: "text/csv" });
  const a = document.createElement("a");
  a.href = URL.createObjectURL(blob);
  a.download = `${currentDB}_${currentTable}.csv`;
  a.click();
  toast("CSV downloaded!");
}

// ---- Modals ----
let columns = [];

function openModal(type) {
  const overlay = document.getElementById("modal-overlay");
  const content = document.getElementById("modal-content");
  overlay.classList.remove("hidden");
  columns = [];

  if (type === "create-db") {
    content.innerHTML = `
      <h3>Create Database</h3>
      <label>Database Name</label>
      <input id="m-db-name" placeholder="e.g. mydb" />
      <div class="modal-actions">
        <button onclick="closeModal()">Cancel</button>
        <button class="primary" onclick="createDB()">Create</button>
      </div>`;

  } else if (type === "drop-db") {
    content.innerHTML = `
      <h3>Drop Database</h3>
      <p style="color:#ef4444;margin-bottom:8px">This will permanently delete <b>${currentDB}</b> and all its data!</p>
      <div class="modal-actions">
        <button onclick="closeModal()">Cancel</button>
        <button class="danger" onclick="dropDB()">Drop</button>
      </div>`;

  } else if (type === "create-table") {
    content.innerHTML = `
      <h3>Create Table in <em>${currentDB}</em></h3>
      <label>Table Name</label>
      <input id="m-table-name" placeholder="e.g. users" />
      <label>Columns</label>
      <div style="display:flex;gap:6px;margin-bottom:6px">
        <input id="m-col-input" placeholder="Column name" onkeydown="if(event.key==='Enter')addColumn()" />
        <button onclick="addColumn()">Add</button>
      </div>
      <div class="columns-input" id="col-tags"></div>
      <div class="modal-actions">
        <button onclick="closeModal()">Cancel</button>
        <button class="primary" onclick="createTable()">Create</button>
      </div>`;

  } else if (type === "delete-table") {
    content.innerHTML = `
      <h3>Delete Table</h3>
      <p style="color:#ef4444;margin-bottom:8px">Delete table <b>${currentTable}</b>?</p>
      <div class="modal-actions">
        <button onclick="closeModal()">Cancel</button>
        <button class="danger" onclick="deleteTable()">Delete</button>
      </div>`;

  } else if (type === "insert") {
    const db = null; // we'll fetch columns
    fetchColumnsAndShowInsert();
    return;

  } else if (type === "search") {
    content.innerHTML = `
      <h3>Search in <em>${currentTable}</em></h3>
      <label>Field</label>
      <input id="m-search-field" placeholder="e.g. name" />
      <label>Value</label>
      <input id="m-search-value" placeholder="search term" />
      <div class="modal-actions">
        <button onclick="closeModal()">Cancel</button>
        <button class="primary" onclick="searchRecords()">Search</button>
      </div>`;
  }
}

async function fetchColumnsAndShowInsert() {
  // Get columns directly from master endpoint (works even if table is empty)
  let cols = await api(MASTER + `/columns?db=${currentDB}&table=${currentTable}`);
  if (!Array.isArray(cols) || cols.length === 0) cols = ["value"];

  const overlay = document.getElementById("modal-overlay");
  const content = document.getElementById("modal-content");
  overlay.classList.remove("hidden");
  content.innerHTML = `
    <h3>Insert into <em>${currentTable}</em></h3>
    ${cols.map(c => `<label>${c}</label><input id="ins_${c}" placeholder="${c}" />`).join("")}
    <div class="modal-actions">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="insertRecord([${cols.map(c=>`'${c}'`).join(",")}])">Insert</button>
    </div>`;
}

function openUpdate(row) {
  const cols = Object.keys(row).filter(c => c !== "id");
  const content = document.getElementById("modal-content");
  document.getElementById("modal-overlay").classList.remove("hidden");
  content.innerHTML = `
    <h3>Update Record #${row.id}</h3>
    ${cols.map(c => `<label>${c}</label><input id="upd_${c}" value="${row[c] || ""}" />`).join("")}
    <div class="modal-actions">
      <button onclick="closeModal()">Cancel</button>
      <button class="primary" onclick="updateRecord('${row.id}',[${cols.map(c=>`'${c}'`).join(",")}])">Update</button>
    </div>`;
}

function closeModal() {
  document.getElementById("modal-overlay").classList.add("hidden");
}

// ---- Column tags ----
function addColumn() {
  const input = document.getElementById("m-col-input");
  const val = input.value.trim();
  if (!val) return;
  columns.push(val);
  input.value = "";
  renderColTags();
}

function removeColumn(i) {
  columns.splice(i, 1);
  renderColTags();
}

function renderColTags() {
  document.getElementById("col-tags").innerHTML = columns.map((c, i) =>
    `<div class="col-tag">${c} <span onclick="removeColumn(${i})">✕</span></div>`
  ).join("");
}

// ---- CRUD ----
async function createDB() {
  const name = document.getElementById("m-db-name").value.trim();
  if (!name) return;
  await api(MASTER + "/db/create", "POST", { name });
  closeModal(); toast("Database created!"); await refreshDBs();
}

async function dropDB() {
  await api(MASTER + "/db/drop", "POST", { name: currentDB });
  currentDB = null; currentTable = null;
  document.getElementById("current-context").textContent = "Select a database";
  document.getElementById("btn-drop-db").style.display = "none";
  document.getElementById("btn-create-table").style.display = "none";
  document.getElementById("tables-bar").innerHTML = "";
  document.getElementById("toolbar").style.display = "none";
  document.getElementById("results").innerHTML = "";
  closeModal(); toast("Database dropped!"); await refreshDBs();
}

async function createTable() {
  const name = document.getElementById("m-table-name").value.trim();
  if (!name || columns.length === 0) { toast("Add a name and at least one column", true); return; }
  await api(MASTER + "/table/create", "POST", { db: currentDB, table: name, columns });
  closeModal(); toast("Table created!"); await refreshTables();
}

async function deleteTable() {
  await api(MASTER + "/table/delete", "POST", { db: currentDB, table: currentTable });
  currentTable = null;
  document.getElementById("toolbar").style.display = "none";
  document.getElementById("results").innerHTML = "";
  closeModal(); toast("Table deleted!"); await refreshTables();
}

async function insertRecord(cols) {
  const record = {};
  cols.forEach(c => { record[c] = document.getElementById(`ins_${c}`).value; });
  await api(MASTER + "/record/insert", "POST", { db: currentDB, table: currentTable, record });
  closeModal(); toast("Record inserted!"); await loadRecords();
}

async function updateRecord(id, cols) {
  const updates = {};
  cols.forEach(c => { updates[c] = document.getElementById(`upd_${c}`).value; });
  await api(MASTER + "/record/update", "POST", { db: currentDB, table: currentTable, id, updates });
  closeModal(); toast("Record updated!"); await loadRecords();
}

async function deleteRecord(id) {
  await api(MASTER + "/record/delete", "POST", { db: currentDB, table: currentTable, id });
  toast("Record deleted!"); await loadRecords();
}

async function searchRecords() {
  const field = document.getElementById("m-search-field").value.trim();
  const value = document.getElementById("m-search-value").value.trim();
  const rows = await api(MASTER + `/record/search?db=${currentDB}&table=${currentTable}&field=${field}&value=${value}`);
  closeModal(); renderTable(rows);
}

// ---- Toast ----
function toast(msg, error = false) {
  const t = document.getElementById("toast");
  t.textContent = msg;
  t.className = error ? "error show" : "show";
  setTimeout(() => t.className = "", 2500);
}
