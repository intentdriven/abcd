package lint

import (
	"fmt"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// ruleRecordSlugCap holds every record in an id-bearing store to the repository's
// slug cap, the one value the minting verbs cut at (layered.RecordSlugCap). A verb
// cannot write past it; a hand rename can, and this rule and the commit-msg hook
// are what refuse that. Every committed name is judged by the value in force:
// a name written under an earlier, wider cap has no standing of its own, so a
// repository that lowers the cap renames what it left over the line
// (cmd/record-slug-rename), or does not lower it.
const ruleRecordSlugCap = "record_slug_cap"

func checkRecordSlugCap(repoRoot string, cfg RuleConfig) ([]Finding, error) {
	rels, err := recordid.RecordFiles(repoRoot)
	if err != nil {
		return nil, err
	}
	return slugCapFindings(repoRoot, rels, cfg.Severity)
}

// SlugCapFindings is the rule over the named repository-relative paths alone, at
// blocker: what the commit-msg hook asks of the records a commit adds or renames.
func SlugCapFindings(repoRoot string, rels []string) ([]Finding, error) {
	return slugCapFindings(repoRoot, rels, "blocker")
}

func slugCapFindings(repoRoot string, rels []string, severity string) ([]Finding, error) {
	slugCap, err := layered.RecordSlugCap(repoRoot)
	if err != nil {
		return nil, err
	}
	var out []Finding
	for _, o := range recordid.SlugsOverCap(repoRoot, rels, slugCap) {
		out = append(out, Finding{
			File: o.Rel, Line: o.Line, RuleID: ruleRecordSlugCap, Severity: severity,
			Message: fmt.Sprintf("%s is %d characters, over this repository's cap of %d (%s in .abcd/config.json, %d when unset); shorten the slug in the filename and the slug field alike, or raise the cap",
				o.Where, o.Len, slugCap, layered.SlugCapKey, recordid.DefaultMaxSlugLen),
		})
	}
	return out, nil
}
