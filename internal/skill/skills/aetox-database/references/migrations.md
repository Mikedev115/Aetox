# Migration discipline

## History is an append-only contract

The repository's migration directory and migration table together describe
what deployed databases may already have seen. Create a new migration for a
new change. Do not edit, reorder, squash or delete an applied migration unless
the repository has an explicit, coordinated history-repair procedure.

Commit schema/model changes and their migration together. Generated SQL must
be read before it is applied; a generator is a diff producer, not an authority
on intent.

Prisma documents the same core rule: migration history is committed source,
and applied migrations should not be edited or deleted:
https://docs.prisma.io/docs/orm/prisma-migrate/understanding-prisma-migrate/migration-histories

## Change sequence

1. Write preconditions and desired postconditions.
2. Generate a migration in create-only, dry-run or preview mode when the tool
   supports it.
3. Inspect the SQL for destructive operations, implicit casts, full-table
   rewrites, locks, defaults, nullability, indexes and engine-specific DDL.
4. Apply the whole history to a new disposable database.
5. Seed or load a small representative data set and apply the migration from
   the previous version.
6. Check schema, constraints and data postconditions, then run application
   tests through the normal data-access path.

Development generation and production-style application are different acts.
For example, Prisma reserves `migrate dev` for development and documents
`migrate deploy` for applying pending migrations outside development:
https://docs.prisma.io/docs/orm/prisma-migrate/workflows/development-and-production

## Incompatible changes use expand and contract

For a rename, type change, new required field, table split or relationship
rewrite:

1. Expand with a compatible nullable column/table/index or dual-readable
   shape.
2. Deploy code that can tolerate old and new data; dual-write only when the
   consistency plan is explicit.
3. Backfill in bounded, restartable batches with progress and invariant
   checks.
4. Switch reads, then stop writing the old shape.
5. Enforce the new constraint after data validates.
6. Contract in a later migration by removing the old shape.

A large backfill does not belong in one long schema transaction. Separate the
schema step from resumable data work.

## Rollback is not a slogan

Some engines implicitly commit DDL; some operations are expensive or
irreversible even inside a transaction. Prefer a tested forward fix when a
down migration would lose data. If rollback is supplied, rehearse both forward
and backward paths and prove what data survives. Otherwise label recovery as
restore-from-backup or forward-fix, not rollback.

