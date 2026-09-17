Upgrade this local Python/SQLite application so email identity is canonical and unique.

Requirements:

1. Canonical form is `email.strip().lower()`; inputs in this benchmark are ASCII.
2. Preserve the original `users.email` text byte-for-byte for display.
3. Add an `email_key` column and database-enforced uniqueness in a NEW migration. Do not edit or replace `migrations/001_initial.sql`.
4. A database that already ran migration 001 must be backfilled. A fresh database must reach the same schema through the full migration chain.
5. `create_user` must write the canonical key using bound parameters. `get_user_by_email` must accept surrounding whitespace and any letter case.
6. Running `migrate` repeatedly must be safe.
7. If legacy rows collide after canonicalization, migration must fail atomically: do not record migration 002 and do not leave a partial `email_key` schema behind.
8. Use only the Python standard library. Keep the public function signatures in `app.py`.

Run the visible tests and finish the implementation. Do not ask a follow-up question.
