package cognitive

import (
	"strings"
	"testing"

	"github.com/Mikedev115/Aetox/internal/think"
	"github.com/Mikedev115/Aetox/internal/turn"
)

// One conversation keeps one cache key and two conversations do not share one.
// That pair is the whole property: the key only selects which machine is asked
// for a cached prefix (model.Request.CacheKey), so a key that drifted mid-chat
// would throw the cache away every turn, and a key two chats shared would send
// them to one shard holding neither's prefix.
func TestEachConversationKeepsItsOwnCacheKey(t *testing.T) {
	one := NewAgent(AgentConfig{Model: "m"})
	two := NewAgent(AgentConfig{Model: "m"})

	if one.cacheKey == "" {
		t.Fatal("an agent was built with no cache key")
	}
	if one.cacheKey == two.cacheKey {
		t.Fatalf("two conversations share the key %q", one.cacheKey)
	}
	before := one.cacheKey
	one.ReplaceModel(one.provider, "another-model")
	if one.cacheKey != before {
		t.Fatal("switching model inside one conversation moved its cache key")
	}

	// It reaches the request, or none of the above matters.
	req := one.buildRequest(nil, 0, 0, nil, "", turn.TurnOptions{ThinkLevel: think.LevelMedium})
	if req.CacheKey != one.cacheKey {
		t.Fatalf("the request carried %q, want the agent's %q", req.CacheKey, one.cacheKey)
	}

	// And it says nothing about the user — see newCacheKey's doc.
	if !strings.HasPrefix(one.cacheKey, "aetox-") {
		t.Fatalf("key %q is not the documented shape", one.cacheKey)
	}
}
