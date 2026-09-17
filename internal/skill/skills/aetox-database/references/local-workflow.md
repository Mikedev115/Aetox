# Local database workflow

## 1. Establish the repository's truth

Inspect, in this order:

1. Dependency and workspace manifests for the driver, ORM and migration tool.
2. Schema/model files and the full ordered migration history.
3. Test and development configuration, example environment files, container
   manifests and startup scripts.
4. The command the repository already uses to create, migrate and test a
   database.

Record the engine and version, the migration authority, and whether schema is
derived from models, migrations, or introspection. Do not introduce a second
migration path.

## 2. Prove the target is local

Resolve configuration without exposing secrets. Accept a target as local only
when evidence identifies one of these:

- a SQLite or equivalent file inside the workspace or an explicit temporary
  directory;
- a Unix-domain socket on this machine;
- `localhost`, `127.0.0.1`, or `::1` with the expected local port;
- a test database created by the current test process;
- a container service whose repository manifest proves it belongs to the
  local development stack.

Names such as `DATABASE_URL`, `DEV_DATABASE_URL`, `.env.local`, database
suffixes such as `_dev`, and a CLI's default profile are not proof. If the
resolved host is external, hidden, ambiguous, or selected by an account
profile, do not connect.

Log or report only a redacted identity such as engine, host class, port and
database name. Never echo a password, token, query-string secret or complete
URL.

## 3. Inspect before mutation

- Read migration status and current schema with the project's own tool when it
  can be done against the proven local target.
- Use bounded read queries: explicit columns, narrow predicates and a limit.
- Capture preconditions that can be checked again: schema version, relevant
  row counts, null counts, uniqueness and orphan counts.
- For an existing non-disposable local database, make and verify a restorable
  backup before destructive or irreversible work.

## 4. Rehearse, apply, verify

Create a fresh temporary database and run the same creation/migration command
the application uses. Apply the proposed change there first. Then verify:

- the complete migration chain succeeds from empty;
- the intended tables, columns, indexes and constraints exist;
- data invariants hold and no unexpected rows changed;
- the application can read and write through its normal data-access layer;
- focused tests and the repository's database integration tests pass.

For a data migration, also test representative empty, duplicate, null,
boundary and already-migrated cases. Make reruns safe when the operation may be
interrupted.

## 5. Report the evidence

State the redacted target, files added or changed, exact repository commands,
pre/postcondition results, tests, and backup/restore rehearsal if one was
needed. Do not say a migration is safe, reversible or idempotent unless that
property was exercised.

