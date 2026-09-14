# Scanners: what to run when one is installed

A scanner finds places; a person, or a model doing the person's job, reads
the place before it becomes a finding. Its output is a list of leads for
the trace in SKILL.md, not a report to forward.

Check with `where <tool>` (Windows) or `which <tool>` first. None of these
ship with Aetox. Installing one changes the user's machine, so it is asked
for, not done; the install line is below so the ask is concrete.

| Tool | Stack | Command | What it finds | Install |
|---|---|---|---|---|
| `govulncheck` | Go | `govulncheck ./...` | known CVEs in dependencies, and only the ones your code actually calls | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| `gosec` | Go | `gosec ./...` | unsafe patterns in your own code: `InsecureSkipVerify`, weak rand, `exec` with variables, file perms | `go install github.com/securego/gosec/v2/cmd/gosec@latest` |
| `staticcheck` | Go | `staticcheck ./...` | correctness, not security; still catches the unchecked error before a security check | `go install honnef.co/go/tools/cmd/staticcheck@latest` |
| `npm audit` | Node | `npm audit --audit-level=high` | known CVEs in the lockfile | comes with npm |
| `pip-audit` | Python | `pip-audit` | known CVEs in installed packages | `pip install pip-audit` |
| `semgrep` | any | `semgrep --config auto .` | pattern rules per language, OWASP rulesets included; noisy until the rules are chosen | `pip install semgrep` |
| `gitleaks` | any repo | `gitleaks detect --source . -v` | secrets in the working tree and the whole history | GitHub release binary |
| `trivy` | containers, IaC, repos | `trivy fs .` / `trivy image <name>` | CVEs in OS packages and app deps, misconfigured Dockerfiles and IaC | GitHub release binary |

## Reading the output

- A dependency CVE is a finding when the vulnerable function is reachable
  from your code. `govulncheck` tells you; `npm audit` does not, so read
  where the package is used before rating it above Medium.
- A `gosec`/`semgrep` hit is the *where*. The *whether* is the trace: does
  input from outside reach it. Most hits do not.
- A `gitleaks` hit in history is a finding even after the line was removed:
  the secret is in every clone. The fix is to rotate the secret, then to
  rewrite history if the repository is public. Report the location and the
  kind, never the value.
- Zero hits from a scanner is a statement about the rules it has, not about
  the code. The report says which scanner ran and at which version.
