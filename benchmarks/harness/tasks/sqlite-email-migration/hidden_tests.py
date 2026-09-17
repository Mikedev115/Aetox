from __future__ import annotations

import json
import sqlite3
import sys
import tempfile
import unittest
from contextlib import closing
from pathlib import Path

import app


ROOT = Path(__file__).resolve().parent
INITIAL = ROOT / "migrations" / "001_initial.sql"


def legacy_database(path: Path, rows: list[tuple[str, str]]) -> None:
    with closing(sqlite3.connect(path)) as connection:
        connection.executescript(INITIAL.read_text(encoding="utf-8"))
        connection.execute(
            """
            CREATE TABLE schema_migrations (
                name TEXT PRIMARY KEY,
                applied_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
            )
            """
        )
        connection.execute(
            "INSERT INTO schema_migrations(name) VALUES (?)",
            ("001_initial.sql",),
        )
        connection.executemany(
            "INSERT INTO users(email, display_name) VALUES (?, ?)",
            rows,
        )
        connection.commit()


class HiddenDatabaseTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.database = Path(self.temp.name) / "app.db"

    def tearDown(self) -> None:
        self.temp.cleanup()

    def test_fresh_install_reaches_both_migrations(self) -> None:
        app.migrate(self.database)
        with closing(sqlite3.connect(self.database)) as connection:
            migrations = connection.execute(
                "SELECT name FROM schema_migrations ORDER BY name"
            ).fetchall()
            columns = {
                row[1] for row in connection.execute("PRAGMA table_info(users)")
            }
        self.assertEqual(2, len(migrations))
        self.assertEqual(("001_initial.sql",), migrations[0])
        self.assertTrue(migrations[1][0].startswith("002"), migrations)
        self.assertIn("email_key", columns)

    def test_upgrade_preserves_display_email_and_backfills_key(self) -> None:
        legacy_database(
            self.database,
            [("  Alice@Example.TEST ", "Alice"), ("Bob@example.test", "Bob")],
        )
        app.migrate(self.database)
        with closing(sqlite3.connect(self.database)) as connection:
            rows = connection.execute(
                "SELECT email, email_key FROM users ORDER BY id"
            ).fetchall()
        self.assertEqual(
            [
                ("  Alice@Example.TEST ", "alice@example.test"),
                ("Bob@example.test", "bob@example.test"),
            ],
            rows,
        )

    def test_create_enforces_canonical_uniqueness(self) -> None:
        app.create_user(self.database, "Alice@Example.TEST", "Alice")
        with self.assertRaises(sqlite3.IntegrityError):
            app.create_user(self.database, "  alice@example.test  ", "Duplicate")
        with closing(sqlite3.connect(self.database)) as connection:
            count = connection.execute("SELECT count(*) FROM users").fetchone()[0]
        self.assertEqual(1, count)

    def test_lookup_uses_canonical_key_and_preserves_display_value(self) -> None:
        app.create_user(self.database, "Alice@Example.TEST", "Alice")
        row = app.get_user_by_email(self.database, "  ALICE@example.test ")
        self.assertIsNotNone(row)
        self.assertEqual("Alice@Example.TEST", row["email"])

    def test_migrate_is_idempotent(self) -> None:
        app.migrate(self.database)
        app.migrate(self.database)
        with closing(sqlite3.connect(self.database)) as connection:
            migrations = connection.execute(
                "SELECT name, count(*) FROM schema_migrations GROUP BY name"
            ).fetchall()
        self.assertEqual(2, len(migrations))
        self.assertEqual(("001_initial.sql", 1), migrations[0])
        self.assertTrue(migrations[1][0].startswith("002"), migrations)
        self.assertEqual(1, migrations[1][1])

    def test_collision_rolls_back_schema_and_history(self) -> None:
        legacy_database(
            self.database,
            [("Alice@example.test", "One"), (" alice@EXAMPLE.test ", "Two")],
        )
        with self.assertRaises(Exception):
            app.migrate(self.database)
        with closing(sqlite3.connect(self.database)) as connection:
            columns = {
                row[1] for row in connection.execute("PRAGMA table_info(users)")
            }
            migrations = connection.execute(
                "SELECT name FROM schema_migrations ORDER BY name"
            ).fetchall()
            count = connection.execute("SELECT count(*) FROM users").fetchone()[0]
        self.assertNotIn("email_key", columns)
        self.assertEqual([("001_initial.sql",)], migrations)
        self.assertEqual(2, count)


if __name__ == "__main__":
    suite = unittest.defaultTestLoader.loadTestsFromTestCase(HiddenDatabaseTests)
    result = unittest.TextTestRunner(verbosity=2, stream=sys.stderr).run(suite)
    print(
        json.dumps(
            {
                "tests": result.testsRun,
                "passed": result.testsRun - len(result.failures) - len(result.errors),
                "failures": len(result.failures),
                "errors": len(result.errors),
                "successful": result.wasSuccessful(),
            },
            separators=(",", ":"),
        )
    )
    raise SystemExit(0 if result.wasSuccessful() else 1)
