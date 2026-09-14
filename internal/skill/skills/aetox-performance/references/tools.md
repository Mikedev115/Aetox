# Tools: how to get the number, per stack

The first column is what is slow; the second is the command that says
where the time goes. Every one of these produces a file or a table that
names a function or a step, which is where reading starts.

## Go

| What is slow | Command | Read |
|---|---|---|
| A function or a request path | `go test -run '^$' -bench 'Name' -count=10 -cpuprofile cpu.out ./pkg` then `go tool pprof -top cpu.out` | the `flat%` column top down; `cum%` for the callers that own the time |
| Memory, GC pressure | the same with `-memprofile mem.out`, then `go tool pprof -sample_index=alloc_space -top mem.out` | allocations per op; `-benchmem` prints `allocs/op` beside `ns/op` |
| Waiting on locks or I/O | `-blockprofile block.out` / `-mutexprofile mutex.out`; a running process: `runtime/pprof` or `net/http/pprof` on a local port | wall time above CPU time is the sign it is waiting, not working |
| Goroutines over time, scheduler | `go test -trace trace.out` then `go tool trace trace.out` | the timeline: gaps are waiting, stacked bars are contention |
| Before vs after | `go test -bench . -count=10 > old.txt`, change, `> new.txt`, then `benchstat old.txt new.txt` | benchstat prints the delta with a p-value; `~` means the change is inside the noise |
| Escapes to the heap | `go build -gcflags='-m' ./pkg 2>&1 \| grep 'escapes'` | only after the profile says allocation is the cost |
| The build | `go build -x ./... 2>&1 \| head` shows each step; `go list -deps ./cmd/x \| wc -l` counts what is compiled; `go build -a -v` after `go clean -cache` times a cold build per package | one package or cgo step that dominates; a generated file rebuilt every time |
| Profile-guided optimisation | `go build -pgo=cpu.out` | measure it with benchstat; on this repository it was inside the noise (BENCHMARK.md) |

`benchstat`: `go install golang.org/x/perf/cmd/benchstat@latest`. Ten runs
is the floor for a claim; five for a first look.

## Node, TypeScript, Vite, Svelte, React

| What is slow | Command | Read |
|---|---|---|
| A script or server | `node --cpu-prof script.js` writes `.cpuprofile`; open in Chrome DevTools Performance (load profile) | self time top down |
| Memory | `node --heap-prof`, or DevTools Memory: two heap snapshots, compare | what grew between them |
| The page | DevTools Performance, record the interaction; `performance.mark()`/`measure()` around the suspect | long tasks, layout thrash, the main thread blocked |
| First load | Lighthouse (DevTools Lighthouse tab), or `web-vitals` in the page | LCP, INP, CLS; the request waterfall in the Network tab, look for serial chains |
| The bundle | `vite build` with `rollup-plugin-visualizer`, or `npx vite-bundle-visualizer`; esbuild `--metafile` | the largest chunk, and the library pulled in for one function |
| The build | `vite build --profile` writes a `.cpuprofile`; `tsc --extendedDiagnostics`, `tsc --generateTrace dir` | which plugin or which type check owns the time |
| Re-renders | React DevTools Profiler; Svelte: `$inspect` and DevTools Performance | a component rendered on every keystroke |
| Waterfalls | Network tab, or read the code: an `await` before a fetch that did not need it | independent fetches started one after another: `Promise.all` them |

## SQL and SQLite

| What is slow | Command | Read |
|---|---|---|
| A query | SQLite: `EXPLAIN QUERY PLAN <query>`; Postgres: `EXPLAIN (ANALYZE, BUFFERS) <query>` | `SCAN` on a big table where the filter column has no index; a nested loop over a large row count |
| Many queries | count them: log every query for one action, or trace the driver | the same statement N times with a different id is the N+1 |
| Writes | SQLite: are they in one transaction? `PRAGMA journal_mode`, `synchronous` | one transaction per row is a fsync per row |

## Python

| What is slow | Command | Read |
|---|---|---|
| A script | `python -m cProfile -s cumtime script.py` | cumulative time top down |
| A running process | `py-spy top --pid N`, `py-spy record -o out.svg --pid N` | the flame graph |

## The whole machine (the USE pass)

When nothing in the code is obviously it, walk each resource and ask
three questions: Utilisation (how busy), Saturation (how much is queued),
Errors. CPU, memory, disk, network, and for a desktop app the GPU and the
UI thread. Task Manager, `top`/`htop`, Resource Monitor. A resource at
100% utilisation or with a queue is the bottleneck; the code that uses it
is where to profile next. (Brendan Gregg's USE method.)

## Wall time versus CPU time

The one distinction that sorts most cases: a profile with the CPU busy
the whole time is work, and the fix is doing less of it; a profile with
the CPU mostly idle across a long wall time is waiting, and the fix is
starting things at once, or not waiting for something that is not needed
yet. Measure both before choosing which list to read.
