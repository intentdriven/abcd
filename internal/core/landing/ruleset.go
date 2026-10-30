package landing

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// RulesetsRelDir is the committed mirror of the default branch's rulesets.
const RulesetsRelDir = ".abcd/work/rulesets"

// Bounds on the ruleset mirror's read.
const (
	maxRulesetBytes = 256 << 10
	maxRulesets     = 32
	maxListing      = 1 << 20
)

// MergeMethods maps a merge queue's method to the forge client's flag.
var MergeMethods = map[string]string{"MERGE": "--merge", "SQUASH": "--squash", "REBASE": "--rebase"}

// MergeRule is the ruleset mirror read at a commit: the merge queue's method
// ("" when no merge queue gates the default branch). A pull-request rule's
// approval requirement is not read: ruling AM1, which armed only where one
// required a person's approval, is retired, and the merge gate
// (facilitator.go) reads approvals from the forge itself.
type MergeRule struct {
	Method string
}

// MirrorError is a ruleset mirror that cannot be read into a merge rule:
// Reason says what is wrong and Fix what settles it. A caller adds what to run
// once it is settled.
type MirrorError struct {
	Reason string
	Fix    string
}

func (e *MirrorError) Error() string { return e.Reason + "; " + e.Fix }

// ListError is a ruleset mirror whose listing git refused.
type ListError struct{ Err error }

func (e *ListError) Error() string { return "listing the ruleset mirror: " + e.Err.Error() }
func (e *ListError) Unwrap() error { return e.Err }

// ruleset is the part of a ruleset mirror the merge rule is read from.
type ruleset struct {
	Enforcement string `json:"enforcement"`
	Conditions  struct {
		RefName struct {
			Include []string `json:"include"`
			Exclude []string `json:"exclude"`
		} `json:"ref_name"`
	} `json:"conditions"`
	Rules []struct {
		Type       string          `json:"type"`
		Parameters json.RawMessage `json:"parameters"`
	} `json:"rules"`
}

// targets reports whether the ruleset applies to the default branch def.
func (r ruleset) targets(def string) bool {
	match := func(pats []string) bool {
		for _, p := range pats {
			if p == "~DEFAULT_BRANCH" || p == "~ALL" || p == "refs/heads/"+def {
				return true
			}
		}
		return false
	}
	return r.Enforcement == "active" && match(r.Conditions.RefName.Include) && !match(r.Conditions.RefName.Exclude)
}

// ReadMergeRule reads the ruleset mirror at the commit base in the repository
// at repoRoot, for the default branch def. A missing mirror requires nothing.
// The base is a commit of the default branch (a lane's base, or the merge base
// of a branch with the default branch), so the branch's own commits cannot
// change the rule it lands by. at names base in a refusal ("the lane's base").
// A mirror that cannot be read is a *MirrorError, a listing git refused a
// *ListError.
func ReadMergeRule(repoRoot, base, def, at string) (MergeRule, error) {
	names, err := gitutil.RunCappedBytes(repoRoot, maxListing, "ls-tree", "-z", "--name-only", base, "--", RulesetsRelDir+"/")
	if err != nil {
		return MergeRule{}, &ListError{Err: err}
	}
	var r MergeRule
	count := 0
	for _, name := range strings.Split(string(names), "\x00") {
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		if count++; count > maxRulesets {
			return MergeRule{}, &MirrorError{fmt.Sprintf("the ruleset mirror holds more than %d files", maxRulesets), "trim the mirror"}
		}
		raw, err := gitutil.RunCappedBytes(repoRoot, maxRulesetBytes, "cat-file", "blob", base+":"+name)
		if err != nil {
			return MergeRule{}, &MirrorError{name + " cannot be read at " + at, "restore the ruleset mirror"}
		}
		var rs ruleset
		if err := json.Unmarshal(raw, &rs); err != nil {
			return MergeRule{}, &MirrorError{name + " does not parse as a ruleset, so the merge rule cannot be read",
				"restore the ruleset mirror (its README says how to refresh it)"}
		}
		if !rs.targets(def) {
			continue
		}
		for _, rule := range rs.Rules {
			if rule.Type != "merge_queue" {
				continue
			}
			var p struct {
				MergeMethod string `json:"merge_method"`
			}
			_ = json.Unmarshal(rule.Parameters, &p)
			m := strings.ToUpper(p.MergeMethod)
			if m == "" {
				m = "MERGE"
			}
			if _, ok := MergeMethods[m]; !ok {
				return MergeRule{}, &MirrorError{name + " names a merge-queue method the forge client has no flag for", "correct the ruleset mirror"}
			}
			if r.Method != "" && r.Method != m {
				return MergeRule{}, &MirrorError{"the ruleset mirror gates the default branch through merge queues with different methods", "correct the ruleset mirror"}
			}
			r.Method = m
		}
	}
	return r, nil
}

// ownerName reports whether s is a forge account name: letters, digits and
// hyphens, not empty, so it can stand in a forge API path as it is.
func ownerName(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
			return false
		}
	}
	return true
}

// repoName reports whether s is a forge repository name (letters, digits,
// '-', '_' and '.', and neither "." nor ".."), so it can stand in a forge API
// path as it is.
func repoName(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}
