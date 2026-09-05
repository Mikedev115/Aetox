package cognitive

import (
	"testing"

	"github.com/Mikedev115/Aetox/internal/memory"
	"github.com/Mikedev115/Aetox/internal/model"
)

// The catalog knows how much a model will write in one reply; the per-provider
// table only knows what each API was once seen to accept. When the catalog has
// a row, its figure is the ceiling — a landing page on glm-5.3-flash was cut
// at the table's 8,192 twice over, four minutes for no file, on a model whose
// row says 131,072.
func TestOutputCeilingComesFromTheCatalogWhenItHasARow(t *testing.T) {
	a := &Agent{
		provider:  fakeProv{name: "opencode-go"},
		model:     "glm-5.3-flash",
		context:   memory.NewContext("sys", 50, 4_000_000),
		lastUsage: model.Usage{PromptTokens: 10_000},
	}
	if got := a.providerOutputCeiling(); got != 131072 {
		t.Fatalf("ceiling = %d, want the catalog's 131072 over the table's 8192", got)
	}
	// The window still has the last word, and here it has room to spare.
	if got := a.toolLoopMaxTokens(); got != 131072 {
		t.Fatalf("max_tokens = %d, want the catalog's ceiling intact inside a 1M window", got)
	}
}

// A model the catalog has no row for keeps the floor the table always gave it:
// nothing changes for the providers this never knew about.
func TestOutputCeilingKeepsTheFloorWithoutACatalogRow(t *testing.T) {
	a := &Agent{
		provider: fakeProv{name: "opencode-go"},
		model:    "some-model-the-catalog-never-heard-of",
		context:  memory.NewContext("sys", 50, 100_000),
	}
	if got := a.providerOutputCeiling(); got != 8192 {
		t.Fatalf("ceiling = %d, want the unknown-provider floor 8192", got)
	}
}
