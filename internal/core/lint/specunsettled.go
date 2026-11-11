package lint

import (
	"os"
	"path/filepath"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// ruleSpecUnsettled is the record-lint half of iss-2611101318042398: an open
// spec that says in its own text that it is not ready to build — a sentence
// opening "Not READY", or a bold "Open:" label outside its `## Open Questions`
// section — is refused where it is written. The readiness gate's spec_body
// row reads the same statements through the same reader (spec.Unsettled), and
// the open items under the section besides, so a spec this rule passes asks
// its questions where the build reads them.
const ruleSpecUnsettled = "spec_unsettled"

// checkSpecUnsettled names each statement spec.Unsettled finds in an open
// spec, one finding on its line, at the configured severity.
//
// WHY open/ alone: a closed spec is never built again, so what it said about
// its readiness is history. A content-exempt spec is skipped as every spec
// content rule skips it.
func checkSpecUnsettled(repoRoot, rootAbs string, cfg RuleConfig, top Config) ([]Finding, error) {
	specsDir := cfg.SpecsDir
	if specsDir == "" {
		specsDir = "specs"
	}
	if _, err := os.Stat(filepath.Join(rootAbs, specsDir)); err != nil {
		if os.IsNotExist(err) {
			return nil, nil // missing specs/ is soft, mirroring spec_lifecycle
		}
		return nil, err
	}
	rootRel, err := filepath.Rel(repoRoot, rootAbs)
	if err != nil {
		return nil, err
	}
	idx, err := ScanSpecLinks(repoRoot,
		filepath.ToSlash(filepath.Join(rootRel, intentsDirOf(cfg))),
		filepath.ToSlash(filepath.Join(rootRel, specsDir)), top)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, s := range idx.Specs {
		if s.exempt || s.Bucket != "open" {
			continue
		}
		content, err := readRepoAbs(repoRoot, filepath.Join(repoRoot, s.Path), maxRepoFileBytes)
		if err != nil {
			return nil, err
		}
		for _, st := range spec.Unsettled(string(content)) {
			out = append(out, Finding{
				File: s.Path, Line: st.Line, RuleID: ruleSpecUnsettled, Severity: cfg.Severity,
				Message: "open spec says it is not ready to build: " + st.Describe() + " — " + spec.UnsettledRemedy,
			})
		}
	}
	return out, nil
}
