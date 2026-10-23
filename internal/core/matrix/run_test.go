package matrix

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gittest"
)

// TestMain points HOME at a throwaway directory, so no test here reaches the
// operator's real lab store.
func TestMain(m *testing.M) {
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
			{Cell: "or-cheap", Role: RoleBuilder, Source: SourceOpenRouter, Runner: "opencode", Model: "vendor/model",
				HarnessVersion: "1.2.3", HarnessDefault: []string{"top_p"},
				ProviderPin: &ProviderPin{Order: []string{"provider-a"}, AllowFallbacks: false}},
			{Cell: "local-rev", Role: RoleReviewer, Source: SourceLocal, Runner: "opencode", Model: "qwen", HarnessVersion: "1.2.3",
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
		rec.Concurrency != 2 || rec.LoadCap != 4 || rec.SpendCap == nil || rec.SpendCap.Amount != 25 ||
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
		"run id":               func(in *runInputs) { in.RunID = "mx-1" },
		"no lab":               func(in *runInputs) { in.Lab = "" },
		"frozen commit":        func(in *runInputs) { in.FrozenCommit = "HEAD" },
		"binary digest":        func(in *runInputs) { in.Binary.SHA256 = "" },
		"binary version":       func(in *runInputs) { in.Binary.Version = "" },
		"prompt hash shape":    func(in *runInputs) { in.PromptSHA256["builder"] = "not-a-hash" },
		"machine cores":        func(in *runInputs) { in.Machine.Cores = 0 },
		"load cap below cores": func(in *runInputs) { in.Machine.Cores = 4 },
		"metered cell no cap":  func(in *runInputs) { in.Declaration = noCap() },
		"machine cap unit":     func(in *runInputs) { in.Declaration = noCap(); in.MachineSpendCap = &Money{Amount: 5, Unit: "credits"} },
		"missing cell pin":     func(in *runInputs) { in.Cells = in.Cells[:2] },
		"unknown cell pin":     func(in *runInputs) { in.Cells[0].Cell = "nobody" },
		"wrong role":           func(in *runInputs) { in.Cells[2].Role = RoleBuilder },
		"unknown source":       func(in *runInputs) { in.Cells[0].Source = "cloud" },
		"openrouter no pin":    func(in *runInputs) { in.Cells[1].ProviderPin = nil },
		"openrouter fallback":  func(in *runInputs) { in.Cells[1].ProviderPin.AllowFallbacks = true },
		"pin on wrong source":  func(in *runInputs) { in.Cells[0].LocalPin = &LocalPin{} },
		"model missing":        func(in *runInputs) { in.Cells[1].Model = "" },
		"setting both ways":    func(in *runInputs) { in.Cells[1].HarnessDefault = []string{"temperature"} },
		"content ref":          func(in *runInputs) { in.HiddenTests[0].SHA256 = "x" },
		"no declaration":       func(in *runInputs) { in.Declaration = nil },
		"declaration refused":  func(in *runInputs) { in.Declaration = []byte(`{}`) },
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
	host, _ := os.Hostname()
	home, _ := os.UserHomeDir()
	for _, leak := range []string{host, home, os.Getenv("USER")} {
		if len(leak) > 2 && strings.Contains(string(js), leak) {
			t.Errorf("the run record carries %q:\n%s", leak, js)
		}
	}
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
