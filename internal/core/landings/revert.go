package landings

// revert.go is the revert trace (spc-2610301950480609 approach 7): which rows
// a later commit on the default branch reverts, read from git's own words over
// every commit the range reaches and from the forge's own revert line in a
// squashed revert's pull request body; the positional trace of one commit to
// the row that brought it; and the reverted rows grouped by the rule that
// admitted them.
//
// A revert line is text whoever made the commit typed, so it is read to git
// revert's own shape or not at all, and the commit it names is resolved by
// git from stored objects alone, in at most two calls for the whole range,
// never guessed: a name that is ambiguous, names nothing, names no commit,
// names a commit outside the range, the reverting commit itself, a commit its
// own row brought, or a commit that is not an ancestor of the reverting one,
// marks no row and says so on the row that carries it. A git line marks only
// a strict ancestor of the commit carrying it, and a forge line only an older
// row whose pull request's answer was read; either way a mark points to an
// older row, so the reverts form no cycle, and a revert of a revert is
// followed to its end. A line naming a row's own commit reverts every commit
// that row brought. A mark changes no row's class, rule or Unattended: a
// reverted row is read as it landed, and a reverting commit is a row like any
// other; the forge is asked for a body only after every row is classed, in a
// document of its own, and nothing it answers is kept. A body is asked only
// for a row whose subject is git revert's own, so a squashed revert whose
// title was reworded is not seen at all: nothing marks the row it reverts, and
// no untraceable line says it was not read; that keeps the asking bounded and
// step 2's answers as they were. dcn-2611120948130053 records these readings
// and their grounds, as dcn-2611121039108572 and dcn-2611121125486678
// correct it.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/gitutil"
)

// The kinds of a revert line, and of a mark.
const (
	// RevertGit is git revert's own line, "This reverts commit <sha>".
	RevertGit = "revert"
	// RevertForge is the forge's own line in a squashed revert's pull request
	// body, "Reverts #N" or "Reverts <owner>/<repo>#N".
	RevertForge = "forge"
	// RevertUntraceable is a commit whose subject is git revert's, `Revert
	// "<subject>"`, with neither line: it marks nothing.
	RevertUntraceable = "untraceable"
)

// GroupUntraceable is the group the reverted groups hold untraceable reverts
// in, apart from the groups by rule.
const GroupUntraceable = "untraceable"

// The reasons a line marks no row that the positional trace widens its range
// for (olderTargets): they are matched whole, so a reason reworded here is
// reworded for both.
const (
	unmarkedOutside = "names a commit outside the listed range"
	unmarkedNoRow   = "names a pull request no row of the listing merged as the forge's answer says"
)

// SaysNotOnDefault is what the positional trace says of a commit no
// first-parent commit of the default branch brought.
const SaysNotOnDefault = "not on the default branch"

// Revert is the mark on a row a later commit reverts: the reverting commit,
// the row that brought it, the line's kind, and, where that revert is itself
// reverted by a revert that stands, the commit that withdrew it (the row's
// change is then reinstated, as RS001's withdrawn_by reads it).
type Revert struct {
	By          string `json:"by"`
	ByRow       string `json:"by_row"`
	Kind        string `json:"kind"`
	WithdrawnBy string `json:"withdrawn_by,omitempty"`
}

// RevertLine is one revert line a row's commits carry, or one untraceable
// revert: the commit carrying it (the row's own commit for the forge's line),
// its kind, the name as written (a hex name, or "#N" or "<owner>/<repo>#N"),
// the full name it resolved to, and the row it marks, or why it marks none.
type RevertLine struct {
	Commit   string `json:"commit"`
	Kind     string `json:"kind"`
	Names    string `json:"names,omitempty"`
	Target   string `json:"target,omitempty"`
	Row      string `json:"row,omitempty"`
	Unmarked string `json:"unmarked,omitempty"`
}

// maxRevertLinesPerCommit bounds the revert lines read from one commit's
// message, or one pull request's body: a squash of many reverts carries one
// line each, and the lines past the bound are counted on the row, not read.
const maxRevertLinesPerCommit = 64

// maxRevertLines bounds the git revert lines one listing follows, oldest row
// first: each is resolved and judged against the history in memory, and the
// lines past it are said on their rows, not followed. A variable only so a
// test can hold a listing to a small one.
var maxRevertLines = 1024

// The lines read. A revert line starts its line, as git revert writes it, so
// a quoted or indented one is not read; it names one commit, by seven to
// sixty-four lower-case hex characters ending the word, and only the first
// name on the line (git's merge revert goes on "reversing changes made to
// <sha>", which names the side kept, not a commit reverted). The forge's line
// is the whole line.
var (
	revertLineRe    = regexp.MustCompile(`^This reverts commit ([0-9a-f]{7,64})(?:[^0-9A-Za-z]|$)`)
	revertSubjectRe = regexp.MustCompile(`^Revert ".*"(?: \(#[1-9][0-9]{0,8}\))?$`)
	forgeLineRe     = regexp.MustCompile(`^Reverts ((?:([A-Za-z0-9][A-Za-z0-9-]{0,38}/[A-Za-z0-9._-]{1,100}))?#([1-9][0-9]{0,8}))\s*$`)
	hexNameRe       = regexp.MustCompile(`^[0-9a-f]{7,64}$`)
)

// revertNames is the commits message names in git revert's own line, in
// order, up to maxRevertLinesPerCommit, and how many lines past that bound it
// carries.
func revertNames(message string) (names []string, past int) {
	for line := range strings.SplitSeq(message, "\n") {
		m := revertLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		if len(names) == maxRevertLinesPerCommit {
			past++
			continue
		}
		names = append(names, m[1])
	}
	return names, past
}

// forgeRef is one forge revert line: as written, the repository it names (""
// for this one), and the pull request.
type forgeRef struct {
	text string
	repo string
	pr   int
}

// forgeRevertLines is the forge's revert lines a pull request's body carries,
// up to maxRevertLinesPerCommit, and how many lines past that bound.
func forgeRevertLines(body string) (refs []forgeRef, past int) {
	for line := range strings.SplitSeq(body, "\n") {
		m := forgeLineRe.FindStringSubmatch(strings.TrimSuffix(line, "\r"))
		if m == nil {
			continue
		}
		if len(refs) == maxRevertLinesPerCommit {
			past++
			continue
		}
		n, _ := strconv.Atoi(m[3])
		refs = append(refs, forgeRef{text: m[1], repo: m[2], pr: n})
	}
	return refs, past
}

// revertSubject says whether message's subject is git revert's.
func revertSubject(message string) bool {
	subject, _, _ := strings.Cut(strings.TrimLeft(message, "\n"), "\n")
	return revertSubjectRe.MatchString(strings.TrimSpace(subject))
}

// resolved is what git says a name names: a commit's full name, or why it
// names none.
type resolved struct {
	full string
	says string
}

// resolveNames asks git what each name names, as a prefix of the object names
// git stores and of nothing else, in at most two calls however many names it
// is given: one listing, for every name at once, the stored objects whose
// names begin with it (rev-parse --disambiguate=<name>, each name hex-checked
// before it is an argument, in a form no option reads), and one asking the
// type of each object named by exactly one, by its full name on standard
// input. Both read stored objects only (gitutil.RunCappedBytesAsStoredContext:
// no replace ref, graft or lazy fetch). A name begun by no stored object names
// nothing, one begun by two or more is ambiguous, and one whose type git does
// not give leaves that name alone unresolved; a git call that fails leaves
// every name asked of it unresolved, and never a line read for another name.
// Every name must be a hex name (hexNameRe) or nothing is asked. A variable
// only so a test can count its calls.
var resolveNames = func(ctx context.Context, root string, names []string) (map[string]resolved, error) {
	args := []string{"rev-parse"}
	for _, n := range names {
		if !hexNameRe.MatchString(n) {
			return nil, errors.New("a name asked of git is no hex object name")
		}
		args = append(args, "--disambiguate="+n)
	}
	got := make(map[string]resolved, len(names))
	if len(names) == 0 {
		return got, nil
	}
	out, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", maxListBytes, args...)
	if err != nil {
		return nil, err
	}
	var objects []string
	for o := range strings.FieldsSeq(string(out)) {
		if !gitutil.IsFullSHA(o) {
			return nil, errors.New("git listed the objects a name begins in a shape it does not document")
		}
		objects = append(objects, o)
	}
	// Sorted and unique, so the objects a name begins are one run.
	slices.Sort(objects)
	objects = slices.Compact(objects)
	match := make(map[string]string, len(names))
	asked := map[string]bool{}
	var ask []string
	for _, n := range names {
		i, _ := slices.BinarySearch(objects, n)
		hits := 0
		for j := i; j < len(objects) && strings.HasPrefix(objects[j], n) && hits < 2; j++ {
			hits++
		}
		switch hits {
		case 0:
			got[n] = resolved{says: "names no object"}
		case 1:
			match[n] = objects[i]
			if !asked[objects[i]] {
				asked[objects[i]] = true
				ask = append(ask, objects[i])
			}
		default:
			got[n] = resolved{says: "is ambiguous: it names more than one object, and none is guessed"}
		}
	}
	if len(ask) == 0 {
		return got, nil
	}
	in := strings.Join(ask, "\n") + "\n"
	out, err = gitutil.RunCappedBytesAsStoredContext(ctx, root, in, len(ask)*96+64,
		"cat-file", "--batch-check=%(objectname) %(objecttype)")
	if err != nil {
		return nil, err
	}
	kinds := make(map[string]string, len(ask))
	for line := range strings.SplitSeq(strings.TrimSuffix(string(out), "\n"), "\n") {
		full, kind, _ := strings.Cut(line, " ")
		if asked[full] && kinds[full] == "" {
			kinds[full] = kind
		}
	}
	for n, full := range match {
		switch kind := kinds[full]; kind {
		case "commit":
			got[n] = resolved{full: full}
		case "", "missing":
			got[n] = resolved{says: "names an object git did not say the type of"}
		default:
			got[n] = resolved{says: "names a " + boundText(kind, 16) + ", not a commit"}
		}
	}
	return got, nil
}

// revertEdge is one line that marks a row: the commit reverting, the commit
// it reverts (for the forge's line, the row it names), its kind, and where the
// line sits (rows[row].Reverts[line]).
type revertEdge struct {
	from, to  string
	kind      string
	row, line int
}

// traceReverts marks rows (the listing's, newest first, each row's commit
// rows[i].Commit) with what later commits revert, and gives each row the
// revert lines its commits carry. accepted holds the row commits whose pull
// request's answer was read and confirmed the row: only those rows are named
// by the forge's line, and only their bodies are asked for it (l.bodies), and
// of those only a row that merged with one parent, whose subject is git
// revert's, and whose commits carry no git revert line git resolved. It asks
// git at most twice (resolveNames), for the names the history alone cannot
// resolve, and the forge only for those bodies; a body that is not read
// leaves the row's revert untraceable, saying why, and changes nothing else.
func (l *lister) traceReverts(rows []Row, accepted map[string]bool) {
	h := l.h
	at := make(map[string]int, len(rows))
	byPR := map[int]int{}
	for i, r := range rows {
		at[r.Commit] = i
		if accepted[r.Commit] {
			byPR[r.PR] = i
		}
	}
	type pending struct {
		row, line    int
		name, commit string
	}
	var git []pending
	plain := make([][]string, len(rows))
	budget := maxRevertLines
	for i := len(rows) - 1; i >= 0; i-- {
		r := &rows[i]
		over := 0
		for _, c := range append([]string{r.Commit}, h.branch[r.Commit]...) {
			msg := h.commits[c].message
			names, past := revertNames(msg)
			for _, n := range names {
				if budget == 0 {
					over++
					continue
				}
				budget--
				git = append(git, pending{row: i, line: len(r.Reverts), name: n, commit: c})
				r.Reverts = append(r.Reverts, RevertLine{Commit: c, Kind: RevertGit, Names: n})
			}
			if past > 0 {
				r.Reverts = append(r.Reverts, RevertLine{Commit: c, Kind: RevertGit,
					Unmarked: fmt.Sprintf("%d more revert lines past the %d a commit is read for", past, maxRevertLinesPerCommit)})
			}
			if len(names) == 0 && past == 0 && revertSubject(msg) {
				plain[i] = append(plain[i], c)
			}
		}
		if over > 0 {
			// Counted once for the row, so lines past the bound add one line
			// to the listing, not one each.
			r.Reverts = append(r.Reverts, RevertLine{Commit: r.Commit, Kind: RevertGit,
				Unmarked: fmt.Sprintf("%d revert lines its commits carry are past the listing's bound of %d revert lines, so they are not followed", over, maxRevertLines)})
		}
	}

	asked := map[string]bool{}
	var need []string
	for _, p := range git {
		if h.commits[p.name] == nil && !asked[p.name] {
			asked[p.name] = true
			need = append(need, p.name)
		}
	}
	var res map[string]resolved
	var resErr error
	if len(need) > 0 {
		res, resErr = resolveNames(l.ctx, l.root, need)
	}
	var gen map[string]int
	var edges []revertEdge
	for _, p := range git {
		line := &rows[p.row].Reverts[p.line]
		full := p.name
		if h.commits[full] == nil {
			switch r := res[p.name]; {
			case resErr != nil:
				line.Unmarked = "cannot be resolved: " + whyText(resErr)
				continue
			case r.full == "":
				line.Unmarked = r.says
				continue
			default:
				full = r.full
			}
		}
		line.Target = full
		switch {
		case full == p.commit:
			line.Unmarked = "names the reverting commit itself"
		case h.commits[full] == nil:
			line.Unmarked = unmarkedOutside
		case h.rowOf[full] == h.rowOf[p.commit]:
			line.Unmarked = "names a commit its own row brought, so it reverts nothing that landed"
		default:
			if gen == nil {
				if gen = h.generations(l.ctx); gen == nil {
					return
				}
			}
			if !h.isAncestor(l.ctx, gen, full, p.commit) {
				if l.ctx.Err() != nil {
					return
				}
				line.Unmarked = "names a commit that is not an ancestor of the reverting one"
				continue
			}
			edges = append(edges, revertEdge{from: p.commit, to: full, kind: RevertGit, row: p.row, line: p.line})
		}
	}

	wantBody := map[int]bool{}
	var bodyAsks []prAsk
	for i := len(rows) - 1; i >= 0; i-- {
		r := rows[i]
		c := h.commits[r.Commit]
		if !accepted[r.Commit] || len(c.parents) > 1 || !revertSubject(c.message) ||
			slices.ContainsFunc(r.Reverts, func(x RevertLine) bool { return x.Kind == RevertGit && x.Target != "" }) {
			continue
		}
		wantBody[i] = true
		bodyAsks = append(bodyAsks, prAsk{pr: r.PR, commit: r.Commit})
	}
	bodies := bodyAnswers{}
	if len(bodyAsks) > 0 && l.bodies != nil {
		bodies = l.bodies(bodyAsks)
	}

	for i := len(rows) - 1; i >= 0; i-- {
		r := &rows[i]
		forgeLines, unread := false, ""
		if wantBody[i] {
			b := bodies.got[r.PR]
			if b == nil {
				unread = boundText(bodies.why[r.PR], 600)
				if unread == "" {
					unread = "it was not asked"
				}
			} else {
				refs, past := forgeRevertLines(b.Body)
				forgeLines = len(refs) > 0 || past > 0
				for _, ref := range refs {
					line := RevertLine{Commit: r.Commit, Kind: RevertForge, Names: ref.text}
					j, merged := byPR[ref.pr]
					switch {
					case ref.repo != "" && (b.BaseRepository == nil || b.BaseRepository.NameWithOwner == ""):
						line.Unmarked = "names a repository by name, and the forge's answer names no repository the pull request merged into"
					case ref.repo != "" && !strings.EqualFold(ref.repo, b.BaseRepository.NameWithOwner):
						line.Unmarked = "names another repository's pull request"
					case ref.pr == r.PR:
						line.Unmarked = "names its own pull request"
					case !merged:
						line.Unmarked = unmarkedNoRow
					case j < i:
						line.Unmarked = "names a pull request merged after it"
					default:
						edges = append(edges, revertEdge{from: r.Commit, to: rows[j].Commit, kind: RevertForge, row: i, line: len(r.Reverts)})
					}
					r.Reverts = append(r.Reverts, line)
				}
				if past > 0 {
					r.Reverts = append(r.Reverts, RevertLine{Commit: r.Commit, Kind: RevertForge,
						Unmarked: fmt.Sprintf("%d more revert lines past the %d a body is read for", past, maxRevertLinesPerCommit)})
				}
			}
		}
		bodyUnread := ", and its pull request's body, which may carry the forge's revert line, was not read: " + unread
		for _, c := range plain[i] {
			if forgeLines && c == r.Commit {
				continue
			}
			why := "its subject reads as git's revert, but no line names what it reverts"
			switch {
			case c == r.Commit && r.ForgeUnreadable != "":
				why += ", and the forge's answer for its pull request, whose body may carry the forge's revert line, was not read"
			case c == r.Commit && unread != "":
				why += bodyUnread
			}
			r.Reverts = append(r.Reverts, RevertLine{Commit: c, Kind: RevertUntraceable, Unmarked: why})
		}
		if unread != "" && !slices.Contains(plain[i], r.Commit) {
			// Its git lines resolved nothing, and the body that might have
			// named what it reverts was not read.
			r.Reverts = append(r.Reverts, RevertLine{Commit: r.Commit, Kind: RevertUntraceable,
				Unmarked: "its subject reads as git's revert, but no line it carries resolved" + bodyUnread})
		}
	}
	// Oldest reverting row first, then in the order the lines were read.
	slices.SortStableFunc(edges, func(a, b revertEdge) int { return b.row - a.row })
	l.mark(rows, at, edges)
}

// mark sets each reverted row's mark from edges, oldest reverting row first:
// the first revert of the row that stands, else the first, with the commit
// that withdrew it. A revert stands when no revert that stands reverts it. An
// edge to a row's own commit (a merge reverted by its own name, or a pull
// request named by the forge's line) reverts every commit that row brought,
// so a revert one of those commits carries is withdrawn by it too.
func (l *lister) mark(rows []Row, at map[string]int, edges []revertEdge) {
	h := l.h
	reverters := map[string][]int{}
	for i, e := range edges {
		reverters[e.to] = append(reverters[e.to], i)
		if h.onLine[e.to] {
			for _, c := range h.branch[e.to] {
				reverters[c] = append(reverters[c], i)
			}
		}
	}
	memo := map[string]int8{}
	var standing func(c string) bool
	standing = func(c string) bool {
		if v, ok := memo[c]; ok {
			// 2 is a commit being judged: no edge forms a cycle (each goes to
			// a commit an older row brought), and were one to, it reads as
			// standing.
			return v != 0
		}
		memo[c] = 2
		s := true
		for _, e := range reverters[c] {
			if standing(edges[e].from) {
				s = false
				break
			}
		}
		memo[c] = 0
		if s {
			memo[c] = 1
		}
		return s
	}
	withdrawnBy := func(c string) string {
		for _, e := range reverters[c] {
			if standing(edges[e].from) {
				return edges[e].from
			}
		}
		return ""
	}
	chosen := map[int]revertEdge{}
	for _, e := range edges {
		target := at[h.rowOf[e.to]]
		rows[e.row].Reverts[e.line].Row = rows[target].Commit
		prev, ok := chosen[target]
		if !ok || (!standing(prev.from) && standing(e.from)) {
			chosen[target] = e
		}
	}
	for target, e := range chosen {
		m := &Revert{By: e.from, ByRow: rows[e.row].Commit, Kind: e.kind}
		if !standing(e.from) {
			m.WithdrawnBy = withdrawnBy(e.from)
		}
		rows[target].Reverted = m
	}
}

// ctxCheckSteps is how many steps generations and isAncestor take between
// looks at the listing's ctx: both run in memory, after git has answered, so
// the ctx is all that bounds them in time.
const ctxCheckSteps = 4096

// generations is each commit's generation within the range: one more than
// its highest parent in the range, 1 for a commit with none there, so an
// ancestor's generation is always below its descendant's. One pass, without
// recursion, each commit and parent link once; a parent met while still being
// judged (a cycle git cannot store) is read as outside the range. nil when ctx
// ends first.
func (h *history) generations(ctx context.Context) map[string]int {
	gen := make(map[string]int, len(h.commits))
	open := map[string]bool{}
	steps := 0
	for start := range h.commits {
		if gen[start] != 0 {
			continue
		}
		stack := []string{start}
		for len(stack) > 0 {
			if steps++; steps%ctxCheckSteps == 0 && ctx.Err() != nil {
				return nil
			}
			c := stack[len(stack)-1]
			if gen[c] != 0 {
				stack = stack[:len(stack)-1]
				continue
			}
			open[c] = true
			ready, g := true, 0
			for _, p := range h.commits[c].parents {
				switch {
				case h.commits[p] == nil:
				case gen[p] != 0:
					g = max(g, gen[p])
				case open[p]:
					// Being judged below c on the stack: in a history git
					// stores, only a cycle puts a parent there.
				default:
					ready = false
					stack = append(stack, p)
				}
			}
			if ready {
				gen[c] = g + 1
				delete(open, c)
				stack = stack[:len(stack)-1]
			}
		}
	}
	return gen
}

// isAncestor says whether a is a strict ancestor of c within the range. Every
// commit between them is in the range too (one reachable from the range's
// From would make a reachable as well), so the walk stays in the history read,
// and skips every commit no newer than a by generation: it visits each commit
// of the range at most once, and a listing walks at most maxRevertLines times.
// It answers false once ctx ends, which the caller reads from ctx.
func (h *history) isAncestor(ctx context.Context, gen map[string]int, a, c string) bool {
	if a == c || gen[a] >= gen[c] {
		return false
	}
	seen := map[string]bool{c: true}
	stack := []string{c}
	for steps := 1; len(stack) > 0; steps++ {
		if steps%ctxCheckSteps == 0 && ctx.Err() != nil {
			return false
		}
		x := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, p := range h.commits[x].parents {
			if p == a {
				return true
			}
			if h.commits[p] == nil || seen[p] || gen[p] <= gen[a] {
				continue
			}
			seen[p] = true
			stack = append(stack, p)
		}
	}
	return false
}

// RevertGroup is one group of reverted rows: the rule that admitted them (as
// Rule.String names it, "person-push" for a person's push), or
// GroupUntraceable for the rows carrying an untraceable revert; how many; and
// the rows, newest first.
type RevertGroup struct {
	Rule  string `json:"rule"`
	Count int    `json:"count"`
	Rows  []Row  `json:"rows"`
}

// RevertedGroups is l's reverted rows grouped by rule, each group where its
// newest row falls, and the untraceable reverts in a group of their own, last.
// A row whose revert was withdrawn is grouped with the rest: its change was
// reverted, and its mark says it was reinstated. A group is the rule's, not
// the class's, so one may hold attended and unattended rows (a row a person
// approved under auto shares the auto group with the automatic rows); each
// row carries its own Unattended, which a reader splits the group by.
func RevertedGroups(l Listing) []RevertGroup {
	var groups []RevertGroup
	at := map[string]int{}
	var untraceable []Row
	for _, r := range l.Rows {
		if r.Reverted != nil {
			key := r.Rule.String()
			if r.Class == ClassPersonPush {
				key = ClassPersonPush
			}
			i, ok := at[key]
			if !ok {
				i = len(groups)
				at[key] = i
				groups = append(groups, RevertGroup{Rule: key})
			}
			groups[i].Rows = append(groups[i].Rows, r)
		}
		if slices.ContainsFunc(r.Reverts, func(x RevertLine) bool { return x.Kind == RevertUntraceable }) {
			untraceable = append(untraceable, r)
		}
	}
	if len(untraceable) > 0 {
		groups = append(groups, RevertGroup{Rule: GroupUntraceable, Rows: untraceable})
	}
	for i := range groups {
		groups[i].Count = len(groups[i].Rows)
	}
	return groups
}

// Trace is what the positional trace says of one commit: its full name; what
// stops it having a row (SaysNotOnDefault, or why the name names no commit);
// the row that brought it onto the default branch, as a listing over the
// range From says classes and marks it; and the rows that row's revert lines
// mark in that listing, each classed and marked as the listing has it.
// RangeNote says why the range may not hold a row one of its lines names, ""
// when it holds every one. Tip and To are the tip read, and the notes are the
// listing's.
//
// A traced row reads as a listing over a longer range reads it but in these
// ways. The reverted rows' own revert lines are read over the trace's range,
// which may start at them, so their lines can say "outside the listed range".
// A line past the maxTraceTargets the trace looks for keeps its reason, which
// then says the bound was reached. And every bound a listing holds is the
// listing's own and spent oldest row first: the maxRevertLines git lines it
// follows, and the maxHalvingCalls of asking again that its pull request
// batches and then its bodies share. So a listing from an older start can
// leave unfollowed or unread a line or a body that the trace's shorter range
// follows or reads, and the widened listing can do the same to the traced
// row's own lines.
type Trace struct {
	Commit         string `json:"commit,omitempty"`
	Says           string `json:"says,omitempty"`
	Row            *Row   `json:"row,omitempty"`
	Reverts        []Row  `json:"reverts,omitempty"`
	Tip            string `json:"tip"`
	To             string `json:"to"`
	From           string `json:"from,omitempty"`
	RangeNote      string `json:"range_note,omitempty"`
	ForgeUnread    string `json:"forge_unread,omitempty"`
	CacheUnwritten string `json:"cache_unwritten,omitempty"`
	ModeUnread     string `json:"mode_unread,omitempty"`
}

// maxTraceTargets bounds the rows the positional trace looks for before it
// widens its range: the distinct commits outside its first range, and the
// distinct pull requests no row of it merged, that the traced row's lines
// name. One git call looks for each commit, and one for every pull request.
const maxTraceTargets = 16

// The bounds on the older commits' messages the trace reads to find the row
// that merged a pull request a forge line names: the most commits, and the most
// bytes, read. Past either, RangeNote says so and the range is not widened for
// them. Variables only so a test can hold a trace to small ones.
var (
	maxOlderCommits = 4096
	maxOlderBytes   = maxListBytes
)

// traceGit is the positional trace's own reads of git, each as stored
// (gitutil.RunCappedBytesAsStoredContext: no replace ref or graft). A
// variable only so a test can count them.
var traceGit = gitutil.RunCappedBytesAsStoredContext

// TraceCommit is the positional trace (the spec's Trace): name, a full or
// abbreviated hex name, resolved by git from stored objects alone, followed to
// the first-parent commit of the default branch's tip, as last fetched, that
// brought it, however far back. Its row is read from a listing, so it is
// classed and marked as the listing classes and marks it, the forge asked as
// the listing asks it, and its reverts are the rows that listing's lines of
// that row mark, by the listing's own rules. The listing runs from the traced
// row to the tip; where the row's lines name a commit before it, or a pull
// request no row of it merged, the range is widened once to the oldest row
// they name (at most maxTraceTargets looked for), and RangeNote says what it
// could not place. So a trace reads at most two listings, and beside them asks
// git, before the first: the default branch's ref questions
// (gitutil.DefaultRefContext), the tip's commit (commitOf), the name's
// objects and their types (resolveNames, at most two calls), the first-parent
// line (firstParentLine) and the traced commit's ancestry path (rowOn); and
// between the two: one ancestry path for each commit looked for (at most
// maxTraceTargets) and one log of the older commits' messages. Every one of
// them but the ref questions, as every call of the listings', reads objects as
// stored. The ref questions read refs, the type of the object each names and,
// for a tag, of the object it points at, and peel a ref to its commit as git
// resolves it, only to choose the default branch's ref, whose tip commitOf
// then reads as stored. It fetches nothing, writes only the listing's forge
// cache, and binds every call to ctx as List does.
func TraceCommit(ctx context.Context, repoRoot, name string, o Options) (Trace, error) {
	ctx, cancel := bounded(ctx)
	defer cancel()
	t, err := traceCommit(ctx, repoRoot, name, o)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return Trace{}, ctxErr
	}
	return t, err
}

// traceCommit is trace's work, under the ctx trace bounds.
func traceCommit(ctx context.Context, repoRoot, name string, o Options) (Trace, error) {
	ref := gitutil.DefaultRefContext(ctx, repoRoot)
	if ref == "" {
		return Trace{}, ErrNoDefaultBranch
	}
	to, err := commitOf(ctx, repoRoot, ref)
	if err != nil {
		return Trace{}, ErrNoDefaultBranch
	}
	t := Trace{Tip: gitutil.ShortRef(ref), To: to}
	if !hexNameRe.MatchString(name) {
		t.Says = "is no object name: a commit is named by seven to sixty-four lower-case hex characters"
		return t, nil
	}
	res, err := resolveNames(ctx, repoRoot, []string{name})
	if err != nil {
		return Trace{}, masked("resolving the commit named", err)
	}
	if res[name].full == "" {
		t.Says = res[name].says
		return t, nil
	}
	t.Commit = res[name].full
	line, err := firstParentLine(ctx, repoRoot, to)
	if err != nil {
		return Trace{}, err
	}
	idx := make(map[string]int, len(line))
	for i, c := range line {
		idx[c.sha] = i
	}
	own, err := rowOn(ctx, repoRoot, to, t.Commit, line, idx)
	if err != nil {
		return Trace{}, err
	}
	if own < 0 {
		t.Says = SaysNotOnDefault
		return t, nil
	}
	from := func(i int) string {
		if i+1 < len(line) {
			return line[i+1].sha
		}
		return ""
	}

	l, err := List(ctx, repoRoot, Range{From: from(own), To: to, Tip: ref}, o)
	if err != nil {
		return Trace{}, err
	}
	r, err := rowIn(l, line[own].sha)
	if err != nil {
		return Trace{}, err
	}
	oldest, note, past, err := olderTargets(ctx, repoRoot, to, line, idx, own, r)
	if err != nil {
		return Trace{}, err
	}
	t.RangeNote = note
	if oldest > own {
		if l, err = List(ctx, repoRoot, Range{From: from(oldest), To: to, Tip: ref}, o); err != nil {
			return Trace{}, err
		}
		if r, err = rowIn(l, line[own].sha); err != nil {
			return Trace{}, err
		}
	}
	t.From = l.From
	t.ForgeUnread, t.CacheUnwritten, t.ModeUnread = l.ForgeUnread, l.CacheUnwritten, l.ModeUnread
	r.Reverts = slices.Clone(r.Reverts)
	for i := range r.Reverts {
		if past.holds(r.Reverts[i]) {
			r.Reverts[i].Unmarked += fmt.Sprintf("; the trace looks for at most %d such rows, a bound this line is past, so its range was not widened to it", maxTraceTargets)
		}
	}
	t.Row = &r
	var marked []string
	for _, x := range r.Reverts {
		if x.Row != "" && !slices.Contains(marked, x.Row) {
			marked = append(marked, x.Row)
		}
	}
	for _, c := range marked {
		m, err := rowIn(l, c)
		if err != nil {
			return Trace{}, err
		}
		t.Reverts = append(t.Reverts, m)
	}
	return t, nil
}

// rowIn is l's row for commit.
func rowIn(l Listing, commit string) (Row, error) {
	for _, r := range l.Rows {
		if r.Commit == commit {
			return r, nil
		}
	}
	return Row{}, errors.New("the listing for the trace holds no row for a commit it names")
}

// pastTargets are the commits and pull requests a traced row's lines name
// past the maxTraceTargets the trace looks for.
type pastTargets struct {
	commits map[string]bool
	prs     map[int]bool
}

// widenable is the target x names that the trace widens its range for: a
// commit a git line resolved to outside the listing's range, or a pull request
// a forge line names that no row of it merged, by the line's reason, matched
// whole (unmarkedOutside, unmarkedNoRow); "" and 0 for any other line.
func widenable(x RevertLine) (commit string, pr int) {
	switch {
	case x.Kind == RevertGit && x.Target != "" && x.Unmarked == unmarkedOutside:
		return x.Target, 0
	case x.Kind == RevertForge && x.Unmarked == unmarkedNoRow:
		_, num, _ := strings.Cut(x.Names, "#")
		n, _ := strconv.Atoi(num)
		return "", n
	}
	return "", 0
}

// holds says whether x names a target past the bound, read as widenable
// reads it.
func (p pastTargets) holds(x RevertLine) bool {
	c, n := widenable(x)
	return (c != "" && p.commits[c]) || (n != 0 && p.prs[n])
}

// olderTargets is the index on line of the oldest row older than own that
// r's revert lines name and the listing from own could not read (widenable);
// own when there is none. note says what it could not place: a commit no
// first-parent commit brought, a pull request no older first-parent commit's
// subject names, or one the older commits' messages, read to their bounds
// (maxOlderCommits, maxOlderBytes), did not reach; that fewer older commits
// were read than exist, whenever they were, even with every pull request
// placed; and the targets past
// maxTraceTargets, which past holds. The subject is each message's first line,
// as the listing reads it (subjectPR).
func olderTargets(ctx context.Context, repoRoot, to string, line []lineCommit, idx map[string]int, own int, r Row) (int, string, pastTargets, error) {
	var commits []string
	var prs []int
	past := pastTargets{commits: map[string]bool{}, prs: map[int]bool{}}
	for _, x := range r.Reverts {
		c, pr := widenable(x)
		switch {
		case c == "" && pr == 0, slices.Contains(commits, c), slices.Contains(prs, pr),
			past.commits[c], past.prs[pr]:
			continue
		case len(commits)+len(prs) == maxTraceTargets && pr != 0:
			past.prs[pr] = true
		case len(commits)+len(prs) == maxTraceTargets:
			past.commits[c] = true
		case pr != 0:
			prs = append(prs, pr)
		default:
			commits = append(commits, c)
		}
	}
	oldest := own
	var notes []string
	if n := len(past.commits) + len(past.prs); n > 0 {
		notes = append(notes, fmt.Sprintf("%d more commits or pull requests its lines name are past the %d looked for, so the range may not hold their rows", n, maxTraceTargets))
	}
	off := 0
	for _, c := range commits {
		i, err := rowOn(ctx, repoRoot, to, c, line, idx)
		if err != nil {
			return 0, "", past, err
		}
		if i < 0 {
			off++
		}
		oldest = max(oldest, i)
	}
	if off > 0 {
		notes = append(notes, fmt.Sprintf("%d commits its lines name are on no first-parent commit of the default branch", off))
	}
	if len(prs) == 0 {
		return oldest, strings.Join(notes, "; "), past, nil
	}
	older := len(line) - own - 1
	if older <= 0 {
		notes = append(notes, fmt.Sprintf("%d pull requests its lines name are named by no older first-parent commit's subject", len(prs)))
		return oldest, strings.Join(notes, "; "), past, nil
	}
	out, err := traceGit(ctx, repoRoot, "", maxOlderBytes,
		"log", "-z", "--first-parent", "--no-show-signature", "--max-count="+strconv.Itoa(maxOlderCommits), "--format=%H%x1f%B", line[own+1].sha, "--")
	if err != nil {
		if ctx.Err() != nil {
			return 0, "", past, ctx.Err()
		}
		notes = append(notes, fmt.Sprintf("%d pull requests its lines name were not looked for: the older first-parent commits' messages were not read (%s)", len(prs), whyText(err)))
		return oldest, strings.Join(notes, "; "), past, nil
	}
	found := map[int]bool{}
	for rec := range bytes.SplitSeq(out, []byte{0}) {
		sha, message, ok := strings.Cut(strings.TrimPrefix(string(rec), "\n"), "\x1f")
		i, on := idx[sha]
		if !ok || !on {
			continue
		}
		if n := subjectPR(message); n != 0 && slices.Contains(prs, n) {
			// The oldest row naming it, as the listing's own byPR reads it.
			found[n] = true
			oldest = max(oldest, i)
		}
	}
	// Said whenever more older commits exist than were read, whether or not
	// every pull request was placed: a row past them may be the oldest that
	// merged one, which the listing's own byPR would take.
	if older > maxOlderCommits {
		notes = append(notes, fmt.Sprintf("the trace read the messages of %d of the %d older first-parent commits, the most it reads, so a row older than those it read may have merged a pull request its lines name", maxOlderCommits, older))
	}
	switch missing := len(prs) - len(found); {
	case missing > 0 && older > maxOlderCommits:
		notes = append(notes, fmt.Sprintf("%d pull requests its lines name are named by no subject of the %d older first-parent commits the trace read", missing, maxOlderCommits))
	case missing > 0:
		notes = append(notes, fmt.Sprintf("%d pull requests its lines name are named by no older first-parent commit's subject", missing))
	}
	return oldest, strings.Join(notes, "; "), past, nil
}

// rowOn is the index on line (to's first-parent line, newest first, idx its
// index by commit) of the commit that brought sha onto it: sha's own where it
// is on the line, else the oldest commit on the line that sha is an ancestor
// of, which brought it through a parent other than its first; -1 where to does
// not reach sha. It asks git one question, sha..to's ancestry path, read
// against the line.
func rowOn(ctx context.Context, repoRoot, to, sha string, line []lineCommit, idx map[string]int) (int, error) {
	if sha == to {
		return 0, nil
	}
	if !gitutil.IsFullSHA(sha) || !gitutil.IsFullSHA(to) {
		return 0, errors.New("the trace names a commit by no full object name")
	}
	out, err := traceGit(ctx, repoRoot, "", maxListBytes, "rev-list", "--ancestry-path", sha+".."+to, "--")
	if err != nil {
		return 0, masked("finding the first-parent commit that brought the traced commit", err)
	}
	oldest := -1
	for c := range strings.FieldsSeq(string(out)) {
		if i, ok := idx[c]; ok {
			oldest = max(oldest, i)
		}
	}
	if oldest < 0 {
		return -1, nil
	}
	if oldest+1 < len(line) && line[oldest+1].sha == sha {
		return oldest + 1, nil
	}
	return oldest, nil
}
