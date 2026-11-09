package loop

// findings.go is the per-reviewer findings record (itd-2610240922461281,
// spc-2610240951109420 step 4): what each reviewer of a round found, by
// location, so a later choice of review set is made from counts.
//
//   - The `file:line` each finding bullet under a return's Findings heading
//     leads with (`- **file:line** — ...`, the single location each
//     reviewer's definition asks it to cite) is one located finding of that
//     reviewer, read when the loop records the verdict. A location cited in a
//     bullet's text or its nested Evidence and Attack-path lines is not one,
//     and a line inside a fenced code block or an HTML comment is neither a
//     heading nor one. A
//     return citing none records zero and is not refused: a reviewer that
//     found nothing, or wrote its findings without locations, still returns.
//     One location led with twice in one return is one finding, and a range
//     or a column (`a.go:12-20`, `a.go:12:5`) is its first line.
//   - Once the round's fix report is verified, each finding of a reviewer
//     whose verdict failed the round, the findings the fix brief handed on,
//     carries applied or rejected as the report says it of the finding's
//     location (reportStatuses), and unknown when no line does, or the lines
//     that do say both or a negated word. A line naming the location with no
//     status word says nothing of it.
//   - A location two or more reviewers of the round cite is marked, on each
//     of their findings, with every reviewer that cites it.
//   - The state keeps each reviewer's count on every round, and its findings
//     only on each lane's latest round, the one a fix brief or a judge reads
//     (keepLatestFindings); an earlier round's are dropped when the state is
//     written. A state that would still outgrow what its read accepts has
//     its findings dropped a round at a time, the oldest first, its counts
//     kept (trimOldestFindings), and is not written when that is not enough,
//     so the loop never writes a state its own read refuses.
//
// A location is a path of at most 255 bytes relative to the lane's worktree:
// a path the reviewer wrote under the worktree, in any of its spellings
// (relativeTo), is taken relative to it, and any other absolute path, a path
// climbing out with `..`, and a word with no `/` or `.` in it (`round:2`) are
// no location, so no text a reviewer wrote outside a plain repository path
// reaches the state or a --json stream. Two locations are one only when
// their paths match byte for byte, as git's paths do. Every read is one pass
// over the text, and so bounded: the passes here are linear, and mdrecord's
// fence and comment pass, which pairs a line's backtick runs, is O(L^1.5) on
// one line of length L at worst (unpaired runs differ in length, so a line
// holds about the square root of 2L of them), so a return or report of the
// most bytes it may be is read in well under a second, though not strictly in
// linear time.

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/core/mdrecord"
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

// maxLocatedCount bounds a stored count of located findings: the most
// distinct lead bullets a return of maxReturnBytes could hold, each at least
// the 7 bytes of `- .a:1` and its newline (a location's path is at least a
// dot and a letter). recordFindings stores no count above it, so the loop
// never writes a state its own read refuses.
const maxLocatedCount = maxReturnBytes / len("- .a:1\n")

// maxFindingLine bounds a cited line: a number past it is no line.
const maxFindingLine = 10_000_000

// maxLocationPath bounds a location's path, in bytes: a longer path is no
// location.
const maxLocationPath = 255

// citationRe is a `file:line` citation: a path token standing after the
// start of a line, a space or an opening mark, then a colon and a line, and
// an optional range end or column.
var citationRe = regexp.MustCompile("(?:^|[\\s(\\[{<\"'`*,;])(/?[A-Za-z0-9_.][A-Za-z0-9_./+@-]*):([0-9]+)(?:[-:][0-9]+)?")

// leadRe is the opening of a finding bullet: a list marker at the top level,
// then any opening marks its location stands behind.
var leadRe = regexp.MustCompile("^ ?(?:[-*+]|[0-9]{1,9}[.)])[ \\t]+[*_`\\[(<\"']*")

// leadCitationRe is the `file:line` a finding bullet leads with.
var leadCitationRe = regexp.MustCompile(`^(/?[A-Za-z0-9_.][A-Za-z0-9_./+@-]*):([0-9]+)(?:[-:][0-9]+)?`)

// statusRe is, on a line of a fix report, one of three tokens, read forward
// in one pass: a status word, with the prefix that negates it (`unapplied`,
// `non-applied`) as its first group and the word as its second; a word that
// negates a status word standing after it in its clause, as its third; or a
// mark that ends a clause, so a negation before it does not reach a word
// after it. Only a sentence mark ends a clause: `.`, `;`, `:`, `!` or `?`
// before a space or the stretch's end, an em or en dash, or `--`. A comma
// does not, nor the dot inside a file name, so `not, on reflection, applied`
// stays negated: a negation read too far makes a finding unknown, where one
// cut short would record a fix that was not made.
var statusRe = regexp.MustCompile(`(?i)\b(un-?|non-?)?(applied|rejected)\b` +
	`|(\b(?:not|cannot|never|neither|nor|no|nothing|failed)\b|n['\x{2019}]t\b)` +
	`|[.;:!?](?:\s|$)|[\x{2014}\x{2013}]|--`)

// hasLetter is a path's need of a letter: `12:30` is a time, not a location.
var hasLetter = regexp.MustCompile(`[A-Za-z]`)

// location is one file and line.
type location struct {
	file string
	line int
}

// at is f's location.
func (f Finding) at() location { return location{f.File, f.Line} }

// eachLine calls fn with each line of text, without its newline, reading the
// text once.
func eachLine(text string, fn func(string)) {
	for text != "" {
		ln, rest, _ := strings.Cut(text, "\n")
		fn(ln)
		text = rest
	}
}

// relativeTo is text with every spelling of the lane's worktree before a `/`
// taken out where it starts a token, so a path under it reads relative to
// it: the worktree as given, the directory its links resolve to, and the
// /private twin macOS gives /var, /tmp and /etc, each matched byte for byte.
// A spelling standing after a path's own characters (`x/work/a.go`) is part
// of that path and stays. A spelling may hold a space.
func relativeTo(text, worktree string) string {
	var spellings []string
	var add func(string)
	add = func(p string) {
		p = strings.TrimSuffix(p, "/")
		if p == "" || slices.Contains(spellings, p) {
			return
		}
		spellings = append(spellings, p)
		if rest, ok := strings.CutPrefix(p, "/private/"); ok {
			add("/" + rest)
		} else if strings.HasPrefix(p, "/var/") || strings.HasPrefix(p, "/tmp/") || strings.HasPrefix(p, "/etc/") {
			add("/private" + p)
		}
	}
	add(worktree)
	if len(spellings) == 0 {
		return text
	}
	if resolved, err := filepath.EvalSymlinks(filepath.FromSlash(worktree)); err == nil {
		add(filepath.ToSlash(resolved))
	}
	slices.SortFunc(spellings, func(a, b string) int { return len(b) - len(a) })
	alt := make([]string, len(spellings))
	for i, s := range spellings {
		alt[i] = regexp.QuoteMeta(s)
	}
	// A spelling that is no valid UTF-8 (a link target's name on a
	// filesystem that allows one) compiles to no pattern; the text is then
	// read as written, so a path under the worktree is no location rather
	// than the verifier stopping.
	re, err := regexp.Compile(`(^|[^A-Za-z0-9_./+@-])(?:` + strings.Join(alt, "|") + `)/`)
	if err != nil {
		return text
	}
	return re.ReplaceAllString(text, "${1}")
}

// toLocation is the location a cited path and line name, and whether they
// name one.
func toLocation(path, line string) (location, bool) {
	p := strings.TrimPrefix(path, "./")
	p = strings.TrimRight(p, ".")
	n, err := strconv.Atoi(line)
	if !validLocation(p) || err != nil || n < 1 || n > maxFindingLine {
		return location{}, false
	}
	return location{p, n}, true
}

// validLocation reports whether a stored finding's file is one the loop
// writes.
func validLocation(p string) bool {
	return len(p) <= maxLocationPath && fsutil.ValidRelPath(p) && strings.ContainsAny(p, "/.") && hasLetter.MatchString(p) &&
		!strings.HasSuffix(p, ".") && !strings.ContainsFunc(p, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("_./+@-", r))
	})
}

// leadLocation is the location a finding bullet line leads with, and whether
// the line is one: a top-level list item whose first token, behind any
// opening marks, is a `file:line`.
func leadLocation(ln string) (location, bool) {
	m := leadRe.FindStringIndex(ln)
	if m == nil {
		return location{}, false
	}
	c := leadCitationRe.FindStringSubmatch(ln[m[1]:])
	if c == nil {
		return location{}, false
	}
	return toLocation(c[1], c[2])
}

// eachFindingsLine calls fn with each line of text under a Findings heading,
// down to the next heading at the same depth or above. A line inside a fenced
// code block or an HTML comment, as mdrecord reads both under its ListNested
// rule (a fence nested in a list item included, and each delimiter line, as
// mdrecord.BulletBlocks skips them), is neither a heading nor passed on, so a
// `# comment` in a quoted snippet does not end the section, nor a bullet in a
// snippet or a commented-out note count as a finding. The text is split once
// and read in one pass.
func eachFindingsLine(text string, fn func(string)) {
	lines := strings.Split(text, "\n")
	mask := mdrecord.Read(lines, mdrecord.ListNested).Mask
	depth := 0
	for i, ln := range lines {
		if mask[i]&(mdrecord.MaskFence|mdrecord.MaskComment) != 0 {
			continue
		}
		d := headingDepth(ln)
		switch {
		case d > 0 && findingsHeading.MatchString(ln):
			if depth == 0 || d < depth {
				depth = d
			}
		case d > 0 && d <= depth:
			depth = 0
		case depth > 0:
			fn(ln)
		}
	}
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

// recordFindings records on v the located findings its return cites: the
// location each finding bullet under its Findings headings leads with, each
// once, their count, and the first maxLocatedFindings of them.
func (v *ValidatorRun) recordFindings(text, worktree string) {
	var found []Finding
	seen := map[location]bool{}
	eachFindingsLine(relativeTo(text, worktree), func(ln string) {
		l, ok := leadLocation(ln)
		if !ok || seen[l] {
			return
		}
		seen[l] = true
		if len(found) < maxLocatedFindings {
			found = append(found, Finding{File: l.file, Line: l.line})
		}
	})
	n := min(len(seen), maxLocatedCount)
	v.Located = &n
	v.Findings = found
}

// markCitedBy marks, on every located finding of round r, the reviewers of
// the round that cite its file and line, where more than one does.
func markCitedBy(r *ValidationRound) {
	by := map[location][]Citer{}
	for _, v := range r.Validators {
		for _, f := range v.Findings {
			by[f.at()] = append(by[f.at()], Citer{Role: v.Role, Reviewer: v.Reviewer})
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
			if c := by[f.at()]; len(c) > 1 {
				f.CitedBy = slices.Clone(c)
			}
		}
	}
}

// joinFixReport records, on each finding of a reviewer whose verdict failed
// round r, what the fix report says of it (reportStatuses).
func joinFixReport(r *ValidationRound, report, worktree string) {
	said := reportStatuses(report, worktree)
	for i := range r.Validators {
		v := &r.Validators[i]
		if v.Pass || len(v.Findings) == 0 {
			continue
		}
		v.Findings = slices.Clone(v.Findings)
		for j := range v.Findings {
			s, ok := said[v.Findings[j].at()]
			if !ok {
				s = FindingUnknown
			}
			v.Findings[j].Status = s
		}
	}
}

// The words a stretch of a fix report line says, as bits.
const (
	saysApplied = 1 << iota
	saysRejected
	saysNegated
)

// reportStatuses is what a fix report says of each location it cites, read
// in one pass. On a line, a status word speaks of the location it follows,
// and the words before the line's first location of that one; a word with a
// negation before it in its clause (`not`, `no`, `nothing`, `cannot`,
// `never`, `neither`, `nor`, `failed`, `n't`, an `un` or `non` prefix) says
// neither. A line that names a location but says no status word of it says
// nothing of it, so a report repeating the brief's list changes nothing. A
// location is applied or rejected when every line saying a word of it says
// that word alone, and unknown otherwise.
func reportStatuses(report, worktree string) map[location]string {
	out := map[location]string{}
	eachLine(relativeTo(report, worktree), func(ln string) {
		type cite struct {
			at         location
			start, end int
		}
		var cites []cite
		for _, m := range citationRe.FindAllStringSubmatchIndex(ln, -1) {
			if l, ok := toLocation(ln[m[2]:m[3]], ln[m[4]:m[5]]); ok {
				cites = append(cites, cite{l, m[2], m[1]})
			}
		}
		if len(cites) == 0 {
			return
		}
		words := map[location]int{}
		for i, c := range cites {
			end := len(ln)
			if i+1 < len(cites) {
				end = cites[i+1].start
			}
			w := statusWords(ln[c.end:end])
			if i == 0 {
				w |= statusWords(ln[:c.start])
			}
			words[c.at] |= w
		}
		for l, w := range words {
			if w == 0 {
				continue
			}
			s := FindingUnknown
			switch w {
			case saysApplied:
				s = FindingApplied
			case saysRejected:
				s = FindingRejected
			}
			if prev, ok := out[l]; ok && prev != s {
				s = FindingUnknown
			}
			out[l] = s
		}
	})
	return out
}

// statusWords are the status words a stretch of a line says, read forward
// once: whether a negation has stood since the clause's last mark is carried
// along, never looked back for.
func statusWords(text string) int {
	w, negated := 0, false
	for _, m := range statusRe.FindAllStringSubmatchIndex(text, -1) {
		switch {
		case m[4] >= 0 && (m[2] >= 0 || negated):
			w |= saysNegated
		case m[4] >= 0 && strings.EqualFold(text[m[4]:m[5]], FindingApplied):
			w |= saysApplied
		case m[4] >= 0:
			w |= saysRejected
		case m[6] >= 0:
			negated = true
		default:
			negated = false
		}
	}
	return w
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
// one the loop writes: on a reviewer whose verdict is recorded, and only on
// its lane's latest round, none or as many as its count up to the bound
// (none where the writer trimmed them), each a location citations writes,
// once; a status exactly where a verified fix report answered a failing
// reviewer; and the reviewers citing each location as markCitedBy marks them.
func (s State) findingsWritten() bool {
	for _, l := range s.Lanes {
		for ri, r := range l.Validation {
			for _, v := range r.Validators {
				if len(v.Findings) > 0 && ri < len(l.Validation)-1 {
					return false
				}
				if v.Located == nil {
					if len(v.Findings) > 0 {
						return false
					}
					continue
				}
				if !slices.Contains(reviewRoles, v.Role) || v.Verdict == "" || *v.Located < 0 || *v.Located > maxLocatedCount ||
					len(v.Findings) != 0 && len(v.Findings) != min(*v.Located, maxLocatedFindings) {
					return false
				}
				answered := r.Fix != "" && !v.Pass
				seen := make(map[location]bool, len(v.Findings))
				for _, f := range v.Findings {
					if !validLocation(f.File) || f.Line < 1 || f.Line > maxFindingLine || seen[f.at()] {
						return false
					}
					seen[f.at()] = true
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

// keepLatestFindings drops, from every validation round but each lane's
// latest, its reviewers' located findings, keeping their counts: a fix brief
// and a judge read only the latest round's, and a finding kept on every
// round would grow the state by a return's worth each round.
func (st *State) keepLatestFindings() {
	for i := range st.Lanes {
		rs := st.Lanes[i].Validation
		for j := 0; j+1 < len(rs); j++ {
			dropFindings(&rs[j])
		}
	}
}

// trimOldestFindings drops the located findings of the oldest round still
// carrying any, by round number and then lane order, keeping its counts, and
// reports whether there was one. A round's findings go whole, so the
// reviewers left citing a location are the ones markCitedBy marks.
func (st *State) trimOldestFindings() bool {
	var oldest *ValidationRound
	for i := range st.Lanes {
		for j := range st.Lanes[i].Validation {
			r := &st.Lanes[i].Validation[j]
			if r.carriesFindings() && (oldest == nil || r.Round < oldest.Round) {
				oldest = r
			}
		}
	}
	if oldest == nil {
		return false
	}
	dropFindings(oldest)
	return true
}

// carriesFindings reports whether any reviewer of r keeps a located finding.
func (r ValidationRound) carriesFindings() bool {
	for _, v := range r.Validators {
		if len(v.Findings) > 0 {
			return true
		}
	}
	return false
}

// dropFindings drops every reviewer's located findings from r, keeping each
// count. The validators are copied first, so a round sharing them is not
// changed.
func dropFindings(r *ValidationRound) {
	if !r.carriesFindings() {
		return
	}
	r.Validators = slices.Clone(r.Validators)
	for k := range r.Validators {
		r.Validators[k].Findings = nil
	}
}
