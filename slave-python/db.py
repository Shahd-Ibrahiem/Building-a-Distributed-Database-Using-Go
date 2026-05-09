import json
import os
import threading
from copy import deepcopy

databases = {}
db_lock = threading.RLock()

def save_db(name):
    db = databases.get(name)
    if db is None:
        return
    with open(f"data_{name}.json", "w") as f:
        json.dump(db, f, indent=2)

def load_all_dbs():
    for fname in os.listdir("."):
        if fname.startswith("data_") and fname.endswith(".json"):
            with open(fname) as f:
                try:
                    db = json.load(f)
                    databases[db["name"]] = db
                except:
                    pass

def apply_create_db(name):
    with db_lock:
        if name not in databases:
            databases[name] = {"name": name, "tables": {}}
            save_db(name)

def apply_drop_db(name):
    with db_lock:
        databases.pop(name, None)
        try: os.remove(f"data_{name}.json")
        except: pass

def apply_create_table(db_name, table_name, columns):
    with db_lock:
        db = databases.get(db_name)
        if db and table_name not in db["tables"]:
            db["tables"][table_name] = {"columns": columns, "rows": {}, "next_id": 1}
            save_db(db_name)

def apply_delete_table(db_name, table_name):
    with db_lock:
        db = databases.get(db_name)
        if db:
            db["tables"].pop(table_name, None)
            save_db(db_name)

def apply_insert(db_name, table_name, record_id, record):
    with db_lock:
        db = databases.get(db_name)
        if db:
            table = db["tables"].get(table_name)
            if table:
                table["rows"][record_id] = record
                save_db(db_name)

def apply_update(db_name, table_name, record_id, updates):
    with db_lock:
        db = databases.get(db_name)
        if db:
            table = db["tables"].get(table_name)
            if table and record_id in table["rows"]:
                table["rows"][record_id].update(updates)
                save_db(db_name)

def apply_delete(db_name, table_name, record_id):
    with db_lock:
        db = databases.get(db_name)
        if db:
            table = db["tables"].get(table_name)
            if table:
                table["rows"].pop(record_id, None)
                save_db(db_name)

def apply_full_sync(snapshot):
    with db_lock:
        databases.clear()
        databases.update(snapshot)
        for name in databases:
            save_db(name)

def select_records(db_name, table_name):
    with db_lock:
        db = databases.get(db_name)
        if not db:
            return None, "database not found"
        table = db["tables"].get(table_name)
        if not table:
            return None, "table not found"
        results = []
        for rid, row in table["rows"].items():
            r = {"id": rid, **row}
            results.append(r)
        return results, None

def search_records(db_name, table_name, field, value):
    with db_lock:
        db = databases.get(db_name)
        if not db:
            return None, "database not found"
        table = db["tables"].get(table_name)
        if not table:
            return None, "table not found"
        results = []
        for rid, row in table["rows"].items():
            if field in row and value.lower() in row[field].lower():
                results.append({"id": rid, **row})
        return results, None

def get_databases():
    with db_lock:
        return list(databases.keys())

def get_tables(db_name):
    with db_lock:
        db = databases.get(db_name)
        if not db:
            return None
        return list(db["tables"].keys())

# Special feature: return stats about a table (Python's unique contribution)
def get_table_stats(db_name, table_name):
    with db_lock:
        db = databases.get(db_name)
        if not db:
            return None, "database not found"
        table = db["tables"].get(table_name)
        if not table:
            return None, "table not found"
        row_count = len(table["rows"])
        col_count = len(table["columns"])
        return {
            "db": db_name,
            "table": table_name,
            "row_count": row_count,
            "column_count": col_count,
            "columns": table["columns"],
        }, None
