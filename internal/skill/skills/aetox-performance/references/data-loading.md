# Data loading: the shapes that are slow by design

Each row is a shape, the sign that it is present, how to see it in a
measurement, and the shape that replaces it. A shape is a finding only
with its number; the third column says where the number comes from.

| Shape | The sign | How to see it | The shape that fixes it |
|---|---|---|---|
| N+1 | one query or call per row after a query for the rows | log the queries for one action: the same statement N times with a different id | one query with `IN (...)` or a join; a batch call; a loader that collects ids and fetches once |
| Waterfall | fetch, await, fetch, await, for things that do not depend on each other | Network tab or a trace: requests start only when the previous ends; wall time is the sum, not the max | start them together (`Promise.all`, an `errgroup`), await where the value is used |
| Fetch on render | each component fetches its own data when it mounts, so the tree loads in layers | the same waterfall, one layer per nesting level | hoist the fetch to the route or the page, pass data down; or a loader that runs before the tree renders |
| Over-fetch | the whole row, table, file or tree loaded to use one field or one node | memory or transfer size far above the bytes on screen | select the columns; paginate; read the header, not the file; a summary endpoint |
| No pagination | a list endpoint or query with no limit | time and memory grow with the table; fine on the dev machine, slow on the user's | limit + cursor; virtualised list on screen |
| Load everything at startup | the app reads every project, every session, every file before the first screen | startup time grows with the user's data; a profile of startup shows I/O, not rendering | load what the first screen shows; the rest on demand or in the background |
| No shared cache | three screens fetch the same data, each with its own copy | the same request three times in one action | one store, one fetch, subscribers; a request-scoped dedup |
| Cache with no invalidation | a cache that is never cleared, or cleared on every write | stale reads, or a cache that never hits | invalidate by key on write; a short TTL where stale is acceptable, said out loud |
| No index | the filter column of the hottest query has no index | `EXPLAIN QUERY PLAN` says `SCAN`; time grows with the table | an index on the column, or a composite one in the query's order; measured after |
| Write shape read often | data stored the way it arrives (a log, an event stream) and reassembled on every read | the read does the same fold every time | a read model: materialise the shape the reads want, updated on write |
| Chatty boundary | one call per item across a process boundary (IPC, RPC, a worker, a native bridge) | count the calls per action; each has a fixed cost that dominates the work inside | one call with the list; a batch API |
| Sync I/O on the UI thread | a file read, a parse, or a query on the thread that paints | frames drop while data loads; the profile shows I/O inside the render | move it off the thread; stream it; show the shell first |
| Re-parse | the same bytes parsed on every call (JSON, YAML, a big config) | the parser at the top of the profile with the same input every time | parse once, keep the value; parse lazily per section |
| Unbounded growth | a list, a map, a log that only grows for the life of the process | memory rising over a session; a heap profile with one type dominating | a bound, an eviction, a ring; write to disk past the bound |
| Polling | a loop asking every N ms whether anything changed | CPU and I/O steady while idle | an event, a watch, a push; a longer interval with a wake on change |

## When it is the architecture, not a line

One of these shapes fixed in one place is a line fix. The same shape in
every screen, or a shape that a fix in one place cannot remove because
the callers all assume it, is an architecture finding. It is written as:

- the shape, named from the table above;
- the number it costs, on the action the user pointed at;
- the shape that replaces it, and the boundary that moves (where the
  fetch lives, where the cache lives, what the store's read model is);
- what else changes: every caller that assumed the old shape, the
  behaviour that is now different (staleness, page boundaries);
- and no rebuild in the same turn: `aetox-brainstorm` for the yes.

## A quick pass on a codebase with no profile yet

When there is no number yet and the user asks where to look, these three
counts give a first map in minutes, and each is a lead, not a finding:

1. Count fetches or queries per screen or handler; anything above a
   handful, and anything inside a loop, is a lead.
2. Count `await`s in sequence that do not use each other's results.
3. Count what runs at startup before the first paint or the first
   response.

Then measure the largest lead, and only that one, before reading further.
