package model

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// "Is the model still loading?" is asked of the same two endpoints that answer
// "which model is loaded", because it is the same question read the other way
// round — no local runtime reports load progress, so residency is the whole of
// the signal the loading row is drawn from (desktop/model_load.go).
func TestLocalModelResident(t *testing.T) {
	lmstudio := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v0/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
			{"id": "google/gemma-3-4b", "type": "llm", "state": "not-loaded"},
			{"id": "text-embedding-nomic", "type": "embeddings", "state": "loaded"},
			{"id": "qwen/qwen3-8b", "type": "llm", "state": "loaded"},
		}})
	}))
	defer lmstudio.Close()

	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ps" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{
			{"name": "qwen3:8b", "model": "qwen3:8b"},
		}})
	}))
	defer ollama.Close()

	empty := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"models": []map[string]any{}})
	}))
	defer empty.Close()

	cases := []struct {
		name     string
		provider string
		base     string
		want     string
		resident bool
	}{
		{"lmstudio serves the model asked for", "lmstudio", lmstudio.URL + "/v1", "qwen/qwen3-8b", true},
		{"lmstudio publisher prefix is not part of the name", "lmstudio", lmstudio.URL + "/v1", "qwen3-8b", true},
		{"lmstudio has the other model on disk only", "lmstudio", lmstudio.URL + "/v1", "google/gemma-3-4b", false},
		{"an embedding model cannot answer a chat", "lmstudio", lmstudio.URL + "/v1", "text-embedding-nomic", false},
		{"ollama tag defaults to the same weights", "ollama", ollama.URL, "qwen3", true},
		{"ollama with nothing pinned asks only whether anything is up", "ollama", ollama.URL, "", true},
		{"ollama holding nothing is still loading", "ollama", empty.URL, "qwen3", false},
		{"a remote provider is never answered for", "deepseek", ollama.URL, "qwen3", false},
		{"no endpoint is not a claim either way", "ollama", "", "qwen3", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := LocalModelResident(tc.provider, tc.base, "", tc.want); got != tc.resident {
				t.Fatalf("LocalModelResident(%q, %q) = %v, want %v", tc.provider, tc.want, got, tc.resident)
			}
		})
	}
}
