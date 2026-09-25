package lint_test

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestBuildProvenanceUsesActionsAttestDirectly holds every workflow, and the
// scaffold templates they render from, to one attestation action
// (iss-273). Upstream states that from v4 actions/attest-build-provenance is a
// wrapper over actions/attest and that new use should call actions/attest.
// This repository called both — the wrapper for build provenance and
// actions/attest for the semantic-gate receipts — so it ran two versions of
// one action (the wrapper pinned its own, older, actions/attest), carried a
// second trust layer and doubled the dependency-bump churn that breaks
// template parity.
//
// A build-provenance step is an actions/attest step with no predicate input:
// with none, actions/attest generates SLSA build provenance with the same
// generator the wrapper reached. The wrapper also set NODE_OPTIONS to raise
// the HTTP header limit on its attest call, so every such step carries that
// env too and the emitted attestation is produced exactly as before.
func TestBuildProvenanceUsesActionsAttestDirectly(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	files, err := filepath.Glob(filepath.Join(root, ".github", "workflows", "*.yml"))
	if err != nil || len(files) == 0 {
		t.Fatalf("no workflows found (%v)", err)
	}
	tmpls, _ := filepath.Glob(filepath.Join(root, "internal", "core", "launch", "scaffold", "templates", "*.yml.tmpl"))
	files = append(files, tmpls...)

	provenance := 0
	for _, f := range files {
		rel, _ := filepath.Rel(root, f)
		rel = filepath.ToSlash(rel)
		wf := readRepoFile(t, root, rel)
		if strings.Contains(wf, "actions/attest-build-provenance@") {
			t.Errorf("%s still uses the actions/attest-build-provenance wrapper; call actions/attest directly", rel)
		}
		steps, err := workflowSteps(wf)
		if err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
		for _, s := range steps {
			if !strings.Contains(s.body, "uses: actions/attest@") {
				continue
			}
			if strings.Contains(s.body, "predicate-type:") || strings.Contains(s.body, "predicate:") ||
				strings.Contains(s.body, "predicate-path:") || strings.Contains(s.body, "sbom-path:") {
				continue
			}
			provenance++
			if !strings.Contains(s.body, "subject-path:") {
				t.Errorf("%s line %d, step %q: a build-provenance attestation needs subject-path", rel, s.line, s.name)
			}
			if s.env["NODE_OPTIONS"] != `"--max-http-header-size=32768"` {
				t.Errorf("%s line %d, step %q: carry NODE_OPTIONS \"--max-http-header-size=32768\", as the wrapper's attest call did (got %q)",
					rel, s.line, s.name, s.env["NODE_OPTIONS"])
			}
		}
	}
	// release.yml, its template and site.yml each attest build provenance; a
	// count below that means a step was lost or stopped being recognised.
	if provenance < 3 {
		t.Errorf("found %d build-provenance steps on actions/attest; want at least 3 (release.yml, its template, site.yml)", provenance)
	}
}
