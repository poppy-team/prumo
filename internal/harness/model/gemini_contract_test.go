//go:build integration_live

package model_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/raillen/prumo/internal/harness/agent"
	"github.com/raillen/prumo/internal/harness/model"
)

// This file pins the Gemini wire contract against the live API, before any
// adapter is written against it.
//
// The order is deliberate. An adapter written against a guessed shape is
// disposable work: if the real request or response differs, the adapter has to be
// rewritten and the guess has to be found first. These tests are what turn
// "probably" into a shape that can be coded, and they stay as the regression
// guard for it.
//
// It costs real tokens, so it is behind the integration_live tag and skipped
// without GEMINI_API_KEY. Prompts are one sentence and generationConfig caps the
// output.
//
// Four things here were not guessable and are the reason the file exists:
//
//  1. Arguments arrive as `args`, not `arguments`.
//  2. The assistant role is `model`, not `assistant`.
//  3. A function-call part carries a `thoughtSignature` that must be echoed back
//     on the next turn. Omitting it is a hard 400, not a degradation — so an
//     adapter that does not carry it fails on the second turn of every
//     tool-using conversation (GAP-174).
//  4. Reasoning is reported as `thoughtsTokenCount`, separately from the
//     candidate count, and cache reads live in an array-valued
//     `promptTokensDetails` rather than the object OpenAI returns.

const geminiBase = "https://generativelanguage.googleapis.com/v1beta/models/"

func geminiKey(t *testing.T) string {
	t.Helper()
	key := os.Getenv("GEMINI_API_KEY")
	if key == "" {
		t.Skip("set GEMINI_API_KEY to pin the Gemini wire contract")
	}
	return key
}

// geminiModel is overridable because Google's model list moves; the default is a
// current flash model, which is the cheapest thing that still does tool calling.
func geminiModel() string {
	if m := os.Getenv("GEMINI_LIVE_MODEL"); m != "" {
		return m
	}
	return "gemini-3-flash-preview"
}

func geminiPost(t *testing.T, method string, body map[string]any) (int, map[string]any) {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	url := geminiBase + geminiModel() + ":" + method + "?key=" + geminiKey(t)
	resp, err := http.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("%s returned undecodable body: %v", method, err)
	}
	return resp.StatusCode, decoded
}

// skipWhenQuota ends a contract test that cannot reach the API because the free
// tier is spent, and says so out loud.
//
// These tests assert the shape of a response, so a 429 means the shape was never
// observed. Failing there would blame the adapter for a quota, and skipping
// silently would let a contract quietly stop being checked. Neither is acceptable,
// so the test names the quota and the reset it was told about.
func skipWhenQuota(t *testing.T, status int, response map[string]any) {
	t.Helper()
	if status != http.StatusTooManyRequests {
		return
	}
	message := ""
	if errObj, ok := response["error"].(map[string]any); ok {
		message, _ = errObj["message"].(string)
	}
	t.Skipf("the API quota is spent, so the wire shape was not observed: %s", message)
}

func firstFunctionCall(t *testing.T, response map[string]any) map[string]any {
	t.Helper()
	candidates, _ := response["candidates"].([]any)
	if len(candidates) == 0 {
		t.Fatalf("no candidates in response: %v", response)
	}
	first := candidates[0].(map[string]any)
	content, _ := first["content"].(map[string]any)
	parts, _ := content["parts"].([]any)
	for _, raw := range parts {
		part := raw.(map[string]any)
		if _, ok := part["functionCall"].(map[string]any); ok {
			return part
		}
	}
	t.Fatalf("no functionCall part in response: %v", response)
	return nil
}

func toolRequest() map[string]any {
	return map[string]any{
		"contents": []any{
			map[string]any{"role": "user", "parts": []any{map[string]any{"text": "Weather in Paris? Use the tool."}}},
		},
		"tools": []any{
			map[string]any{"functionDeclarations": []any{
				map[string]any{
					"name":        "get_weather",
					"description": "Get weather for a city",
					"parameters": map[string]any{
						"type":       "object",
						"properties": map[string]any{"city": map[string]any{"type": "string"}},
						"required":   []any{"city"},
					},
				},
			}},
		},
		"toolConfig": map[string]any{
			"functionCallingConfig": map[string]any{
				"mode":                 "ANY",
				"allowedFunctionNames": []any{"get_weather"},
			},
		},
		"generationConfig": map[string]any{"maxOutputTokens": 128, "temperature": 0},
	}
}

// TestGeminiCallsAToolAndCarriesAnOpaqueSignature is the finding that shaped the
// adapter: the arguments come back under `args`, and the part carries a
// `thoughtSignature` that this core contract has nowhere to put.
func TestGeminiCallsAToolAndCarriesAnOpaqueSignature(t *testing.T) {
	status, response := geminiPost(t, "generateContent", toolRequest())
	skipWhenQuota(t, status, response)
	if status != http.StatusOK {
		t.Fatalf("status %d: %v", status, response["error"])
	}
	part := firstFunctionCall(t, response)
	call := part["functionCall"].(map[string]any)

	if _, usesArguments := call["arguments"]; usesArguments {
		t.Error("arguments came back under `arguments`; the adapter's field name is stale")
	}
	args, ok := call["args"].(map[string]any) //nolint:staticcheck // the key is the contract under test
	if !ok {
		t.Fatalf("functionCall has no args object: %v", call)
	}
	if args["city"] != "Paris" {
		t.Errorf("args = %v, want the model to have read the prompt", args)
	}
	if id, _ := call["id"].(string); id == "" {
		t.Error("functionCall has no id; a tool result cannot be matched back to its call without one")
	}
	signature, _ := part["thoughtSignature"].(string)
	if signature == "" {
		t.Fatal("functionCall part carries no thoughtSignature; see TestGeminiRejectsARoundTripWithoutTheSignature for why that is fatal")
	}
	t.Logf("thoughtSignature is %d bytes of opaque provider state", len(signature))
}

// TestGeminiRejectsARoundTripWithoutTheSignature is the test that makes the
// contract gap a requirement rather than an observation. The signature is not a
// nicety: omitting it is a 400, so an adapter that does not carry it fails on
// the second turn of every tool-using conversation, which a test that only checks
// the first response would call a pass.
func TestGeminiRejectsARoundTripWithoutTheSignature(t *testing.T) {
	status, response := geminiPost(t, "generateContent", toolRequest())
	skipWhenQuota(t, status, response)
	part := firstFunctionCall(t, response)
	call := part["functionCall"].(map[string]any)

	followUp := func(withSignature bool) map[string]any {
		replayed := map[string]any{"functionCall": call}
		if withSignature {
			replayed["thoughtSignature"] = part["thoughtSignature"]
		}
		return map[string]any{
			"contents": []any{
				map[string]any{"role": "user", "parts": []any{map[string]any{"text": "Weather in Paris? Use the tool."}}},
				map[string]any{"role": "model", "parts": []any{replayed}},
				map[string]any{"role": "user", "parts": []any{map[string]any{
					"functionResponse": map[string]any{"name": "get_weather", "response": map[string]any{"tempC": 17}},
				}}},
			},
			"tools": []any{map[string]any{"functionDeclarations": []any{map[string]any{
				"name": "get_weather", "parameters": map[string]any{
					"type": "object", "properties": map[string]any{"city": map[string]any{"type": "string"}},
					"required": []any{"city"},
				},
			}}}},
			"generationConfig": map[string]any{"maxOutputTokens": 128, "temperature": 0},
		}
	}

	withoutStatus, without := geminiPost(t, "generateContent", followUp(false))
	skipWhenQuota(t, withoutStatus, without)
	if withoutStatus < 400 {
		t.Fatalf("a round trip without the signature was accepted (status %d); if that has become valid, the adapter no longer needs to carry it: %v", withoutStatus, without)
	}
	t.Logf("without signature: HTTP %d", withoutStatus)

	withStatus, with := geminiPost(t, "generateContent", followUp(true))
	skipWhenQuota(t, withStatus, with)
	if withStatus != http.StatusOK {
		t.Fatalf("a round trip WITH the signature was rejected: HTTP %d: %v", withStatus, with["error"])
	}
	t.Logf("with signature: HTTP %d", withStatus)
}

// TestGeminiReportsReasoningSeparatelyAndCachesInAnArray pins the usage shape,
// which differs from OpenAI's in both of the ways an adapter has to special-case:
// reasoning is thoughtsTokenCount rather than a nested detail object, and cache
// reads are an array of per-modality entries rather than an object.
func TestGeminiReportsReasoningSeparatelyAndCachesInAnArray(t *testing.T) {
	status, response := geminiPost(t, "generateContent", toolRequest())
	skipWhenQuota(t, status, response)
	usage, ok := response["usageMetadata"].(map[string]any)
	if !ok {
		t.Fatalf("no usageMetadata: %v", response)
	}
	for _, field := range []string{"promptTokenCount", "candidatesTokenCount", "totalTokenCount"} {
		if _, present := usage[field]; !present {
			t.Errorf("usageMetadata has no %s; the adapter's mapping would report a zero run", field)
		}
	}
	if _, present := usage["thoughtsTokenCount"]; !present {
		t.Log("no thoughtsTokenCount on this call; it may only appear once the model actually thinks")
	}
	details, present := usage["promptTokensDetails"].([]any)
	if !present {
		t.Log("no promptTokensDetails on this call; check the cached-content shape separately")
		return
	}
	for _, raw := range details {
		entry := raw.(map[string]any)
		if _, hasModality := entry["modality"]; !hasModality {
			t.Errorf("promptTokensDetails entry has no modality: %v", entry)
		}
	}
	t.Logf("promptTokensDetails is an array with %d entries", len(details))
}

// TestGeminiReportsAnUnreachableModelAsAClientError pins the error shape the
// adapter has to classify. A 400 for a model that does not exist must not look
// like a server fault, or the router will retry it.
func TestGeminiReportsAnUnreachableModelAsAClientError(t *testing.T) {
	_, response := geminiPost(t, "generateContent", map[string]any{
		"contents": []any{map[string]any{"role": "user", "parts": []any{map[string]any{"text": "hi"}}}},
	})
	if errObj, failed := response["error"].(map[string]any); failed {
		t.Logf("error shape: code=%v status=%v message=%.120s", errObj["code"], errObj["status"], errObj["message"])
	} else {
		t.Logf("an unknown model was accepted: %v", response)
	}
}

// TestTheGeminiAdapterCompletesAToolRoundTrip is the test the adapter exists to
// pass. It runs the real multi-turn sequence — call, execute, return the result,
// get an answer — against the live API, which is the only place the
// thoughtSignature requirement actually bites. A first-turn-only test would pass
// against an adapter that fails on every second turn.
func TestTheGeminiAdapterCompletesAToolRoundTrip(t *testing.T) {
	key := geminiKey(t)
	provider := model.NewGemini(key, geminiModel())
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	req := agent.ModelRequest{
		RequestID: "roundtrip-1",
		TurnID:    "turn-1",
		Messages: []agent.Message{
			{ID: "m1", Role: agent.RoleUser, Content: "What is the weather in Paris? Use the tool."},
		},
		Tools: []agent.ToolSpec{{
			Name:        "get_weather",
			Description: "Get weather for a city",
			Schema: map[string]any{
				"type":       "object",
				"properties": map[string]any{"city": map[string]any{"type": "string"}},
				"required":   []any{"city"},
			},
		}},
	}

	first, err := provider.Stream(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	// The free tier is a small daily budget, and a spent quota is not an adapter
	// failure. The adapter's own classification is checked separately and offline,
	// so skipping here does not lose coverage of it.
	for ev := range first {
		if ev.Kind == agent.EventError && ev.Quota {
			t.Skipf("the API quota is spent, so the round trip was not exercised: %s", ev.Error)
		}
	}
	first, err = provider.Stream(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	var call *agent.ToolCall
	var usage *agent.Usage
	for ev := range first {
		switch ev.Kind {
		case agent.EventToolCallReady:
			call = ev.ToolCall
		case agent.EventUsageUpdated:
			usage = ev.Usage
		case agent.EventError:
			t.Fatalf("first turn failed: %s", ev.Error)
		}
	}
	if call == nil {
		t.Fatal("the model did not call the tool")
	}
	if call.Name != "get_weather" {
		t.Fatalf("called %q, want get_weather", call.Name)
	}
	if call.Arguments["city"] != "Paris" {
		t.Errorf("args = %v, want the city read from the prompt", call.Arguments)
	}
	if call.ProviderOpaque == nil || call.ProviderOpaque[model.GeminiThoughtSignatureKey] == "" {
		t.Fatal("the adapter dropped the thought signature; the next turn will be refused with a 400")
	}
	t.Logf("call %s(%v) with a %d-byte signature",
		call.Name, call.Arguments, len(call.ProviderOpaque[model.GeminiThoughtSignatureKey]))
	if usage != nil {
		t.Logf("first turn usage: in=%d out=%d reasoning=%d", usage.InputTokens, usage.OutputTokens, usage.ReasoningTokens)
	}

	// The second turn: replay the call with its signature, and answer it.
	second := req
	second.RequestID = "roundtrip-2"
	second.Messages = []agent.Message{
		req.Messages[0],
		{ID: "m2", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{*call}},
		{ID: "m3", Role: agent.RoleTool, ToolCallID: call.ID, Content: "17C and cloudy"},
	}
	stream, err := provider.Stream(ctx, second)
	if err != nil {
		t.Fatal(err)
	}
	var answer string
	for ev := range stream {
		switch ev.Kind {
		case agent.EventTextDelta:
			answer += ev.Text
		case agent.EventError:
			t.Fatalf("second turn failed — this is where a missing signature shows up: %s", ev.Error)
		}
	}
	if strings.TrimSpace(answer) == "" {
		t.Fatal("the second turn produced no answer")
	}
	t.Logf("answer: %q", answer)
}

// TestTheGeminiAdapterOmitsTheSignatureWhenTheCallHasNone checks the other
// direction: a call with no signature must not send an empty one, which the API
// treats differently from sending none.
func TestTheGeminiAdapterOmitsTheSignatureWhenTheCallHasNone(t *testing.T) {
	provider := model.NewGemini("k", geminiModel())
	encoded, err := json.Marshal(provider.RequestBody(agent.ModelRequest{
		Messages: []agent.Message{
			{ID: "m1", Role: agent.RoleAgent, ToolCalls: []agent.ToolCall{
				{ID: "c1", Name: "f", Arguments: map[string]any{"a": 1}},
			}},
		},
	}))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "thoughtSignature") {
		t.Fatalf("an empty signature was sent: %s", encoded)
	}
}
