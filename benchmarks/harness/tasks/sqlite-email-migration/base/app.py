from __future__ import annotations

import sqlite3
from pathlib import Path


MIGRATIONS = Path(__file__).with_name("migrations")


def connect(database: str | Path) -> sqlite3.Connection:
    connection = sqlite3.connect(database)
    connection.row_factory = sqlite3.Row
    connection.execute("PRAGMA foreign_keys = ON")
    return connection


def migrate(database: str | Path) -> None:
    connection = connect(database)
    try:
        connection.execute(
            """
            CREATE TABLE IF NOT EXISTS schema_migrations (
                name TEXT PRIMARY KEY,
                applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """
        )
        applied = {
            row["name"]
            for row in connection.execute("SELECT name FROM schema_migrations")
        }
        for migration in sorted(MIGRATIONS.glob("*.sql")):
            if migration.name in applied:
                continue
            connection.executescript(migration.read_text(encoding="utf-8"))
            connection.execute(
                "INSERT INTO schema_migrations(name) VALUES (?)",
                (migration.name,),
            )
        connection.commit()
    finally:
        connection.close()

def create_user(database: str | Path, email: str, display_name: str) -> int:
    migrate(database)
    connection = connect(database)
    try:
        cursor = connection.execute(
            "INSERT INTO users(email, display_name) VALUES (?, ?)",
            (email, display_name),
        )
        connection.commit()
        return int(cursor.lastrowid)
    finally:
        connection.close()


def get_user_by_email(database: str | Path, email: str) -> sqlite3.Row | None:
    migrate(database)
    connection = connect(database)
    try:
        return connection.execute(
            "SELECT id, email, display_name FROM users WHERE email = ?",
            (email,),
        ).fetchone()
    finally:
        connection.close()
