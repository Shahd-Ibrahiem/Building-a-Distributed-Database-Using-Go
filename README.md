# Distributed Database System — Go

A distributed database system with 1 Master and 3 Slave nodes, built using Go, Python, and Node.js.

## Architecture

```
Master (Go :8080)
    ├── Slave 1 (Go     :8081)
    ├── Slave 2 (Python :8082)  ← Special: Table Stats API
    └── Slave 3 (Node.js:8083)  ← Special: CSV Export API
```

## How to Run

### 1. Master (Go)
```bash
cd master
go run .
```

### 2. Slave Go
```bash
cd slave-go
PORT=8081 go run .
```

### 3. Slave Python
```bash
cd slave-python
pip install -r requirements.txt
PORT=8082 python main.py
```

### 4. Slave Node.js
```bash
cd slave-node
npm install
PORT=8083 node main.js
```

### 5. GUI
Open `gui/index.html` in your browser.

---

## Register Slaves with Master

After all nodes are running, register the slaves:

```bash
curl -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{"address":"localhost:8081"}'
curl -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{"address":"localhost:8082"}'
curl -X POST http://localhost:8080/register -H "Content-Type: application/json" -d '{"address":"localhost:8083"}'
```

---

## API Reference

### Master Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/db/create` | Create a new database |
| POST | `/db/drop` | Drop a database (Master only) |
| POST | `/table/create` | Create a table dynamically |
| POST | `/table/delete` | Delete a table |
| GET  | `/tables?db=` | List tables in a database |
| GET  | `/databases` | List all databases |
| POST | `/record/insert` | Insert a record |
| GET  | `/record/select?db=&table=` | Select all records |
| GET  | `/record/search?db=&table=&field=&value=` | Search records |
| POST | `/record/update` | Update a record |
| POST | `/record/delete` | Delete a record |
| POST | `/register` | Register a slave node |

### Special Slave Features

| Node | Endpoint | Feature |
|------|----------|---------|
| Python (:8082) | `GET /stats?db=&table=` | Table statistics |
| Node.js (:8083) | `GET /export/csv?db=&table=` | Export table as CSV |

---

## Usage Examples

```bash
# Create database
curl -X POST http://localhost:8080/db/create -H "Content-Type: application/json" \
  -d '{"name":"school"}'

# Create table
curl -X POST http://localhost:8080/table/create -H "Content-Type: application/json" \
  -d '{"db":"school","table":"students","columns":["name","age","grade"]}'

# Insert record
curl -X POST http://localhost:8080/record/insert -H "Content-Type: application/json" \
  -d '{"db":"school","table":"students","record":{"name":"Ali","age":"20","grade":"A"}}'

# Select all
curl "http://localhost:8080/record/select?db=school&table=students"

# Search
curl "http://localhost:8080/record/search?db=school&table=students&field=name&value=ali"

# Update
curl -X POST http://localhost:8080/record/update -H "Content-Type: application/json" \
  -d '{"db":"school","table":"students","id":"1","updates":{"grade":"A+"}}'

# Delete record
curl -X POST http://localhost:8080/record/delete -H "Content-Type: application/json" \
  -d '{"db":"school","table":"students","id":"1"}'

# Drop database (Master only)
curl -X POST http://localhost:8080/db/drop -H "Content-Type: application/json" \
  -d '{"name":"school"}'
```

---

## Features

- Master/Slave replication (automatic sync on every write)
- Fault tolerance: slaves keep serving reads if master goes down
- Persistence: data saved to JSON files on disk
- Dynamic database and table creation
- Full CRUD operations on all nodes
- Two bonus slave technologies (Python + Node.js)
- Web GUI dashboard
