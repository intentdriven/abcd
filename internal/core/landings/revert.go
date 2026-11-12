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
// git from stored objects, in one call for the whole range, never guessed: a
// name that is ambiguous, names nothing, names no commit, names a commit
// outside the range, the reverting commit itself, a commit its own row
// brought, or a commit that is not an ancestor of the reverting one, marks no
// row and says so on the row that carries it. Since every line that marks a
// row names a strict ancestor of the commit carrying it, the reverts form no
// cycle, and a revert of a revert is followed to its end. A mark changes no
// row's class, rule or Unattended: a reverted row is read as it landed, and a
// reverting commit is a row like any other. dcn-2611120948130053 records each
// of these readings and its grounds.

import (
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
// nothing, one begun by two or more is ambiguous, and an answer that does not
// read for one name leaves that name alone unresolved. Every name must be a
// hex name (hexNameRe) or nothing is asked. A variable only so a test can
// count its calls.
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
// revert lines its commits carry. accepted holds, by row commit, the forge's
// answer each row's pull request was read for and confirmed by; only those
// rows' bodies are read for the forge's line, and only those rows are named by
// it. It asks git once, for the names the history alone cannot resolve.
func (l *lister) traceReverts(rows []Row, accepted map[string]*prAnswer) {
	h := l.h
	at := make(map[string]int, len(rows))
	byPR := map[int]int{}
	for i, r := range rows {
		at[r.Commit] = i
		if accepted[r.Commit] != nil {
			byPR[r.PR] = i
		}
	}
	type pending struct {
		row, line    int
		name, commit string
	}
	var git []pending
	var edges []revertEdge
	budget := maxRevertLines
	for i := len(rows) - 1; i >= 0; i-- {
		r := &rows[i]
		var plain []string
		for _, c := range append([]string{r.Commit}, h.branch[r.Commit]...) {
			msg := h.commits[c].message
			names, past := revertNames(msg)
			for _, n := range names {
				line := RevertLine{Commit: c, Kind: RevertGit, Names: n}
				if budget == 0 {
					line.Unmarked = fmt.Sprintf("past the listing's bound of %d revert lines, so it is not followed", maxRevertLines)
				} else {
					budget--
					git = append(git, pending{row: i, line: len(r.Reverts), name: n, commit: c})
				}
				r.Reverts = append(r.Reverts, line)
			}
			if past > 0 {
				r.Reverts = append(r.Reverts, RevertLine{Commit: c, Kind: RevertGit,
					Unmarked: fmt.Sprintf("%d more revert lines past the %d a commit is read for", past, maxRevertLinesPerCommit)})
			}
			if len(names) == 0 && past == 0 && revertSubject(msg) {
				plain = append(plain, c)
			}
		}
		forgeLines := false
		if a := accepted[r.Commit]; a != nil && len(h.commits[r.Commit].parents) < 2 {
			refs, past := forgeRevertLines(a.Body)
			forgeLines = len(refs) > 0 || past > 0
			for _, ref := range refs {
				line := RevertLine{Commit: r.Commit, Kind: RevertForge, Names: ref.text}
				j, merged := byPR[ref.pr]
				switch {
				case ref.repo != "" && (a.BaseRepository == nil || !strings.EqualFold(ref.repo, a.BaseRepository.NameWithOwner)):
					line.Unmarked = "names another repository's pull request"
				case ref.pr == r.PR:
					line.Unmarked = "names its own pull request"
				case !merged:
					line.Unmarked = "names a pull request no row of the listing merged as the forge's answer says"
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
		for _, c := range plain {
			if forgeLines && c == r.Commit {
				continue
			}
			why := "its subject reads as git's revert, but no line names what it reverts"
			if c == r.Commit && r.ForgeUnreadable != "" {
				why += ", and the forge's answer for its pull request, whose body may carry the forge's revert line, was not read"
			}
			r.Reverts = append(r.Reverts, RevertLine{Commit: c, Kind: RevertUntraceable, Unmarked: why})
		}
	}

	var need []string
	for _, p := range git {
		if h.commits[p.name] == nil && !slices.Contains(need, p.name) {
			need = append(need, p.name)
		}
	}
	var res map[string]resolved
	var resErr error
	if len(need) > 0 {
		res, resErr = resolveNames(l.ctx, l.root, need)
	}
	var gen map[string]int
	var gitEdges []revertEdge
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
			line.Unmarked = "names a commit outside the listed range"
		case h.rowOf[full] == h.rowOf[p.commit]:
			line.Unmarked = "names a commit its own row brought, so it reverts nothing that landed"
		default:
			if gen == nil {
				gen = h.generations()
			}
			if !h.isAncestor(gen, full, p.commit) {
				line.Unmarked = "names a commit that is not an ancestor of the reverting one"
				continue
			}
			gitEdges = append(gitEdges, revertEdge{from: p.commit, to: full, kind: RevertGit, row: p.row, line: p.line})
		}
	}
	edges = append(gitEdges, edges...)
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

// generations is each commit's generation within the range: one more than
// its highest parent in the range, 1 for a commit with none there, so an
// ancestor's generation is always below its descendant's. One pass, without
// recursion; a parent met while still being judged (a cycle git cannot store)
// is read as outside the range.
func (h *history) generations() map[string]int {
	gen := make(map[string]int, len(h.commits))
	open := map[string]bool{}
	for start := range h.commits {
		if gen[start] != 0 {
			continue
		}
		stack := []string{start}
		for len(stack) > 0 {
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
// and skips every commit no newer than a by generation.
func (h *history) isAncestor(gen map[string]int, a, c string) bool {
	if a == c || gen[a] >= gen[c] {
		return false
	}
	seen := map[string]bool{c: true}
	stack := []string{c}
	for len(stack) > 0 {
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

// revertedGroups is l's reverted rows grouped by rule, each group where its
// newest row falls, and the untraceable reverts in a group of their own, last.
// A row whose revert was withdrawn is grouped with the rest: its change was
// reverted, and its mark says it was reinstated. It stays unexported for this
// step: step 4 exports it when it wires the verb.
func revertedGroups(l Listing) []RevertGroup {
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
// the row that brought it onto the default branch, as a listing from that row
// to the tip classes and marks it; and, for a commit whose message reverts
// others, the rows that brought each commit it names that is on the default
// branch. Tip and To are the tip read, and the notes are the listing's.
type Trace struct {
	Commit         string `json:"commit,omitempty"`
	Says           string `json:"says,omitempty"`
	Row            *Row   `json:"row,omitempty"`
	Reverts        []Row  `json:"reverts,omitempty"`
	Tip            string `json:"tip"`
	To             string `json:"to"`
	ForgeUnread    string `json:"forge_unread,omitempty"`
	CacheUnwritten string `json:"cache_unwritten,omitempty"`
	ModeUnread     string `json:"mode_unread,omitempty"`
}

// maxMessageBytes bounds the traced commit's message.
const maxMessageBytes = 4 << 20

// trace is the positional trace (the spec's Trace): name, a full or
// abbreviated hex name, resolved by git, followed to the first-parent commit
// of the default branch's tip, as last fetched, that brought it, however far
// back. It reads the listing from the oldest row it needs (that row's, or a
// reverted commit's row) to the tip, so its rows are classed and marked as
// the listing classes and marks them, the forge asked as the listing asks it.
// It fetches nothing, writes only the listing's forge cache, and binds every
// call to ctx as list does. It stays unexported for this step: step 4 exports
// it when it wires the verb.
func trace(ctx context.Context, repoRoot, name string, o Options) (Trace, error) {
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

	// The rows of the commits its own message reverts, on the default branch.
	raw, err := gitutil.RunCappedBytesContext(ctx, repoRoot, maxMessageBytes, "log", "-1", "--no-show-signature", "--format=%B", t.Commit, "--")
	if err != nil {
		return Trace{}, masked("reading the traced commit's message", err)
	}
	oldest := own
	var reverted []int
	if names, _ := revertNames(string(raw)); len(names) > 0 {
		got, err := resolveNames(ctx, repoRoot, names)
		if err != nil {
			return Trace{}, masked("resolving the commits the traced commit reverts", err)
		}
		for _, n := range names {
			full := got[n].full
			if full == "" || full == t.Commit {
				continue
			}
			i, err := rowOn(ctx, repoRoot, to, full, line, idx)
			if err != nil {
				return Trace{}, err
			}
			if i >= 0 && i != own && !slices.Contains(reverted, i) {
				reverted = append(reverted, i)
				oldest = max(oldest, i)
			}
		}
	}

	from := ""
	if oldest+1 < len(line) {
		from = line[oldest+1].sha
	}
	l, err := list(ctx, repoRoot, Range{From: from, To: to, Tip: ref}, o)
	if err != nil {
		return Trace{}, err
	}
	t.ForgeUnread, t.CacheUnwritten, t.ModeUnread = l.ForgeUnread, l.CacheUnwritten, l.ModeUnread
	byCommit := make(map[string]int, len(l.Rows))
	for i, r := range l.Rows {
		byCommit[r.Commit] = i
	}
	pick := func(i int) (*Row, error) {
		j, ok := byCommit[line[i].sha]
		if !ok {
			return nil, errors.New("the listing from the traced commit's row holds no row for it")
		}
		r := l.Rows[j]
		return &r, nil
	}
	if t.Row, err = pick(own); err != nil {
		return Trace{}, err
	}
	for _, i := range reverted {
		r, err := pick(i)
		if err != nil {
			return Trace{}, err
		}
		t.Reverts = append(t.Reverts, *r)
	}
	return t, nil
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
	out, err := gitutil.RunCappedContext(ctx, repoRoot, maxListBytes, "rev-list", "--ancestry-path", sha+".."+to, "--")
	if err != nil {
		return 0, masked("finding the first-parent commit that brought the traced commit", err)
	}
	oldest := -1
	for c := range strings.FieldsSeq(out) {
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
