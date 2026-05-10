from flask import Flask, request, jsonify
import os
from db import (
    apply_create_db, apply_drop_db, apply_create_table, apply_delete_table,
    apply_insert, apply_update, apply_delete, apply_full_sync,
    select_records, search_records, list_dbs, list_tables, get_columns, get_table_stats
)

app = Flask(__name__)
PORT = int(os.environ.get("PORT", 8082))

@app.after_request
def add_cors(response):
    response.headers["Access-Control-Allow-Origin"] = "*"
    response.headers["Access-Control-Allow-Methods"] = "GET, POST, DELETE, OPTIONS"
    response.headers["Access-Control-Allow-Headers"] = "Content-Type"
    return response

@app.route("/health")
def health():
    return jsonify({"status": "ok", "role": "slave-python"})

@app.route("/sync", methods=["POST", "OPTIONS"])
def sync():
    if request.method == "OPTIONS": return jsonify({}), 200
    apply_full_sync(request.get_json())
    return jsonify({"status": "synced"})

@app.route("/replicate", methods=["POST", "OPTIONS"])
def replicate():
    if request.method == "OPTIONS": return jsonify({}), 200
    p = request.get_json()
    action = p.get("action")
    if action == "create_db":    apply_create_db(p["db"])
    elif action == "drop_db":    apply_drop_db(p["db"])
    elif action == "create_table": apply_create_table(p["db"], p["table"], p.get("columns", []))
    elif action == "delete_table": apply_delete_table(p["db"], p["table"])
    elif action == "insert":     apply_insert(p["db"], p["table"], p["id"], p.get("data", {}))
    elif action == "update":     apply_update(p["db"], p["table"], p["id"], p.get("data", {}))
    elif action == "delete":     apply_delete(p["db"], p["table"], p["id"])
    return jsonify({"status": "applied"})

@app.route("/record/select")
def select():
    rows, err = select_records(request.args.get("db"), request.args.get("table"))
    if err: return jsonify({"error": err}), 400
    return jsonify(rows)

@app.route("/record/search")
def search():
    rows, err = search_records(request.args.get("db"), request.args.get("table"), request.args.get("field"), request.args.get("value"))
    if err: return jsonify({"error": err}), 400
    return jsonify(rows)

@app.route("/databases")
def databases():
    return jsonify(list_dbs())

@app.route("/tables")
def tables():
    t, err = list_tables(request.args.get("db"))
    if err: return jsonify({"error": err}), 404
    return jsonify(t)

@app.route("/columns")
def columns():
    cols, err = get_columns(request.args.get("db"), request.args.get("table"))
    if err: return jsonify({"error": err}), 400
    return jsonify(cols)

@app.route("/stats")
def stats():
    result, err = get_table_stats(request.args.get("db"), request.args.get("table"))
    if err: return jsonify({"error": err}), 400
    return jsonify(result)

if __name__ == "__main__":
    print(f"[SLAVE-PYTHON] Running on port {PORT}")
    app.run(host="0.0.0.0", port=PORT)
