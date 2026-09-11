package scaffold

// abcdSemanticGates is the required host-run detector list abcd-cli's release
// gate arms (see .abcd/development/release-gate/README.md). It is the reference
// semantic-gate set the self-scaffold parity rendering reproduces.
var abcdSemanticGates = []string{"docs-currency-reviewer", "iss35-brief-surface-crosscheck"}

// abcdExtraGates are the deterministic verify steps abcd-cli runs beyond the
// generic Go leg (gofmt/build/vet/test/race). They render into both the verify
// job and the runbook's numbered gate list from this one source, so the
// gate_lockstep invariant (runbook list == workflow steps) holds by construction.
var abcdExtraGates = []Gate{
	{Name: "Record-lint (design-record drift gate)", Run: "go run ./cmd/record-lint"},
	{Name: "Docs-lint (docs-currency gate)", Run: "go run ./cmd/abcd docs lint"},
	{Name: "Reviews-charter discipline (RD001-RD003)", Run: "bash scripts/check-reviews.sh"},
	{Name: "Smoke every command (self-discovering harness)", Run: "make smoke"},
}

// AbcdSubstitutions is abcd-cli's own fact set: the fixed inputs that regenerate
// its live release.yml / auto-release.yml byte-for-byte (self-scaffold parity,
// spc-14 decision b). It is deliberately hard-coded, not derived — it is the
// reference rendering the parity test pins, and deriving it would let the pin
// track the derivation rather than the proven workflow.
func AbcdSubstitutions() Substitutions {
	return Substitutions{
		DefaultBranch: "main",
		Abcd:          true,
		ExtraGates:    abcdExtraGates,
		SemanticGates: abcdSemanticGates,
	}
}

// BareSubstitutions is the degraded fact set a managed repo with no semantic
// detectors receives: the deterministic Go gates alone, a generic build, and no
// host-run semantic gate (spc-14 clean degradation). DefaultBranch is the repo's
// own fact, derived by the caller; the Go toolchain is not a substitution at all,
// because the rendered workflows read it out of the adopter's go.mod.
func BareSubstitutions(defaultBranch string) Substitutions {
	return Substitutions{
		DefaultBranch: defaultBranch,
		Abcd:          false,
		ExtraGates:    nil,
		SemanticGates: nil,
	}
}
