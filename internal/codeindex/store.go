package codeindex

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Mikedev115/Aetox/internal/config"
	_ "modernc.org/sqlite"
)

const cacheSchemaVersion = 3

const schema = `
CREATE TABLE IF NOT EXISTS facts (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  from_path   TEXT NOT NULL,
  from_line   INTEGER NOT NULL,
  from_end    INTEGER NOT NULL DEFAULT 0,
  from_name   TEXT NOT NULL DEFAULT '',
  to_path     TEXT NOT NULL,
  to_line     INTEGER NOT NULL,
  to_end      INTEGER NOT NULL DEFAULT 0,
  to_name     TEXT NOT NULL DEFAULT '',
  relation    TEXT NOT NULL,
  strength    TEXT NOT NULL,
  producer    TEXT NOT NULL,
  from_size   INTEGER NOT NULL,
  from_mtime  INTEGER NOT NULL,
  from_hash   TEXT NOT NULL,
  to_size     INTEGER NOT NULL,
  to_mtime    INTEGER NOT NULL,
  to_hash     TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_facts_from ON facts(from_path);
CREATE INDEX IF NOT EXISTS idx_facts_to ON facts(to_path);
CREATE INDEX IF NOT EXISTS idx_facts_producer ON facts(producer);

-- Which analyzers have run against this project. Row count cannot answer it: a
-- project with no imports at all legitimately stores nothing, and reading that
-- as "never indexed" would rebuild the whole project on every single question.
CREATE TABLE IF NOT EXISTS producers (
  name       TEXT PRIMARY KEY,
  -- The version of the analyzer that wrote the rows stored under this name. It
  -- is the only thing here that can tell one reading apart from the same
  -- reading taken by different code: a fact carries the content stamps of its
  -- two endpoints and nothing about the build that decided it existed, so a
  -- change to an analyzer used to leave its facts valid, served as fresh, and
  -- indistinguishable from a reading taken a minute ago. A row whose version is
  -- not the running build's is not this build's answer.
  version    TEXT NOT NULL DEFAULT '',
  indexed_at TEXT NOT NULL,
  -- How many relationships this analyzer could see and could not establish
  -- when it last ran. It belongs beside the row that says the analyzer ran:
  -- the count is a fact about that pass, and nothing that happens later can
  -- recompute it without walking the project again.
  unknown    INTEGER NOT NULL DEFAULT 0
);

-- Every file the walk actually mapped, whether or not it produced a fact.
-- Without this a file created after the index is indistinguishable from a file
-- with no relationships: neither has rows, and the query answers "nothing is
-- connected to this" about a file nobody has ever looked at.
CREATE TABLE IF NOT EXISTS sources (
  path       TEXT PRIMARY KEY,
  indexed_at TEXT NOT NULL
);
`

// Store owns one repository's derived fact cache.
//
// Every write goes through ReplaceProducer: a producer hands over its WHOLE
// reading of the project in one transaction, so the rows it owns are exactly
// the facts it just produced. A per-file replace cannot express that — a file
// whose imports were all deleted produces no facts at all, and with nothing to
// replace the row it left behind lives on, failing the content check on every
// later read and being counted as staleness that only ever grows.
type Store struct {
	root string
	db   *sql.DB
	mu   sync.Mutex
}

type fileStamp struct {
	size    int64
	mtime   int64
	sha256  string
	present bool
}

// Open opens the cache for root under Aetox's DataRoot.
func Open(root string) (*Store, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		return nil, fmt.Errorf("not a folder to index: %s", root)
	}
	dataRoot, err := config.DataRoot()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(dataRoot, "code-index")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, config.ProjectKey(abs)+".db")
	db, err := openCache(path)
	if err != nil {
		return nil, err
	}
	var version int
	if err := db.QueryRow(`PRAGMA user_version`).Scan(&version); err != nil {
		_ = db.Close()
		return nil, err
	}
	if version != 0 && version != cacheSchemaVersion {
		_ = db.Close()
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		db, err = openCache(path)
		if err != nil {
			return nil, err
		}
	}
	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, cacheSchemaVersion)); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{root: abs, db: db}, nil
}

func openCache(path string) (*sql.DB, error) {
	dsn := "file:" + filepath.ToSlash(path) + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	return sql.Open("sqlite", dsn)
}

// Close releases the SQLite handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// factRow is one row ready to write: both paths resolved inside the project and
// both endpoints stamped. Built outside the lock, because it reads and hashes
// files.
type factRow struct {
	fact     Fact
	fromPath string
	toPath   string
	from     fileStamp
	to       fileStamp
}

// ReplaceProducer replaces everything one analyzer has stored for this
// project. version is the version of the code that produced those facts, and it
// is written with them because the rows cannot say it themselves. unknown is how
// many relationships that pass saw and could not establish, carried with the
// rows for the same reason: it describes this pass, and nothing later can
// recompute it.
func (s *Store) ReplaceProducer(producer, version string, unknown int, bySource map[string][]Fact) error {
	producer = strings.TrimSpace(producer)
	if producer == "" {
		return errors.New("producer is required")
	}
	var rows []factRow
	for source, facts := range bySource {
		if len(facts) == 0 {
			return fmt.Errorf("source %q was handed an empty fact list; a producer that found nothing for a file simply leaves it out", source)
		}
		for _, fact := range facts {
			if fact.Producer != producer {
				return fmt.Errorf("fact producer %q does not match replacement producer %q", fact.Producer, producer)
			}
			row, err := s.prepareRow(source, fact)
			if err != nil {
				return err
			}
			rows = append(rows, row)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM facts WHERE producer = ?`, producer); err != nil {
		return err
	}
	if err := markIndexed(tx, producer, version, unknown); err != nil {
		return err
	}
	for _, row := range rows {
		if _, err := tx.Exec(insertFactSQL,
			row.fromPath, row.fact.From.Line, row.fact.From.EndLine, row.fact.From.Name,
			row.toPath, row.fact.To.Line, row.fact.To.EndLine, row.fact.To.Name,
			row.fact.Relation, row.fact.Strength, row.fact.Producer,
			row.from.size, row.from.mtime, row.from.sha256,
			row.to.size, row.to.mtime, row.to.sha256,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func markIndexed(tx *sql.Tx, producer, version string, unknown int) error {
	_, err := tx.Exec(`
INSERT INTO producers (name, version, indexed_at, unknown) VALUES (?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET version = excluded.version, indexed_at = excluded.indexed_at, unknown = excluded.unknown`,
		producer, version, time.Now().UTC().Format(time.RFC3339Nano), unknown)
	return err
}

// MarkIndexed records that an analyzer ran and found nothing to store — a
// project with no generated bindings, say. Without it, "this analyzer has
// nothing here" and "this analyzer has never run" are the same state, and the
// second one costs a full walk every time it is assumed.
func (s *Store) MarkIndexed(producer, version string, unknown int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := markIndexed(tx, producer, version, unknown); err != nil {
		return err
	}
	return tx.Commit()
}

// ReplaceSources records the files a full-project walk mapped. It is replaced
// wholesale with the pass that produced it, so a deleted file stops being
// listed as indexed.
func (s *Store) ReplaceSources(files []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM sources`); err != nil {
		return err
	}
	stamp := time.Now().UTC().Format(time.RFC3339Nano)
	for _, path := range files {
		if _, err := tx.Exec(`INSERT OR REPLACE INTO sources (path, indexed_at) VALUES (?, ?)`, path, stamp); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// Indexed reports whether the walk that built this cache had mapped this file.
func (s *Store) IndexedSource(path string) (bool, error) {
	resolved, _, err := s.resolve(path)
	if err != nil {
		return false, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var found string
	err = s.db.QueryRow(`SELECT path FROM sources WHERE path = ?`, resolved).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// Unknown is what the last pass of this analyzer could not establish.
func (s *Store) Unknown(producer string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var unknown int
	err := s.db.QueryRow(`SELECT unknown FROM producers WHERE name = ?`, producer).Scan(&unknown)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return unknown, nil
}

// Indexed reports whether this analyzer has run against this project and left
// behind the reading the version that is asking would have written. A row from
// an older build answers the first half of that and not the second: it is a
// reading, but not this one. Collapsing the two is how a cache goes on
// answering as the build before the analyzer changed.
func (s *Store) Indexed(producer, version string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var stored string
	err := s.db.QueryRow(`SELECT version FROM producers WHERE name = ?`, producer).Scan(&stored)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return stored == version, nil
}

const insertFactSQL = `
INSERT INTO facts (
  from_path, from_line, from_end, from_name,
  to_path, to_line, to_end, to_name,
  relation, strength, producer,
  from_size, from_mtime, from_hash,
  to_size, to_mtime, to_hash
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

func (s *Store) prepareRow(source string, fact Fact) (factRow, error) {
	fromPath, fromAbs, err := s.resolve(fact.From.Path)
	if err != nil {
		return factRow{}, err
	}
	resolvedSource, _, err := s.resolve(source)
	if err != nil {
		return factRow{}, err
	}
	if fromPath != resolvedSource {
		return factRow{}, fmt.Errorf("fact source %q does not match replacement source %q", fromPath, resolvedSource)
	}
	toPath, toAbs, err := s.resolve(fact.To.Path)
	if err != nil {
		return factRow{}, err
	}
	from, err := stampFile(fromAbs)
	if err != nil {
		return factRow{}, err
	}
	to, err := stampFile(toAbs)
	if err != nil {
		return factRow{}, err
	}
	return factRow{fact: fact, fromPath: fromPath, toPath: toPath, from: from, to: to}, nil
}

// FactsFrom returns valid facts produced from path and how many cached rows
// were dropped as no longer matching the files.
func (s *Store) FactsFrom(path string) ([]Fact, int, error) {
	path, _, err := s.resolve(path)
	if err != nil {
		return nil, 0, err
	}
	return s.readFacts(`SELECT `+factColumns+` FROM facts WHERE from_path = ? ORDER BY id`, path)
}

// FactsTo returns valid facts whose destination is path.
func (s *Store) FactsTo(path string) ([]Fact, int, error) {
	path, _, err := s.resolve(path)
	if err != nil {
		return nil, 0, err
	}
	return s.readFacts(`SELECT `+factColumns+` FROM facts WHERE to_path = ? ORDER BY id`, path)
}

// FactsBetween returns valid facts running from one file to another.
func (s *Store) FactsBetween(from, to string) ([]Fact, int, error) {
	from, _, err := s.resolve(from)
	if err != nil {
		return nil, 0, err
	}
	to, _, err = s.resolve(to)
	if err != nil {
		return nil, 0, err
	}
	return s.readFacts(`SELECT `+factColumns+` FROM facts WHERE from_path = ? AND to_path = ? ORDER BY id`, from, to)
}

const factColumns = `from_path, from_line, from_end, from_name, to_path, to_line, to_end, to_name, relation, strength, producer, from_size, from_mtime, from_hash, to_size, to_mtime, to_hash`

// readFacts reads the rows under the lock and decides about them outside it.
// The decision hashes files, and doing that while holding the store's lock
// would make every other caller wait on somebody else's disk.
func (s *Store) readFacts(query string, args ...any) ([]Fact, int, error) {
	type cachedRow struct {
		fact Fact
		from fileStamp
		to   fileStamp
	}
	rows, err := func() ([]cachedRow, error) {
		s.mu.Lock()
		defer s.mu.Unlock()
		cursor, err := s.db.Query(query, args...)
		if err != nil {
			return nil, err
		}
		defer cursor.Close()
		var out []cachedRow
		for cursor.Next() {
			var row cachedRow
			if err := cursor.Scan(
				&row.fact.From.Path,
				&row.fact.From.Line,
				&row.fact.From.EndLine,
				&row.fact.From.Name,
				&row.fact.To.Path,
				&row.fact.To.Line,
				&row.fact.To.EndLine,
				&row.fact.To.Name,
				&row.fact.Relation,
				&row.fact.Strength,
				&row.fact.Producer,
				&row.from.size,
				&row.from.mtime,
				&row.from.sha256,
				&row.to.size,
				&row.to.mtime,
				&row.to.sha256,
			); err != nil {
				return nil, err
			}
			out = append(out, row)
		}
		return out, cursor.Err()
	}()
	if err != nil {
		return nil, 0, err
	}

	var facts []Fact
	stale := 0
	for _, row := range rows {
		current, err := s.rowIsCurrent(row.fact, row.from, row.to)
		if err != nil {
			// A file that cannot be read at all is not the same statement as a
			// file that changed: the first says the answer is unknown, and
			// folding it into the second would keep reporting an old fact as
			// merely out of date. One unreadable endpoint is reported, not
			// swallowed.
			return nil, stale, err
		}
		if !current {
			stale++
			continue
		}
		facts = append(facts, row.fact)
	}
	return facts, stale, nil
}

func (s *Store) rowIsCurrent(fact Fact, from, to fileStamp) (bool, error) {
	_, fromAbs, err := s.resolve(fact.From.Path)
	if err != nil {
		return false, err
	}
	_, toAbs, err := s.resolve(fact.To.Path)
	if err != nil {
		return false, err
	}
	fromCurrent, err := stampMatches(fromAbs, from)
	if err != nil || !fromCurrent {
		return false, err
	}
	return stampMatches(toAbs, to)
}

func (s *Store) resolve(path string) (string, string, error) {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) {
		return "", "", fmt.Errorf("path must be project-relative: %q", path)
	}
	abs := filepath.Join(s.root, filepath.FromSlash(path))
	rel, err := filepath.Rel(s.root, abs)
	if err != nil {
		return "", "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("path leaves the project: %q", path)
	}
	return filepath.ToSlash(rel), abs, nil
}

func stampFile(path string) (fileStamp, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fileStamp{}, err
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return fileStamp{}, err
	}
	sum := sha256.Sum256(body)
	return fileStamp{size: info.Size(), mtime: info.ModTime().UnixNano(), sha256: hex.EncodeToString(sum[:])}, nil
}

// stampMatches is the cheap gate first and the content second: a file whose
// size and modification time are untouched is the file that was read, and only
// a visible change costs a hash. A file that has gone is not current — and it
// is not an error either, since "this was deleted" is a fact about the project
// that its own cache has to accept.
func stampMatches(path string, expected fileStamp) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if info.Size() == expected.size && info.ModTime().UnixNano() == expected.mtime {
		return true, nil
	}
	actual, err := stampFile(path)
	if err != nil {
		return false, err
	}
	return actual.sha256 == expected.sha256, nil
}
