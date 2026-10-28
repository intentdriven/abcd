package lint

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/intentdriven/abcd/internal/core/spec"
)

// ruleSpecStepsSerial is the record-lint half of dcn-2610230857411093 (on
// iss-2610230016561927): an open spec of two or more steps where no step
// declares `needs:` runs fully serial, each step waiting for every earlier one
// (ruling DR6b), so a build spends most of its time waiting on serial merges.
// The readiness gate's steps row warns on the same shape; both read it through
// spec.SerialByDefault over spec.ParseSteps, so the two never disagree and there
// is one step parser.
const ruleSpecStepsSerial = "spec_steps_serial"

// checkSpecStepsSerial flags each open spec whose steps run serial by default,
// at the configured severity (warn in the shipped config: the ruling is a
// warning, never a refusal).
//
// WHY open/ alone: a closed spec is never built again, so naming its steps'
// needs is work nobody may act on; a remainder carries its unlanded steps into a
// new open spec, which this rule then reads. WHY a section the parser refuses is
// skipped: its shape is the readiness gate's steps row to name, with the shape it
// expects, and this rule judges needs, not shape. A content-exempt spec is
// skipped as every spec content rule skips it.
func checkSpecStepsSerial(repoRoot, rootAbs string, cfg RuleConfig, top Config) ([]Finding, error) {
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
		steps, perr := spec.ParseSteps(string(content))
		if perr != nil || !spec.SerialByDefault(steps) {
			continue
		}
		out = append(out, Finding{
			File: s.Path, Line: 1, RuleID: ruleSpecStepsSerial, Severity: cfg.Severity,
			Message: "open spec lists " + strconv.Itoa(len(steps)) + " steps and " + spec.SerialByDefaultWarning +
				" — " + spec.SerialByDefaultRemedy("the spec"),
		})
	}
	return out, nil
}
