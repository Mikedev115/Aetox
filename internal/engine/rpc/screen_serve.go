package rpc

// The whole Screen, served: what the desktop's in-process engine.Screen
// answered by method call, answered over the wire by the same value.

import (
	"context"
	"encoding/json"

	"github.com/Mikedev115/Aetox/internal/engine"
)

// ServeScreen registers a Screen's every answer on the client — the desk
// questions, the provider signer, the window tools — so an engine across
// the socket sees exactly the window an in-process one did. Call before
// Connect; Hello announces the tools.
func ServeScreen(c *Client, s engine.Screen) {
	c.Handle(MethodAgentTab, func(context.Context, string, json.RawMessage) (any, error) {
		return s.AgentTab(), nil
	})
	c.Handle(MethodProviderEndpoint, func(_ context.Context, _ string, params json.RawMessage) (any, error) {
		var provider string
		if err := decodeParams(params, &provider); err != nil {
			return nil, err
		}
		return s.ProviderEndpoint(provider), nil
	})
	c.Handle(MethodDefaultModel, func(_ context.Context, _ string, params json.RawMessage) (any, error) {
		var provider, baseURL string
		if err := decodeParams(params, &provider, &baseURL); err != nil {
			return nil, err
		}
		return s.DefaultModel(provider, baseURL), nil
	})
	c.Handle(MethodProbe, func(_ context.Context, _ string, params json.RawMessage) (any, error) {
		var provider, modelName, baseURL, wireFormat string
		if err := decodeParams(params, &provider, &modelName, &baseURL, &wireFormat); err != nil {
			return nil, err
		}
		return s.Probe(provider, modelName, baseURL, wireFormat)
	})
	c.Handle(MethodModelResident, func(_ context.Context, _ string, params json.RawMessage) (any, error) {
		var provider, baseURL, modelName string
		if err := decodeParams(params, &provider, &baseURL, &modelName); err != nil {
			return nil, err
		}
		return s.ModelResident(provider, baseURL, modelName), nil
	})
	c.Handle(MethodRenderDeck, func(ctx context.Context, _ string, params json.RawMessage) (any, error) {
		var fileURL string
		var req engine.DeckRender
		if err := decodeParams(params, &fileURL, &req); err != nil {
			return nil, err
		}
		return s.RenderDeck(ctx, fileURL, req)
	})
	ServeProvider(c, s.ProviderTransport)
	ServeTools(c, s.WindowTools)
}
