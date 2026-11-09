package loop

// findings.go is the per-reviewer findings record (itd-2610240922461281,
// spc-2610240951109420 step 4): what each reviewer of a round found, by
// location, so a later choice of review set is made from counts.
//
//   - Each `file:line` cited under a return's Findings heading is one located
//     finding of that reviewer, read when the loop records the verdict. A
//     return citing none records zero and is not refused: a reviewer that
//     found nothing, or wrote its findings without locations, still returns.
//     One location cited twice in one return is one finding, and a range or a
//     column (`a.go:12-20`, `a.go:12:5`) is its first line.
//   - Once the round's fix report is verified, each finding of a reviewer
//     whose verdict failed the round, the findings the fix brief handed on,
//     carries applied or rejected as the report says it on a line citing the
//     finding's location, and unknown when no line does or the lines that do
//     say both or neither.
//   - A location two or more reviewers of the round cite is marked, on each
//     of their findings, with every reviewer that cites it.
//
// A location is a path relative to the lane's worktree: a path the reviewer
// wrote under the worktree is taken relative to it, and any other absolute
// path, a path climbing out with `..`, and a word with no `/` or `.` in it
// (`round:2`) are no location, so no text a reviewer wrote outside a plain
// repository path reaches the state or a --json stream.

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Finding is one located finding a reviewer's return cites.
type Finding struct {
	// File is the path the finding cites, relative to the lane's worktree,
	// and Line the line, from 1.
	File string `json:"file"`
	Line int    `json:"line"`
	// Status is what the round's fix report says of it: FindingApplied,
	// FindingRejected or FindingUnknown; empty until a fix report is
	// verified, and on the findings of a reviewer that passed the round,
	// which no fix brief hands on.
	Status string `json:"status,omitempty"`
	// CitedBy names every reviewer of the round citing the same file and
	// line, this one included, in the round's validator order; absent when
	// only this reviewer cites it.
	CitedBy []Citer `json:"cited_by,omitempty"`
}

// Citer is one reviewer of a round: its role and its index among the role's
// reviewers.
type Citer struct {
	Role     string `json:"role"`
	Reviewer int    `json:"reviewer"`
}

// The statuses a fix report gives a finding.
const (
	FindingApplied  = "applied"
	FindingRejected = "rejected"
	FindingUnknown  = "unknown"
)

// maxLocatedFindings bounds the findings one reviewer's record keeps; its
// count still counts every location the return cites.
const maxLocatedFindings = 256

// maxFindingLine bounds a cited line: a number past it is no line.
const maxFindingLine = 10_000_000

// citationRe is a `file:line` citation: a path token standing after the
// start of a line, a space or an opening mark, then a colon and a line, and
// an optional range end or column.
var citationRe = regexp.MustCompile("(?:^|[\\s(\\[{<\"'`*,;])(/?[A-Za-z0-9_.][A-Za-z0-9_./+@-]*):([0-9]+)(?:[-:][0-9]+)?")

// statusRe is a status word on a line of a fix report.
var statusRe = regexp.MustCompile(`(?i)\b(applied|rejected)\b`)

// hasLetter is a path's need of a letter: `12:30` is a time, not a location.
var hasLetter = regexp.MustCompile(`[A-Za-z]`)

// citations are the locations text cites, in order, each once, with
// worktree the lane's worktree a path under it is taken relative to.
func citations(text, worktree string) []Finding {
	var out []Finding
	for _, m := range citationRe.FindAllStringSubmatch(text, -1) {
		file, ok := locationPath(m[1], worktree)
		if !ok {
			continue
		}
		n, err := strconv.Atoi(m[2])
		if err != nil || n < 1 || n > maxFindingLine {
			continue
		}
		if slices.ContainsFunc(out, func(f Finding) bool { return f.File == file && f.Line == n }) {
			continue
		}
		out = append(out, Finding{File: file, Line: n})
	}
	return out
}

// locationPath is p as a location relative to the worktree, and whether it
// is one.
func locationPath(p, worktree string) (string, bool) {
	if wt := strings.TrimSuffix(worktree, "/"); wt != "" {
		if rest, ok := strings.CutPrefix(p, wt+"/"); ok {
			p = rest
		}
	}
	p = strings.TrimPrefix(p, "./")
	p = strings.TrimRight(p, ".")
	return p, validLocation(p)
}

// validLocation reports whether a stored finding's file is one citations
// writes.
func validLocation(p string) bool {
	return fsutil.ValidRelPath(p) && strings.ContainsAny(p, "/.") && hasLetter.MatchString(p) &&
		!strings.HasSuffix(p, ".") && !strings.ContainsFunc(p, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./+@-", r))
	})
}

// findingsSections are the parts of a return under its Findings headings:
// each runs to the next heading at the same depth or above.
func findingsSections(text string) []string {
	var out []string
	for _, at := range findingsHeading.FindAllStringIndex(text, -1) {
		depth := strings.Count(strings.TrimSpace(text[at[0]:at[1]]), "#")
		rest := text[at[1]:]
		end := len(rest)
		off := 0
		for _, ln := range strings.SplitAfter(rest, "\n") {
			if d := headingDepth(ln); off > 0 && d > 0 && d <= depth {
				end = off
				break
			}
			off += len(ln)
		}
		out = append(out, rest[:end])
	}
	return out
}

// headingDepth is the depth of a markdown heading line, 0 for any other line.
func headingDepth(ln string) int {
	t := strings.TrimLeft(ln, " \t")
	if len(ln)-len(t) > 3 {
		return 0
	}
	d := 0
	for d < len(t) && t[d] == '#' {
		d++
	}
	if d == 0 || d > 6 || d < len(t) && t[d] != ' ' && t[d] != '\t' && t[d] != '\r' && t[d] != '\n' {
		return 0
	}
	return d
}

// recordFindings records on v the located findings its return cites: their
// count, and the first maxLocatedFindings of them.
func (v *ValidatorRun) recordFindings(text, worktree string) {
	var found []Finding
	for _, s := range findingsSections(text) {
		for _, f := range citations(s, worktree) {
			if !slices.ContainsFunc(found, func(o Finding) bool { return o.File == f.File && o.Line == f.Line }) {
				found = append(found, f)
			}
		}
	}
	n := len(found)
	v.Located = &n
	v.Findings = nil
	if n > 0 {
		v.Findings = found[:min(n, maxLocatedFindings)]
	}
}

// markCitedBy marks, on every located finding of round r, the reviewers of
// the round that cite its file and line, where more than one does.
func markCitedBy(r *ValidationRound) {
	type at struct {
		file string
		line int
	}
	by := map[at][]Citer{}
	for _, v := range r.Validators {
		for _, f := range v.Findings {
			k := at{f.File, f.Line}
			by[k] = append(by[k], Citer{Role: v.Role, Reviewer: v.Reviewer})
		}
	}
	for i := range r.Validators {
		v := &r.Validators[i]
		if len(v.Findings) == 0 {
			continue
		}
		v.Findings = slices.Clone(v.Findings)
		for j := range v.Findings {
			f := &v.Findings[j]
			f.CitedBy = nil
			if c := by[at{f.File, f.Line}]; len(c) > 1 {
				f.CitedBy = slices.Clone(c)
			}
		}
	}
}

// joinFixReport records, on each finding of a reviewer whose verdict failed
// round r, what the fix report says of it (fixStatus).
func joinFixReport(r *ValidationRound, report, worktree string) {
	lines := strings.Split(report, "\n")
	for i := range r.Validators {
		v := &r.Validators[i]
		if v.Pass || len(v.Findings) == 0 {
			continue
		}
		v.Findings = slices.Clone(v.Findings)
		for j := range v.Findings {
			v.Findings[j].Status = fixStatus(lines, v.Findings[j], worktree)
		}
	}
}

// fixStatus is what the lines of a fix report say of f: applied or rejected
// when every line citing its location says that word and not the other, and
// unknown otherwise.
func fixStatus(lines []string, f Finding, worktree string) string {
	said := ""
	for _, ln := range lines {
		if !slices.ContainsFunc(citations(ln, worktree), func(o Finding) bool { return o.File == f.File && o.Line == f.Line }) {
			continue
		}
		words := map[string]bool{}
		for _, m := range statusRe.FindAllString(ln, -1) {
			words[strings.ToLower(m)] = true
		}
		if len(words) != 1 {
			return FindingUnknown
		}
		for w := range words {
			if said != "" && said != w {
				return FindingUnknown
			}
			said = w
		}
	}
	if said == "" {
		return FindingUnknown
	}
	return said
}

// findingLocations names each finding by its `file:line`.
func findingLocations(fs []Finding) []string {
	out := make([]string, 0, len(fs))
	for _, f := range fs {
		out = append(out, fmt.Sprintf("%s:%d", f.File, f.Line))
	}
	return out
}

// fixTally is the record's words for what round r's fix report said of the
// located findings it was handed; "" when it was handed none.
func fixTally(r ValidationRound) string {
	handed, applied, rejected, unknown := 0, 0, 0, 0
	for _, v := range r.Validators {
		for _, f := range v.Findings {
			switch f.Status {
			case FindingApplied:
				applied++
			case FindingRejected:
				rejected++
			case FindingUnknown:
				unknown++
			default:
				continue
			}
			handed++
		}
	}
	if handed == 0 {
		return ""
	}
	return fmt.Sprintf("of the %d located finding(s) handed on, %d applied, %d rejected, %d the fix report does not say", handed, applied, rejected, unknown)
}

// located reports the first thing, in a state, that only version 24 and later
// write: a validator's located findings; "" when there is none.
func (s State) located() string {
	for _, l := range s.Lanes {
		for _, r := range l.Validation {
			for i, v := range r.Validators {
				if v.Located != nil || len(v.Findings) > 0 {
					return fmt.Sprintf("`located` findings on %s's round %d validator %d", l.ID, r.Round, i+1)
				}
			}
		}
	}
	return ""
}

// findingsWritten reports whether every located finding a state carries is
// one the loop writes: on a reviewer whose verdict is recorded, as many as
// its count up to the bound, each a location citations writes, once; a
// status exactly where a verified fix report answered a failing reviewer;
// and the reviewers citing each location as markCitedBy marks them.
func (s State) findingsWritten() bool {
	for _, l := range s.Lanes {
		for _, r := range l.Validation {
			for _, v := range r.Validators {
				if v.Located == nil {
					if len(v.Findings) > 0 {
						return false
					}
					continue
				}
				if !slices.Contains(reviewRoles, v.Role) || v.Verdict == "" || *v.Located < 0 || len(v.Findings) != min(*v.Located, maxLocatedFindings) {
					return false
				}
				answered := r.Fix != "" && !v.Pass
				for i, f := range v.Findings {
					if !validLocation(f.File) || f.Line < 1 || f.Line > maxFindingLine ||
						slices.ContainsFunc(v.Findings[:i], func(o Finding) bool { return o.File == f.File && o.Line == f.Line }) {
						return false
					}
					switch f.Status {
					case "":
						if answered {
							return false
						}
					case FindingApplied, FindingRejected, FindingUnknown:
						if !answered {
							return false
						}
					default:
						return false
					}
				}
			}
			again := r
			again.Validators = slices.Clone(r.Validators)
			markCitedBy(&again)
			for i, v := range r.Validators {
				for j, f := range v.Findings {
					if !slices.Equal(f.CitedBy, again.Validators[i].Findings[j].CitedBy) {
						return false
					}
				}
			}
		}
	}
	return true
}
