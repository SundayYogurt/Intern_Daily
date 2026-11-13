package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAIChatOpenAICompatible(t *testing.T) {
	var gotPath, gotAuth, gotModel string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		gotModel, _ = body["model"].(string)
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"<think>x</think>สวัสดี"}}]}`))
	}))
	defer srv.Close()
	t.Setenv("AI_PROVIDER", "typhoon")
	t.Setenv("AI_API_KEY", "ty-key")
	t.Setenv("AI_BASE_URL", srv.URL+"/v1")
	t.Setenv("OLLAMA_API_KEY", "")

	text, err := aiChat(context.Background(), "hi")
	if err != nil || text != "สวัสดี" {
		t.Fatalf("got %q, %v", text, err)
	}
	if gotPath != "/v1/chat/completions" || gotAuth != "Bearer ty-key" || gotModel != "typhoon-v2.5-30b-a3b-instruct" {
		t.Fatalf("path=%s auth=%s model=%s", gotPath, gotAuth, gotModel)
	}
}

func TestAIChatOllama(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" || r.Header.Get("Authorization") != "Bearer ol-key" {
			w.WriteHeader(401)
			return
		}
		w.Write([]byte(`{"message":{"role":"assistant","content":"ok"}}`))
	}))
	defer srv.Close()
	t.Setenv("AI_PROVIDER", "")
	t.Setenv("OLLAMA_API_KEY", "ol-key")
	t.Setenv("OLLAMA_HOST", srv.URL)

	if text, err := aiChat(context.Background(), "hi"); err != nil || text != "ok" {
		t.Fatalf("got %q, %v", text, err)
	}
}

func TestAIChatErrorMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
		w.Write([]byte(`{"error":{"message":"rate limit exceeded"}}`))
	}))
	defer srv.Close()
	t.Setenv("AI_PROVIDER", "openai")
	t.Setenv("AI_API_KEY", "k")
	t.Setenv("AI_BASE_URL", srv.URL)

	_, err := aiChat(context.Background(), "hi")
	if err == nil || !strings.Contains(err.Error(), "429") || !strings.Contains(err.Error(), "rate limit exceeded") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAIEnabledNeedsProviderKey(t *testing.T) {
	t.Setenv("AI_PROVIDER", "typhoon")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("OLLAMA_API_KEY", "ol-key")
	if aiEnabled() {
		t.Fatal("typhoon provider must not fall back to the Ollama key")
	}
}

func TestReportGuideCoversEveryReportSection(t *testing.T) {
	general, _ := reportGuide("background")
	if !strings.Contains(general, "บุคคลที่สาม") {
		t.Fatalf("general guide missing: %q", general)
	}
	for _, ch := range reportChapters {
		for _, f := range ch.Fields {
			if _, section := reportGuide(f.Key); section == "" {
				t.Errorf("no writing guide for report key %q (%s)", f.Key, f.Label)
			}
		}
	}
	if _, section := reportGuide("nope"); section != "" {
		t.Fatal("unknown key should have no guide")
	}
}
