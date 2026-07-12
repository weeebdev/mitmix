import json
import os
import sqlite3
import threading

STORE_DIR = os.path.expanduser("~/.mitm-agent")
STORE_PATH = os.path.join(STORE_DIR, "store.db")


class LocalStore:
    def __init__(self, db_path=None):
        self.db_path = db_path or STORE_PATH
        if self.db_path and self.db_path != ":memory:":
            os.makedirs(os.path.dirname(self.db_path), exist_ok=True)
        self._lock = threading.Lock()
        self._init_db()

    def _connect(self):
        return sqlite3.connect(self.db_path)

    def _init_db(self):
        with self._lock:
            conn = self._connect()
            conn.execute("PRAGMA journal_mode=WAL")
            conn.execute("CREATE TABLE IF NOT EXISTS rules (id TEXT PRIMARY KEY, data TEXT NOT NULL)")
            conn.execute("CREATE TABLE IF NOT EXISTS flows (id INTEGER PRIMARY KEY AUTOINCREMENT, data TEXT NOT NULL, created_at TEXT DEFAULT (datetime('now')))")
            conn.execute("CREATE TABLE IF NOT EXISTS meta (key TEXT PRIMARY KEY, value TEXT)")
            conn.execute("INSERT OR IGNORE INTO meta (key, value) VALUES ('last_synced_id', '0')")
            conn.commit()
            conn.close()

    def save_rules(self, rules):
        with self._lock:
            conn = self._connect()
            conn.execute("DELETE FROM rules")
            conn.executemany(
                "INSERT INTO rules (id, data) VALUES (?, ?)",
                [(r.get("id", ""), json.dumps(r)) for r in rules],
            )
            conn.commit()
            conn.close()

    def load_rules(self):
        with self._lock:
            conn = self._connect()
            rows = conn.execute("SELECT data FROM rules ORDER BY rowid").fetchall()
            conn.close()
            return [json.loads(r[0]) for r in rows]

    def rule_count(self):
        with self._lock:
            conn = self._connect()
            row = conn.execute("SELECT COUNT(*) FROM rules").fetchone()
            conn.close()
            return row[0] if row else 0

    def append_flow(self, flow):
        with self._lock:
            conn = self._connect()
            conn.execute("INSERT INTO flows (data) VALUES (?)", (json.dumps(flow),))
            conn.commit()
            conn.close()

    def get_unsynced(self, limit=100):
        with self._lock:
            conn = self._connect()
            last_id = conn.execute("SELECT value FROM meta WHERE key = 'last_synced_id'").fetchone()
            cursor = int(last_id[0]) if last_id else 0
            rows = conn.execute(
                "SELECT id, data FROM flows WHERE id > ? ORDER BY id LIMIT ?",
                (cursor, limit),
            ).fetchall()
            conn.close()
            return [(r[0], json.loads(r[1])) for r in rows]

    def unsynced_count(self):
        with self._lock:
            conn = self._connect()
            last_id = conn.execute("SELECT value FROM meta WHERE key = 'last_synced_id'").fetchone()
            cursor = int(last_id[0]) if last_id else 0
            row = conn.execute("SELECT COUNT(*) FROM flows WHERE id > ?", (cursor,)).fetchone()
            conn.close()
            return row[0] if row else 0

    def mark_synced(self, max_local_id):
        with self._lock:
            conn = self._connect()
            conn.execute(
                "UPDATE meta SET value = ? WHERE key = 'last_synced_id' AND CAST(value AS INTEGER) < ?",
                (str(max_local_id), max_local_id),
            )
            conn.commit()
            conn.close()

    def cleanup_old(self, older_than_hours=24):
        with self._lock:
            conn = self._connect()
            last_id = conn.execute("SELECT value FROM meta WHERE key = 'last_synced_id'").fetchone()
            cursor = int(last_id[0]) if last_id else 0
            conn.execute(
                "DELETE FROM flows WHERE id <= ? AND created_at < datetime('now', ?)",
                (cursor, f"-{older_than_hours} hours"),
            )
            conn.commit()
            conn.close()
