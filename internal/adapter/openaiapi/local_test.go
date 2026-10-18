package openaiapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The tests in this file are iss-2610161734428291's adapter half: the probe
// reads what a server on this machine says it serves, through each runtime's
// own endpoints, and never anything off this machine. Each fake is shaped like
// the runtime's real answer (Ollama 0.14, llama-server b9860, mlx_lm.server
// 0.31); a live probe against a real local server is owed.

const (
	ollamaModel  = "qwen3-coder:30b"
	ollamaDigest = "06c1097efce0431c2045fe7b2e5108366e43bee1b4603a7aded8f21689e90bca"
)

// ollamaFake is an Ollama server: the version, the model's details, the
// pulled models with their digests and the loaded ones with their context.
type ollamaFake struct {
	show, ps   string
	showStatus int
	asked      []string
}

func (o *ollamaFake) serve(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		o.asked = append(o.asked, r.Method+" "+r.URL.Path)
		switch r.Method + " " + r.URL.Path {
		case "GET /api/version":
			_, _ = io.WriteString(w, `{"version":"0.14.2"}`)
		case "POST /api/show":
			body, _ := io.ReadAll(r.Body)
			if !strings.Contains(string(body), `"model":"`+ollamaModel+`"`) || o.showStatus != 0 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = io.WriteString(w, `{"error":"model 'x' not found"}`)
				return
			}
			_, _ = io.WriteString(w, o.show)
		case "GET /api/tags":
			_, _ = io.WriteString(w, `{"models":[{"name":"llama3.2:latest","model":"llama3.2:latest","digest":"a80c4f17acd55265feec403c7aef86be0c25983ab279d83f3bcd3abbcb5b8b72"},`+
				`{"name":"`+ollamaModel+`","model":"`+ollamaModel+`","modified_at":"2026-10-16T09:00:00Z","size":18556688736,`+
				`"digest":"`+ollamaDigest+`","details":{"format":"gguf","family":"qwen3moe","quantization_level":"Q4_K_M"}}]}`)
		case "GET /api/ps":
			_, _ = io.WriteString(w, o.ps)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ollamaShow is /api/show's answer for a model whose Modelfile sets params.
func ollamaShow(params string, caps string) string {
	return `{"license":"Apache 2.0","modelfile":"FROM x","parameters":"` + params + `","template":"{{ .Prompt }}",` +
		`"details":{"parent_model":"","format":"gguf","family":"qwen3moe","families":["qwen3moe"],"parameter_size":"30.5B","quantization_level":"Q4_K_M"},` +
		`"model_info":{"general.architecture":"qwen3moe","qwen3moe.context_length":262144},` + caps + `"modified_at":"2026-10-16T09:00:00Z"}`
}

const ollamaLoaded = `{"models":[{"name":"` + ollamaModel + `","model":"` + ollamaModel + `","size":20000000000,"digest":"` + ollamaDigest +
	`","details":{"quantization_level":"Q4_K_M"},"expires_at":"2026-10-16T09:05:00Z","size_vram":20000000000,"context_length":32768}]}`

func probeClient(t *testing.T, base, key string) *Client {
	t.Helper()
	c, err := New(base, key)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// TestTheOllamaProbeReadsWhatItServes: the version, the model's digest, its
// quantisation, the context the loaded model is served at, and whether its
// template takes tools, each from Ollama's own endpoints.
func TestTheOllamaProbeReadsWhatItServes(t *testing.T) {
	o := &ollamaFake{show: ollamaShow(`num_ctx 8192`, `"capabilities":["completion","tools"],`), ps: ollamaLoaded}
	srv := o.serve(t)
	rep, err := probeClient(t, srv.URL+"/v1", "").ProbeLocal(context.Background(), RuntimeOllama, ollamaModel)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Version != "0.14.2" || rep.Digest != ollamaDigest || rep.Quantisation != "Q4_K_M" || rep.Model != ollamaModel {
		t.Fatalf("report = %+v", rep)
	}
	// The loaded model's context is the one served, over the Modelfile's.
	if rep.Context != 32768 {
		t.Fatalf("context = %d, want the loaded model's 32768", rep.Context)
	}
	if rep.ToolCalls == nil || !*rep.ToolCalls {
		t.Fatalf("tool calls = %v, want true", rep.ToolCalls)
	}
	if rep.KVCacheType != "" {
		t.Fatalf("kv cache type = %q: Ollama reports none", rep.KVCacheType)
	}
}

// TestAnOllamaModelNotLoadedReadsItsModelfileContext: a model not loaded
// has no served context in /api/ps, so the Modelfile's num_ctx is read; with
// neither the context is not reported, never the trained maximum, and a
// template without tools reads as no tool calls.
func TestAnOllamaModelNotLoadedReadsItsModelfileContext(t *testing.T) {
	o := &ollamaFake{show: ollamaShow(`stop \"<|im_end|>\"\nnum_ctx 16384`, `"capabilities":["completion"],`), ps: `{"models":[]}`}
	rep, err := probeClient(t, o.serve(t).URL+"/v1", "").ProbeLocal(context.Background(), RuntimeOllama, ollamaModel)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Context != 16384 {
		t.Fatalf("context = %d, want the Modelfile's 16384", rep.Context)
	}
	if rep.ToolCalls == nil || *rep.ToolCalls {
		t.Fatalf("tool calls = %v, want false", rep.ToolCalls)
	}
	o = &ollamaFake{show: ollamaShow(`stop \"<|im_end|>\"`, ``), ps: `{"models":[]}`}
	rep, err = probeClient(t, o.serve(t).URL+"/v1", "").ProbeLocal(context.Background(), RuntimeOllama, ollamaModel)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Context != 0 || rep.ToolCalls != nil {
		t.Fatalf("context %d, tool calls %v: want neither reported", rep.Context, rep.ToolCalls)
	}
}

// TestAnOllamaWithoutTheModelIsRefused: a model Ollama has not pulled, and a
// server that does not answer as Ollama, fail the probe, naming why.
func TestAnOllamaWithoutTheModelIsRefused(t *testing.T) {
	o := &ollamaFake{show: ollamaShow(``, ``), ps: `{"models":[]}`, showStatus: http.StatusNotFound}
	_, err := probeClient(t, o.serve(t).URL+"/v1", "").ProbeLocal(context.Background(), RuntimeOllama, ollamaModel)
	if err == nil || !strings.Contains(err.Error(), "ollama pull") {
		t.Fatalf("err = %v, want the pull named", err)
	}
	mlx := mlxFake(t)
	_, err = probeClient(t, mlx.URL+"/v1", "").ProbeLocal(context.Background(), RuntimeOllama, ollamaModel)
	if err == nil || !strings.Contains(err.Error(), "does not answer as Ollama") {
		t.Fatalf("err = %v, want the runtime named", err)
	}
}

const llamaProps = `{"default_generation_settings":{"id":0,"n_ctx":32768,"speculative":false,"is_processing":false,"params":{"n_predict":-1,"seed":4294967295,"temperature":0.8}},` +
	`"total_slots":1,"model_alias":"","model_path":"/Users/someone/models/Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf",` +
	`"modalities":{"vision":false,"audio":false},"chat_template":"{% for m in messages %}{% endfor %}",` +
	`"chat_template_caps":{"supports_parallel_tool_calls":true,"supports_preserve_reasoning":false,"supports_string_content":true,` +
	`"supports_system_role":true,"supports_tool_calls":true,"supports_tools":true,"supports_typed_content":false},` +
	`"bos_token":"","eos_token":"<|im_end|>","build_info":"b9860-1a2b3c4d","is_sleeping":false}`

// TestTheLlamaServerProbeReadsItsProps: the build, the context each slot
// serves, the model file's name and whether its chat template calls tools,
// from /props, with the block's key as the bearer token.
func TestTheLlamaServerProbeReadsItsProps(t *testing.T) {
	var auth atomic.Value
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/props" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		auth.Store(r.Header.Get("Authorization"))
		_, _ = io.WriteString(w, llamaProps)
	}))
	t.Cleanup(srv.Close)
	rep, err := probeClient(t, srv.URL+"/v1", "sk-local-1234567890").ProbeLocal(context.Background(), RuntimeLlamaCpp, "qwen3-coder")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Version != "b9860" || rep.Context != 32768 || rep.Model != "Qwen3-Coder-30B-A3B-Instruct-Q4_K_M.gguf" {
		t.Fatalf("report = %+v", rep)
	}
	if rep.ToolCalls == nil || !*rep.ToolCalls {
		t.Fatalf("tool calls = %v", rep.ToolCalls)
	}
	if rep.Digest != "" || rep.Quantisation != "" || rep.KVCacheType != "" {
		t.Fatalf("report = %+v: llama-server reports no digest, quantisation or cache type", rep)
	}
	if got, _ := auth.Load().(string); got != "Bearer sk-local-1234567890" {
		t.Fatalf("authorization = %q", got)
	}
}

func mlxFake(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, _ = io.WriteString(w, `{"object":"list","data":[{"id":"mlx-community/Qwen3-Coder-30B-A3B-Instruct-4bit","object":"model","created":1760000000}]}`)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// TestTheMLXProbeReadsItsModelList: mlx_lm.server names its models and
// nothing else, so a listed model is all it reports, and an unlisted one
// fails the probe.
func TestTheMLXProbeReadsItsModelList(t *testing.T) {
	srv := mlxFake(t)
	c := probeClient(t, srv.URL+"/v1", "")
	rep, err := c.ProbeLocal(context.Background(), RuntimeMLX, "mlx-community/Qwen3-Coder-30B-A3B-Instruct-4bit")
	if err != nil {
		t.Fatal(err)
	}
	if rep.Model != "mlx-community/Qwen3-Coder-30B-A3B-Instruct-4bit" || rep.Version != "" || rep.Context != 0 || rep.ToolCalls != nil {
		t.Fatalf("report = %+v", rep)
	}
	if _, err := c.ProbeLocal(context.Background(), RuntimeMLX, "other/model"); err == nil || !strings.Contains(err.Error(), "does not list") {
		t.Fatalf("err = %v", err)
	}
}

// TestTheProbeNeverLeavesThisMachine: a base URL off this machine is never
// probed, an unknown runtime is refused, and a redirect is never followed.
func TestTheProbeNeverLeavesThisMachine(t *testing.T) {
	c := probeClient(t, "https://api.example.com/v1", "")
	if _, err := c.ProbeLocal(context.Background(), RuntimeOllama, ollamaModel); err == nil || !strings.Contains(err.Error(), "this machine") {
		t.Fatalf("err = %v", err)
	}
	var hits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits.Add(1) }))
	t.Cleanup(other.Close)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/props", http.StatusFound)
	}))
	t.Cleanup(srv.Close)
	c = probeClient(t, srv.URL+"/v1", "")
	if _, err := c.ProbeLocal(context.Background(), RuntimeLlamaCpp, "m"); err == nil || !strings.Contains(err.Error(), "redirect") {
		t.Fatalf("err = %v", err)
	}
	if hits.Load() != 0 {
		t.Fatal("the redirect was followed")
	}
	if _, err := c.ProbeLocal(context.Background(), "vllm", "m"); err == nil {
		t.Fatal("an unknown runtime was probed")
	}
}

// TestTheProbeIsBounded: a server that never answers ends the probe at its
// bound.
func TestTheProbeIsBounded(t *testing.T) {
	stop := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-stop:
		case <-r.Context().Done():
		}
	}))
	t.Cleanup(func() { close(stop); srv.Close() })
	c := probeClient(t, srv.URL+"/v1", "")
	c.probeWait = 200 * time.Millisecond
	start := time.Now()
	_, err := c.ProbeLocal(context.Background(), RuntimeLlamaCpp, "m")
	if err == nil || !strings.Contains(err.Error(), "within") {
		t.Fatalf("err = %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatalf("the probe took %s", time.Since(start))
	}
}
