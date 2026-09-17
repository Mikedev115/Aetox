# MySQL notes

- Confirm MySQL/MariaDB product, server version, storage engine and `sql_mode`;
  they materially change DDL and validation behavior.
- Do not treat a transaction wrapper as rollback protection for DDL. MySQL
  documents many statements that implicitly commit, and DDL changes generally
  cannot be rolled back as ordinary transactional changes can.
- Preview generated SQL, rehearse it against a disposable database restored
  from representative structure/data, and back up a non-disposable local
  database before irreversible work.
- For changes to a populated table, inspect whether the exact server version
  supports the requested online DDL algorithm and lock mode. Do not claim a
  migration is online merely because `ALGORITHM` or `LOCK` syntax was present;
  inspect warnings and the observed behavior.
- Verify with `SHOW CREATE TABLE`, index/constraint metadata, affected row
  counts and application tests. Read `SHOW WARNINGS` after statements whose
  coercion or online-DDL fallback matters.
- Keep data backfills bounded and restartable. Autocommit is normally enabled;
  make transaction boundaries explicit for data work instead of assuming a
  CLI invocation is one atomic unit.

Official references:
https://dev.mysql.com/doc/refman/8.4/en/implicit-commit.html
https://dev.mysql.com/doc/refman/8.4/en/cannot-roll-back.html
https://dev.mysql.com/doc/refman/8.4/en/innodb-autocommit-commit-rollback.html

