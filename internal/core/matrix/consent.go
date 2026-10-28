package matrix

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// ErrNotConsented is a run asked to start with no approval: nothing starts and
// nothing is written until the person has seen the plan and said yes to it
// (acceptance criterion 1).
var ErrNotConsented = errors.New("the run has no consent: show the person the plan, and start it only with the digest of the plan they said yes to")

// Starter starts a consented run. It is the seam the run's trial mode fills
// (spec step 3); consent calls it once, after the run's record is written.
type Starter interface {
	Start(Plan, RunRecord) error
}

// ConsentInputs is what a consented run's record is written from, beside the
// plan: the run's id, the repository's root commit, the work binary, the
// declaration's bytes and the hashes and machine the record names.
type ConsentInputs struct {
	RunID                 string
	RootSHA               string
	Binary                BinaryPin
	Declaration           []byte
	PromptSHA256          map[string]string
	ToolDefinitionsSHA256 map[string]string
	RenderTemplate        *ContentRef
	HiddenTests           []ContentRef
	SeededMaterial        []ContentRef
	Machine               Machine
}

// consent starts the run p describes, once the person has said yes to exactly
// that plan: approve must be its digest. An empty approval is ErrNotConsented;
// an approval naming another plan, a plan whose content no longer matches its
// own digest, and a declaration other than the one the plan was built from are
// refused. In every refusal nothing is written and nothing starts. On a match
// the run's record and declaration are written to the lab, then s starts the
// run, once.
func consent(repoRoot string, p Plan, approve string, in ConsentInputs, s Starter) (RunRecord, error) {
	if approve == "" {
		return RunRecord{}, ErrNotConsented
	}
	if approve != p.Digest {
		return RunRecord{}, fmt.Errorf("the approval %s is not the plan's digest %s: it names another plan; show the person this one and start it only with its own digest", clip(approve), p.Digest)
	}
	if got, err := p.digest(); err != nil || got != p.Digest {
		return RunRecord{}, fmt.Errorf("the plan no longer matches the digest it was shown with; build and show it again")
	}
	if contentOf("declaration.json", in.Declaration) != p.Declaration {
		return RunRecord{}, errors.New("the declaration is not the one the plan was built from; build and show the plan again")
	}
	if s == nil {
		return RunRecord{}, errors.New("no starter was given, so a consented run could not start; nothing was written")
	}
	pins := make([]CellPin, 0, len(p.Cells))
	for _, c := range p.Cells {
		cp := CellPin{Cell: c.Cell, Role: c.Role, Source: c.Source, Runner: c.Runner, Model: c.Model,
			ProviderPin: c.ProviderPin, LocalPin: c.LocalPin}
		if c.Source == SourceHost {
			// The host names its model only in its own receipt, after the call.
			cp.Model = ""
			if cp.Runner == "" {
				cp.Runner = "host"
			}
		}
		pins = append(pins, cp)
	}
	var machine *Money
	if p.SpendCapOrigin == CapFromMachine {
		machine = p.SpendCap
	}
	rec, err := newRunRecord(runInputs{
		RunID: in.RunID, Lab: p.Lab, RootSHA: in.RootSHA, FrozenCommit: p.FrozenCommit,
		Binary: in.Binary, Declaration: in.Declaration, MachineSpendCap: machine,
		PromptSHA256: in.PromptSHA256, ToolDefinitionsSHA256: in.ToolDefinitionsSHA256,
		RenderTemplate: in.RenderTemplate, HiddenTests: in.HiddenTests, SeededMaterial: in.SeededMaterial,
		Machine: in.Machine, Cells: pins,
	})
	if err != nil {
		return RunRecord{}, err
	}
	if !sameCap(rec.SpendCap, p.SpendCap) || rec.SpendCapOrigin != p.SpendCapOrigin {
		return RunRecord{}, errors.New("the run's record would spend under another cap than the plan shows; nothing was written")
	}
	if err := writeRun(repoRoot, rec, in.Declaration); err != nil {
		return RunRecord{}, err
	}
	if err := s.Start(p, rec); err != nil {
		return rec, fmt.Errorf("the run's record is written, and the run did not start: %w", err)
	}
	return rec, nil
}

// sameCap reports whether two caps are the same figure, both absent included.
func sameCap(a, b *Money) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

// loopRole is the build loop's role a matrix cell plays: the runner
// configuration knows the loop's roles, not the matrix's.
var loopRole = map[Role]string{RoleBuilder: "implementer", RoleReviewer: "ruthless-reviewer"}

// PlanForLab builds the plan for a run of the declaration in the lab labID of
// the repository at repoRoot, on a machine of cores cores: the lab's pin is the
// frozen commit, the provider blocks and the spending cap are read from the
// layered configuration, and each cell's route is checked against a freshly
// loaded runner configuration, so building the plan changes none. The notes
// are the configuration reader's, for the front door to print.
func PlanForLab(repoRoot, labID string, declaration []byte, cores int) (Plan, []string, error) {
	p, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return Plan{}, nil, err
	}
	entry := p.Entry()
	p.Close()
	roots, notes := layered.RootsFor(repoRoot)
	d, err := parseDeclaration(declaration)
	if err != nil {
		return Plan{}, notes, err
	}
	api, err := oracle.LoadAPI(roots)
	if err != nil {
		return Plan{}, notes, err
	}
	cap, err := resolveCap(roots, d.SpendCap)
	if err != nil {
		return Plan{}, notes, err
	}
	check := func(fr oracle.FlagRoute) error {
		rc, err := runner.Load(roots)
		if err != nil {
			return err
		}
		// A matrix cell's role is checked as the loop role it plays: a
		// builder is the implementer, a reviewer the ruthless reviewer.
		fr.Agent = loopRole[Role(fr.Agent)]
		return rc.Override([]oracle.FlagRoute{fr})
	}
	plan, err := buildPlan(PlanInputs{Lab: entry.ID, FrozenCommit: entry.Pin, Declaration: declaration,
		API: api, CheckRoute: check, Cap: cap, Cores: cores})
	return plan, notes, err
}

// StartRun builds the plan again and starts the run with approve as the
// person's consent: the digest of the plan they were shown. The record pins
// the binary at executable and this machine; prompt, tool-definition and
// material hashes not yet produced are listed as unpinned.
func StartRun(repoRoot, labID string, declaration []byte, approve string, cores int, executable, version string, s Starter) (RunRecord, Plan, error) {
	plan, _, err := PlanForLab(repoRoot, labID, declaration, cores)
	if err != nil {
		return RunRecord{}, Plan{}, err
	}
	if approve == "" {
		return RunRecord{}, plan, ErrNotConsented
	}
	p, err := lab.OpenProcedure(repoRoot, labID)
	if err != nil {
		return RunRecord{}, plan, err
	}
	entry := p.Entry()
	p.Close()
	f, err := os.Open(executable)
	if err != nil {
		return RunRecord{}, plan, fmt.Errorf("the binary that drives the run cannot be read: %w", err)
	}
	defer f.Close()
	bin, err := pinBinary(f, version)
	if err != nil {
		return RunRecord{}, plan, err
	}
	var suffix [2]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return RunRecord{}, plan, err
	}
	runID := "mx-" + time.Now().UTC().Format("060102150405") + "-" + hex.EncodeToString(suffix[:])
	rec, err := consent(repoRoot, plan, approve, ConsentInputs{RunID: runID, RootSHA: entry.RootSHA, Binary: bin,
		Declaration: declaration, Machine: Machine{OS: runtime.GOOS, Arch: runtime.GOARCH, Cores: cores}}, s)
	return rec, plan, err
}
