package embargo

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The kinds of finding the outbound check reports (adr-2610120558328615,
// decision 3).
const (
	KindAdvisoryID         = "advisory_id"          // an unpublished advisory's GHSA id
	KindReservedCVE        = "reserved_cve_id"      // a CVE id reserved for one
	KindAdvisoryTitle      = "advisory_title"       // its title, unless it was moved to the public track
	KindStorePath          = "store_path"           // where the private records live
	KindUnlistedAdvisoryID = "unlisted_advisory_id" // an advisory id the store does not hold, not --allow'ed
)

// maxCheckText caps one text the check reads (a file, stdin, a commit
// message); maxCheckGit caps one git answer (a range's commit list, one
// commit's patch). Over either, the check refuses rather than read part.
const (
	maxCheckText = 1 << 20
	maxCheckGit  = 64 << 20
)

// staleIntake is how old an intake may be before the check warns that the
// deny set may miss an advisory reported since.
const staleIntake = 24 * time.Hour

// snippetRunes caps a finding's masked line.
const snippetRunes = 200

// minTitleRunes is the shortest title the check matches on its own. A title
// of a word or two ("Parser bug") would match ordinary prose everywhere, so
// below this length only the advisory's id and reserved CVE id are refused;
// the command page names the limit.
const minTitleRunes = 12

// ghsaShapeRe is any text of a GitHub advisory id's shape, in the lower case
// the check matches in.
var ghsaShapeRe = regexp.MustCompile(`ghsa-[2-9cfghjmpqrvwx]{4}-[2-9cfghjmpqrvwx]{4}-[2-9cfghjmpqrvwx]{4}`)

// CheckRequest is one outbound check. Range is a commit range `<a>..<b>`
// whose messages and added lines are read; each of Files is a path, or "-"
// for Stdin. Allow names advisory ids of other projects, already public, that
// may appear. Now is the clock; nil is the system clock.
type CheckRequest struct {
	Home     string
	RepoRoot string
	Range    string
	Files    []string
	Stdin    io.Reader
	Allow    []string
	Now      func() time.Time
}

// CheckFinding is one place the check refuses. It names where the match is
// (Source, Where, Line) and never repeats it: Snippet is the line with every
// matched span starred out, whole, the way the scanner masks an identifier.
type CheckFinding struct {
	Kind    string `json:"kind"`
	Source  string `json:"source"`
	Where   string `json:"where,omitempty"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

// CheckResult is what the check read and found. UnderEmbargo counts the
// advisories whose id is refused; IntakeStale warns that the last intake is
// more than a day old, so an advisory reported since is not yet in the set.
// The result never carries the store's location.
type CheckResult struct {
	NoStore      bool           `json:"no_store"`
	Note         string         `json:"note,omitempty"`
	UnderEmbargo int            `json:"under_embargo"`
	IntakeAt     *time.Time     `json:"intake_at"`
	IntakeStale  bool           `json:"intake_stale"`
	Commits      int            `json:"commits"`
	Files        int            `json:"files"`
	Allowed      []string       `json:"allowed"`
	Findings     []CheckFinding `json:"findings"`
}

// Clean reports a check with nothing to refuse.
func (r CheckResult) Clean() bool { return len(r.Findings) == 0 }

// needle is one entry of the deny set, normalised the way the text is.
type needle struct {
	text     string
	kind     string
	boundary bool // the match must not continue a word on either side
	pathEnd  bool // the match must not continue a path segment's name after it
}

// denySet is what the check refuses, built from the store's state.
type denySet struct {
	needles  []needle
	held     map[string]bool // every advisory id the store holds, canonGHSA's form
	allowed  map[string]bool // --allow'ed ids, canonGHSA's form
	unlisted bool            // refuse an advisory id the store does not hold
	embargo  int
}

// Check is the outbound embargo check. It reads the commit range and the
// files, and reports every unpublished advisory's id, its reserved CVE id, its
// title (unless the advisory was moved to the public track, which is fixed in
// the open) and the store's location, plus any advisory id the store does not
// hold that --allow does not name. Inputs are read before the store, so a
// refusal to read one is the same with or without a store. With no store on
// this machine nothing is under embargo, and the check passes saying so.
func Check(req CheckRequest) (CheckResult, error) {
	now := clock(req.Now)
	if strings.TrimSpace(req.Range) == "" && len(req.Files) == 0 {
		return CheckResult{}, refuse("there is nothing to check", "name a commit range with --range <a>..<b>, or a text with --file <path> (- for stdin)")
	}
	top, err := gitutil.Toplevel(req.RepoRoot)
	if err != nil {
		return CheckResult{}, refuse("this is not inside a git checkout", "run it from the repository's checkout")
	}
	allowed, allowList, err := parseAllow(req.Allow)
	if err != nil {
		return CheckResult{}, err
	}
	texts, err := readCheckFiles(req.Files, req.Stdin)
	if err != nil {
		return CheckResult{}, err
	}
	commits, err := readRange(top, req.Range)
	if err != nil {
		return CheckResult{}, err
	}
	res := CheckResult{Allowed: allowList, Findings: []CheckFinding{}, Commits: len(commits), Files: len(texts)}

	p, err := StorePath(req.Home, top)
	if err != nil {
		return CheckResult{}, err
	}
	switch _, err := os.Lstat(p); {
	case errors.Is(err, fs.ErrNotExist):
		res.NoStore = true
		res.Note = "nothing is under embargo on this machine: there is no embargo store for this repository"
		return res, nil
	case err != nil:
		return CheckResult{}, refuse("the embargo store's folder could not be examined", "check the permissions under abcd's home")
	}
	s, st, err := OpenExisting(req.Home, top)
	if err != nil {
		return CheckResult{}, err
	}
	ds, err := buildDenySet(s, st, allowed)
	if err != nil {
		return CheckResult{}, err
	}
	res.UnderEmbargo = ds.embargo
	res.IntakeAt = st.IntakeAt
	res.IntakeStale = st.IntakeAt == nil || now.Sub(*st.IntakeAt) > staleIntake

	for _, c := range commits {
		res.Findings = append(res.Findings, ds.scan(c.message, "commit "+c.sha, "message", 1)...)
		for _, add := range c.added {
			where := "added line in " + ds.nameOr(add.path, "a file the commit changes")
			for _, run := range add.runs {
				res.Findings = append(res.Findings, ds.scan(run.text, "commit "+c.sha, where, run.line)...)
			}
			named := ds.scan(add.path, "commit "+c.sha, "a file name the commit adds", 0)
			for i := range named {
				named[i].Snippet = ""
			}
			res.Findings = append(res.Findings, named...)
		}
	}
	for i, t := range texts {
		src := "stdin"
		if t.name != "-" {
			src = "file " + ds.nameOr(t.name, "argument "+strconv.Itoa(i+1))
		}
		res.Findings = append(res.Findings, ds.scan(t.body, src, "", 1)...)
	}
	return res, nil
}

// canonGHSA is an advisory id in GitHub's own spelling, however it was
// typed: the GHSA prefix upper case, the three groups lower case.
func canonGHSA(id string) string {
	id = strings.TrimSpace(id)
	if len(id) < 5 || !strings.EqualFold(id[:5], "GHSA-") {
		return id
	}
	return "GHSA-" + strings.ToLower(id[5:])
}

// parseAllow reads the --allow ids: each of an advisory id's shape, returned
// as a set and as a sorted list to print.
func parseAllow(ids []string) (map[string]bool, []string, error) {
	set := map[string]bool{}
	for _, id := range ids {
		up := canonGHSA(id)
		if !ValidGHSAID(up) {
			return nil, nil, refuse("an --allow value is not a GitHub advisory id", "pass each as GHSA-xxxx-xxxx-xxxx")
		}
		set[up] = true
	}
	list := make([]string, 0, len(set))
	for id := range set {
		list = append(list, id)
	}
	slices.Sort(list)
	return set, list, nil
}

type checkText struct {
	name string
	body string
}

// readCheckFiles reads each named text whole, within its cap; "-" is stdin.
// A text that cannot be read, or is over the cap, refuses the check: a part
// read is a part unchecked.
func readCheckFiles(names []string, stdin io.Reader) ([]checkText, error) {
	var out []checkText
	for _, name := range names {
		var data []byte
		var err error
		if name == "-" {
			if stdin == nil {
				return nil, refuse("--file - names stdin, and there is no stdin to read", "pipe the text in, or name a file")
			}
			data, err = io.ReadAll(io.LimitReader(stdin, maxCheckText+1))
			if err == nil && len(data) > maxCheckText {
				err = fsutil.ErrTooBig
			}
		} else {
			p := name
			if r, rerr := filepath.EvalSymlinks(name); rerr == nil {
				p = r
			}
			data, err = fsutil.ReadGuarded(p, maxCheckText)
		}
		if err != nil {
			why := "could not be read"
			if errors.Is(err, fsutil.ErrTooBig) {
				why = "could not be read within the check's 1 MiB limit"
			}
			return nil, refuse(fmt.Sprintf("text %d of the check %s; nothing was checked", len(out)+1, why), "name a readable regular file, or split the text")
		}
		out = append(out, checkText{name: name, body: string(data)})
	}
	return out, nil
}

// commitText is what one commit of the range makes public: its message, and
// the lines it adds, file by file.
type commitText struct {
	sha     string
	message string
	added   []fileAdds
}

type fileAdds struct {
	path string
	runs []addedRun
}

// addedRun is a hunk's added lines, joined, and the new-side line of the first.
type addedRun struct {
	line int
	text string
}

// readRange reads every commit of `<a>..<b>`, oldest first. Both ends are
// resolved to commits before git sees the range, so neither can reach git as
// an option. Every git call is isolated (no hook, no repository-configured
// program), and the patch passes --no-ext-diff and --no-textconv so no diff
// driver or text conversion the repository configures runs, or decides what
// the check reads; --text keeps a file marked binary from hiding its lines.
// A merge commit's diff is read against its first parent, so a line the merge
// itself adds (a conflict resolution, an evil merge) is read too; a line it
// brings in from a merged commit of the range is then read twice, which can
// only add findings.
func readRange(top, rng string) ([]commitText, error) {
	rng = strings.TrimSpace(rng)
	if rng == "" {
		return nil, nil
	}
	a, b, ok := strings.Cut(rng, "..")
	if !ok || a == "" || b == "" || strings.HasPrefix(b, ".") {
		return nil, refuse("--range is not of the form <a>..<b>", "name the range's two ends, for example origin/main..HEAD")
	}
	from, err1 := gitutil.ResolveCommit(top, a)
	to, err2 := gitutil.ResolveCommit(top, b)
	if err1 != nil || err2 != nil {
		return nil, refuse("an end of --range names no commit in this repository", "name two commits, branches or tags this checkout holds")
	}
	list, err := gitutil.RunCappedBytes(top, maxCheckGit, "rev-list", "--reverse", from+".."+to, "--")
	if err != nil {
		return nil, refuse("git could not list the range's commits", "narrow the range, then run it again")
	}
	var out []commitText
	for _, sha := range strings.Fields(string(list)) {
		if !gitutil.IsFullSHA(sha) {
			return nil, refuse("git listed the range in a form this abcd does not read", "run it again; if it persists, report it")
		}
		msg, err := gitutil.RunCappedBytes(top, maxCheckText, "log", "-1", "--no-show-signature", "--format=%B", sha, "--")
		if err != nil {
			return nil, refuse("a commit message of the range could not be read within the check's limit", "narrow the range, then run it again")
		}
		patch, err := gitutil.RunCappedBytes(top, maxCheckGit, "log", "-1", "--no-show-signature", "--format=", "-p",
			"--no-ext-diff", "--no-textconv", "--text", "--no-color", "--no-renames", "--no-relative",
			"--diff-merges=first-parent", "--unified=0", "--src-prefix=a/", "--dst-prefix=b/", sha, "--")
		if err != nil {
			return nil, refuse("a commit of the range could not be read within the check's limit", "narrow the range, then run it again")
		}
		added, err := parseAddedLines(patch)
		if err != nil {
			return nil, err
		}
		out = append(out, commitText{sha: sha, message: string(msg), added: added})
	}
	return out, nil
}

var hunkRe = regexp.MustCompile(`^@@ -\d+(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// parseAddedLines reads a --unified=0 patch: per file, each hunk's added
// lines and where they start. A hunk's header says how many lines it holds,
// and exactly those are read as its body, so an added line that itself
// begins "+++ " or "diff --git" is never taken for a header.
func parseAddedLines(patch []byte) ([]fileAdds, error) {
	var out []fileAdds
	bad := refuse("git's patch for a commit of the range is not in the form this abcd reads; nothing was judged clean", "run it again; if it persists, report it")
	sc := bufio.NewScanner(bytes.NewReader(patch))
	sc.Buffer(make([]byte, 64<<10), maxCheckGit)
	cur := -1
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "diff --git "):
			out = append(out, fileAdds{})
			cur = len(out) - 1
		case strings.HasPrefix(line, "+++ ") && cur >= 0:
			out[cur].path = patchPath(strings.TrimPrefix(line, "+++ "))
		case strings.HasPrefix(line, "@@ ") && cur >= 0:
			m := hunkRe.FindStringSubmatch(line)
			if m == nil {
				return nil, bad
			}
			removed, added := count(m[1]), count(m[3])
			start, _ := strconv.Atoi(m[2])
			var b strings.Builder
			for removed+added > 0 {
				if !sc.Scan() {
					return nil, bad
				}
				body := sc.Text()
				switch {
				case strings.HasPrefix(body, "\\"):
				case strings.HasPrefix(body, "-") && removed > 0:
					removed--
				case strings.HasPrefix(body, "+") && added > 0:
					if b.Len() > 0 {
						b.WriteByte('\n')
					}
					b.WriteString(body[1:])
					added--
				default:
					return nil, bad
				}
			}
			if b.Len() > 0 || count(m[3]) > 0 {
				out[cur].runs = append(out[cur].runs, addedRun{line: start, text: b.String()})
			}
		}
	}
	if sc.Err() != nil {
		return nil, bad
	}
	return out, nil
}

// count is a hunk header's line count: absent means one.
func count(s string) int {
	if s == "" {
		return 1
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// patchPath is a "+++ " header's path: the b/ prefix dropped, a C-quoted
// path unquoted. A deleted file's /dev/null is no path.
func patchPath(p string) string {
	if strings.HasPrefix(p, `"`) {
		if u, err := strconv.Unquote(p); err == nil {
			p = u
		}
	}
	if p == "/dev/null" {
		return ""
	}
	return strings.TrimPrefix(p, "b/")
}

// buildDenySet is the deny set of a store's state: every advisory under
// embargo (embargoed), and, while any is, the store's location on this
// machine in every form a text could name it: absolute, home-relative, and
// the worktree-store fragment keyed on this repository's root commit. The
// relative paths inside any store (StateRel, DraftDirRel) are not in the set:
// abcd's own source names them, and they say nothing of where this machine
// keeps its records.
func buildDenySet(s Store, st State, allowed map[string]bool) (denySet, error) {
	ds := denySet{held: map[string]bool{}, allowed: allowed, unlisted: true}
	for _, a := range st.Advisories {
		ds.held[canonGHSA(a.GHSAID)] = true
		if a.PublishedAt == nil && allowed[canonGHSA(a.GHSAID)] {
			return denySet{}, refuse("an --allow value names an unpublished advisory this store holds, which stays under embargo",
				"drop it from --allow; only an advisory already public may be allowed")
		}
	}
	ds.addEmbargoed(st, "")
	if ds.embargo == 0 {
		return ds, nil
	}
	sha := gitutil.RootCommit(s.Checkout)
	forms := []string{s.Root, resolve(s.Root), "worktrees/" + sha + "/" + StoreName}
	if home, err := os.UserHomeDir(); err == nil {
		for _, h := range []string{filepath.Clean(home), resolve(home)} {
			for _, root := range []string{s.Root, resolve(s.Root)} {
				if rel, err := filepath.Rel(h, root); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
					forms = append(forms, "~/"+filepath.ToSlash(rel))
				}
			}
		}
	}
	for _, f := range forms {
		ds.add(filepath.ToSlash(f), KindStorePath, false)
	}
	return ds, nil
}

// addEmbargoed adds every advisory under embargo but the one with id skip:
// each not yet published (a closed one stays private on GitHub, so it stays
// in the set), by its id and reserved CVE id, and by its title unless it was
// moved to the public track (that issue is fixed in the open) or is shorter
// than minTitleRunes.
func (ds *denySet) addEmbargoed(st State, skip string) {
	for _, a := range st.Advisories {
		if a.PublishedAt != nil || (skip != "" && canonGHSA(a.GHSAID) == canonGHSA(skip)) {
			continue
		}
		ds.embargo++
		ds.add(a.GHSAID, KindAdvisoryID, true)
		if a.CVEID != nil {
			ds.add(*a.CVEID, KindReservedCVE, true)
		}
		if a.Verdict == nil || *a.Verdict != VerdictPublicTrack {
			if n, _ := normalise(a.Title); utf8.RuneCountInString(strings.TrimSpace(n)) >= minTitleRunes {
				ds.add(a.Title, KindAdvisoryTitle, true)
			}
		}
	}
}

// othersDenySet is the deny set an advisory's own text is held to before it
// is published: every other advisory still under embargo, by id, reserved
// CVE id and title, so publishing one never discloses another. An advisory id
// the store does not hold is not refused here: an advisory may cite another
// project's, already public.
func othersDenySet(st State, self string) denySet {
	ds := denySet{}
	ds.addEmbargoed(st, self)
	return ds
}

func (ds *denySet) add(text, kind string, boundary bool) {
	n, _ := normalise(text)
	n = strings.TrimSpace(n)
	if n == "" {
		return
	}
	for _, have := range ds.needles {
		if have.text == n && have.kind == kind {
			return
		}
	}
	ds.needles = append(ds.needles, needle{text: n, kind: kind, boundary: boundary, pathEnd: kind == KindStorePath})
}

// span is one match, in byte offsets of the original text.
type span struct {
	start, end int
	kind       string
}

// matches finds every match of the deny set in text, overlapping matches
// merged into the one that starts first (the longer on a tie).
func (ds denySet) matches(text string) []span {
	norm, at := normalise(text)
	var found []span
	toOrig := func(s, e int) span {
		end := len(text)
		if e < len(at) {
			end = at[e]
		}
		return span{start: at[s], end: end}
	}
	for _, n := range ds.needles {
		for off := 0; ; {
			i := strings.Index(norm[off:], n.text)
			if i < 0 {
				break
			}
			s, e := off+i, off+i+len(n.text)
			off = s + 1
			if n.boundary && !wordEdge(norm, s, e) {
				continue
			}
			if n.pathEnd && continuesName(norm, e) {
				continue
			}
			sp := toOrig(s, e)
			sp.kind = n.kind
			found = append(found, sp)
		}
	}
	for _, loc := range ghsaShapeRe.FindAllStringIndex(norm, -1) {
		if !ds.unlisted {
			break
		}
		id := canonGHSA(norm[loc[0]:loc[1]])
		if ds.held[id] || ds.allowed[id] || !wordEdge(norm, loc[0], loc[1]) {
			continue
		}
		sp := toOrig(loc[0], loc[1])
		sp.kind = KindUnlistedAdvisoryID
		found = append(found, sp)
	}
	slices.SortFunc(found, func(a, b span) int {
		if a.start != b.start {
			return a.start - b.start
		}
		return b.end - a.end
	})
	var merged []span
	for _, sp := range found {
		if n := len(merged); n > 0 && sp.start < merged[n-1].end {
			if sp.end > merged[n-1].end {
				merged[n-1].end = sp.end
			}
			continue
		}
		merged = append(merged, sp)
	}
	return merged
}

// scan reports the matches in one text. firstLine is the line the text's
// first line has in its source; 0 means the text is a name, not lines.
func (ds denySet) scan(text, source, where string, firstLine int) []CheckFinding {
	spans := ds.matches(text)
	if len(spans) == 0 {
		return nil
	}
	masked := []byte(text)
	for _, sp := range spans {
		for i := sp.start; i < sp.end; {
			r, size := utf8.DecodeRune(masked[i:])
			if r != '\n' {
				for j := 0; j < size; j++ {
					masked[i+j] = 0
				}
				masked[i] = '*'
			}
			i += size
		}
	}
	lines := strings.Split(strings.ReplaceAll(string(masked), "\x00", ""), "\n")
	var out []CheckFinding
	for _, sp := range spans {
		n := strings.Count(text[:sp.start], "\n")
		line := 0
		if firstLine > 0 {
			line = firstLine + n
		}
		out = append(out, CheckFinding{Kind: sp.kind, Source: source, Where: where, Line: line,
			Snippet: capRunes(strings.TrimRight(lines[n], "\r"), snippetRunes)})
	}
	return out
}

// nameOr is name, or fallback when name itself carries a match: a finding
// never repeats what it found, even in the name of where it found it.
func (ds denySet) nameOr(name, fallback string) string {
	if name == "" || len(ds.matches(name)) > 0 {
		return fallback
	}
	return name
}

// normalise is text case-folded with every run of white space one space,
// and the offset in text each byte of the result came from.
func normalise(text string) (string, []int) {
	var b strings.Builder
	at := make([]int, 0, len(text))
	space := false
	for i, r := range text {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				at = append(at, i)
			}
			space = true
			continue
		}
		space = false
		var buf [utf8.UTFMax]byte
		n := utf8.EncodeRune(buf[:], unicode.ToLower(r))
		b.Write(buf[:n])
		for j := 0; j < n; j++ {
			at = append(at, i)
		}
	}
	return b.String(), at
}

// wordEdge reports whether norm[s:e] stands on its own: where the match
// begins or ends with a letter or digit, the text does not continue it.
func wordEdge(norm string, s, e int) bool {
	first, _ := utf8.DecodeRuneInString(norm[s:])
	last, _ := utf8.DecodeLastRuneInString(norm[:e])
	if isWord(first) && s > 0 {
		if prev, _ := utf8.DecodeLastRuneInString(norm[:s]); isWord(prev) {
			return false
		}
	}
	if isWord(last) && e < len(norm) {
		if next, _ := utf8.DecodeRuneInString(norm[e:]); isWord(next) {
			return false
		}
	}
	return true
}

func isWord(r rune) bool { return unicode.IsLetter(r) || unicode.IsDigit(r) }

// continuesName reports whether the path segment ending at norm[e] goes on
// past it: a store path followed by more of a name (embargo-old, embargo.bak)
// names a sibling folder, not the store. A dot that ends a sentence does not
// continue it.
func continuesName(norm string, e int) bool {
	name := func(b byte) bool {
		return b >= 'a' && b <= 'z' || b >= '0' && b <= '9' || b == '-' || b == '_' || b >= 0x80
	}
	for ; e < len(norm) && norm[e] == '.'; e++ {
	}
	return e < len(norm) && name(norm[e])
}

func capRunes(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	r := []rune(s)
	return string(r[:n]) + "…"
}
