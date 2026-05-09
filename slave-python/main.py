from flask import Flask, request, jsonify
import os
from db import (
    load_all_dbs, apply_create_db, apply_drop_db,
    apply_create_table, apply_delete_table,
    apply_insert, apply_update, apply_delete,
    apply_full_sync, select_records, search_records,
    get_databases, get_tables, get_table_stats
)

app = Flask(__name__)
PORT = int(os.environ.get("PORT", 8082))

# ---- CORS ----
@app.after_request
def add_cors(response):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Methods"] = "GET, POST, DELETE, OPTIONS"
    response.headers["Access-Control-Allow-Headers"] = "Content-Type"
    return response

@app.route("/health")
def health():
    return jsonify({"status": "ok", "role": "slave-python"})

# ---- Sync from master ----
@app.route("/sync", methods=["POST", "OPTIONS"])
def sync():
    if request.method == "OPTIONS": return jsonify({}), 200
    snapshot = request.get_json()
    apply_full_sync(snapshot)
    return jsonify({"status": "synced"})

# ---- Replicate ----
@app.route("/replicate", methods=["POST", "OPTIONS"])
def replicate():
    if request.method == "OPTIONS": return jsonify({}), 200
    payload = request.get_json()
    action = payload.get("action")

    if action == "create_db":
        apply_create_db(payload["db"])
    elif action == "drop_db":
        apply_drop_db(payload["db"])
    elif action == "create_table":
        apply_create_table(payload["db"], payload["table"], payload.get("columns", []))
    elif action == "delete_table":
        apply_delete_table(payload["db"], payload["table"])
    elif action == "insert":
        apply_insert(payload["db"], payload["table"], payload["id"], payload.get("data", {}))
    elif action == "update":
        apply_update(payload["db"], payload["table"], payload["id"], payload.get("data", {}))
    elif action == "delete":
        apply_delete(payload["db"], payload["table"], payload["id"])

    return jsonify({"status": "applied"})

# ---- Read queries ----
@app.route("/record/select")
def select():
    db = request.args.get("db")
    table = request.args.get("table")
    rows, err = select_records(db, table)
    if err:
        return jsonify({"error": err}), 400
    return jsonify(rows)

@app.route("/record/search")
def search():
    db = request.args.get("db")
    table = request.args.get("table")
    field = request.args.get("field")
    value = request.args.get("value")
    rows, err = search_records(db, table, field, value)
    if err:
        return jsonify({"error": err}), 400
    return jsonify(rows)

@app.route("/databases")
def databases():
    return jsonify(get_databases())

@app.route("/tables")
def tables():
    db = request.args.get("db")
    t = get_tables(db)
    if t is None:
        return jsonify({"error": "db not found"}), 404
    return jsonify(t)

# ---- Special Feature: Table Stats (Python's unique contribution) ----
@app.route("/stats")
def stats():
    db = request.args.get("db")
    table = request.args.get("table")
    result, err = get_table_stats(db, table)
    if err:
        return jsonify({"error": err}), 400
    return jsonify(result)

if __name__ == "__main__":
    load_all_dbs()
    print(f"[SLAVE-PYTHON] Running on port {PORT}")
    app.run(host="0.0.0.0", port=PORT)
