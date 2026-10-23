package matrix

import (
	"errors"
	"fmt"
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

// Consent starts the run p describes, once the person has said yes to exactly
// that plan: approve must be its digest. An empty approval is ErrNotConsented;
// an approval naming another plan, a plan whose content no longer matches its
// own digest, and a declaration other than the one the plan was built from are
// refused. In every refusal nothing is written and nothing starts. On a match
// the run's record and declaration are written to the lab, then s starts the
// run, once.
func Consent(repoRoot string, p Plan, approve string, in ConsentInputs, s Starter) (RunRecord, error) {
	if approve == "" {
		return RunRecord{}, ErrNotConsented
	}
	if approve != p.Digest {
		return RunRecord{}, fmt.Errorf("the approval %s is not the plan's digest %s: it names another plan; show the person this one and start it only with its own digest", clip(approve), p.Digest)
	}
	if got := p.digest(); got != p.Digest {
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
