package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/reading"
)

// Provider dispatch at the delegating verbs (spc-2609251028149555 AC 3,
// itd-2609081951381895 criteria 1 and 5, ruling DR5 of 2026-09-29). Every
// provider is an httptest fake on this machine, and the key is built at run
// time, so no test reaches a network or reads a real key.

var cliDispatchKey = "dk-" + strings.Repeat("5e", 16) + "-not-a-real-key"

// chatFake is an OpenAI-compatible chat endpoint answering content as
// model reported, recording what it was sent.
type chatFake struct {
	srv   *httptest.Server
	calls atomic.Int32
	auth  atomic.Value
	body  atomic.Value
}

func newChatFake(t *testing.T, model string, content func(body string) string) *chatFake {
	t.Helper()
	p := &chatFake{}
	p.auth.Store("")
	p.body.Store("")
	p.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.calls.Add(1)
		p.auth.Store(r.Header.Get("Authorization"))
		raw, _ := io.ReadAll(r.Body)
		p.body.Store(string(raw))
		out, _ := json.Marshal(map[string]any{"model": model,
			"choices": []any{map[string]any{"message": map[string]any{"role": "assistant", "content": content(string(raw))}}}})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(out)
	}))
	t.Cleanup(p.srv.Close)
	return p
}

// pointMachine writes home's ~/.abcd/config.json with provider openrouter at
// base (keyed, its key stored when keyed) and each agent pointed at its model.
func pointMachine(t *testing.T, home, base string, keyed bool, extra string, agents ...string) {
	t.Helper()
	keyField := ""
	if keyed {
		keyField = `"key":"openrouter",`
		if _, err := credential.SetMachine(home, "openrouter", cliDispatchKey); err != nil {
			t.Fatal(err)
		}
	}
	roles := make([]string, 0, len(agents))
	for _, a := range agents {
		roles = append(roles, `"`+a+`":"openrouter/typesafe/jev-1.13"`)
	}
	if extra != "" {
		extra = "," + extra
	}
	body := `{"oracle":{"api":{"openrouter":{"base_url":"` + base + `/v1",` + keyField +
		`"models":["typesafe/jev-1.13"]}},"roles":{` + strings.Join(roles, ",") + `}` + extra + `}}`
	if err := os.MkdirAll(filepath.Join(home, ".abcd"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".abcd", "config.json"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// TestReadingIngestDispatchesAPointedPosition is criterion 1 and AC 3 at a
// verb: a cold-reading position pointed at a keyed provider is self-contained
// under DR5, so `reading ingest --dispatch <run>` sends the definition and the
// parked bundle under the key, ingests the answer, and the receipt names the
// provider, the model asked and the model reported (criterion 5).
func TestReadingIngestDispatchesAPointedPosition(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(repo)
	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	outPath := detectionPayloadFile(t, runID, manifestHash, def.Regime, def)
	answer, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	p := newChatFake(t, "typesafe/jev-1.13-20260915", func(string) string { return string(answer) })
	pointMachine(t, home, p.srv.URL, true, "", "cold-reading-detection")

	stdout, stderr, err := runCLISplit(t, "reading", "ingest", "--dispatch", runID, "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	if p.auth.Load() != "Bearer "+cliDispatchKey {
		t.Fatal("the key stored under the provider's credential name was not the one sent")
	}
	body := p.body.Load().(string)
	for _, want := range []string{runID, manifestHash, def.SHA256, "context_stamp", `"model":"typesafe/jev-1.13"`, "cold-reading-detection"} {
		if !strings.Contains(body, want) {
			t.Fatalf("the request does not carry %q", want)
		}
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.ConnectionUsed != "openrouter" || rc.ProviderCall == nil || rc.ProviderCall.Provider != "openrouter" ||
		rc.ProviderCall.ModelAsked != "typesafe/jev-1.13" || rc.ProviderCall.ModelReported != "typesafe/jev-1.13-20260915" {
		t.Fatalf("receipt = %+v, provider call %+v", rc, rc.ProviderCall)
	}
	if strings.Contains(stdout+stderr, cliDispatchKey) {
		t.Fatal("the key reached the verb's output")
	}
	var res struct {
		RunID   string            `json:"run_id"`
		Records []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || res.RunID != runID || len(res.Records) != 1 {
		t.Fatalf("ingest result = %+v, %v\n%s", res, err, stdout)
	}
}

// TestADispatchSaysHowManyItemsTravelUnscanned: before a parked run is sent
// under the person's key, the send names on stderr how many of the bundle's
// items the exclusion floor never examined, the count its manifest marks
// `unscanned`, so a paid send never carries that figure silently
// (review-providerDispatch2 point 3).
func TestADispatchSaysHowManyItemsTravelUnscanned(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(repo)
	runID, _, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	manifestPath := filepath.Join(repo, reading.DefaultRunDir, runID, reading.ManifestFileName)
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture's material is all parsed, so one item is marked unscanned,
	// as a preset admitting a test or source row would mark it, and the
	// output cites the manifest as it now stands.
	raw = []byte(strings.Replace(string(raw), `"scan": "parsed"`, `"scan": "unscanned"`, 1))
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(raw)
	manifestHash := hex.EncodeToString(sum[:])
	m, err := reading.DecodeManifest(raw)
	if err != nil {
		t.Fatal(err)
	}
	unscanned := 0
	for _, it := range m.Items {
		if it.Scan == reading.ScanUnscanned {
			unscanned++
		}
	}
	if unscanned != 1 {
		t.Fatalf("the fixture marks %d item(s) unscanned, want 1", unscanned)
	}
	outPath := detectionPayloadFile(t, runID, manifestHash, def.Regime, def)
	answer, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatal(err)
	}
	p := newChatFake(t, "typesafe/jev-1.13", func(string) string { return string(answer) })
	pointMachine(t, home, p.srv.URL, true, "", "cold-reading-detection")
	_, stderr, err := runCLISplit(t, "reading", "ingest", "--dispatch", runID, "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	want := regexp.MustCompile(`(?m)^reading ingest: run ` + runID + ` sends ` + strconv.Itoa(len(m.Items)) +
		` item\(s\) to openrouter, ` + strconv.Itoa(unscanned) + ` of them unscanned`)
	if !want.MatchString(stderr) {
		t.Fatalf("stderr does not name the unscanned count (%d of %d):\n%s", unscanned, len(m.Items), stderr)
	}
}

// TestReadingIngestDispatchNeedsAPointedPosition: with nothing pointed, the
// host runs the reading, so --dispatch is refused at exit 2 naming the host's
// path, and --dispatch with --reading-json is refused as two outputs.
func TestReadingIngestDispatchNeedsAPointedPosition(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	t.Setenv("HOME", t.TempDir())
	t.Chdir(repo)
	runID, _, _ := parkedRunForIngest(t, srcRoot, repo, "detection")
	_, _, err := runCLISplit(t, "reading", "ingest", "--dispatch", runID)
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "--reading-json") || !strings.Contains(err.Error(), "harness") {
		t.Fatalf("err = %v", err)
	}
	for _, bad := range []string{"../escape", "rdg-1"} {
		_, _, err = runCLISplit(t, "reading", "ingest", "--dispatch", bad)
		if exitCodeOf(err) != 2 || !(strings.Contains(err.Error(), "not a reading run id") || strings.Contains(err.Error(), "not parked")) {
			t.Fatalf("--dispatch %s: err = %v", bad, err)
		}
	}
	_, _, err = runCLISplit(t, "reading", "ingest", "--dispatch", runID, "--reading-json", "x.json")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "--dispatch") {
		t.Fatalf("err = %v", err)
	}
}

// TestAHostPayloadOnAProviderRouteIsRefused: a reading pointed at a provider
// is sent there by abcd, so an output the host produced is refused rather than
// recorded under the provider's name, and --route <agent>=host-decides keeps
// one run on the harness.
func TestAHostPayloadOnAProviderRouteIsRefused(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := readingRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Chdir(repo)
	runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
	outPath := detectionPayloadFile(t, runID, manifestHash, def.Regime, def)
	p := newChatFake(t, "typesafe/jev-1.13", func(string) string { return "{}" })
	pointMachine(t, home, p.srv.URL, true, "", "cold-reading-detection")
	_, _, err := runCLISplit(t, "reading", "ingest", "--reading-json", outPath)
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "--dispatch") ||
		!strings.Contains(err.Error(), "cold-reading-detection=host-decides") {
		t.Fatalf("err = %v", err)
	}
	stdout, stderr, err := runCLISplit(t, "reading", "ingest", "--reading-json", outPath, "--json",
		"--route", "cold-reading-detection=host-decides")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.ConnectionUsed != oracle.Harness {
		t.Fatalf("receipt = %+v; want the harness", rc)
	}
	if n := p.calls.Load(); n != 0 {
		t.Fatalf("provider called %d times", n)
	}
}

// TestIntentAuditOnAKeyedProviderIsRefusedByDR5: the intent auditor reads
// files, so pointed at a provider that holds a key it is refused at exit 2
// before anything is written or sent, naming the rule and the override.
func TestIntentAuditOnAKeyedProviderIsRefusedByDR5(t *testing.T) {
	root := intentTestRepo(t)
	writeRepoFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md", conditionedIntent)
	p := newChatFake(t, "typesafe/jev-1.13", func(string) string { return "{}" })
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, true, "", "intent-auditor")
	_, _, err := runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if exitCodeOf(err) != 2 {
		t.Fatalf("err = %v, want exit 2", err)
	}
	for _, want := range []string{"intent-auditor", "DR5", "oracle.bundled_context_providers"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("refusal %q does not name %q", err, want)
		}
	}
	if n := p.calls.Load(); n != 0 {
		t.Fatalf("provider called %d times", n)
	}
	if _, serr := os.Stat(filepath.Join(root, ".abcd", "work", "reviews")); serr == nil {
		entries, _ := os.ReadDir(filepath.Join(root, ".abcd", "work", "reviews"))
		if len(entries) != 0 {
			t.Fatalf("the refused emit wrote %d review entries", len(entries))
		}
	}
}

// TestIntentAuditOnAKeylessProviderDispatches: a provider whose block names
// no key is outside DR5, so the auditor's request is sent there, the answer
// is ingested, and the receipt names the provider as used.
func TestIntentAuditOnAKeylessProviderDispatches(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	root := intentTestRepo(t)
	writeRepoFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md", conditionedIntent)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	p := newChatFake(t, "typesafe/jev-1.13", func(body string) string {
		return `{"_type":"abcd/intent-fidelity-verdict/v1","receipt_id":"` + regexp.MustCompile(`rcp-[0-9a-f]{12}`).FindString(body) + `"}`
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	if a := p.auth.Load(); a != "" {
		t.Fatalf("Authorization = %v on a keyless provider", a)
	}
	body := p.body.Load().(string)
	if !strings.Contains(body, "itd-10") || !strings.Contains(body, "intent-auditor") {
		t.Fatal("the request does not carry the emitted review request and the auditor's prompt")
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.ConnectionUsed != "openrouter" || rc.ProviderCall == nil {
		t.Fatalf("receipt = %+v", rc)
	}
	var res struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || res.Status != "dead_letter" {
		t.Fatalf("ingest status = %q, %v; want the malformed answer quarantined", res.Status, err)
	}
}

// TestEveryEmittingVerbRefusesAFileReadingAgentOnAKeyedProvider: DR5 at every
// verb that emits a file-reading agent's request. Pointed at a provider that
// holds a key, each exits 2 before any call, naming the rule and the override.
func TestEveryEmittingVerbRefusesAFileReadingAgentOnAKeyedProvider(t *testing.T) {
	cases := []struct {
		name  string
		agent string
		setup func(t *testing.T)
		args  []string
	}{
		{"intent consistency", "intent-auditor", func(t *testing.T) { consistencyCLIRepo(t) }, []string{"intent", "consistency", "--json"}},
		{"intent audit --owed", "intent-auditor", func(t *testing.T) { drainRepo(t) }, []string{"intent", "audit", "--owed", "--json"}},
		{"launch ship", "release-changelog-composer", func(t *testing.T) {
			r := shipReadyRepo(t)
			t.Setenv("HOME", t.TempDir())
			t.Chdir(r.Root())
		}, []string{"launch", "ship", "--json"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.setup(t)
			p := newChatFake(t, "typesafe/jev-1.13", func(string) string { return "{}" })
			pointMachine(t, os.Getenv("HOME"), p.srv.URL, true, "", tc.agent)
			_, _, err := runCLISplit(t, tc.args...)
			if exitCodeOf(err) != 2 {
				t.Fatalf("err = %v, want exit 2", err)
			}
			for _, want := range []string{tc.agent, "DR5", "oracle.bundled_context_providers"} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("refusal %q does not name %q", err, want)
				}
			}
			if n := p.calls.Load(); n != 0 {
				t.Fatalf("provider called %d times", n)
			}
		})
	}
}

// TestIntentConsistencyOnAKeylessProviderDispatches: the pass's request and
// corpus are sent to a keyless provider, and its findings are ingested.
func TestIntentConsistencyOnAKeylessProviderDispatches(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	repo := consistencyCLIRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	var em consistencyEmitted
	if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
		t.Fatal(err)
	}
	findings, err := os.ReadFile(consistencyFindingsFile(t, repo, em))
	if err != nil {
		t.Fatal(err)
	}
	p := newChatFake(t, "local-model", func(string) string { return string(findings) })
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "consistency", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	if body := p.body.Load().(string); !strings.Contains(body, filepath.Base(em.CorpusPath)) || !strings.Contains(body, filepath.Base(em.RequestPath)) {
		t.Fatal("the request does not carry the emitted request and corpus")
	}
	var res struct {
		Status string   `json:"status"`
		Filed  []string `json:"filed"`
	}
	if err := json.Unmarshal([]byte(stdout), &res); err != nil || res.Status != "ingested" || len(res.Filed) != 1 {
		t.Fatalf("ingest = %+v, %v\n%s", res, err, stdout)
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.ConnectionUsed != "openrouter" || rc.ProviderCall == nil || rc.ProviderCall.ModelReported != "local-model" {
		t.Fatalf("receipt = %+v", rc)
	}
}

// TestLaunchShipOnAKeylessProviderDispatches: the emitted cut is sent to a
// keyless provider and the release is ingested from its answer.
func TestLaunchShipOnAKeylessProviderDispatches(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	r := shipReadyRepo(t)
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	t.Chdir(r.Root())
	composed, err := os.ReadFile(composedPayload(t, t.TempDir(), "v0.4.1", "itd-73"))
	if err != nil {
		t.Fatal(err)
	}
	p := newChatFake(t, "local-model", func(string) string { return string(composed) })
	pointMachine(t, home, p.srv.URL, false, "", "release-changelog-composer")
	stdout, stderr, err := runCLISplit(t, "launch", "ship", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	if body := p.body.Load().(string); !strings.Contains(body, "next_tag") || !strings.Contains(body, "release-changelog-composer") {
		t.Fatal("the request does not carry the emitted cut and the composer's prompt")
	}
	var rc oracle.ReceiptRoute
	member(t, []byte(stdout), "route", &rc)
	if rc.ConnectionUsed != "openrouter" || rc.ProviderCall == nil {
		t.Fatalf("receipt = %+v", rc)
	}
}

// TestDrainOnAKeylessProviderRunsTheHead: the drain's head is sent to a
// keyless provider and its verdict ingested, so that entry ran.
func TestDrainOnAKeylessProviderRunsTheHead(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	drainRepo(t)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	p := newChatFake(t, "local-model", func(body string) string {
		return `{"_type":"abcd/intent-fidelity-verdict/v1","receipt_id":"` + regexp.MustCompile(`rcp-[0-9a-f]{12}`).FindString(body) + `"}`
	})
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", "intent-auditor")
	out, stderr, err := runCLISplit(t, "intent", "audit", "--owed")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if n := p.calls.Load(); n != 1 {
		t.Fatalf("provider called %d times, want 1", n)
	}
	if !strings.Contains(out, "ran: itd-21") || !strings.Contains(out, "route: intent-auditor ran on openrouter") ||
		strings.Contains(out, "this command runs no reviewer") {
		t.Fatalf("drain render:\n%s", out)
	}
}

// TestAnUnreachableProviderLeavesTheStepToTheHost: a provider that cannot be
// reached at all (nothing was sent) leaves the written request to the host,
// with one stderr line and the request block naming the harness as used.
func TestAnUnreachableProviderLeavesTheStepToTheHost(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	root := intentTestRepo(t)
	writeRepoFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md", conditionedIntent)
	t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
	gone := httptest.NewServer(http.NotFoundHandler())
	base := gone.URL
	gone.Close()
	pointMachine(t, os.Getenv("HOME"), base, false, "", "intent-auditor")
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "could not be reached") || strings.Count(stderr, "\n") != 1 {
		t.Fatalf("stderr %q", stderr)
	}
	var rr oracle.RequestRouting
	member(t, []byte(stdout), "routing", &rr)
	if rr.Connection != oracle.Harness {
		t.Fatalf("routing = %+v; want the harness after the fallback", rr)
	}
}

// TestHostPayloadsAreRefusedOnAProviderRoute: every ingest handed a payload
// the host produced refuses it when the agent is routed to a provider, before
// anything is read or written, naming the escape to the harness.
func TestHostPayloadsAreRefusedOnAProviderRoute(t *testing.T) {
	cases := []struct {
		agent string
		args  []string
	}{
		{"intent-auditor", []string{"intent", "audit", "ingest", "--verdict-json", "v.json"}},
		{"intent-auditor", []string{"intent", "consistency", "ingest", "--findings-json", "f.json"}},
		{"release-changelog-composer", []string{"launch", "ship", "--changelog-json", "c.json"}},
		{"principle-distiller", []string{"disembark", "principles", "lb", "--principles-json", "p.json"}},
		{"press-release-composer", []string{"disembark", "press-release", "lb", "--press-release-json", "p.json"}},
		{"lifeboat-reviewer", []string{"disembark", "review", "lb", "src", "--review-json", "r.json"}},
		{"graveyard-interpreter", []string{"disembark", "graveyard", "lb", "--lessons-json", "l.json"}},
	}
	for _, tc := range cases {
		t.Run(strings.Join(tc.args[:2], " "), func(t *testing.T) {
			intentTestRepo(t)
			p := newChatFake(t, "local-model", func(string) string { return "{}" })
			pointMachine(t, os.Getenv("HOME"), p.srv.URL, false, "", tc.agent)
			_, _, err := runCLISplit(t, tc.args...)
			if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "routed to provider openrouter") ||
				!strings.Contains(err.Error(), tc.agent+"=host-decides") {
				t.Fatalf("err = %v", err)
			}
			if n := p.calls.Load(); n != 0 {
				t.Fatalf("provider called %d times", n)
			}
		})
	}
}

// TestSpecCloseSendsTheReviewAndTheCloseStands: the close ships the intent
// whatever the review does. Pointed at a keyless provider the review runs
// there and is ingested, said on stderr; pointed at a keyed one DR5 refuses it
// and the review stays owed, said as a warning.
func TestSpecCloseSendsTheReviewAndTheCloseStands(t *testing.T) {
	srcRoot := repoRootFromTest(t)
	for _, keyed := range []bool{false, true} {
		t.Run(map[bool]string{false: "keyless", true: "keyed"}[keyed], func(t *testing.T) {
			repo := specCloseWorld(t)
			t.Setenv("ABCD_PLUGIN_ROOT", srcRoot)
			p := newChatFake(t, "local-model", func(body string) string {
				return `{"_type":"abcd/intent-fidelity-verdict/v1","receipt_id":"` + regexp.MustCompile(`rcp-[0-9a-f]{12}`).FindString(body) + `"}`
			})
			pointMachine(t, os.Getenv("HOME"), p.srv.URL, keyed, "", "intent-auditor")
			_, stderr := closeRequest(t, repo)
			if keyed {
				if !strings.Contains(stderr, "WARNING") || !strings.Contains(stderr, "DR5") || p.calls.Load() != 0 {
					t.Fatalf("stderr %q, calls %d", stderr, p.calls.Load())
				}
				return
			}
			if !strings.Contains(stderr, "ran on openrouter") || p.calls.Load() != 1 {
				t.Fatalf("stderr %q, calls %d", stderr, p.calls.Load())
			}
		})
	}
}

// TestTheVerbsReadTheMachinesProviderConfiguration: a delegating verb reads
// the provider configuration before it resolves a route. A fault in it exits
// 2 before anything is written, and its diagnostics (here, a repository's
// route to a keyed provider, skipped under ruling CD2) are printed on stderr.
func TestTheVerbsReadTheMachinesProviderConfiguration(t *testing.T) {
	root := intentTestRepo(t)
	writeRepoFile(t, root, ".abcd/development/intents/shipped/itd-10-alpha.md", conditionedIntent)
	p := newChatFake(t, "typesafe/jev-1.13", func(string) string { return "{}" })
	pointMachine(t, os.Getenv("HOME"), p.srv.URL, true, "")

	writeRepoFile(t, root, ".abcd/config.json", `{"oracle":{"roles":{"intent-auditor":"openrouter/typesafe/jev-1.13"}}}`)
	stdout, stderr, err := runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "only a route set on this machine may spend that key") {
		t.Fatalf("stderr %q does not carry the skip diagnostic", stderr)
	}
	var rr oracle.RequestRouting
	member(t, []byte(stdout), "routing", &rr)
	if rr.Connection != oracle.Harness || p.calls.Load() != 0 {
		t.Fatalf("routing = %+v, calls %d; want the harness", rr, p.calls.Load())
	}

	writeRepoFile(t, root, ".abcd/config.json", `{"oracle":{"bundled_context_providers":["openrouter"]}}`)
	_, _, err = runCLISplit(t, "intent", "audit", "itd-10", "--json")
	if exitCodeOf(err) != 2 || !strings.Contains(err.Error(), "oracle.bundled_context_providers") {
		t.Fatalf("err = %v; want the repository's override refused at exit 2", err)
	}
}

// TestAhoyProvidersSaysWhatDispatchSends: the providers board's dispatch line
// states what the verbs do with a pointed role, and which agents a paid
// provider takes under DR5, never that dispatch is still to come.
func TestAhoyProvidersSaysWhatDispatchSends(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	out, err := runCLIErr(t, "ahoy", "--providers", "--json")
	if err != nil {
		t.Fatalf("ahoy --providers: %v\n%s", err, out)
	}
	var board struct {
		Dispatch string `json:"dispatch"`
	}
	if err := json.Unmarshal(out, &board); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, want := range []string{"sends the step there itself", "DR5", "cold-reading", "oracle.bundled_context_providers"} {
		if !strings.Contains(board.Dispatch, want) {
			t.Fatalf("dispatch %q does not say %q", board.Dispatch, want)
		}
	}
	if strings.Contains(board.Dispatch, "lands") {
		t.Fatalf("dispatch %q still says dispatch is to come", board.Dispatch)
	}
}
