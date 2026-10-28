package matrix

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// The machine's own home, user and host name, read before TestMain repoints
// HOME, so the leak checks below look for the values a record would leak.
var realHome, realUser, realHost string

// TestMain points HOME at a throwaway directory, so no test here reaches the
// operator's real lab store.
func TestMain(m *testing.M) {
	realHome, _ = os.UserHomeDir()
	realHost, _ = os.Hostname()
	if u, err := user.Current(); err == nil {
		realUser = u.Username
	}
	dir, err := os.MkdirTemp("", "abcd-matrix-home-")
	if err != nil {
		panic(err)
	}
	os.Setenv("HOME", dir)
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

var (
	sha40 = strings.Repeat("a", 40)
	sha64 = strings.Repeat("b", 64)
	hex64 = func(c string) string { return strings.Repeat(c, 64) }
)

// fixtureRoutes are goodDeclaration's runner routes, by their cell's index in
// inputs.
var fixtureRoutes = map[int]string{1: "builder=runner:opencode@openrouter/vendor/model", 2: "reviewer=runner:opencode@local/qwen"}

// remodel points cell i's declared route and its pin at model together, on the
// route's provider, so the pin still agrees with its route.
func remodel(in *runInputs, i int, model string) {
	route := fixtureRoutes[i]
	head, provider, _ := strings.Cut(route, "@")
	provider, _, _ = strings.Cut(provider, "/")
	in.Declaration = []byte(strings.Replace(string(in.Declaration), route, head+"@"+provider+"/"+model, 1))
	in.Cells[i].Model = provider + "/" + model
}

// A pin is held to what its cell's declared route fixes: a route naming a
// model is never pinned host, the host leg is never pinned anything else, and
// a runner route is pinned on its own runner and model (dcn-2610280629294458).
func TestACellPinThatDisagreesWithItsRouteIsRefused(t *testing.T) {
	for name, tc := range map[string]struct {
		mutate func(*runInputs)
		want   string
	}{
		"a provider route pinned host": {func(in *runInputs) {
			in.Cells[1].Source, in.Cells[1].ProviderPin = SourceHost, nil
		}, "names the host as its source"},
		"the host leg pinned openrouter": {func(in *runInputs) {
			in.Cells[0].Source, in.Cells[0].Model = SourceOpenRouter, "openrouter/vendor/model"
			in.Cells[0].ProviderPin = &ProviderPin{Order: []string{"provider-a"}}
		}, "the route is the host's"},
		"another runner": {func(in *runInputs) { in.Cells[2].Runner = "claude" }, "names runner opencode"},
		"another model":  {func(in *runInputs) { in.Cells[1].Model = "openrouter/vendor/other" }, "names the model openrouter/vendor/model"},
	} {
		t.Run(name, func(t *testing.T) {
			in := inputs(t)
			tc.mutate(&in)
			if _, err := newRunRecord(in); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("newRunRecord = %v; want a refusal saying %q", err, tc.want)
			}
		})
	}
}

// inputs is a complete run's inputs: every hash given, every cell pinned.
func inputs(t *testing.T) runInputs {
	t.Helper()
	return runInputs{
		RunID:                 "mx-261023012949-ab12",
		Lab:                   "lab-261023012949-aaaaaaa",
		RootSHA:               sha40,
		FrozenCommit:          strings.Repeat("c", 40),
		Binary:                BinaryPin{SHA256: sha64, Version: "0.16.0", Revision: strings.Repeat("d", 40), Known: true},
		Declaration:           []byte(goodDeclaration),
		PromptSHA256:          map[string]string{"builder": hex64("1"), "reviewer": hex64("2")},
		ToolDefinitionsSHA256: map[string]string{"opencode": hex64("3")},
		RenderTemplate:        &ContentRef{Name: "template.html", SHA256: hex64("4"), Bytes: 10},
		HiddenTests:           []ContentRef{{Name: "hidden_test.go", SHA256: hex64("5"), Bytes: 20}},
		SeededMaterial:        []ContentRef{{Name: "seeded.md", SHA256: hex64("6"), Bytes: 30}},
		Machine:               Machine{OS: "darwin", Arch: "arm64", Cores: 12},
		Cells: []CellPin{
			{Cell: "host-big", Role: RoleBuilder, Source: SourceHost, Runner: "host"},
			{Cell: "or-cheap", Role: RoleBuilder, Source: SourceOpenRouter, Runner: "opencode", Model: "openrouter/vendor/model",
				HarnessVersion: "1.2.3", HarnessDefault: []string{"top_p"},
				ProviderPin: &ProviderPin{Order: []string{"provider-a"}, AllowFallbacks: false}},
			{Cell: "local-rev", Role: RoleReviewer, Source: SourceLocal, Runner: "opencode", Model: "local/qwen", HarnessVersion: "1.2.3",
				LocalPin: &LocalPin{Runtime: "llama-server", RuntimeVersion: "b6000", WeightsSHA256: hex64("7"), Quantisation: "Q4_K_M"}},
		},
	}
}

func TestTheRunRecordNamesCommitBinaryHashesTemplateMachineConcurrencyAndCap(t *testing.T) {
	rec, err := newRunRecord(inputs(t))
	if err != nil {
		t.Fatalf("newRunRecord: %v", err)
	}
	js, err := json.Marshal(rec)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(js, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"schema_version", "run_id", "lab", "root_sha", "frozen_commit", "binary", "declaration",
		"prompt_sha256", "tool_definitions_sha256", "render_template", "hidden_tests", "seeded_material",
		"machine", "concurrency", "load_cap", "spend_cap", "spend_cap_origin", "cells", "unpinned"} {
		if _, ok := m[k]; !ok {
			t.Errorf("the run record lacks %q:\n%s", k, js)
		}
	}
	if rec.FrozenCommit != strings.Repeat("c", 40) || rec.Binary.SHA256 != sha64 || rec.Binary.Version != "0.16.0" ||
		rec.Concurrency != 2 || rec.LoadCap == nil || *rec.LoadCap != 4 || rec.SpendCap == nil || rec.SpendCap.Amount != 25 ||
		rec.Machine.Cores != 12 || rec.RenderTemplate == nil || rec.Declaration.SHA256 == "" || rec.Declaration.Bytes != int64(len(goodDeclaration)) {
		t.Fatalf("record = %+v", rec)
	}
	// The machine's model and memory are never read (they need a per-OS
	// exec), so they are listed, not left out; the host cell cannot report
	// its model, nor the openrouter cell the setting it leaves to the harness.
	for _, want := range []oracle.Unpinned{
		{Field: RunMachineCPU, Reason: oracle.NotReported},
		{Field: RunMachineMemory, Reason: oracle.NotReported},
		{Field: cellField("host-big", "model"), Reason: oracle.NotReported},
		{Field: cellField("or-cheap", "settings.top_p"), Reason: oracle.NotConfigured},
	} {
		if !hasUnpinned(rec.Unpinned, want) {
			t.Errorf("Unpinned lacks %+v: %+v", want, rec.Unpinned)
		}
	}
	// Round trip: the record is a record, not a view.
	var back RunRecord
	if err := json.Unmarshal(js, &back); err != nil {
		t.Fatal(err)
	}
	js2, _ := json.Marshal(back)
	if string(js) != string(js2) {
		t.Fatalf("the run record does not round-trip:\n%s\n%s", js, js2)
	}
}

func hasUnpinned(us []oracle.Unpinned, want oracle.Unpinned) bool {
	for _, u := range us {
		if u == want {
			return true
		}
	}
	return false
}

func TestAHashTheRunCannotReadIsListedUnpinnedNeverOmitted(t *testing.T) {
	in := inputs(t)
	in.PromptSHA256 = nil
	in.ToolDefinitionsSHA256 = map[string]string{}
	in.RenderTemplate = nil
	in.Binary.Known = false
	in.Cells[2].LocalPin = nil
	in.Cells[1].HarnessVersion = ""
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatalf("newRunRecord: %v", err)
	}
	for _, want := range []oracle.Unpinned{
		{Field: RunPromptSHA256, Reason: oracle.NotReported},
		{Field: RunToolDefinitionsSHA256, Reason: oracle.NotReported},
		{Field: RunRenderTemplate, Reason: oracle.NotConfigured},
		{Field: RunBinaryRevision, Reason: oracle.NotReported},
		{Field: cellField("local-rev", "local_pin"), Reason: oracle.NotConfigured},
		{Field: cellField("or-cheap", "harness_version"), Reason: oracle.NotReported},
	} {
		if !hasUnpinned(rec.Unpinned, want) {
			t.Errorf("Unpinned lacks %+v: %+v", want, rec.Unpinned)
		}
	}
	js, _ := json.Marshal(rec)
	for _, k := range []string{`"prompt_sha256":{}`, `"tool_definitions_sha256":{}`, `"render_template":null`} {
		if !strings.Contains(string(js), k) {
			t.Errorf("the record omits rather than empties %s:\n%s", k, js)
		}
	}
}

func TestTheRunRecordIsRefusedWhatItCannotStandOn(t *testing.T) {
	cases := map[string]func(*runInputs){
		"run id":         func(in *runInputs) { in.RunID = "mx-1" },
		"no lab":         func(in *runInputs) { in.Lab = "" },
		"frozen commit":  func(in *runInputs) { in.FrozenCommit = "HEAD" },
		"binary digest":  func(in *runInputs) { in.Binary.SHA256 = "" },
		"binary version": func(in *runInputs) { in.Binary.Version = "" },
		// The binary's version and revision are recorded names, as every
		// other identity field is.
		"escape in binary version":   func(in *runInputs) { in.Binary.Version = "0.16.0\x1b]0;x\a" },
		"space in binary version":    func(in *runInputs) { in.Binary.Version = "0.16.0 (dev)" },
		"long binary version":        func(in *runInputs) { in.Binary.Version = strings.Repeat("9", oracle.MaxModelBytes+1) },
		"escape in binary revision":  func(in *runInputs) { in.Binary.Revision = "dddd\u202e" },
		"newline in binary revision": func(in *runInputs) { in.Binary.Revision = "dddd\nforged" },
		"prompt hash shape":          func(in *runInputs) { in.PromptSHA256["builder"] = "not-a-hash" },
		"machine cores":              func(in *runInputs) { in.Machine.Cores = 0 },
		"load cap below cores":       func(in *runInputs) { in.Machine.Cores = 4 },
		"metered cell no cap":        func(in *runInputs) { in.Declaration = noCap() },
		"machine cap unit":           func(in *runInputs) { in.Declaration = noCap(); in.MachineSpendCap = &Money{Amount: 5, Unit: "credits"} },
		"missing cell pin":           func(in *runInputs) { in.Cells = in.Cells[:2] },
		"unknown cell pin":           func(in *runInputs) { in.Cells[0].Cell = "nobody" },
		"wrong role":                 func(in *runInputs) { in.Cells[2].Role = RoleBuilder },
		"unknown source":             func(in *runInputs) { in.Cells[0].Source = "cloud" },
		"openrouter no pin":          func(in *runInputs) { in.Cells[1].ProviderPin = nil },
		"openrouter fallback":        func(in *runInputs) { in.Cells[1].ProviderPin.AllowFallbacks = true },
		"pin on wrong source":        func(in *runInputs) { in.Cells[0].LocalPin = &LocalPin{} },
		"model missing":              func(in *runInputs) { in.Cells[1].Model = "" },
		"setting both ways":          func(in *runInputs) { in.Cells[1].HarnessDefault = []string{"temperature"} },
		"content ref":                func(in *runInputs) { in.HiddenTests[0].SHA256 = "x" },
		"no declaration":             func(in *runInputs) { in.Declaration = nil },
		"declaration refused":        func(in *runInputs) { in.Declaration = []byte(`{}`) },
		// Every pin string is a bounded, plain recorded name; the weight
		// file is a base name, never a path on the person's machine.
		"absolute weights file":  func(in *runInputs) { in.Cells[2].LocalPin.WeightsFile = "/Users/someone/models/x.gguf" },
		"weights file in a dir":  func(in *runInputs) { in.Cells[2].LocalPin.WeightsFile = "models/x.gguf" },
		"weights file home":      func(in *runInputs) { in.Cells[2].LocalPin.WeightsFile = "~" },
		"long runtime":           func(in *runInputs) { in.Cells[2].LocalPin.Runtime = strings.Repeat("r", oracle.MaxModelBytes+1) },
		"escape in quantisation": func(in *runInputs) { in.Cells[2].LocalPin.Quantisation = "Q4\x1b[2J" },
		"escape in kv cache":     func(in *runInputs) { in.Cells[2].LocalPin.KVCacheType = "f16\u202e" },
		"negative context":       func(in *runInputs) { in.Cells[2].LocalPin.ContextLength = -1 },
		"escape in order":        func(in *runInputs) { in.Cells[1].ProviderPin.Order = []string{"a\x1b]0;x\a"} },
		"too many orders": func(in *runInputs) {
			for range maxPinList + 1 {
				in.Cells[1].ProviderPin.Order = append(in.Cells[1].ProviderPin.Order, "p")
			}
		},
		"too many quantizations": func(in *runInputs) {
			for range maxPinList + 1 {
				in.Cells[1].ProviderPin.Quantizations = append(in.Cells[1].ProviderPin.Quantizations, "fp8")
			}
		},
		"long canonical slug": func(in *runInputs) {
			in.Cells[1].ProviderPin.CanonicalSlug = strings.Repeat("s", oracle.MaxModelBytes+1)
		},
		"escape in data policy": func(in *runInputs) { in.Cells[1].ProviderPin.DataCollection = "deny\x07" },
		"too many harness defaults": func(in *runInputs) {
			for i := range oracle.MaxSettings + 1 {
				in.Cells[1].HarnessDefault = append(in.Cells[1].HarnessDefault, fmt.Sprintf("s%d", i))
			}
		},
		"too many hidden tests": func(in *runInputs) {
			for len(in.HiddenTests) <= maxRefs {
				in.HiddenTests = append(in.HiddenTests, in.HiddenTests[0])
			}
		},
		"escape in a model":  func(in *runInputs) { remodel(in, 1, "vendor/\u200bmodel") },
		"local cell, no cap": func(in *runInputs) { in.Declaration = noLoadCap() },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			in := inputs(t)
			mutate(&in)
			if _, err := newRunRecord(in); err == nil {
				t.Fatal("newRunRecord admitted it")
			}
		})
	}
	// The person's own configuration supplies the cap when the run names none.
	in := inputs(t)
	in.Declaration = noCap()
	in.MachineSpendCap = &Money{Amount: 9, Unit: "usd"}
	rec, err := newRunRecord(in)
	if err != nil || rec.SpendCap == nil || rec.SpendCap.Amount != 9 || rec.SpendCapOrigin != CapFromMachine {
		t.Fatalf("machine cap: %+v %q, %v", rec.SpendCap, rec.SpendCapOrigin, err)
	}
	// The run's own cap overrides it (decision 17).
	in = inputs(t)
	in.MachineSpendCap = &Money{Amount: 9, Unit: "usd"}
	if rec, err := newRunRecord(in); err != nil || rec.SpendCap.Amount != 25 || rec.SpendCapOrigin != CapFromRun {
		t.Fatalf("run cap: %+v %q, %v", rec.SpendCap, rec.SpendCapOrigin, err)
	}
	// A run of host and local cells alone needs no cap.
	in = inputs(t)
	in.Declaration = []byte(strings.Replace(string(noCap()),
		`,
    {"id": "or-cheap", "route": "builder=runner:opencode@openrouter/vendor/model", "settings": {"temperature": 0.6}}`, "", 1))
	in.Cells = []CellPin{in.Cells[0], in.Cells[2]}
	if rec, err := newRunRecord(in); err != nil || rec.SpendCap != nil || rec.SpendCapOrigin != "" {
		t.Fatalf("a run with no metered cell: %+v, %v; want no cap and no refusal", rec.SpendCap, err)
	}
}

// noLoadCap is the good declaration without its load cap.
func noLoadCap() []byte {
	return []byte(strings.Replace(goodDeclaration, `"load_cap": 4,
`, "", 1))
}

// Decision 12: the load cap is the local cells'. A run with none names no load
// cap and is not refused for it.
func TestARunWithNoLocalCellNeedsNoLoadCap(t *testing.T) {
	in := inputs(t)
	in.Declaration = []byte(strings.Replace(string(noLoadCap()), `,
  "reviewers": [
    {"id": "local-rev", "route": "reviewer=runner:opencode@local/qwen"}
  ]`, "", 1))
	in.Cells = in.Cells[:2]
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatalf("a run with no local cell and no load cap: %v", err)
	}
	js, _ := json.Marshal(rec)
	if rec.LoadCap != nil || !strings.Contains(string(js), `"load_cap":null`) {
		t.Fatalf("load cap = %v; want none, recorded as null:\n%s", rec.LoadCap, js)
	}
}

// The largest run record a declaration can produce fits the bound its lab
// reads a record back within: a record is never written that cannot be read.
// Every string is at its bound and made of the character JSON escapes the
// widest that a recorded name may carry: '"', written as two bytes (a control
// character, which JSON writes as six, is refused as a name). Settings values
// are kept as sent, so their bound is on the bytes as written.
func TestTheLargestRunRecordFitsTheLabsReadBound(t *testing.T) {
	const q = `"`
	long := func(c string) string { return strings.Repeat(c, oracle.MaxModelBytes) }
	var sjs []string
	for i := range 8 {
		k := fmt.Sprintf("s%d_%s", i, strings.Repeat("k", 60))
		sjs = append(sjs, fmt.Sprintf("%q: %q", k, strings.Repeat(q, (oracle.MaxSettingBytes-2)/2)))
	}
	var list, defaults []string
	for i := range maxPinList {
		list = append(list, long(q))
		defaults = append(defaults, fmt.Sprintf("d%02d%s", i, long(q)[3:]))
	}
	// A route is kept as the declaration writes it, so its escapes are paid
	// in the declaration's own bound too: each route carries as many '"' as
	// that bound leaves room for, and the rest of it is plain.
	declare := func(quotes int) (string, []CellPin) {
		body := oracle.MaxRouteBytes - len("builder=runner:")
		route := "builder=runner:" + strings.Repeat(q, quotes) + strings.Repeat("&", body-quotes)
		var cells []string
		var pins []CellPin
		for i := range maxCells {
			id := fmt.Sprintf("c-%02d-%s", i, strings.Repeat("x", maxIDBytes-5))
			cells = append(cells, fmt.Sprintf(`{"id": %q, "route": %q, "settings": {%s}}`, id, route, strings.Join(sjs, ", ")))
			pins = append(pins, CellPin{Cell: id, Role: RoleBuilder, Source: SourceOpenRouter, Runner: long(q), Model: long(q),
				HarnessVersion: long(q), HarnessDefault: defaults,
				ProviderPin: &ProviderPin{Order: list, Quantizations: list, DataCollection: long(q), CanonicalSlug: long(q)}})
		}
		doc := strings.Replace(goodDeclaration, `{"id": "host-big", "route": "host"},
    {"id": "or-cheap", "route": "builder=runner:opencode@openrouter/vendor/model", "settings": {"temperature": 0.6}}`, strings.Join(cells, ",\n"), 1)
		doc = strings.Replace(doc, `,
  "reviewers": [
    {"id": "local-rev", "route": "reviewer=runner:opencode@local/qwen"}
  ]`, "", 1)
		return doc, pins
	}
	base, _ := declare(0)
	quotes := min(oracle.MaxRouteBytes-len("builder=runner:"), (MaxDeclarationBytes-len(base))/maxCells)
	doc, pins := declare(quotes)
	if len(doc) > MaxDeclarationBytes || quotes < 1 {
		t.Fatalf("the test's declaration is %d bytes with %d quotes a route, past the declaration's own bound", len(doc), quotes)
	}
	in := inputs(t)
	in.Declaration, in.Cells = []byte(doc), pins
	in.Binary.Version, in.Binary.Revision = long(q), long(q)
	in.PromptSHA256, in.ToolDefinitionsSHA256 = map[string]string{}, map[string]string{}
	for i := range maxHashes {
		in.PromptSHA256[fmt.Sprintf("p%03d%s", i, long(q)[4:])] = hex64("1")
		in.ToolDefinitionsSHA256[fmt.Sprintf("t%03d%s", i, long(q)[4:])] = hex64("2")
	}
	ref := ContentRef{Name: strings.Repeat(q, maxRefName), SHA256: hex64("5"), Bytes: 1 << 40}
	in.HiddenTests, in.SeededMaterial = nil, nil
	for range maxRefs {
		in.HiddenTests = append(in.HiddenTests, ref)
		in.SeededMaterial = append(in.SeededMaterial, ref)
	}
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatalf("the largest record is refused: %v", err)
	}
	js, err := encodeRun(rec)
	if err != nil {
		t.Fatal(err)
	}
	if len(js) > lab.MaxStateBytes {
		t.Fatalf("the largest run record is %d bytes, past the %d its lab reads back", len(js), lab.MaxStateBytes)
	}
	t.Logf("the largest run record is %d bytes of %d (declaration %d bytes, %d quotes a route)", len(js), lab.MaxStateBytes, len(doc), quotes)
}

// noCap is the good declaration without its spending cap.
func noCap() []byte {
	return []byte(strings.Replace(goodDeclaration, `"spend_cap": {"amount": 25, "unit": "usd"},
`, "", 1))
}

// The machine is named by what bears on a run (OS, architecture, cores) and
// never by who or where: no host name, no home path (the PII domain).
func TestTheRunRecordCarriesNoHostnameOrHomePath(t *testing.T) {
	m := currentMachine()
	if m.OS == "" || m.Arch == "" || m.Cores < 1 {
		t.Fatalf("currentMachine = %+v", m)
	}
	in := inputs(t)
	in.Machine = m
	in.Machine.Cores = max(m.Cores, 5)
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	js, _ := json.Marshal(rec)
	if realHome == "" || realHome == os.Getenv("HOME") {
		t.Fatalf("the machine's own home %q was not read before HOME was repointed", realHome)
	}
	if leaks := leaked(js); len(leaks) > 0 {
		t.Errorf("the run record carries %q:\n%s", leaks, js)
	}
	// The check can fire: a record that did carry the home is caught.
	if leaks := leaked([]byte(`{"model":"` + realHome + `/models/x.gguf"}`)); len(leaks) == 0 {
		t.Fatal("the leak check misses the machine's own home path")
	}
}

// leaked is every one of the machine's own home, user and host name that js
// carries.
func leaked(js []byte) []string {
	var out []string
	for _, leak := range []string{realHost, realHome, realUser} {
		if len(leak) > 2 && strings.Contains(string(js), leak) {
			out = append(out, leak)
		}
	}
	return out
}

// A run is written into its lab through the procedure seam: under
// state/matrix/<run-id>/, once, at the lab's own pin.
func TestARunIsWrittenUnderItsLabOnceAtThePin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write("README.md", "root\n")
	r.Commit("root")
	t.Cleanup(lab.SetAbcdRootCommitForTest(strings.Repeat("e", 40)))
	m, err := lab.Mint(r.Root(), "does a cheaper model build this as well?", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lab.Preflight(r.Root(), m.ID); err != nil {
		t.Fatal(err)
	}
	in := inputs(t)
	in.Lab, in.RootSHA, in.FrozenCommit = m.ID, m.RootSHA, m.Pin
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(r.Root(), rec, in.Declaration); err != nil {
		t.Fatalf("writeRun: %v", err)
	}
	dir := abcdhome.Path(home, "lab", m.RootSHA, m.ID, "state", "matrix", rec.RunID)
	got, err := os.ReadFile(filepath.Join(dir, "declaration.json"))
	if err != nil || string(got) != goodDeclaration {
		t.Fatalf("declaration.json = %q, %v", got, err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "run.json"))
	if err != nil {
		t.Fatal(err)
	}
	var back RunRecord
	if err := json.Unmarshal(data, &back); err != nil || back.RunID != rec.RunID || back.FrozenCommit != m.Pin {
		t.Fatalf("run.json = %+v, %v", back, err)
	}
	if err := writeRun(r.Root(), rec, in.Declaration); !errors.Is(err, lab.ErrRefused) {
		t.Fatalf("a second write = %v; want a refusal, never an overwrite", err)
	}

	// A record naming no lab of this repository is refused before any path.
	bad := rec
	bad.Lab = "lab-../../x"
	if err := writeRun(r.Root(), bad, in.Declaration); !errors.Is(err, lab.ErrRefused) {
		t.Fatalf("writeRun into a malformed lab id = %v; want ErrRefused", err)
	}

	// A record whose frozen commit is not the lab's pin is not this lab's run.
	in.RunID = "mx-261023012950-ab13"
	in.FrozenCommit = strings.Repeat("f", 40)
	other, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(r.Root(), other, in.Declaration); err == nil || !strings.Contains(err.Error(), "pin") {
		t.Fatalf("writeRun at another commit = %v; want a refusal naming the pin", err)
	}
}

// The binary that drives a run is pinned through the lab's one hashing path:
// its sha256 and its build vintage, beside the version it reports.
func TestTheDrivingBinaryIsPinnedByItsHash(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(exe)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pin, err := pinBinary(f, "0.16.0")
	if err != nil {
		t.Fatalf("pinBinary: %v", err)
	}
	data, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	if pin.SHA256 != hex.EncodeToString(sum[:]) || pin.Version != "0.16.0" {
		t.Fatalf("pin = %+v; want the file's sha256", pin)
	}
	if _, err := pinBinary(strings.NewReader("not a binary"), "0.16.0"); err == nil {
		t.Fatal("a file with no build metadata was pinned")
	}
}

// matrixLab mints a lab of a throwaway repository and passes its preflight,
// and returns the repository's root, the home and the lab, with inputs for a
// run at the lab's pin.
func matrixLab(t *testing.T) (string, string, lab.Minted, runInputs) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	r := gittest.NewRepo(t)
	r.Write("README.md", "root\n")
	r.Commit("root")
	t.Cleanup(lab.SetAbcdRootCommitForTest(strings.Repeat("e", 40)))
	m, err := lab.Mint(r.Root(), "does a cheaper model build this as well?", "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lab.Preflight(r.Root(), m.ID); err != nil {
		t.Fatal(err)
	}
	in := inputs(t)
	in.Lab, in.RootSHA, in.FrozenCommit = m.ID, m.RootSHA, m.Pin
	return r.Root(), home, m, in
}

func runDir(home string, m lab.Minted, run string) string {
	return abcdhome.Path(home, "lab", m.RootSHA, m.ID, "state", "matrix", run)
}

// Where the lab pinned a work binary, the run's binary is that one; and the
// declaration written beside the record is the one the record keeps.
func TestARunIsRefusedAnotherBinaryOrAnotherDeclaration(t *testing.T) {
	root, home, m, in := matrixLab(t)
	pinned := hex64("9")
	pin := filepath.Join(abcdhome.Path(home, "lab", m.RootSHA, m.ID), "state", "work-binary.json")
	if err := os.WriteFile(pin, []byte(`{"sha256":"`+pinned+`","revision":"`+m.Pin+`","at":"2026-10-23T00:00:00Z"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(root, rec, in.Declaration); err == nil || !strings.Contains(err.Error(), "work binary") {
		t.Fatalf("writeRun with a binary not the lab's = %v; want a refusal naming the work binary", err)
	}
	in.Binary.SHA256 = pinned
	if rec, err = newRunRecord(in); err != nil {
		t.Fatal(err)
	}
	other := []byte(strings.Replace(goodDeclaration, "routing check", "another check", 1))
	if err := writeRun(root, rec, other); err == nil || !strings.Contains(err.Error(), "declaration") {
		t.Fatalf("writeRun with another declaration = %v; want a refusal naming the declaration", err)
	}
	if _, err := os.Stat(runDir(home, m, rec.RunID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused run left %s behind: %v", runDir(home, m, rec.RunID), err)
	}
	if err := writeRun(root, rec, in.Declaration); err != nil {
		t.Fatalf("writeRun with the lab's binary and the record's declaration: %v", err)
	}
}

// The record names every route and launches none: it carries no credential
// and no home path, whatever a pin or a route was given, so a record that would
// is refused whole, with nothing written.
func TestARunCarryingACredentialOrAHomePathIsRefusedUnwritten(t *testing.T) {
	for name, mutate := range map[string]func(*runInputs){
		"a token as a model":     func(in *runInputs) { remodel(in, 1, "ghp_"+testsecret.Synthetic(7, 36)) },
		"a home path in a model": func(in *runInputs) { remodel(in, 2, "file:/Users/alice/models/qwen.gguf") },
		"the home in a route": func(in *runInputs) {
			in.Declaration = []byte(strings.Replace(goodDeclaration, "reviewer=runner:opencode@local/qwen",
				"reviewer=runner:opencode@local"+os.Getenv("HOME")+"/models/qwen", 1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, home, m, in := matrixLab(t)
			mutate(&in)
			rec, err := newRunRecord(in)
			if err != nil {
				t.Fatalf("newRunRecord: %v", err)
			}
			if err := writeRun(root, rec, in.Declaration); err == nil {
				t.Fatal("writeRun wrote it")
			}
			if _, err := os.Stat(runDir(home, m, rec.RunID)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("a refused run left %s behind: %v", runDir(home, m, rec.RunID), err)
			}
		})
	}
}

// A run's two files are written together or not at all: a write that fails
// part-way leaves nothing, so the run id can be written again.
func TestARunWrittenPartWayLeavesNothing(t *testing.T) {
	root, home, m, in := matrixLab(t)
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	p, err := lab.OpenProcedure(root, m.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := p.WriteState(rec.RunID+"/run.json", []byte("{}\n")); err != nil {
		t.Fatal(err)
	}
	p.Close()
	if err := writeRun(root, rec, in.Declaration); !errors.Is(err, lab.ErrRefused) {
		t.Fatalf("writeRun over a written run.json = %v; want a refusal", err)
	}
	if _, err := os.Stat(filepath.Join(runDir(home, m, rec.RunID), "declaration.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a write that failed part-way left declaration.json behind: %v", err)
	}
}

// A version is screened as a version, never as an address: a four-part
// runtime, harness or binary version is written, while the token screen still
// covers those fields and an address in free text (a route) still refuses.
func TestAFourPartVersionIsNotAnAddress(t *testing.T) {
	root, home, m, in := matrixLab(t)
	in.Binary.Version = "1.10.2.3"
	in.Cells[1].HarnessVersion = "2.10.3.4"
	in.Cells[2].HarnessVersion = "2.10.3.4"
	in.Cells[2].LocalPin.RuntimeVersion = "0.10.1.1"
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(root, rec, in.Declaration); err != nil {
		t.Fatalf("writeRun with four-part versions: %v", err)
	}
	// The network screen reads the record line for line as written, with
	// only its versions and digests emptied.
	js, err := encodeRun(rec)
	if err != nil {
		t.Fatal(err)
	}
	vs, err := encodeRun(rec.withoutVersionsAndDigests())
	if err != nil {
		t.Fatal(err)
	}
	jl, vl := strings.Split(string(js), "\n"), strings.Split(string(vs), "\n")
	if len(jl) != len(vl) {
		t.Fatalf("the screened record has %d lines; the record has %d", len(vl), len(jl))
	}
	for i := range jl {
		if jl[i] != vl[i] && (strings.Contains(vl[i], "1.10.2.3") || !strings.HasSuffix(strings.TrimRight(vl[i], ","), `""`)) {
			t.Errorf("line %d differs other than by an emptied value: %q vs %q", i+1, jl[i], vl[i])
		}
	}
	if again, _ := encodeRun(rec); string(again) != string(js) {
		t.Fatal("screening changed the record")
	}
	if _, err := os.Stat(filepath.Join(runDir(home, m, rec.RunID), "run.json")); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*runInputs){
		"a token as a version":        func(in *runInputs) { in.Cells[2].LocalPin.RuntimeVersion = "ghp_" + testsecret.Synthetic(8, 36) },
		"a token as a binary version": func(in *runInputs) { in.Binary.Version = "ghp_" + testsecret.Synthetic(9, 36) },
		"an address in a route": func(in *runInputs) {
			in.Declaration = []byte(strings.Replace(goodDeclaration, "reviewer=runner:opencode@local/qwen",
				"reviewer=runner:opencode@local/qwen@81.2.69.142", 1))
		},
	} {
		t.Run(name, func(t *testing.T) {
			root, home, m, in := matrixLab(t)
			mutate(&in)
			rec, err := newRunRecord(in)
			if err != nil {
				t.Fatalf("newRunRecord: %v", err)
			}
			if err := writeRun(root, rec, in.Declaration); err == nil {
				t.Fatal("writeRun wrote it")
			}
			if _, err := os.Stat(runDir(home, m, rec.RunID)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("a refused run left %s behind: %v", runDir(home, m, rec.RunID), err)
			}
		})
	}
}

// fakeAugmenter is a repository's configured augmenter whose Available
// answers err and that flags every occurrence of flag.
type fakeAugmenter struct {
	err  error
	flag string
	// kind is the finding's kind; "" is gitleaks' generic one.
	kind string
}

func (a fakeAugmenter) Available() error { return a.err }

func (a fakeAugmenter) Scan(text, _ string) []scanner.Finding {
	if a.flag == "" {
		return nil
	}
	var out []scanner.Finding
	for i, line := range strings.Split(text, "\n") {
		for from := 0; ; {
			at := strings.Index(line[from:], a.flag)
			if at < 0 {
				break
			}
			out = append(out, scanner.Finding{Line: i + 1, Column: from + at + 1, Kind: cmp.Or(a.kind, "gitleaks:generic-api-key"), Matched: a.flag})
			from += at + len(a.flag)
		}
	}
	return out
}

// What the repository's configured augmenter flags refuses the run, as it
// fails the launch bundle's scan, whatever kind it names.
func TestARunTheAugmenterFlagsIsRefusedUnwritten(t *testing.T) {
	const flagged = "zq7flaggedvalue9"
	root, home, m, in := matrixLab(t)
	t.Cleanup(scanner.SetDefaultAugmenter(func(string) scanner.Augmenter { return fakeAugmenter{flag: flagged} }))
	remodel(&in, 1, "vendor/"+flagged)
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(root, rec, in.Declaration); err == nil || !strings.Contains(err.Error(), "gitleaks:generic-api-key") {
		t.Fatalf("writeRun = %v; want a refusal naming the augmenter's finding", err)
	}
	if _, err := os.Stat(runDir(home, m, rec.RunID)); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("a refused run left %s behind: %v", runDir(home, m, rec.RunID), err)
	}
}

// An augmenter's kind is the repository's own rule id, so the refusal names
// it with a control or direction character masked, never raw.
func TestARefusalNamesAnAugmentersKindMasked(t *testing.T) {
	const flagged = "zq7flaggedvalue9"
	root, _, _, in := matrixLab(t)
	t.Cleanup(scanner.SetDefaultAugmenter(func(string) scanner.Augmenter {
		return fakeAugmenter{flag: flagged, kind: "repo:rule\x1b]0;x\a\u202e"}
	}))
	remodel(&in, 1, "vendor/"+flagged)
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	err = writeRun(root, rec, in.Declaration)
	if err == nil || !strings.Contains(err.Error(), "repo:rule") {
		t.Fatalf("writeRun = %v; want a refusal naming the augmenter's kind", err)
	}
	if strings.ContainsAny(err.Error(), "\x1b\a\u202e") {
		t.Errorf("the refusal carries the kind's control or direction character raw: %q", err)
	}
}

// The role a pin names is echoed quoted, so a control character in it is
// shown escaped.
func TestAPinsRoleIsEchoedQuoted(t *testing.T) {
	in := inputs(t)
	in.Cells[2].Role = "reviewer\x1b[2J"
	_, err := newRunRecord(in)
	if err == nil || !strings.Contains(err.Error(), `"reviewer\x1b[2J"`) || strings.ContainsRune(err.Error(), 0x1b) {
		t.Fatalf("newRunRecord = %v; want the role echoed quoted and escaped", err)
	}
}

// The run is screened by the launch bundle's scanner on the launch bundle's
// terms: an augmenter the repository configured that is not installed is a
// coverage gap, and a scanner that runs degraded is no screen; either refuses
// the run, with nothing written.
func TestARunTheScannerCannotFullyScreenIsRefusedUnwritten(t *testing.T) {
	for name, err := range map[string]error{
		"augmenter not installed": fmt.Errorf("gitleaks: %w", scanner.ErrAugmenterNotFound),
		"augmenter failing":       errors.New("gitleaks exited 2"),
	} {
		t.Run(name, func(t *testing.T) {
			root, home, m, in := matrixLab(t)
			t.Cleanup(scanner.SetDefaultAugmenter(func(string) scanner.Augmenter { return fakeAugmenter{err: err} }))
			rec, rerr := newRunRecord(in)
			if rerr != nil {
				t.Fatal(rerr)
			}
			if werr := writeRun(root, rec, in.Declaration); werr == nil || !strings.Contains(werr.Error(), "secret scanner") {
				t.Fatalf("writeRun = %v; want a refusal naming the secret scanner", werr)
			}
			if _, serr := os.Stat(runDir(home, m, rec.RunID)); !errors.Is(serr, os.ErrNotExist) {
				t.Fatalf("a refused run left %s behind: %v", runDir(home, m, rec.RunID), serr)
			}
		})
	}
	// The same lab with the scanner whole writes the run.
	root, _, _, in := matrixLab(t)
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(root, rec, in.Declaration); err != nil {
		t.Fatalf("writeRun with the scanner whole: %v", err)
	}
}

// An unpinned field is named by abcd from a validated cell id
// (cell.<id>.<field>), so a cell id such as local-rev must not read as the
// host name cell.local to the network screen: the record is written. The
// credential screen still reads the names.
func TestAnUnpinnedFieldNamedForACellIsNotReadAsAHostName(t *testing.T) {
	root, _, _, in := matrixLab(t)
	in.Cells[2].HarnessVersion = "" // local-rev: its harness version is then listed as unpinned
	rec, err := newRunRecord(in)
	if err != nil {
		t.Fatal(err)
	}
	if err := writeRun(root, rec, in.Declaration); err != nil {
		t.Fatalf("writeRun with cell.local-rev.harness_version unpinned: %v", err)
	}
}
