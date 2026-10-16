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
// ("" when no merge queue gates the default branch), whether an active
// ruleset on the default branch requires a person's approval (ruling AM1),
// and whether a mirror exists at all.
type MergeRule struct {
	Method   string
	Approval bool
	Mirrored bool
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
			if rule.Type == "pull_request" && requiresApproval(repoRoot, base, rule.Parameters) {
				r.Approval = true
			}
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
	r.Mirrored = count > 0
	return r, nil
}

// requiresApproval reports whether a pull_request rule's parameters require a
// person's approval: an approving count of one or more, or a code-owner review
// where a CODEOWNERS file at base names an owner. Parameters that do not parse
// require nothing, so the merge is left for a person.
func requiresApproval(repoRoot, base string, params json.RawMessage) bool {
	var p struct {
		Count     int  `json:"required_approving_review_count"`
		CodeOwner bool `json:"require_code_owner_review"`
	}
	if json.Unmarshal(params, &p) != nil {
		return false
	}
	return p.Count >= 1 || (p.CodeOwner && namesCodeOwners(repoRoot, base))
}

// codeOwnersPaths are where the forge reads a CODEOWNERS file from, in the
// order it looks: the first file found is the only one it reads.
var codeOwnersPaths = []string{".github/CODEOWNERS", "CODEOWNERS", "docs/CODEOWNERS"}

// namesCodeOwners reports whether the CODEOWNERS file the forge reads at base
// — the first found in codeOwnersPaths, so a file there shadows the later ones
// even when it names nobody — names at least one owner: a line whose pattern
// is followed by an owner token (codeOwner).
func namesCodeOwners(repoRoot, base string) bool {
	for _, p := range codeOwnersPaths {
		raw, err := gitutil.RunCappedBytes(repoRoot, maxRulesetBytes, "cat-file", "blob", base+":"+p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(raw), "\n") {
			if lineNamesOwner(line) {
				return true
			}
		}
		return false
	}
	return false
}

// lineNamesOwner reports whether one CODEOWNERS line names an owner: after its
// pattern, a field before any comment is an owner token. A blank line, a
// comment and a pattern with no owner name nobody.
func lineNamesOwner(line string) bool {
	fields := strings.Fields(line)
	if len(fields) < 2 || strings.HasPrefix(fields[0], "#") {
		return false
	}
	for _, f := range fields[1:] {
		if strings.HasPrefix(f, "#") {
			return false
		}
		if codeOwner(f) {
			return true
		}
	}
	return false
}

// codeOwner reports whether a token is one the forge takes as an owner:
// @username, @org/team-name, or an e-mail address.
func codeOwner(tok string) bool {
	if name, ok := strings.CutPrefix(tok, "@"); ok {
		org, team, isTeam := strings.Cut(name, "/")
		if isTeam {
			return ownerName(org) && team != "" && !strings.ContainsAny(team, "/@")
		}
		return ownerName(name)
	}
	local, domain, ok := strings.Cut(tok, "@")
	return ok && local != "" && strings.Contains(domain, ".") && !strings.HasPrefix(domain, ".") &&
		!strings.HasSuffix(domain, ".") && !strings.Contains(domain, "@")
}

// ownerName reports whether s is a forge account name: letters, digits and
// hyphens, not empty.
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
