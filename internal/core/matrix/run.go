package matrix

// run.go is the run's record: what a second person needs to start the same run
// on a machine with the same routes and see where it drifted (decisions 4, 15
// and the technical criterion naming the frozen commit, the binary's digest
// and version, the prompt and tool-definition hashes, the render template, the
// machine, the concurrency and the load cap). A model run is re-runnable, not
// reproducible, so the record pins what can be pinned and lists every field it
// could not in Unpinned, with oracle's four reasons (itd-166), never leaving
// one out.
//
// The record names every route but launches none (scope condition 1): it
// carries no credential, no route file's content, no host name and no home
// path.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strings"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// RunSchema is the run record's schema version.
const RunSchema = 1

// The run-level fields a run lists when it cannot pin them. Their reasons are
// oracle's; a run adds fields, never reasons.
const (
	RunPromptSHA256          oracle.Field = "run.prompt_sha256"
	RunToolDefinitionsSHA256 oracle.Field = "run.tool_definitions_sha256"
	RunRenderTemplate        oracle.Field = "run.render_template"
	RunBinaryRevision        oracle.Field = "run.binary.revision"
	// The machine's processor model and memory need a per-OS exec to read,
	// so a run lists them rather than reading them.
	RunMachineCPU    oracle.Field = "run.machine.cpu_model"
	RunMachineMemory oracle.Field = "run.machine.memory"
)

// cellField is a cell's own field, as its unpinned entry names it.
func cellField(cell, f string) oracle.Field { return oracle.Field("cell." + cell + "." + f) }

// ContentRef is an artefact a run keeps by content (decision 4: the hidden
// tests, the seeded material and the render template), named and hashed.
type ContentRef struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}

// contentOf is data's reference.
func contentOf(name string, data []byte) ContentRef {
	sum := sha256.Sum256(data)
	return ContentRef{Name: name, SHA256: hex.EncodeToString(sum[:]), Bytes: int64(len(data))}
}

// BinaryPin is the abcd binary that drives a run: its sha256, the version it
// reports and its build vintage. Known is false for a binary whose vintage
// cannot be read or that was built from a modified tree.
type BinaryPin struct {
	SHA256   string `json:"sha256"`
	Version  string `json:"version"`
	Revision string `json:"revision"`
	Known    bool   `json:"known"`
}

// pinBinary pins the binary f holds through the lab's one hashing path.
func pinBinary(f interface {
	io.Reader
	io.ReaderAt
}, version string) (BinaryPin, error) {
	cur, sha, err := lab.BinaryIdentity(f)
	if err != nil {
		return BinaryPin{}, fmt.Errorf("cannot pin the binary: %w", err)
	}
	return BinaryPin{SHA256: sha, Version: version, Revision: cur.Revision, Known: cur.Known}, nil
}

// Machine is the machine a run ran on, named by what bears on the run and
// never by who or where: no host name, no user, no path.
type Machine struct {
	OS    string `json:"os"`
	Arch  string `json:"arch"`
	Cores int    `json:"cores"`
	// CoresFallback is true when the online core count could not be read and
	// Cores is the runtime's count instead.
	CoresFallback bool `json:"cores_fallback"`
}

// currentMachine reads this machine.
func currentMachine() Machine {
	m := Machine{OS: runtime.GOOS, Arch: runtime.GOARCH}
	if snap, err := machineload.Read(); err == nil && snap.Cores > 0 {
		m.Cores, m.CoresFallback = snap.Cores, snap.CoresFallback
	} else {
		m.Cores, m.CoresFallback = runtime.NumCPU(), true
	}
	return m
}

// Source is where a cell's model comes from (decision 2). Model and harness
// travel together by source (scope condition 4): host models in the host's
// harness, every other model in a command-line runner.
type Source string

const (
	SourceHost       Source = "host"
	SourceOpenRouter Source = "openrouter"
	SourceLocal      Source = "local"
	// SourceProvider is any other provider connection.
	SourceProvider Source = "provider"
)

// Metered reports whether a cell from this source spends money a cap bounds
// (decision 17): a host or local cell does not.
func (s Source) Metered() bool { return s == SourceOpenRouter || s == SourceProvider }

func (s Source) valid() bool {
	return s == SourceHost || s == SourceOpenRouter || s == SourceLocal || s == SourceProvider
}

// ProviderPin is an OpenRouter cell's provider pin (decision 6: fallbacks off,
// so an outage is retried on the same pin).
//
// Typed here until abcd-f1's oracle.Pin (feat/local-model-pin-r2) reaches the
// default branch; its fields and tags are that type's, so the reconciliation
// replaces this declaration with an alias and changes no record.
type ProviderPin struct {
	Order             []string `json:"order"`
	AllowFallbacks    bool     `json:"allow_fallbacks"`
	RequireParameters bool     `json:"require_parameters"`
	DataCollection    string   `json:"data_collection"`
	Quantizations     []string `json:"quantizations"`
	CanonicalSlug     string   `json:"canonical_slug"`
}

// LocalPin is a local cell's pin, down to its weight file (scope condition 3).
//
// Typed here until abcd-f1's oracle.LocalPin (feat/local-model-pin-r2) reaches
// the default branch; its fields and tags are that type's, so the
// reconciliation replaces this declaration with an alias.
type LocalPin struct {
	Runtime        string `json:"runtime"`
	RuntimeVersion string `json:"runtime_version"`
	WeightsSHA256  string `json:"weights_sha256"`
	WeightsFile    string `json:"weights_file,omitempty"`
	Quantisation   string `json:"quantisation"`
	ContextLength  int    `json:"context_length"`
	KVCacheType    string `json:"kv_cache_type"`
}

// CellPin is one cell as the run pinned it: the declared cell's route and
// settings, the source and runner it resolved to, the model, and the pin its
// source allows.
type CellPin struct {
	Cell     string          `json:"cell"`
	Role     Role            `json:"role"`
	Source   Source          `json:"source"`
	Route    string          `json:"route"`
	Runner   string          `json:"runner"`
	Model    string          `json:"model"`
	Settings oracle.Settings `json:"settings"`
	// HarnessDefault names each sampling setting the cell cannot set, which
	// the harness's own default decides (decision 5), at HarnessVersion.
	HarnessDefault []string     `json:"harness_default"`
	HarnessVersion string       `json:"harness_version"`
	ProviderPin    *ProviderPin `json:"provider_pin"`
	LocalPin       *LocalPin    `json:"local_pin"`
}

// Where a run's spending cap came from (decision 17): the run's own
// declaration, which overrides the person's machine-wide configuration. A
// repository never sets a person's cap.
const (
	CapFromRun     = "run"
	CapFromMachine = "machine"
)

// RunRecord is a run's record.
type RunRecord struct {
	SchemaVersion int    `json:"schema_version"`
	RunID         string `json:"run_id"`
	Lab           string `json:"lab"`
	RootSHA       string `json:"root_sha"`
	// FrozenCommit is the commit every copy starts from: the lab's pin.
	FrozenCommit string     `json:"frozen_commit"`
	Binary       BinaryPin  `json:"binary"`
	Declaration  ContentRef `json:"declaration"`
	// PromptSHA256 is each role's prompt hash, and ToolDefinitionsSHA256
	// each runner's tool definitions' hash (prompts are pinned by hash until
	// itd-14 versions them).
	PromptSHA256          map[string]string `json:"prompt_sha256"`
	ToolDefinitionsSHA256 map[string]string `json:"tool_definitions_sha256"`
	RenderTemplate        *ContentRef       `json:"render_template"`
	HiddenTests           []ContentRef      `json:"hidden_tests"`
	SeededMaterial        []ContentRef      `json:"seeded_material"`
	Machine               Machine           `json:"machine"`
	Concurrency           int               `json:"concurrency"`
	LoadCap               int               `json:"load_cap"`
	SpendCap              *Money            `json:"spend_cap"`
	SpendCapOrigin        string            `json:"spend_cap_origin"`
	Cells                 []CellPin         `json:"cells"`
	Unpinned              []oracle.Unpinned `json:"unpinned"`
}

// runInputs is what a run's record is built from. Declaration is the person's
// file as read: the record keeps it by content and reads the matrix from it,
// so the two cannot disagree.
type runInputs struct {
	RunID, Lab, RootSHA, FrozenCommit string
	Binary                            BinaryPin
	Declaration                       []byte
	// MachineSpendCap is the cap the person's machine-wide configuration
	// names, nil when it names none.
	MachineSpendCap       *Money
	PromptSHA256          map[string]string
	ToolDefinitionsSHA256 map[string]string
	RenderTemplate        *ContentRef
	HiddenTests           []ContentRef
	SeededMaterial        []ContentRef
	Machine               Machine
	Cells                 []CellPin
}

// runIDRe is a run's id: lab-local, the UTC start time and four hex digits.
var runIDRe = regexp.MustCompile(`^mx-[0-9]{12}-[0-9a-f]{4}$`)

var sha256Re = regexp.MustCompile(`^[0-9a-f]{64}$`)

const maxRefName = 256

// newRunRecord builds and checks a run's record. It refuses what a re-run could
// not stand on (an unpinned commit, a binary with no digest, a declaration that
// does not parse, a cell with no pin, a metered cell with no cap) and lists
// every hash it was not given as unpinned.
func newRunRecord(in runInputs) (RunRecord, error) {
	if !runIDRe.MatchString(in.RunID) {
		return RunRecord{}, fmt.Errorf("%q is not a run id (mx-<yymmddHHMMSS>-<4 hex>)", clip(in.RunID))
	}
	// The lab id's shape is held where the run is written: lab.OpenProcedure
	// refuses one that is not a lab's before it touches a path.
	if err := text("lab", in.Lab); err != nil {
		return RunRecord{}, err
	}
	if !gitutil.IsFullSHA(in.RootSHA) || !gitutil.IsFullSHA(in.FrozenCommit) {
		return RunRecord{}, errors.New("the root commit and the frozen commit are each a full object name")
	}
	if !sha256Re.MatchString(in.Binary.SHA256) || strings.TrimSpace(in.Binary.Version) == "" {
		return RunRecord{}, errors.New("the binary's sha256 and version are both named; a run is never recorded against an unpinned binary")
	}
	if len(in.Declaration) == 0 {
		return RunRecord{}, errors.New("the run has no declaration")
	}
	d, err := parseDeclaration(in.Declaration)
	if err != nil {
		return RunRecord{}, err
	}
	var un oracle.Unpin
	rec := RunRecord{
		SchemaVersion: RunSchema, RunID: in.RunID, Lab: in.Lab, RootSHA: in.RootSHA,
		FrozenCommit: in.FrozenCommit, Binary: in.Binary,
		Declaration: contentOf("declaration.json", in.Declaration),
		Concurrency: d.Concurrency, LoadCap: d.LoadCap,
	}
	un.Unless(in.Binary.Known && gitutil.IsFullSHA(in.Binary.Revision), RunBinaryRevision, oracle.NotReported)

	if rec.PromptSHA256, err = hashes("prompt_sha256", in.PromptSHA256); err != nil {
		return RunRecord{}, err
	}
	un.Unless(len(rec.PromptSHA256) > 0, RunPromptSHA256, oracle.NotReported)
	if rec.ToolDefinitionsSHA256, err = hashes("tool_definitions_sha256", in.ToolDefinitionsSHA256); err != nil {
		return RunRecord{}, err
	}
	un.Unless(len(rec.ToolDefinitionsSHA256) > 0, RunToolDefinitionsSHA256, oracle.NotReported)
	if in.RenderTemplate != nil {
		if err := in.RenderTemplate.validate(); err != nil {
			return RunRecord{}, fmt.Errorf("render_template: %w", err)
		}
		t := *in.RenderTemplate
		rec.RenderTemplate = &t
	}
	un.Unless(rec.RenderTemplate != nil, RunRenderTemplate, oracle.NotConfigured)
	if rec.HiddenTests, err = refs("hidden_tests", in.HiddenTests); err != nil {
		return RunRecord{}, err
	}
	if rec.SeededMaterial, err = refs("seeded_material", in.SeededMaterial); err != nil {
		return RunRecord{}, err
	}

	m := in.Machine
	if m.OS == "" || m.Arch == "" || m.Cores < 1 || len(m.OS) > 32 || len(m.Arch) > 32 {
		return RunRecord{}, fmt.Errorf("the machine is named by its OS, architecture and core count; have %+v", m)
	}
	if d.LoadCap >= m.Cores {
		return RunRecord{}, fmt.Errorf("the load cap of %d is not below the machine's %d cores", d.LoadCap, m.Cores)
	}
	rec.Machine = m
	un.Add(RunMachineCPU, oracle.NotReported)
	un.Add(RunMachineMemory, oracle.NotReported)

	if rec.Cells, err = pinCells(d, in.Cells, &un); err != nil {
		return RunRecord{}, err
	}
	switch {
	case d.SpendCap != nil:
		c := *d.SpendCap
		rec.SpendCap, rec.SpendCapOrigin = &c, CapFromRun
	case in.MachineSpendCap != nil:
		if err := in.MachineSpendCap.validate(); err != nil {
			return RunRecord{}, fmt.Errorf("the machine's spending cap: %w", err)
		}
		c := *in.MachineSpendCap
		rec.SpendCap, rec.SpendCapOrigin = &c, CapFromMachine
	}
	if rec.SpendCap == nil {
		for _, c := range rec.Cells {
			if c.Source.Metered() {
				return RunRecord{}, fmt.Errorf("the cell %s is metered (%s) and the run names no spending cap; name one in the declaration or your own configuration", c.Cell, c.Source)
			}
		}
	}
	rec.Unpinned = un.List()
	return rec, nil
}

// pinCells checks the cells' pins against the declaration: one pin a declared
// cell, in its role, each with what its source allows.
func pinCells(d Declaration, pins []CellPin, un *oracle.Unpin) ([]CellPin, error) {
	declared := d.cells()
	byID := map[string]CellPin{}
	for _, p := range pins {
		if _, dup := byID[p.Cell]; dup {
			return nil, fmt.Errorf("the cell %s is pinned twice", clip(p.Cell))
		}
		byID[p.Cell] = p
	}
	if len(byID) != len(declared) {
		return nil, fmt.Errorf("the run pins %d cells and the declaration names %d", len(byID), len(declared))
	}
	out := make([]CellPin, 0, len(declared))
	for _, dc := range declared {
		p, ok := byID[dc.ID]
		if !ok {
			return nil, fmt.Errorf("the declared cell %s has no pin", dc.ID)
		}
		if p.Role != dc.role {
			return nil, fmt.Errorf("the cell %s is declared a %s and pinned a %s", dc.ID, dc.role, p.Role)
		}
		if !p.Source.valid() {
			return nil, fmt.Errorf("the cell %s names the source %q; a source is host, openrouter, local or provider", dc.ID, clip(string(p.Source)))
		}
		if err := text("runner", p.Runner); err != nil {
			return nil, fmt.Errorf("cell %s: %w", dc.ID, err)
		}
		p.Route = dc.Route
		p.Settings = dc.Settings
		if p.Settings == nil {
			p.Settings = oracle.Settings{}
		}
		switch {
		case p.Model != "":
			if err := text("model", p.Model); err != nil {
				return nil, fmt.Errorf("cell %s: %w", dc.ID, err)
			}
		case p.Source == SourceHost:
			// The host names its model only in its own receipt, after the
			// call (scope condition 3).
			un.Add(cellField(dc.ID, "model"), oracle.NotReported)
		default:
			return nil, fmt.Errorf("the cell %s names no model; only a host cell leaves it to the host", dc.ID)
		}
		if p.HarnessVersion != "" {
			if err := text("harness_version", p.HarnessVersion); err != nil {
				return nil, fmt.Errorf("cell %s: %w", dc.ID, err)
			}
		} else {
			un.Add(cellField(dc.ID, "harness_version"), oracle.NotReported)
		}
		p.HarnessDefault = slices.Clone(p.HarnessDefault)
		if p.HarnessDefault == nil {
			p.HarnessDefault = []string{}
		}
		sort.Strings(p.HarnessDefault)
		for _, s := range p.HarnessDefault {
			if _, set := p.Settings[s]; set || text("harness_default", s) != nil {
				return nil, fmt.Errorf("cell %s: the setting %q is both set and left to the harness, or is not a setting name", dc.ID, clip(s))
			}
			un.Add(cellField(dc.ID, "settings."+s), oracle.NotConfigured)
		}
		if p.ProviderPin != nil && p.Source != SourceOpenRouter || p.LocalPin != nil && p.Source != SourceLocal {
			return nil, fmt.Errorf("the cell %s carries a pin its source (%s) does not take", dc.ID, p.Source)
		}
		switch p.Source {
		case SourceOpenRouter:
			if p.ProviderPin == nil {
				return nil, fmt.Errorf("the OpenRouter cell %s has no provider pin; without one OpenRouter falls back across providers, which changes the cell", dc.ID)
			}
			if p.ProviderPin.AllowFallbacks {
				return nil, fmt.Errorf("the OpenRouter cell %s allows fallbacks; a matrix cell keeps them off and retries an outage on the same pin", dc.ID)
			}
		case SourceLocal:
			if p.LocalPin == nil {
				// The local pin and probe are iss-2610161734428291's.
				un.Add(cellField(dc.ID, "local_pin"), oracle.NotConfigured)
			} else if !sha256Re.MatchString(p.LocalPin.WeightsSHA256) {
				return nil, fmt.Errorf("the local cell %s's pin names no weight file sha256", dc.ID)
			}
		}
		out = append(out, p)
	}
	return out, nil
}

// hashes checks a map of names to sha256 digests, never nil.
func hashes(field string, in map[string]string) (map[string]string, error) {
	out := make(map[string]string, len(in))
	for k, v := range in {
		if text(field, k) != nil || !sha256Re.MatchString(v) {
			return nil, fmt.Errorf("%s: %q => %q is not a name and a sha256", field, clip(k), clip(v))
		}
		out[k] = v
	}
	return out, nil
}

// refs checks a list of content references, never nil.
func refs(field string, in []ContentRef) ([]ContentRef, error) {
	out := make([]ContentRef, 0, len(in))
	for i, r := range in {
		if err := r.validate(); err != nil {
			return nil, fmt.Errorf("%s[%d]: %w", field, i, err)
		}
		out = append(out, r)
	}
	return out, nil
}

func (r ContentRef) validate() error {
	if err := text("name", r.Name); err != nil || len(r.Name) > maxRefName {
		return fmt.Errorf("a kept artefact is named in at most %d bytes", maxRefName)
	}
	if !sha256Re.MatchString(r.SHA256) || r.Bytes < 0 {
		return fmt.Errorf("%s is not kept by a sha256 and a size", clip(r.Name))
	}
	return nil
}

// text holds a recorded name to one non-empty, bounded token with no control
// character or space.
func text(field, s string) error {
	if s == "" || len(s) > oracle.MaxModelBytes || strings.IndexFunc(s, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return fmt.Errorf("%s %q is empty, longer than %d bytes or carries a space or control character", field, clip(s), oracle.MaxModelBytes)
	}
	return nil
}

// writeRun writes a run's record and the declaration it was read from into its
// lab, under state/matrix/<run-id>/, once. The run is the lab's: its frozen
// commit is the lab's pin, and where the lab pinned a work binary, the run's
// binary is that one.
func writeRun(repoRoot string, rec RunRecord, declaration []byte) error {
	p, err := lab.OpenProcedure(repoRoot, rec.Lab)
	if err != nil {
		return err
	}
	defer p.Close()
	e := p.Entry()
	if e.RootSHA != rec.RootSHA || e.Pin != rec.FrozenCommit {
		return fmt.Errorf("the run's frozen commit %s is not the lab %s's pin %s", rec.FrozenCommit, e.ID, e.Pin)
	}
	if sha, _, ok := p.WorkBinary(); ok && sha != rec.Binary.SHA256 {
		return fmt.Errorf("the run's binary %s is not the lab's pinned work binary %s", rec.Binary.SHA256, sha)
	}
	if contentOf("declaration.json", declaration) != rec.Declaration {
		return errors.New("the declaration given is not the one the run record keeps")
	}
	js, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		return err
	}
	if err := p.WriteState(rec.RunID+"/declaration.json", declaration); err != nil {
		return err
	}
	return p.WriteState(rec.RunID+"/run.json", append(js, '\n'))
}
