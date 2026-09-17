# PostgreSQL notes

- Confirm server major version and current database. For inspection use a
  read-only transaction where practical; it is additional containment, not a
  substitute for proving the target is local.
- Set conservative `lock_timeout` and `statement_timeout` for migration
  rehearsal. Keep transactions short: locks are normally held until the
  transaction ends.
- Read the exact `ALTER TABLE` form for its lock level. Adding constraints,
  rewriting a column, changing a type or installing a default can block or
  rewrite more than its syntax suggests.
- For large existing tables, stage compatible changes. Where supported and
  appropriate, add a constraint `NOT VALID`, repair/verify data, then
  `VALIDATE CONSTRAINT`; make required/nullability changes only after the data
  already satisfies them.
- Build large indexes with the engine's concurrency-aware mechanism when the
  deployment needs writes to continue. `CREATE INDEX CONCURRENTLY` has special
  transaction and failure semantics; do not wrap it in an ordinary migration
  transaction without checking tool support.
- `EXPLAIN ANALYZE` executes the query. Use plain `EXPLAIN` first and execute
  only against a proven local data set when runtime evidence is required.
- For a non-disposable local database, take a restorable `pg_dump` (or the
  project's established backup) before irreversible data or schema work and
  test that the artifact can be read/restored.

Official references:
https://www.postgresql.org/docs/current/sql-begin.html
https://www.postgresql.org/docs/current/ddl-alter.html
https://www.postgresql.org/docs/current/explicit-locking.html

