package connect

import (
	"context"
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/config"
)

func isolate(t *testing.T) {
	t.Helper()
	t.Setenv("AETOX_DATA_ROOT", t.TempDir())
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
}

// The catalog is the register. If an entry is malformed the settings page draws
// a row nobody can use, so the shape is checked rather than assumed.
func TestEveryCatalogEntryIsUsable(t *testing.T) {
	for _, p := range catalog {
		if p.ID == "" || p.Label == "" {
			t.Fatalf("entry %+v has no id or label", p)
		}
		if p.connect == nil || p.verify == nil || p.status == nil || p.disconnect == nil {
			t.Fatalf("%s does not implement all four verbs", p.ID)
		}
		if len(p.Tools) == 0 && !p.Channel {
			t.Fatalf("%s contributes no tools — nothing would change when it is switched off", p.ID)
		}
		if p.Channel && (len(p.Tools) != 0 || p.pairing == nil) {
			t.Fatalf("%s channel must have a pairing gate and no desk tools", p.ID)
		}
		// Asking for a pasted token without saying where to get one leaves the
		// user searching a website they may never have opened. A self-hosted
		// service satisfies this differently: the page is inside their own
		// instance, so the address is joined on once they say where that is
		// (statusOf) — which is why TokenPath counts, and why one of the two
		// must be set rather than neither.
		if p.Kind == KindToken && p.TokenURL == "" && p.TokenPath == "" {
			t.Fatalf("%s asks for a pasted token but says nowhere to mint one", p.ID)
		}
		if p.TokenPath != "" && !p.NeedsBaseURL {
			t.Fatalf("%s has a token path but no address to join it to", p.ID)
		}
	}
}

// A self-hosted service's mint page cannot be linked to until the user says
// where their server is — and must be linked to the moment they have.
func TestSelfHostedTokenLinkAppearsWithTheAddress(t *testing.T) {
	isolate(t)

	before, _ := StatusOf("n8n")
	if before.TokenURL != "" {
		t.Fatalf("TokenURL = %q before an address was given; it would open nowhere", before.TokenURL)
	}
	if err := config.SetConnectionBaseURL("n8n", "http://localhost:5678"); err != nil {
		t.Fatalf("SetConnectionBaseURL: %v", err)
	}
	after, _ := StatusOf("n8n")
	if after.TokenURL != "http://localhost:5678/settings/api" {
		t.Fatalf("TokenURL = %q; want the mint page on the user's own instance", after.TokenURL)
	}
	if after.BaseURL != "http://localhost:5678" {
		t.Fatalf("BaseURL = %q; want what the user typed back for the form", after.BaseURL)
	}
}

// The reason every tool name in this catalog carries its vendor: two engines
// doing the same job would both want `workflow_create`, and ProviderOfTool
// answers with the first match in catalog order — so the second vendor added
// would silently lose its tools to the first.
func TestNoTwoProvidersClaimTheSameTool(t *testing.T) {
	owner := map[string]string{}
	for _, p := range catalog {
		for _, tool := range p.Tools {
			if had, clash := owner[strings.ToLower(tool)]; clash {
				t.Fatalf("%q is claimed by both %s and %s — the gate would follow catalog order, not intent", tool, had, p.ID)
			}
			owner[strings.ToLower(tool)] = p.ID
		}
	}
}

func TestNormalizeBaseURLAcceptsWhatPeopleActuallyType(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"localhost:5678", "http://localhost:5678"},
		{"http://localhost:5678/", "http://localhost:5678"},
		{"  https://flows.example.com  ", "https://flows.example.com"},
		// A reverse-proxied instance genuinely lives under a prefix, so a path
		// is kept — but the tail of a page URL somebody copied is not.
		{"https://example.com/n8n/", "https://example.com/n8n"},
		{"http://localhost:5678/home/workflows?x=1", "http://localhost:5678/home/workflows"},
	} {
		got, err := NormalizeBaseURL(c.in)
		if err != nil {
			t.Errorf("NormalizeBaseURL(%q) errored: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("NormalizeBaseURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	// What must be refused is what would send the user's token somewhere it
	// was never meant to go.
	for _, bad := range []string{"", "   ", "file:///etc/passwd", "ftp://host", "http://"} {
		if got, err := NormalizeBaseURL(bad); err == nil {
			t.Errorf("NormalizeBaseURL(%q) = %q, want a refusal", bad, got)
		}
	}
}

// A tool that belongs to no connection is not this package's business. Gating
// it here would take away tools nobody attached to an account.
func TestToolsOutsideTheCatalogAreAlwaysAllowed(t *testing.T) {
	for _, tool := range []string{"read", "shell", "web_search", ""} {
		if !Allows(tool, nil) {
			t.Fatalf("%q was gated, but belongs to no connection", tool)
		}
	}
}

func TestConnectionToolsFollowTheDesksPlacement(t *testing.T) {
	if !Allows("github_search", []string{"github"}) {
		t.Fatal("a desk holding github cannot see github_search")
	}
	if Allows("github_search", []string{"gmail"}) {
		t.Fatal("a desk holding another connection can see github_search")
	}
	if Allows("github_search", nil) {
		t.Fatal("a desk holding nothing can see github_search")
	}
}

// The install tool reaches GitHub like the rest, so a desk with the connection
// off must not keep exactly one tool that still goes there.
func TestPluginInstallIsGatedWithTheRestOfGitHub(t *testing.T) {
	if Allows("plugin_install", nil) {
		t.Fatal("plugin_install survived a desk that does not hold github")
	}
	owner, ok := ProviderOfTool("plugin_install")
	if !ok || owner != "github" {
		t.Fatalf("ProviderOfTool(plugin_install) = %q, %v; want github", owner, ok)
	}
}

func TestProviderOfToolIsCaseInsensitive(t *testing.T) {
	if owner, ok := ProviderOfTool("  GitHub_Search "); !ok || owner != "github" {
		t.Fatalf("owner = %q, %v; want github", owner, ok)
	}
	if _, ok := ProviderOfTool("read"); ok {
		t.Fatal("read was claimed by a connection")
	}
}

// Absent placement means every desk — an install that never opened the page has
// not asked for its tools to go away.
func TestUnplacedConnectionReportsItselfAsUnconfigured(t *testing.T) {
	isolate(t)

	rows := List()
	if len(rows) != len(catalog) {
		t.Fatalf("List returned %d rows, want %d", len(rows), len(catalog))
	}
	for _, row := range rows {
		if row.Configured {
			t.Fatalf("%s reports a placement with an empty config", row.ID)
		}
		if row.For == nil {
			t.Fatalf("%s sent a nil list to the UI, which renders as absent rather than empty", row.ID)
		}
		if row.Connected {
			t.Fatalf("%s reports connected with an empty vault", row.ID)
		}
	}
}

func TestSystemConnectionHasNoPlacementGate(t *testing.T) {
	isolate(t)

	// A placement written by an older release is stale data now. It must not
	// reappear in the UI or narrow the built-in integration.
	if err := config.SetConnectionTargets("github", []string{"coding"}); err != nil {
		t.Fatalf("seed old placement: %v", err)
	}
	row, ok := StatusOf("github")
	if !ok {
		t.Fatal("github vanished from the catalog")
	}
	if !row.System || row.Configured || len(row.For) != 0 {
		t.Fatalf("row = %+v, want a system connection with no placement", row)
	}
	if err := SetTargets("github", []string{"assistant"}); err == nil {
		t.Fatal("SetTargets accepted a placement for a system connection")
	}
	if !Allows("github_search", ConnectionsForDesk("coding")) {
		t.Fatal("the built-in GitHub integration was narrowed by stale placement")
	}
	if !Allows("github_search", ConnectionsForAgent("github")) {
		t.Fatal("the system connection did not reach an agent whose profile permits its tools")
	}
}

// Reconnecting is about the token, not about where the account belongs. Making
// the user choose desks again would treat one decision as two.
func TestDisconnectKeepsThePlacement(t *testing.T) {
	isolate(t)

	if err := SetTargets("n8n", []string{"agent:automation"}); err != nil {
		t.Fatalf("SetTargets: %v", err)
	}
	if err := Disconnect("n8n"); err != nil {
		t.Fatalf("Disconnect: %v", err)
	}
	row, _ := StatusOf("n8n")
	if !row.Configured || len(row.For) != 1 {
		t.Fatalf("placement after disconnect = %+v, want it kept", row.For)
	}
}

func TestUnknownConnectionIsRefusedByEveryVerb(t *testing.T) {
	isolate(t)

	if _, err := Connect(context.Background(), "myspace", "token", "", nil); err == nil {
		t.Fatal("Connect accepted an id the catalog does not have")
	}
	if err := SetTargets("myspace", []string{"coding"}); err == nil {
		t.Fatal("SetTargets accepted an unknown id")
	}
	if err := Disconnect("myspace"); err == nil {
		t.Fatal("Disconnect accepted an unknown id")
	}
	if _, ok := StatusOf("myspace"); ok {
		t.Fatal("StatusOf answered for an unknown id")
	}
}

// A connect that fails must not leave the placement written for an account that
// was never attached — but the reverse order is worse, so the rule is that a
// failed connect stores nothing the user can mistake for success. The placement
// alone is inert: no token, so no reach.
func TestFailedConnectStoresNoAccount(t *testing.T) {
	isolate(t)

	// Points at a token the real GitHub will never accept; the call fails at
	// the network or at the 401, and either way nothing is attached.
	if _, err := Connect(context.Background(), "github", "definitely-not-a-token", "", []string{"coding"}); err == nil {
		t.Skip("network reachable and GitHub accepted a junk token — nothing to assert")
	}
	row, _ := StatusOf("github")
	if row.Connected {
		t.Fatal("a failed connect left an account attached")
	}
}

// A HomeAgent lock has exactly two legal placements: at its agent, or switched
// off. Every other value a caller sends — a desk, another agent, a mixed list —
// is a state 2026-08-10 proved nobody can read: the engines' one reader is the
// automation agent, and a placement anywhere else makes every screen say
// "connected" while that agent holds nothing.
func TestHomeAgentLocksPlacementToTheAgent(t *testing.T) {
	isolate(t)

	// Whatever the caller sends, only the home survives.
	if err := SetTargets("n8n", []string{"coding", "agent:doc", "agent:automation"}); err != nil {
		t.Fatalf("SetTargets: %v", err)
	}
	row, _ := StatusOf("n8n")
	if len(row.For) != 1 || row.For[0] != "agent:automation" {
		t.Fatalf("For = %v, want exactly [agent:automation]", row.For)
	}
	// The desk that was named must not have gained anything.
	if Allows("n8n_workflow_list", config.ConnectionsForDesk("coding", IDs())) {
		t.Fatal("a desk placement survived the lock")
	}

	// A list without the home means switched off — and it must survive as an
	// EMPTY placement, not collapse into the nil that means every desk.
	if err := SetTargets("n8n", []string{"coding"}); err != nil {
		t.Fatalf("SetTargets: %v", err)
	}
	row, _ = StatusOf("n8n")
	if !row.Configured || len(row.For) != 0 {
		t.Fatalf("For = %v (configured=%v), want a configured empty placement", row.For, row.Configured)
	}

	// The page can see the lock, so it can draw a sentence instead of a picker.
	if row.HomeAgent != "automation" {
		t.Fatalf("HomeAgent = %q, want automation", row.HomeAgent)
	}
	// A system connection is a different case: it has no placement at all,
	// rather than inheriting the home-agent lock.
	if gh, _ := StatusOf("github"); !gh.System || gh.Configured || len(gh.For) != 0 {
		t.Fatalf("github = %+v — system state was confused with a home lock", gh)
	}
}

// Placements written before the lock existed stay in the file forever, so the
// sweep at startup is what makes the rule true for a machine that used the old
// picker — not just for fresh installs.
func TestEnforceHomesRepairsPreLockPlacements(t *testing.T) {
	isolate(t)

	// A pre-lock file: n8n pointed at a desk and at its agent, windmill at a
	// desk alone. Written through config directly, the way the old build did.
	if err := config.SetConnectionTargets("n8n", []string{"coding", "agent:automation"}); err != nil {
		t.Fatalf("seed n8n: %v", err)
	}
	if err := config.SetConnectionTargets("windmill", []string{"assistant"}); err != nil {
		t.Fatalf("seed windmill: %v", err)
	}

	EnforceHomes()

	if row, _ := StatusOf("n8n"); len(row.For) != 1 || row.For[0] != "agent:automation" {
		t.Fatalf("n8n For = %v, want [agent:automation]", row.For)
	}
	// The desk-only row is repaired to "switched off", not guessed home: the
	// user pointed it somewhere on purpose, and off-at-its-desk is the state
	// the room's own picker can fix in one click.
	if row, _ := StatusOf("windmill"); !row.Configured || len(row.For) != 0 {
		t.Fatalf("windmill For = %v (configured=%v), want a configured empty placement", row.For, row.Configured)
	}
	// A connection the user never placed is not touched: nothing was written,
	// and writing would turn "never asked" into a choice they did not make.
	if _, configured := config.ConnectionTargets("github"); configured {
		t.Fatal("EnforceHomes wrote a placement for an untouched connection")
	}
}
