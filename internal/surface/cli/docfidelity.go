package cli

// docfidelity.go — the front door of the doc-fidelity gate over the brief
// (itd-60, spc-2609020903498198). The judgement is core/docfidelity's; this
// file derives the command tree from the live binary (the tree commands.md and
// surface.json are generated from), picks each enforcement point's population,
// and formats the verdict. It invents no word of the verdict.

import (
	"strings"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/core/spec"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// closeShips names the intents a `spec close` of specID would move to
// shipped: each member no other open spec still names. A reference the store
// cannot resolve ships nothing here, and the close reports it itself.
func closeShips(repoRoot, specID string) []string {
	store, err := spec.Load(repoRoot)
	if err != nil {
		return nil
	}
	sp, ok := store.Lookup(specID)
	if !ok || sp.Status != spec.StatusOpen {
		return nil
	}
	var out []string
	for _, m := range sp.Members() {
		others := 0
		for _, o := range store.OpenSpecsForIntent(m) {
			if o.ID != sp.ID {
				others++
			}
		}
		if others == 0 {
			out = append(out, m)
		}
	}
	return out
}

// enforceDocFidelity runs the gate for population and returns the refusal the
// verb exits with, or nil. The gate is armed only in the repository that ships
// the binary its brief describes; elsewhere it judges nothing.
func enforceDocFidelity(repoRoot, verb string, population []string) error {
	if len(population) == 0 || !docfidelity.Armed(repoRoot) {
		return nil
	}
	snap, err := SurfaceSnapshot(repoRoot)
	if err != nil {
		return &exitError{Code: 2, Msg: verb + ": the doc-fidelity gate cannot derive the command tree: " + scrubPaths(err)}
	}
	v, _, err := docfidelity.Gate(repoRoot, snap.Commands, population, false)
	if err != nil {
		return &exitError{Code: 2, Msg: verb + ": the doc-fidelity gate cannot read its inputs (nothing moved): " + scrubPaths(err)}
	}
	if !v.Refuse {
		return nil
	}
	var b strings.Builder
	b.WriteString(verb + ": refused by the doc-fidelity gate — the brief lags what " +
		strings.Join(population, ", ") + " delivered (nothing moved):")
	for _, r := range v.Reasons {
		b.WriteString("\n  - " + termsafe.Sanitize(r))
	}
	b.WriteString("\n  (`abcd docs fidelity` shows the whole verdict; `abcd docs fidelity record` saves a docs review for HEAD)")
	return &exitError{Code: 1, Msg: b.String()}
}
