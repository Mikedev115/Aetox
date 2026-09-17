# SQLite notes

- Identify the exact database file and use a new temporary file or safe copy
  for rehearsal. Copy a live database only through the SQLite backup API or a
  documented snapshot method; a plain file copy while it is active may omit
  journal/WAL state.
- Foreign-key enforcement is a per-connection setting and may be off by
  default. Enable `PRAGMA foreign_keys = ON` when the application requires it,
  then verify it on the same connection. Changing it inside a transaction does
  not take effect.
- Run `PRAGMA foreign_key_check` after relationship changes and
  `PRAGMA integrity_check` when file integrity is in scope. Index child-key
  columns used by foreign-key lookups where the workload warrants it.
- SQLite permits many readers but only one simultaneous writer. Keep write
  transactions short and follow the application's WAL and busy-timeout policy;
  do not invent a new one inside a migration.
- `ALTER TABLE` supports a limited set directly. Complex constraint, column or
  type changes commonly require creating a replacement table, copying data,
  rebuilding indexes/triggers and swapping tables. Rehearse that full sequence
  on a copy and verify row counts and constraints.

Official references:
https://www.sqlite.org/lang_transaction.html
https://www.sqlite.org/foreignkeys.html
https://www.sqlite.org/lang_altertable.html

