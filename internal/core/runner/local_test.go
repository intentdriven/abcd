package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core/oracle"
)

// The tests in this file are the runner's half of iss-2610161734428291 and
// iss-2610171545099084: a runner route to a model on this machine lists the
// local fields, not configured until a local pin names them; a pinned route
// is probed before opencode is launched, refused when the server could not
// run a tool-using role, and pointed at the probed server through a sealed
// configuration. The server is an httptest fake shaped like Ollama's real
// endpoints; a live probe is owed.

const (
	localRunModel  = "qwen3-coder"
	localRunDigest = "06c1097efce0431c2045fe7b2e5108366e43bee1b4603a7aded8f21689e90bca"
)

// ollamaRunFake is Ollama serving localRunModel, with or without tools, its
// probes counted.
type ollamaRunFake struct {
	srv    *httptest.Server
	tools  bool
	probes atomic.Int32
}

func newOllamaRunFake(t *testing.T, tools bool) *ollamaRunFake {
	t.Helper()
	o := &ollamaRunFake{tools: tools}
	o.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/version":
			o.probes.Add(1)
			_, _ = io.WriteString(w, `{"version":"0.14.2"}`)
		case "POST /api/show":
			caps := `["completion"]`
			if o.tools {
				caps = `["completion","tools"]`
			}
			_, _ = io.WriteString(w, `{"parameters":"num_ctx 32768","details":{"quantization_level":"Q4_K_M"},"capabilities":`+caps+`}`)
		case "GET /api/tags":
			_, _ = io.WriteString(w, `{"models":[{"name":"`+localRunModel+`:latest","model":"`+localRunModel+`:latest","digest":"`+localRunDigest+`"}]}`)
		case "GET /api/ps":
			_, _ = io.WriteString(w, `{"models":[]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(o.srv.Close)
	return o
}

// localRunSeq names each test's provider apart: the probe's reports are
// the process's, by provider and base URL, and a fake's port can be one an
// earlier test's fake had.
var localRunSeq atomic.Int32

// localRunConfig is the machine's configuration: a block at base serving
// localRunModel, pinned when pinned, and opencode and claude routed to it,
// and the block's name.
func localRunConfig(base string, pinned bool) (string, string) {
	name := fmt.Sprintf("local%d", localRunSeq.Add(1))
	pins := ""
	if pinned {
		pins = `,"pins":{"` + localRunModel + `":{"runtime":"ollama","runtime_version":"0.14.2","weights_sha256":"` + localRunDigest +
			`","quantisation":"Q4_K_M","context_length":32768,"kv_cache_type":"f16"}}`
	}
	return `{"oracle":{"api":{"` + name + `":{"base_url":"` + base + `","models":["` + localRunModel + `"]` + pins + `}}},` +
		`"runner":{"opencode":{"model":"` + name + `/` + localRunModel + `"},"claude":{"model":"` + name + `/` + localRunModel + `"}}}`, name
}

// TestARunnerRouteToALocalModelListsTheLocalFields is iss-2610171545099084:
// opencode, and claude, routed to a model whose provider is on this machine
// list the weights' fingerprint, quantisation and runtime version as not
// configured, as the adapter's route does, and the served local fields as
// not reported; nothing is probed, since nothing is pinned.
func TestARunnerRouteToALocalModelListsTheLocalFields(t *testing.T) {
	o := newOllamaRunFake(t, true)
	cfg, _ := localRunConfig(o.srv.URL+"/v1", false)
	c := mustLoad(t, cfg, "")
	for _, name := range []string{OpenCode, Claude} {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, "ok", name)
			ans, _, err := c.adapter(c.runners[name]).Run(context.Background(), f.request("ruthless-reviewer"))
			if err != nil {
				t.Fatalf("run: %v", err)
			}
			got := reasons(ans.Unpinned)
			for _, fl := range append(append([]oracle.Field{}, oracle.LocalFields...), oracle.AskedContext) {
				if got[fl] != oracle.NotConfigured {
					t.Errorf("%s = %q, want not_configured", fl, got[fl])
				}
			}
			for _, fl := range oracle.LocalServedFields {
				if got[fl] != oracle.NotReported {
					t.Errorf("%s = %q, want not_reported", fl, got[fl])
				}
			}
			if ans.LocalPin != nil || ans.LocalServed != nil {
				t.Fatalf("an unpinned route carries a local pin: %+v %+v", ans.LocalPin, ans.LocalServed)
			}
			assertFixedLists(t, ans.Unpinned)
		})
	}
	if o.probes.Load() != 0 {
		t.Fatal("an unpinned route was probed")
	}
	// A route off this machine lists no local field.
	c = mustLoad(t, `{"oracle":{"api":{"remote":{"base_url":"https://api.example.com/v1","models":["`+localRunModel+`"]}}},`+
		`"runner":{"opencode":{"model":"remote/`+localRunModel+`"}}}`, "")
	f := newFake(t, "ok", OpenCode)
	ans, _, err := c.adapter(c.runners[OpenCode]).Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatal(err)
	}
	if _, listed := reasons(ans.Unpinned)[oracle.AskedWeightsSHA256]; listed {
		t.Fatalf("a remote route lists the local fields: %v", ans.Unpinned)
	}
}

// wantLocalConfig is the sealed configuration of a pinned local run: the
// provider's base URL, the server the probe read, and nothing else.
func wantLocalConfig(name, base string) string {
	return `{"$schema":"https://opencode.ai/config.json","provider":{"` + name + `":{"options":{"baseURL":"` + base +
		`"},"models":{"` + localRunModel + `":{}}}}}` + "\n"
}

// TestAPinnedLocalRunIsProbedAndSealed: opencode routed to a pinned local
// model is launched only after the server's probe holds, against a sealed
// configuration naming the probed server, and the record carries the pin and
// what the server reported.
func TestAPinnedLocalRunIsProbedAndSealed(t *testing.T) {
	own, ownBody := personsOpenCode(t)
	o := newOllamaRunFake(t, true)
	base := o.srv.URL + "/v1"
	cfg, name := localRunConfig(base, true)
	c := mustLoad(t, cfg, "")
	f := newFake(t, "ok", OpenCode)
	ans, _, err := c.adapter(c.runners[OpenCode]).Run(context.Background(), f.request("ruthless-reviewer"))
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if o.probes.Load() != 1 {
		t.Fatalf("probes = %d, want one", o.probes.Load())
	}
	raw, err := os.ReadFile(filepath.Join(f.log, "opencode.config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != wantLocalConfig(name, base) {
		t.Fatalf("sealed configuration =\n%s\nwant\n%s", raw, wantLocalConfig(name, base))
	}
	sum := sha256.Sum256(raw)
	if ans.PinConfigSHA256 != hex.EncodeToString(sum[:]) || ans.HarnessVersion != "1.18.31" {
		t.Fatalf("config sha256 %q, version %q", ans.PinConfigSHA256, ans.HarnessVersion)
	}
	if got, _ := os.ReadFile(own); string(got) != string(ownBody) {
		t.Fatalf("the person's own opencode configuration changed: %s", got)
	}
	if ans.LocalPin == nil || ans.LocalPin.WeightsSHA256 != localRunDigest || ans.ProviderPin != nil {
		t.Fatalf("local pin %+v, provider pin %+v", ans.LocalPin, ans.ProviderPin)
	}
	if s := ans.LocalServed; s == nil || s.WeightsSHA256 != localRunDigest || s.ContextLength != 32768 || s.ToolCalls == nil || !*s.ToolCalls {
		t.Fatalf("local served = %+v", ans.LocalServed)
	}
	// The probe confirmed the pin, but opencode merges the person's
	// ~/.opencode/opencode.json after the sealed file, and an entry there can
	// point the provider elsewhere: the asked local fields stay not verified,
	// as a provider pin on this path does.
	got := reasons(ans.Unpinned)
	for _, fl := range append(append([]oracle.Field{}, oracle.LocalFields...), oracle.AskedContext) {
		if got[fl] != oracle.NotVerified {
			t.Errorf("%s = %q, want not_verified on the runner path", fl, got[fl])
		}
	}
	// No runtime reports its KV cache's type, so the pin's is never confirmed.
	if got[oracle.ServedKVCacheType] != oracle.NotVerified || got[oracle.ServedWeightsSHA256] != oracle.SelfReported {
		t.Errorf("served local fields: %v", ans.Unpinned)
	}
	if got[oracle.AskedProviderPin] != oracle.NotConfigured {
		t.Errorf("provider pin = %q", got[oracle.AskedProviderPin])
	}
	assertFixedLists(t, ans.Unpinned)
}

// TestClaudeRefusesAPinnedRoute: claude can hold neither a local pin nor a
// provider pin, so a route the machine pins is refused before anything is
// launched or probed, naming opencode as the runner that holds it.
func TestClaudeRefusesAPinnedRoute(t *testing.T) {
	o := newOllamaRunFake(t, true)
	cfg, _ := localRunConfig(o.srv.URL+"/v1", true)
	c := mustLoad(t, cfg, "")
	f := newFake(t, "ok", Claude)
	_, _, err := c.adapter(c.runners[Claude]).Run(context.Background(), f.request("ruthless-reviewer"))
	if err == nil || !strings.Contains(err.Error(), "runner.opencode.model") || !strings.Contains(err.Error(), "pins") {
		t.Fatalf("a pinned claude route: err = %v, want a refusal naming opencode", err)
	}
	if f.launched(Claude) {
		t.Fatal("claude was launched on a pinned route")
	}
	if o.probes.Load() != 0 {
		t.Fatal("a pinned claude route was probed")
	}
}

// TestEveryPinnedLocalRunProbesItsServer: each run probes the server
// afresh, so a server changed between two runs is never trusted on the
// strength of the first run's report.
func TestEveryPinnedLocalRunProbesItsServer(t *testing.T) {
	o := newOllamaRunFake(t, true)
	cfg, _ := localRunConfig(o.srv.URL+"/v1", true)
	c := mustLoad(t, cfg, "")
	for range 2 {
		f := newFake(t, "ok", OpenCode)
		if _, _, err := c.adapter(c.runners[OpenCode]).Run(context.Background(), f.request("ruthless-reviewer")); err != nil {
			t.Fatalf("run: %v", err)
		}
	}
	if n := o.probes.Load(); n != 2 {
		t.Fatalf("probes = %d, want one per run", n)
	}
}

// TestAPinnedLocalRunTheServerCannotHoldIsNeverLaunched: a server whose
// model takes no tool calls fails a tool-using role's probe, so opencode is
// never launched and nothing is written; the refusal names the runtime and
// the fix.
func TestAPinnedLocalRunTheServerCannotHoldIsNeverLaunched(t *testing.T) {
	o := newOllamaRunFake(t, false)
	cfg, _ := localRunConfig(o.srv.URL+"/v1", true)
	c := mustLoad(t, cfg, "")
	f := newFake(t, "ok", OpenCode)
	_, _, err := c.adapter(c.runners[OpenCode]).Run(context.Background(), f.request("ruthless-reviewer"))
	if err == nil || !strings.Contains(err.Error(), "Ollama") || !strings.Contains(err.Error(), "tool") {
		t.Fatalf("err = %v", err)
	}
	if f.launched(OpenCode) {
		t.Fatal("opencode was launched against a server the probe refused")
	}
}

// TestAPinnedLocalRunRefusesThePersonsOwnOpenCodeConfiguration: the sealed
// file would set aside, or be overridden by, a configuration the person
// names for opencode, so the run is refused before the probe or a launch.
func TestAPinnedLocalRunRefusesThePersonsOwnOpenCodeConfiguration(t *testing.T) {
	o := newOllamaRunFake(t, true)
	cfg, _ := localRunConfig(o.srv.URL+"/v1", true)
	c := mustLoad(t, cfg, "")
	f := newFake(t, "ok", OpenCode)
	t.Setenv("OPENCODE_CONFIG_DIR", t.TempDir())
	_, _, err := c.adapter(c.runners[OpenCode]).Run(context.Background(), f.request("ruthless-reviewer"))
	if err == nil || !strings.Contains(err.Error(), "OPENCODE_CONFIG_DIR") {
		t.Fatalf("err = %v", err)
	}
	if f.launched(OpenCode) || o.probes.Load() != 0 {
		t.Fatal("the run went ahead")
	}
}
