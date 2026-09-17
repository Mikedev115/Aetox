from __future__ import annotations

import sqlite3
import tempfile
import unittest
from contextlib import closing
from pathlib import Path

import app


class ApplicationTests(unittest.TestCase):
    def setUp(self) -> None:
        self.temp = tempfile.TemporaryDirectory()
        self.database = Path(self.temp.name) / "app.db"

    def tearDown(self) -> None:
        self.temp.cleanup()

    def test_creates_and_reads_a_user(self) -> None:
        user_id = app.create_user(self.database, "one@example.test", "One")
        row = app.get_user_by_email(self.database, "one@example.test")
        self.assertEqual(user_id, row["id"])
        self.assertEqual("One", row["display_name"])

    def test_records_initial_migration(self) -> None:
        app.migrate(self.database)
        with closing(sqlite3.connect(self.database)) as connection:
            names = connection.execute(
                "SELECT name FROM schema_migrations ORDER BY name"
            ).fetchall()
        self.assertIn(("001_initial.sql",), names)


if __name__ == "__main__":
    unittest.main()
