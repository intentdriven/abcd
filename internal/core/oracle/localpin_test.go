package oracle

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/openaiapi"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// The tests in this file are iss-2610161734428291's oracle half: a block on
// this machine can pin a listed model down to its weights, the server is
// probed before the route is used, a field it reports otherwise
// refuses the route, and the record carries the pin on the asked side and the
// server's report on the served side. Each server is an httptest fake shaped
// like the runtime's real endpoints; a live probe is owed.

const (
	localModel  = "qwen3-coder:30b"
	localDigest = "06c1097efce0431c2045fe7b2e5108366e43bee1b4603a7aded8f21689e90bca"
	// llamaFile is the GGUF file a llama.cpp pin names, and llamaPath where
	// the llama-server fakes report it loaded.
	llamaFile = "Qwen3-Coder-Q4_K_M.gguf"
	llamaPath = "/m/" + llamaFile
)

// pinnedLocalBlock is a keyless block at base serving localModel, pinned by pins.
func pinnedLocalBlock(base, pins string) string {
	return `{"oracle":{"api":{"local":{"base_url":"` + base + `","models":["` + localModel + `","other"],"pins":` + pins + `}},` +
		`"roles":{"scribe":"local/` + localModel + `"}}}`
}

// localPinText is a pin of localModel on runtime with the context ctx and
// cache type kv; a llama.cpp pin names llamaFile as its weights file.
func localPinText(runtime, version, ctx, kv string) string {
	file := ""
	if runtime == "llama.cpp" {
		file = `"weights_file":"` + llamaFile + `",`
	}
	return `{"` + localModel + `":{"runtime":"` + runtime + `","runtime_version":"` + version + `","weights_sha256":"sha256:` +
		strings.ToUpper(localDigest) + `",` + file + `"quantisation":"Q4_K_M","context_length":` + ctx + `,"kv_cache_type":"` + kv + `"}}`
}

// llamaPinFile is a llama.cpp pin of localModel naming file as its weights
// file, the key left out when file is empty.
func llamaPinFile(file string) string {
	return strings.Replace(localPinText("llama.cpp", "b9860", "32768", "f16"), `"weights_file":"`+llamaFile+`",`, file, 1)
}

// llamaPropsAt is llama-server's /props loading the model file at path,
// serving each of slots slots (none named when 0) a context of nctx.
func llamaPropsAt(path string, nctx, slots int) string {
	extra := ""
	if path != "" {
		extra += `,"model_path":"` + path + `"`
	}
	if slots != 0 {
		extra += `,"total_slots":` + fmt.Sprint(slots)
	}
	return `{"build_info":"b9860-1a2b3c4d","default_generation_settings":{"n_ctx":` + fmt.Sprint(nctx) + `},` +
		`"chat_template_caps":{"supports_tool_calls":true}` + extra + `}`
}

var ollamaPin = localPinText("ollama", "0.14.2", "32768", "f16")

// localFake is a server on this machine: Ollama's endpoints answering as
// the fields say, llama-server's /props when props is set, and the chat
// endpoint, each request counted by path.
type localFake struct {
	srv                 *httptest.Server
	digest, caps, ps    string
	params, props       string
	chats, versionAsked atomic.Int32
}

// forgetProbes clears this process's probe reports for the length of the
// test: a fake's port can be one an earlier test's fake had.
func forgetProbes(t *testing.T) {
	t.Helper()
	clear := func() {
		probes.Lock()
		probes.m = map[string]probeHeld{}
		probes.Unlock()
	}
	clear()
	t.Cleanup(clear)
}

func newLocalFake(t *testing.T) *localFake {
	t.Helper()
	forgetProbes(t)
	l := &localFake{digest: localDigest, caps: `"capabilities":["completion","tools"],`, params: "",
		ps: `{"models":[{"name":"` + localModel + `","model":"` + localModel + `","digest":"` + localDigest + `","context_length":32768}]}`}
	l.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method + " " + r.URL.Path {
		case "GET /api/version":
			l.versionAsked.Add(1)
			_, _ = io.WriteString(w, `{"version":"0.14.2"}`)
		case "POST /api/show":
			_, _ = io.WriteString(w, `{"parameters":"`+l.params+`","details":{"format":"gguf","quantization_level":"Q4_K_M"},`+l.caps+`"model_info":{}}`)
		case "GET /api/tags":
			_, _ = io.WriteString(w, `{"models":[{"name":"`+localModel+`","model":"`+localModel+`","digest":"`+l.digest+`"}]}`)
		case "GET /api/ps":
			_, _ = io.WriteString(w, l.ps)
		case "GET /props":
			if l.props == "" {
				http.NotFound(w, r)
				return
			}
			_, _ = io.WriteString(w, l.props)
		case "POST /v1/chat/completions":
			l.chats.Add(1)
			_, _ = io.WriteString(w, chat(localModel, `{"verdict":"keep"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(l.srv.Close)
	return l
}

// localCall configures l with pins and makes one call to localModel.
func localCall(t *testing.T, l *localFake, pins string) (CallRecord, error) {
	t.Helper()
	f := newFx(t)
	f.machineConfig(pinnedLocalBlock(l.srv.URL+"/v1", pins))
	c := f.loadAPI()
	_, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "local", Model: localModel}, Brief: sidesBrief})
	return rec, err
}

// TestALocalPinIsReadWithEveryField: a block on this machine pins a listed
// model with its runtime and version, its weights' digest, quantisation,
// context and cache type; the digest is kept in lower-case hex.
func TestALocalPinIsReadWithEveryField(t *testing.T) {
	f := newFx(t)
	f.machineConfig(pinnedLocalBlock("http://127.0.0.1:11434/v1", ollamaPin))
	c := f.loadAPI()
	pin, ok := c.LocalPinFor("local", localModel)
	want := LocalPin{Runtime: "ollama", RuntimeVersion: "0.14.2", WeightsSHA256: localDigest, Quantisation: "Q4_K_M",
		ContextLength: 32768, KVCacheType: "f16"}
	if !ok || pin != want {
		t.Fatalf("LocalPinFor = %+v, %v; want %+v", pin, ok, want)
	}
	if _, ok := c.LocalPinFor("local", "other"); ok {
		t.Fatal("an unpinned model has a local pin")
	}
	if _, ok := c.PinFor("local", localModel); ok {
		t.Fatal("a local pin reads as an OpenRouter one")
	}
	if !c.IsLocal("local") || c.IsLocal("absent") {
		t.Fatal("IsLocal does not name the block on this machine")
	}
}

// TestALocalPinThatNamesTooLittleIsRefused: a pin that leaves a field out,
// names a runtime abcd does not probe, or names an unlisted model is refused
// when the configuration is read; so is OpenRouter's pin on a local block,
// and a local pin on OpenRouter's.
func TestALocalPinThatNamesTooLittleIsRefused(t *testing.T) {
	full := `"runtime":"ollama","runtime_version":"0.14.2","weights_sha256":"` + localDigest + `","quantisation":"Q4_K_M","context_length":32768`
	for _, tc := range []struct{ name, base, pins, want string }{
		{"no cache type", "http://127.0.0.1:11434/v1", `{"` + localModel + `":{` + full + `}}`, "kv_cache_type"},
		{"no digest", "http://localhost:11434/v1", `{"` + localModel + `":{"runtime":"ollama","runtime_version":"0.14.2","quantisation":"Q4_K_M","context_length":32768,"kv_cache_type":"f16"}}`, "weights_sha256"},
		{"short digest", "http://127.0.0.1:11434/v1", `{"` + localModel + `":{"runtime":"ollama","runtime_version":"0.14.2","weights_sha256":"06c1097efce0","quantisation":"Q4_K_M","context_length":32768,"kv_cache_type":"f16"}}`, "weights_sha256"},
		{"unknown runtime", "http://127.0.0.1:8000/v1", localPinText("vllm", "0.28.0", "32768", "f16"), "vllm"},
		{"no context", "http://127.0.0.1:11434/v1", localPinText("ollama", "0.14.2", "0", "f16"), "context_length"},
		{"brace version", "http://127.0.0.1:11434/v1", localPinText("ollama", "{env:HOME}", "32768", "f16"), "runtime_version"},
		{"unlisted model", "http://127.0.0.1:11434/v1", strings.Replace(ollamaPin, localModel, "absent", 1), "not on its models"},
		{"OpenRouter's pin here", "http://127.0.0.1:11434/v1", `{"` + localModel + `":{"order":["a"],"quantizations":["fp8"],"canonical_slug":"a/b"}}`, "local pin"},
		{"local pin on OpenRouter", "https://openrouter.ai/api/v1", ollamaPin, "OpenRouter pin"},
		{"llama.cpp without its weights file", "http://127.0.0.1:8080/v1", llamaPinFile(""), "weights_file"},
		{"weights file as a path", "http://127.0.0.1:8080/v1", llamaPinFile(`"weights_file":"/m/` + llamaFile + `",`), "weights_file"},
		{"weights file not GGUF", "http://127.0.0.1:8080/v1", llamaPinFile(`"weights_file":"qwen3-coder",`), "weights_file"},
		// The probe keeps at most 200 bytes of the served name, so a longer
		// pin could never match; a bidi override or a C1 control would reach
		// a refusal and the record as written.
		{"weights file past the served name's length", "http://127.0.0.1:8080/v1",
			llamaPinFile(`"weights_file":"` + strings.Repeat("q", 196) + `.gguf",`), "weights_file"},
		{"weights file with a bidi override", "http://127.0.0.1:8080/v1",
			llamaPinFile(`"weights_file":"qwen\u202e3.gguf",`), "weights_file"},
		{"weights file with a C1 control", "http://127.0.0.1:8080/v1",
			llamaPinFile(`"weights_file":"qwen\u00853.gguf",`), "weights_file"},
		{"weights file off llama.cpp", "http://127.0.0.1:11434/v1",
			strings.Replace(ollamaPin, `"quantisation"`, `"weights_file":"`+llamaFile+`","quantisation"`, 1), "weights_file"},
		{"neither", "https://api.example.com/v1", ollamaPin, "this machine"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFx(t)
			f.machineConfig(pinnedLocalBlock(tc.base, tc.pins))
			if err := f.loadAPIErr(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to name %q", err, tc.want)
			}
		})
	}
}

// TestALocalPinIsTheMachinesAlone: a repository cannot declare a block, so it
// cannot declare a local pin either.
func TestALocalPinIsTheMachinesAlone(t *testing.T) {
	f := newFx(t)
	f.repoConfig(pinnedLocalBlock("http://127.0.0.1:11434/v1", ollamaPin))
	if err := f.loadAPIErr(); err == nil || !strings.Contains(err.Error(), "repository") {
		t.Fatalf("err = %v", err)
	}
}

// TestAPinnedLocalCallIsProbedAndRecorded: the server is probed before the
// call, and the record carries the pin on the asked side and what the server
// reported on the served side; each pinned field it confirmed is pinned, and
// each served value is its own word.
func TestAPinnedLocalCallIsProbedAndRecorded(t *testing.T) {
	l := newLocalFake(t)
	rec, err := localCall(t, l, ollamaPin)
	if err != nil {
		t.Fatal(err)
	}
	if rec.LocalPin == nil || rec.LocalPin.WeightsSHA256 != localDigest || rec.LocalPin.Runtime != "ollama" {
		t.Fatalf("local pin = %+v", rec.LocalPin)
	}
	s := rec.LocalServed
	if s == nil || s.RuntimeVersion != "0.14.2" || s.WeightsSHA256 != localDigest || s.ContextLength != 32768 ||
		s.Quantisation != "Q4_K_M" || s.ToolCalls == nil || !*s.ToolCalls {
		t.Fatalf("local served = %+v", s)
	}
	for _, f := range append(append([]Field{}, LocalFields...), AskedContext) {
		if r, ok := listed(rec, f); ok {
			t.Errorf("%s is listed %q, but the server confirmed the pin", f, r)
		}
	}
	for f, want := range map[Field]UnpinnedReason{
		ServedWeightsSHA256: SelfReported, ServedQuantisation: SelfReported, ServedRuntimeVersion: SelfReported,
		ServedContextLength: SelfReported, ServedToolCalls: SelfReported, ServedKVCacheType: NotVerified,
		AskedProviderPin: NotConfigured,
	} {
		if r, _ := listed(rec, f); r != want {
			t.Errorf("%s = %q, want %q", f, r, want)
		}
	}
	if l.chats.Load() != 1 {
		t.Fatalf("chats = %d", l.chats.Load())
	}
}

// TestWhatTheServerDoesNotReportIsNotVerified: llama-server reports no
// digest or quantisation, so the pin's are carried and listed as not
// verified, and the served fields as not reported.
func TestWhatTheServerDoesNotReportIsNotVerified(t *testing.T) {
	l := newLocalFake(t)
	l.props = llamaPropsAt(llamaPath, 32768, 0)
	rec, err := localCall(t, l, localPinText("llama.cpp", "b9860", "32768", "q8_0"))
	if err != nil {
		t.Fatal(err)
	}
	for f, want := range map[Field]UnpinnedReason{
		AskedWeightsSHA256: NotVerified, AskedQuantisation: NotVerified,
		ServedWeightsSHA256: NotReported, ServedQuantisation: NotReported, ServedRuntimeVersion: SelfReported,
		ServedContextLength: SelfReported, ServedKVCacheType: NotVerified,
	} {
		if r, _ := listed(rec, f); r != want {
			t.Errorf("%s = %q, want %q", f, r, want)
		}
	}
	for _, f := range []Field{AskedRuntimeVersion, AskedContext} {
		if r, ok := listed(rec, f); ok {
			t.Errorf("%s is listed %q, but llama-server confirmed it", f, r)
		}
	}
	if rec.LocalServed == nil || rec.LocalServed.Model != llamaFile {
		t.Fatalf("local served = %+v", rec.LocalServed)
	}
}

// TestAServerThatDiffersFromThePinIsRefusedBeforeAnyCall: each field the
// server reports otherwise than the pin refuses the route, naming the field,
// and the brief is never sent.
func TestAServerThatDiffersFromThePinIsRefusedBeforeAnyCall(t *testing.T) {
	for _, tc := range []struct {
		name, pins, want string
		set              func(*localFake)
	}{
		{"digest", ollamaPin, "weights_sha256", func(l *localFake) { l.digest = strings.Repeat("a", 64) }},
		{"version", localPinText("ollama", "0.13.0", "32768", "f16"), "runtime_version", nil},
		{"context", localPinText("ollama", "0.14.2", "65536", "f16"), "OLLAMA_CONTEXT_LENGTH", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLocalFake(t)
			if tc.set != nil {
				tc.set(l)
			}
			_, err := localCall(t, l, tc.pins)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want it to name %q", err, tc.want)
			}
			if l.chats.Load() != 0 {
				t.Fatal("the brief was sent to a server the pin does not hold")
			}
		})
	}
}

// holdClock sets the probe's clock to the returned time for the length of
// the test.
func holdClock(t *testing.T) *time.Time {
	t.Helper()
	now := time.Date(2026, 10, 18, 9, 0, 0, 0, time.UTC)
	was := probeNow
	probeNow = func() time.Time { return now }
	t.Cleanup(func() { probeNow = was })
	return &now
}

// TestACallReusesAProbeForAMinuteAtMost: a self-contained call to the same
// pinned route within a minute of the probe the pin held reads its report,
// and one made later probes again; an unpinned route to the same server is
// never probed.
func TestACallReusesAProbeForAMinuteAtMost(t *testing.T) {
	l := newLocalFake(t)
	now := holdClock(t)
	f := newFx(t)
	f.machineConfig(pinnedLocalBlock(l.srv.URL+"/v1", ollamaPin))
	c := f.loadAPI()
	call := func() {
		t.Helper()
		if _, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
			Target: Target{Provider: "local", Model: localModel}, Brief: sidesBrief}); err != nil {
			t.Fatal(err)
		}
	}
	call()
	*now = now.Add(59 * time.Second)
	call()
	if n := l.versionAsked.Load(); n != 1 {
		t.Fatalf("the server was probed %d times within a minute, want once", n)
	}
	*now = now.Add(2 * time.Second)
	call()
	if n := l.versionAsked.Load(); n != 2 {
		t.Fatalf("the server was probed %d times, want a report older than a minute probed again", n)
	}
	before := l.versionAsked.Load()
	_, rec, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
		Target: Target{Provider: "local", Model: "other"}, Brief: sidesBrief})
	if err != nil {
		t.Fatal(err)
	}
	if l.versionAsked.Load() != before || rec.LocalPin != nil || rec.LocalServed != nil {
		t.Fatalf("an unpinned route was probed or carries a pin: %+v", rec)
	}
	for _, fl := range LocalFields {
		if r, _ := listed(rec, fl); r != NotConfigured {
			t.Errorf("%s on an unpinned local route = %q, want not_configured", fl, r)
		}
	}
	if r, _ := listed(rec, AskedContext); r != NotConfigured {
		t.Errorf("asked context on an unpinned local route = %q", r)
	}
}

// TestARefusedReportIsNeverReused: a report the pin refused is not kept, so
// the server, once put right, holds the route at the next call without a
// restart.
func TestARefusedReportIsNeverReused(t *testing.T) {
	l := newLocalFake(t)
	holdClock(t)
	f := newFx(t)
	f.machineConfig(pinnedLocalBlock(l.srv.URL+"/v1", ollamaPin))
	c := f.loadAPI()
	call := func() error {
		_, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
			Target: Target{Provider: "local", Model: localModel}, Brief: sidesBrief})
		return err
	}
	l.digest = strings.Repeat("a", 64)
	if err := call(); err == nil || !strings.Contains(err.Error(), "weights_sha256") {
		t.Fatalf("err = %v", err)
	}
	l.digest = localDigest
	if err := call(); err != nil {
		t.Fatalf("the server put right is still refused: %v", err)
	}
	if n := l.versionAsked.Load(); n != 2 {
		t.Fatalf("probes = %d, want the refused report probed again", n)
	}
}

// TestARunnerRouteIsProbedAtEveryUse: the probe a runner's route takes
// before each run is made afresh every time, never read from an earlier one.
func TestARunnerRouteIsProbedAtEveryUse(t *testing.T) {
	l := newLocalFake(t)
	holdClock(t)
	f := newFx(t)
	f.machineConfig(pinnedLocalBlock(l.srv.URL+"/v1", ollamaPin))
	c := f.loadAPI()
	for range 2 {
		if _, _, err := c.ProbeLocal(context.Background(), credential.Machine(f.roots.Home), "local", localModel, true); err != nil {
			t.Fatal(err)
		}
	}
	if n := l.versionAsked.Load(); n != 2 {
		t.Fatalf("probes = %d, want one per use", n)
	}
}

// TestALlamaPinIsHeldToItsWeightsFile: llama-server reports no digest, so
// the model file it has loaded is what identifies the weights: a file other
// than the one pinned, or none named, refuses the route, naming the field and
// what the server has loaded, before the brief is sent.
func TestALlamaPinIsHeldToItsWeightsFile(t *testing.T) {
	for _, tc := range []struct{ name, path, want string }{
		{"other file", "/m/Qwen3-Coder-Q8_0.gguf", "Qwen3-Coder-Q8_0.gguf"},
		{"no file named", "", "names no model file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLocalFake(t)
			l.props = llamaPropsAt(tc.path, 32768, 0)
			_, err := localCall(t, l, localPinText("llama.cpp", "b9860", "32768", "f16"))
			if err == nil || !strings.Contains(err.Error(), "weights_file") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v, want weights_file and %q named", err, tc.want)
			}
			if l.chats.Load() != 0 {
				t.Fatal("the brief was sent to a server holding other weights")
			}
		})
	}
	l := newLocalFake(t)
	l.props = llamaPropsAt(`C:\\models\\`+llamaFile, 32768, 0)
	if _, err := localCall(t, l, localPinText("llama.cpp", "b9860", "32768", "f16")); err != nil {
		t.Fatalf("the pinned file under a Windows path: %v", err)
	}
}

// TestALlamaContextFixCountsItsSlots: llama-server reports each slot's
// context, and --ctx-size is the total its --parallel slots share, so the
// fix names the pinned context times the slots, or one slot.
func TestALlamaContextFixCountsItsSlots(t *testing.T) {
	for _, tc := range []struct {
		name  string
		slots int
		want  []string
	}{
		{"four slots", 4, []string{"--ctx-size 131072", "--parallel 1 --ctx-size 32768"}},
		{"slots unread", 0, []string{"--ctx-size <32768 × its --parallel slot count>", "--parallel 1 --ctx-size 32768"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLocalFake(t)
			l.props = llamaPropsAt(llamaPath, 8192, tc.slots)
			_, err := localCall(t, l, localPinText("llama.cpp", "b9860", "32768", "f16"))
			if err == nil || !strings.Contains(err.Error(), "context_length") {
				t.Fatalf("err = %v", err)
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("err = %v, want it to name %q", err, w)
				}
			}
		})
	}
}

// TestABuildRouteIsHeldToToolsAndContext: a route that runs a tool-using
// role is refused when the server reports no tool calls, when Ollama cannot
// say what context it serves, or when a llama-server pin declares a q4 KV
// cache, each naming the setting that would fix it; a self-contained call
// over the same server is not.
func TestABuildRouteIsHeldToToolsAndContext(t *testing.T) {
	for _, tc := range []struct {
		name, pins, want string
		set              func(*localFake)
	}{
		{"no tools", ollamaPin, "ollama show", func(l *localFake) { l.caps = `"capabilities":["completion"],` }},
		{"context unread", ollamaPin, "OLLAMA_CONTEXT_LENGTH", func(l *localFake) { l.ps = `{"models":[]}` }},
		{"q4 cache", localPinText("llama.cpp", "b9860", "32768", "q4_0"), "--cache-type-k", func(l *localFake) {
			l.props = llamaPropsAt(llamaPath, 32768, 0)
		}},
		{"llama without jinja", localPinText("llama.cpp", "b9860", "32768", "f16"), "--jinja", func(l *localFake) {
			l.props = strings.Replace(llamaPropsAt(llamaPath, 32768, 0), `"supports_tool_calls":true`, `"supports_tool_calls":false`, 1)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			l := newLocalFake(t)
			tc.set(l)
			f := newFx(t)
			f.machineConfig(pinnedLocalBlock(l.srv.URL+"/v1", tc.pins))
			c := f.loadAPI()
			if _, _, err := c.ProbeLocal(context.Background(), credential.Machine(f.roots.Home), "local", localModel, true); err == nil ||
				!strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "tool-using") {
				t.Fatalf("err = %v, want it to name %q", err, tc.want)
			}
			pin, served, err := c.ProbeLocal(context.Background(), credential.Machine(f.roots.Home), "local", localModel, false)
			if err != nil || pin == nil || served == nil {
				t.Fatalf("a self-contained route: %+v, %+v, %v", pin, served, err)
			}
		})
	}
}

// TestAProbeThatFailsSendsNothing: a server that is not there fails the
// probe, the route is refused naming it, and a later use probes again.
func TestAProbeThatFailsSendsNothing(t *testing.T) {
	var made atomic.Int32
	was := probe
	probe = func(ctx context.Context, c *openaiapi.Client, runtime, model string) (openaiapi.LocalReport, error) {
		made.Add(1)
		return was(ctx, c, runtime, model)
	}
	t.Cleanup(func() { probe = was })
	forgetProbes(t)
	srv := httptest.NewServer(http.NotFoundHandler())
	base := srv.URL + "/v1"
	srv.Close()
	f := newFx(t)
	f.machineConfig(pinnedLocalBlock(base, ollamaPin))
	c := f.loadAPI()
	for range 2 {
		_, _, err := c.Call(context.Background(), credential.Machine(f.roots.Home), CallRequest{
			Target: Target{Provider: "local", Model: localModel}, Brief: sidesBrief})
		if err == nil || !strings.Contains(err.Error(), "probe") {
			t.Fatalf("err = %v", err)
		}
	}
	if made.Load() != 2 {
		t.Fatalf("probes made = %d, want a failed probe made again", made.Load())
	}
}
