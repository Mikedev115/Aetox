# Security checklist: the questions, by category

Each row is a question to ask of the code, and what to search for to find
the places where the question applies. A hit on the search is a place to
read, not a finding; SKILL.md says what a finding is.

Sources: OWASP Top 10 (web), CWE Top 25, the category list of
anthropics/claude-code-security-review, and OWASP Top 10 for Agentic
Applications 2026 (ASI01 to ASI10) for the last section.

## Application code

| Category | The question | Where to look (grep leads) |
|---|---|---|
| Command injection | Does anything from outside reach a shell or a process argument list without being one argument, or being validated against a fixed set? | `exec.Command`, `sh -c`, `subprocess`, `child_process`, `os.system`, `Runtime.exec`, backticks; then trace the arguments backwards |
| SQL / NoSQL injection | Is a query built by string concatenation or format with a value from outside? | `fmt.Sprintf(` next to `SELECT`/`INSERT`/`UPDATE`/`WHERE`, `+ "` in a query, `$where`, `.raw(`, `text(` |
| Path traversal | Can a name from outside reach `open`/`read`/`write`/`delete`/`serve` without being resolved against a root and checked to still be under it? Symlinks resolved? | `filepath.Join(root, userInput)`, `os.Open`, `sendFile`, `path.join`, `../`, `ReadFile` |
| Template / expression injection | Does user input become the template rather than a value in it? | `template.New(...).Parse(userInput)`, `eval`, `new Function`, `render_template_string`, `Jinja2` with `autoescape=False` |
| Deserialisation | Is untrusted bytes decoded into objects by a format that can carry code or references? | `pickle`, `yaml.load` without `SafeLoader`, `unserialize`, `ObjectInputStream`, `Marshal.load`; JSON is fine, what is done with it after may not be |
| XSS | Does a value from outside reach HTML, an attribute or a script without escaping for that context? | `innerHTML`, `dangerouslySetInnerHTML`, `{@html`, `v-html`, `template.HTML(`, `\|safe`, `document.write` |
| SSRF | Does a URL from outside get fetched by the server? Can it reach localhost, link-local (169.254.x.x), or the metadata endpoint? | `http.Get(userURL)`, `fetch(url)` on the server, `requests.get(`, image proxies, webhooks, "import from URL" |
| Authentication | Can the check be skipped: a route without the middleware, a token accepted without signature or `alg: none`, a password compared with `==`, a reset token that is guessable or never expires? | route tables, `jwt.Parse` without key verification, `subtle.ConstantTimeCompare` absent, `math/rand` for tokens |
| Authorization | Does the handler check that the caller may touch *this* object, not just that the caller is logged in? | any handler that takes an id from the request and loads by it; admin-only routes; "is owner" checks |
| Session | Cookie without `HttpOnly`/`Secure`/`SameSite`, session id not rotated at login, logout that does not invalidate | `http.SetCookie`, `session.regenerate`, `Set-Cookie` |
| Secrets in source or history | A key, token, password or private key in a file or in a past commit | `git log -p -S "BEGIN PRIVATE KEY"`, `-S "sk-"`, `-S "AKIA"`, `.env` committed, `password =` in config; a scanner (`gitleaks`) does this well |
| Crypto | MD5/SHA1 for passwords, ECB mode, a fixed IV or nonce, `math/rand` where `crypto/rand` is needed, `InsecureSkipVerify: true`, `verify=False` | those literals |
| Data exposure | Secrets, tokens, or personal data in log lines, error messages, stack traces returned to the client, debug endpoints left on | `log.Printf` with request bodies or headers, `pprof` on a public listener, `DEBUG = True`, verbose error handlers |
| Code execution | `eval`, dynamic import or `require` of a name from input, plugins loaded from a path the user controls | `eval(`, `exec(`, `importlib.import_module(userInput)`, `require(variable)`, `plugin.Open` |
| Supply chain | A dependency with a known vulnerability, unpinned (`latest`, a branch), a package with an install script, a lockfile missing or not used in CI | `go.mod`, `package.json` + lock, `requirements.txt`, `postinstall`, `curl \| sh` in build scripts |
| Configuration | Fail-open defaults: auth off unless configured, CORS `*` with credentials, bound to `0.0.0.0` when local was meant, TLS optional, file permissions world-writable | config defaults, `AllowAllOrigins`, `Listen("0.0.0.0` vs `127.0.0.1`, `0777`, `chmod` |
| Race / TOCTOU | A security decision made, then the thing re-read: check a file then open it by name; check a balance then debit in a second query | check-then-act pairs on files, money, quotas; missing transactions or locks |
| Uploads | Extension trusted, content type trusted, file written under a name from the client, served back from the same origin as the app | upload handlers, `Content-Type` from the request used to decide, `filename` from the form |
| Redirects | Open redirect to a URL from the request; low on its own, high when it carries a token | `Redirect(w, r, userURL` , `next=`, `returnTo=` |

## Apps with an agent, tools or MCP servers

This is the OWASP Top 10 for Agentic Applications 2026, phrased as the
question to ask of an app like this one.

| ID | Name | The question |
|---|---|---|
| ASI01 | Agent goal hijack | Can content the model reads (a tool result, a fetched page, a file, an email, a comment) change what it does next? Is that content marked as data when it is put back into the prompt? Is there any place where instructions found in content are followed without the user? |
| ASI02 | Tool misuse | Can a tool be called with arguments outside what its approval covered: a `read` that reads a secret store, a `write` outside the project, a `shell` that runs something the user did not see? Is the approval prompt shown the real arguments? |
| ASI03 | Identity and privilege abuse | Does a subagent, a worker or a scheduled job carry the parent's full rights? Can a tool filter be widened from inside a session? Is an API key usable for more than the one provider it was given for? |
| ASI04 | Agentic supply chain | Is an MCP server, a skill or a plugin installed from a name alone, with no pin, no hash and no reading of what it declares? Does an installed skill get to rewrite other skills? |
| ASI05 | Unexpected code execution | Is there a road from model output to `eval`, a shell, or a file that is then executed, that skips the gate every other tool passes through? A calculator that can reach the filesystem is this. |
| ASI06 | Memory and context poisoning | Can content the model read end up in a memory file, a persona file or a skill that every later session loads? Who approves the write, and is the source shown? |
| ASI07 | Insecure inter-agent communication | Do agents pass each other text that is then treated as instructions? Is a remote engine socket authenticated, and bound to localhost or a key? |
| ASI08 | Cascading failures | Can one bad tool result make a loop (retry, re-plan, re-delegate) that never stops, or that widens what it tries? Where is the ceiling? |
| ASI09 | Human trust exploitation | Can the model make an approval prompt look safer than it is: a summary that hides the dangerous argument, an action taken as "read" that writes? Does the user see the argument, not the model's description of it? |
| ASI10 | Rogue agents | Is there a kill switch a person can reach while a run is going? Does a background job, a scheduled task or a desktop companion keep rights after the session that made it is gone? |

## Reading a stack quickly

- Go: the sandbox is a `SandboxRoot`-style resolve; check every file tool
  goes through it. `exec.Command` with a slice is safe from shell parsing
  and unsafe when the binary name comes from input. `html/template`
  escapes, `text/template` does not.
- Node / TypeScript: `child_process.exec` parses a shell, `execFile` does
  not. Svelte and React escape by default; `{@html}` and
  `dangerouslySetInnerHTML` are the exceptions to read.
- Python: `subprocess` with `shell=True`, `yaml.load` without a loader,
  `pickle`, f-strings into SQL. Flask `debug=True` on a public bind.
- SQL: any driver has a parameter form; the finding is the query that does
  not use it.
