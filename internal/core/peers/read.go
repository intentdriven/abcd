package peers

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The two record families a holding is read from, each with the folders whose
// membership IS the record's state (adr-3).
var families = []struct {
	prefix  string
	rel     string
	folders []string
}{
	{"iss", capture.LedgerRelPath, []string{"open", "resolved", "wontfix"}},
	{"itd", intent.IntentsRelDir, intent.Buckets},
}

const (
	// maxEntries bounds how many directory or tree entries one tree may
	// contribute, the record resolver's ceiling: a ledger past it is
	// pathological or hostile, and the peer is named and not read.
	maxEntries = 20000
	// maxTitleBytes caps the one read a title costs, the record reader's head cap.
	maxTitleBytes = 256 * 1024
	// maxTitleRunes keeps a row one line: an issue's title is its body's first
	// line, which can be a paragraph.
	maxTitleRunes = 120
	// maxListing caps a git listing (worktrees, branches, one branch's tree).
	maxListing = 16 << 20
)

// holding is one record file a tree holds.
type holding struct {
	folder string // open, resolved, wontfix, or an intent bucket
	rel    string // repo-relative slash path
	blob   string // the blob's object name, branch peers only
}

// holdings is one tree's record ids, each with every file that claims it.
type holdings map[string][]holding

// Location is where a live peer holds one record.
type Location struct {
	Source Source `json:"source"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
	Folder string `json:"folder"`
}

// worktree is one entry of `git worktree list --porcelain -z`.
type worktree struct {
	path   string
	head   string
	branch string // short name; "" when detached
	bare   bool
}

// Read returns the peer picture for the checkout at root, which is a checkout
// root (gitutil.CheckoutRoot's answer), not an arbitrary directory under it.
//
// It fails only when this checkout itself cannot be read: git will not name its
// common dir, or its own record folders cannot be listed. Every fault in a PEER
// is that peer's NotRead, so one bad sibling never hides the others.
func Read(root string) (Report, error) {
	rep := Report{Sources: append([]Source(nil), Sources...), Peers: []Peer{}, Skipped: []Skipped{}, root: root}
	common, err := commonDir(root)
	if err != nil {
		return rep, fmt.Errorf("peers: git could not name this checkout's common dir: %w", err)
	}
	here, _, err := scanDisk(root)
	if err != nil {
		return rep, fmt.Errorf("peers: reading this checkout's records: %w", err)
	}
	rep.here = here
	defaultRef := resolveDefaultRef(root)
	rep.DefaultRef = shortRef(defaultRef)
	merged := mergedBranches(root, defaultRef)

	wts, err := listWorktrees(root)
	if err != nil {
		return rep, fmt.Errorf("peers: listing this repository's worktrees: %w", err)
	}
	checkedOut := map[string]bool{}
	self := realPath(root)
	for _, wt := range wts {
		if wt.branch != "" {
			checkedOut[wt.branch] = true
		}
		if wt.bare || realPath(wt.path) == self {
			continue
		}
		if p, skip := readWorktree(root, common, wt, merged, defaultRef); skip != nil {
			rep.Skipped = append(rep.Skipped, *skip)
		} else {
			rep.Peers = append(rep.Peers, p)
		}
	}

	branches, err := listBranches(root)
	if err != nil {
		return rep, fmt.Errorf("peers: listing this repository's branches: %w", err)
	}
	for _, b := range branches {
		if checkedOut[b] {
			continue
		}
		if merged[b] {
			rep.Skipped = append(rep.Skipped, Skipped{Source: SourceBranch, Branch: b, Reason: SkipMerged})
			continue
		}
		rep.Peers = append(rep.Peers, readBranch(root, b))
	}

	for i := range rep.Peers {
		rep.Peers[i].Rows = rep.diff(rep.Peers[i])
	}
	return rep, nil
}

// readWorktree reads one linked worktree, or says why it is spent.
func readWorktree(root, common string, wt worktree, merged map[string]bool, defaultRef string) (Peer, *Skipped) {
	p := Peer{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Rows: []Row{}}
	if _, err := os.Lstat(wt.path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return p, &Skipped{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Reason: SkipGone}
		}
		p.NotRead = "its directory cannot be read: " + err.Error()
		return p, nil
	}
	// The candidate's OWN answer, not this checkout's: a directory git lists can
	// have been replaced by another repository, and reading it would report that
	// repository's records as this one's.
	theirs, err := commonDir(wt.path)
	if err != nil {
		p.NotRead = "git refused to answer for it: " + firstLine(err.Error())
		return p, nil
	}
	if realPath(theirs) != realPath(common) {
		p.NotRead = "its own common dir is not this checkout's, so it belongs to another repository"
		return p, nil
	}
	spentByAncestry := false
	switch {
	case wt.branch != "":
		spentByAncestry = merged[wt.branch]
	case defaultRef != "" && wt.head != "":
		spentByAncestry, _ = gitutil.IsAncestor(root, wt.head, defaultRef)
	}
	// A branch at the default tip is merged by ancestry the moment it is cut, and
	// an uncommitted capture in it is exactly what this reader exists to show; so
	// a worktree is spent only when its record folders are clean as well.
	if spentByAncestry && recordsClean(wt.path) {
		return p, &Skipped{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Reason: SkipMerged}
	}
	h, present, err := scanDisk(wt.path)
	if err != nil {
		p.NotRead = "its record folders cannot be read: " + err.Error()
		return p, nil
	}
	p.NotRead = judgeHoldings(h, present)
	if p.NotRead == "" {
		p.holdings = h
	}
	return p, nil
}

// readBranch reads one local branch from the common dir.
func readBranch(root, branch string) Peer {
	p := Peer{Source: SourceBranch, Branch: branch, Rows: []Row{}}
	h, present, err := scanTree(root, "refs/heads/"+branch)
	if err != nil {
		p.NotRead = "git could not list its records: " + firstLine(err.Error())
		return p
	}
	p.NotRead = judgeHoldings(h, present)
	if p.NotRead == "" {
		p.holdings = h
	}
	return p
}

// judgeHoldings returns why a peer's holdings must not be read, or "".
//
// One id in two status folders has no status: the folder IS the state, so the
// peer's answer for that id is a contradiction, and a row built from it would
// pass the contradiction on as a fact (iss-2609100507430423). The whole peer is
// marked rather than the one id dropped, because the same merge that split one
// record commonly split others the check cannot tell from legitimate rows.
func judgeHoldings(h holdings, present bool) string {
	if !present {
		return "it holds no records at the committed layout (" + capture.LedgerRelPath + "/, " +
			intent.IntentsRelDir + "/); a checkout from before that layout contributes nothing"
	}
	var split []string
	for _, id := range sortedIDs(h) {
		if hs := h[id]; len(hs) > 1 {
			var where []string
			for _, x := range hs {
				where = append(where, x.folder+"/")
			}
			split = append(split, id+" in "+strings.Join(where, " and "))
		}
	}
	if len(split) == 0 {
		return ""
	}
	return "its ledger holds one id in two places (" + strings.Join(split, "; ") +
		"), which is no status at all; move or remove one of each so the ledger says which status the record is in"
}

// diff computes one peer's rows against this checkout's holdings.
func (r Report) diff(p Peer) []Row {
	rows := []Row{}
	if p.NotRead != "" {
		return rows
	}
	for _, id := range sortedIDs(p.holdings) {
		h := p.holdings[id][0]
		var kind Kind
		switch {
		case strings.HasPrefix(id, "iss-") && h.folder == "open" && !r.HeldHere(id):
			kind = KindOpenThere
		case strings.HasPrefix(id, "iss-") && (h.folder == "resolved" || h.folder == "wontfix") && r.heldHereIn(id, "open"):
			kind = KindTerminalThere
		case strings.HasPrefix(id, "itd-") && h.folder == intent.BucketDrafts && !r.HeldHere(id):
			kind = KindDraftThere
		default:
			continue
		}
		rows = append(rows, Row{ID: id, Kind: kind, Folder: h.folder, Title: r.title(p, id, h)})
	}
	sort.SliceStable(rows, func(i, j int) bool { return kindRank(rows[i].Kind) < kindRank(rows[j].Kind) })
	return rows
}

func kindRank(k Kind) int {
	switch k {
	case KindOpenThere:
		return 0
	case KindTerminalThere:
		return 1
	default:
		return 2
	}
}

// Locate returns every live, read peer that holds id, in any folder of its
// family. It is the not-found paths' question: a verb that cannot find a record
// here asks where it is before answering "not found".
func (r Report) Locate(id string) []Location {
	id = recordid.CanonCitedID(id)
	var out []Location
	for _, p := range r.Peers {
		if p.NotRead != "" {
			continue
		}
		for _, h := range p.holdings[id] {
			out = append(out, Location{Source: p.Source, Branch: p.Branch, Path: p.Path, Folder: h.folder})
		}
	}
	return out
}

// HeldHere reports whether this checkout holds id in any folder of its family.
func (r Report) HeldHere(id string) bool { return len(r.here[recordid.CanonCitedID(id)]) > 0 }

func (r Report) heldHereIn(id, folder string) bool {
	for _, h := range r.here[id] {
		if h.folder == folder {
			return true
		}
	}
	return false
}

// title reads one record's title, or "" when the file is unreadable or
// malformed. A disk read goes through the guarded primitive; a branch read
// through a capped blob read, so neither can be made to block or balloon.
func (r Report) title(p Peer, id string, h holding) string {
	var data []byte
	switch p.Source {
	case SourceWorktree:
		b, err := fsutil.ReadGuarded(filepath.Join(p.Path, filepath.FromSlash(h.rel)), maxTitleBytes)
		if err != nil {
			return ""
		}
		data = b
	case SourceBranch:
		out, err := gitutil.RunCapped(r.root, maxTitleBytes, "cat-file", "blob", h.blob)
		if err != nil {
			return ""
		}
		data = []byte(out)
	}
	return parseTitle(data, id[:3])
}

// parseTitle is the record's title by its family's rule: an intent's first H1,
// an issue's first body line. A file with no closed frontmatter, or bytes that
// are not UTF-8, is malformed and has no title.
func parseTitle(data []byte, family string) string {
	if !utf8.Valid(data) {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return ""
	}
	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			end = i
			break
		}
	}
	if end < 0 {
		return ""
	}
	var title string
	for _, ln := range lines[end+1:] {
		if family == "itd" {
			if strings.HasPrefix(ln, "# ") {
				title = strings.TrimSpace(strings.TrimPrefix(ln, "# "))
				break
			}
			continue
		}
		if f := strings.Fields(ln); len(f) > 0 {
			title = strings.Join(f, " ")
			break
		}
	}
	if utf8.RuneCountInString(title) > maxTitleRunes {
		title = string([]rune(title)[:maxTitleRunes-1]) + "…"
	}
	return title
}

// scanDisk reads a tree's holdings off disk by filename. present reports
// whether either family's root exists at the committed layout.
func scanDisk(root string) (holdings, bool, error) {
	h := holdings{}
	present := false
	budget := maxEntries
	for _, fam := range families {
		if isRealDirChain(root, fam.rel) {
			present = true
		}
		for _, folder := range fam.folders {
			rel := fam.rel + "/" + folder
			if !isRealDirChain(root, rel) {
				continue
			}
			entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(rel)))
			if err != nil {
				return nil, present, err
			}
			for _, e := range entries {
				if budget--; budget < 0 {
					return nil, present, fmt.Errorf("more than %d record files; refusing to read further", maxEntries)
				}
				// Never follow a link: a symlinked file could make a record appear
				// to exist that the tree does not hold (the resolver's rule).
				if e.IsDir() || e.Type()&os.ModeSymlink != 0 {
					continue
				}
				if id := fileID(fam.prefix, e.Name()); id != "" {
					h[id] = append(h[id], holding{folder: folder, rel: rel + "/" + e.Name()})
				}
			}
		}
	}
	return h, present, nil
}

// isRealDirChain reports whether every component of rel under root is a real
// directory, no symlink among them: a symlinked record folder would walk the
// read out of the peer's tree.
func isRealDirChain(root, rel string) bool {
	cur := root
	for _, part := range strings.Split(rel, "/") {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if err != nil || !fi.IsDir() {
			return false
		}
	}
	return true
}

// scanTree reads a ref's holdings from the object store by filename.
func scanTree(root, ref string) (holdings, bool, error) {
	args := []string{"ls-tree", "-r", "-z", "--full-tree", ref, "--"}
	for _, fam := range families {
		args = append(args, fam.rel)
	}
	out, err := gitutil.RunCapped(root, maxListing, args...)
	if err != nil {
		return nil, false, err
	}
	h := holdings{}
	present := false
	n := 0
	for _, entry := range strings.Split(out, "\x00") {
		meta, file, ok := strings.Cut(entry, "\t")
		if !ok {
			continue
		}
		if n++; n > maxEntries {
			return nil, present, fmt.Errorf("more than %d record files; refusing to read further", maxEntries)
		}
		f := strings.Fields(meta) // <mode> <type> <object>
		if len(f) != 3 {
			continue
		}
		for _, fam := range families {
			rest, ok := strings.CutPrefix(file, fam.rel+"/")
			if !ok {
				continue
			}
			present = true
			// A symlink (120000) or a gitlink is not a record file.
			if f[0] != "100644" && f[0] != "100755" || f[1] != "blob" {
				break
			}
			folder, name, ok := strings.Cut(rest, "/")
			if !ok || strings.Contains(name, "/") || !contains(fam.folders, folder) {
				break
			}
			if id := fileID(fam.prefix, name); id != "" {
				h[id] = append(h[id], holding{folder: folder, rel: path.Join(fam.rel, folder, name), blob: f[2]})
			}
			break
		}
	}
	return h, present, nil
}

// fileID derives a record id from a filename by the canonical grammar, rebuilt
// canonically so a zero-padded twin keys with its plain spelling.
func fileID(prefix, name string) string {
	m := recordid.FilenameNumRe(prefix).FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return recordid.CanonCitedID(prefix + "-" + m[1])
}

// commonDir is git's absolute common dir for the tree at dir.
func commonDir(dir string) (string, error) {
	out, err := gitutil.Run(dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", errors.New("git named no common dir")
	}
	return out, nil
}

// recordsClean reports whether a worktree's record folders hold no uncommitted
// change, tracked or not. A git that cannot answer reads as not clean, so the
// peer is read rather than skipped.
func recordsClean(dir string) bool {
	args := []string{"status", "--porcelain", "-z", "--untracked-files=all", "--"}
	for _, fam := range families {
		args = append(args, fam.rel)
	}
	out, err := gitutil.RunCapped(dir, maxListing, args...)
	return err == nil && out == ""
}

// listWorktrees parses `git worktree list --porcelain -z`.
func listWorktrees(root string) ([]worktree, error) {
	out, err := gitutil.RunCapped(root, maxListing, "worktree", "list", "--porcelain", "-z")
	if err != nil {
		return nil, err
	}
	var wts []worktree
	var cur *worktree
	for _, field := range strings.Split(out, "\x00") {
		key, val, _ := strings.Cut(field, " ")
		switch key {
		case "worktree":
			wts = append(wts, worktree{path: val})
			cur = &wts[len(wts)-1]
		case "HEAD":
			if cur != nil {
				cur.head = val
			}
		case "branch":
			if cur != nil {
				cur.branch = strings.TrimPrefix(val, "refs/heads/")
			}
		case "bare":
			if cur != nil {
				cur.bare = true
			}
		}
	}
	return wts, nil
}

// listBranches is every local branch's short name.
func listBranches(root string) ([]string, error) {
	out, err := gitutil.RunCapped(root, maxListing, "for-each-ref", "--format=%(refname)", "refs/heads/")
	if err != nil {
		return nil, err
	}
	var bs []string
	for _, ln := range strings.Split(out, "\n") {
		if b, ok := strings.CutPrefix(strings.TrimSpace(ln), "refs/heads/"); ok && b != "" {
			bs = append(bs, b)
		}
	}
	return bs, nil
}

// mergedBranches is the set of local branches whose tip is an ancestor of the
// default ref, in one listing rather than one ancestry probe per branch.
func mergedBranches(root, defaultRef string) map[string]bool {
	set := map[string]bool{}
	if defaultRef == "" {
		return set
	}
	out, err := gitutil.RunCapped(root, maxListing, "for-each-ref", "--merged="+defaultRef, "--format=%(refname)", "refs/heads/")
	if err != nil {
		return set
	}
	for _, ln := range strings.Split(out, "\n") {
		if b, ok := strings.CutPrefix(strings.TrimSpace(ln), "refs/heads/"); ok && b != "" {
			set[b] = true
		}
	}
	return set
}

// resolveDefaultRef is the ref a branch is judged merged into, as last fetched
// and with no network: origin/HEAD's target, then the conventional names on the
// remote, then the same names locally. "" when none resolves.
func resolveDefaultRef(root string) string {
	if out, err := gitutil.Run(root, "symbolic-ref", "--quiet", "refs/remotes/origin/HEAD"); err == nil && strings.HasPrefix(out, "refs/remotes/origin/") && refExists(root, out) {
		return out
	}
	names := []string{"main", "master", "trunk", "develop"}
	for _, prefix := range []string{"refs/remotes/origin/", "refs/heads/"} {
		for _, n := range names {
			if refExists(root, prefix+n) {
				return prefix + n
			}
		}
	}
	return ""
}

func refExists(root, ref string) bool {
	_, err := gitutil.Run(root, "rev-parse", "--verify", "--quiet", ref+"^{commit}", "--")
	return err == nil
}

func shortRef(ref string) string {
	if s, ok := strings.CutPrefix(ref, "refs/remotes/"); ok {
		return s
	}
	return strings.TrimPrefix(ref, "refs/heads/")
}

// realPath resolves symlinks so two spellings of one directory compare equal;
// a path that cannot be resolved compares by its cleaned spelling.
func realPath(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	return filepath.Clean(p)
}

func sortedIDs(h holdings) []string {
	ids := make([]string, 0, len(h))
	for id := range h {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return idLess(ids[i], ids[j]) })
	return ids
}

// idLess orders ids by family, then by number (shorter digit strings first,
// which is numeric order for canonical ids).
func idLess(a, b string) bool {
	if a[:3] != b[:3] {
		return a[:3] < b[:3]
	}
	if len(a) != len(b) {
		return len(a) < len(b)
	}
	return a < b
}

func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}

func firstLine(s string) string {
	line, _, _ := strings.Cut(s, "\n")
	return strings.TrimSpace(line)
}
