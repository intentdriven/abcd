package peers

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/embargo/storepath"
	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
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
	{"iss", recordid.IssuesRelDir, issueschema.StatusDirs},
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

// The starts of the not-read reasons a caller tells apart to choose a peer's
// repair: git would not answer for a worktree (most often a git file that
// points nowhere), and a ledger holds one id in two folders.
const (
	NotReadGitRefused = "git refused to answer for it: "
	NotReadSplit      = "its ledger holds one id in two places"
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

	// The file the peer holds the record in (its blob for a branch peer), and
	// the commit the peer's merge-base with this checkout is taken from
	// (Peer.base): what Holders judges a stale copy by.
	rel, blob, base string
}

// worktree is one entry of `git worktree list --porcelain`.
type worktree struct {
	path   string
	head   string
	branch string // short name; "" when detached
	bare   bool
}

// SeenWorktree is one linked worktree a scan saw: its path as git reports
// it, its short branch ("" when detached), and how the scan judged it.
type SeenWorktree struct {
	Path   string `json:"path"`
	Branch string `json:"branch,omitempty"`
	// Spent is the scan's merged judgement: its branch (or detached HEAD) is
	// merged into the default branch by ancestry and its record folders are
	// clean (SkipMerged).
	Spent bool `json:"spent,omitempty"`
	// Gone is a worktree git lists whose directory is not there (SkipGone).
	Gone bool `json:"gone,omitempty"`
}

// Worktrees lists every linked worktree the scan saw, sorted by path: each
// worktree peer, read or not, and each worktree skipped as merged or gone.
// The main checkout is not one, and neither is the embargo store, which is
// skipped unnamed. It reads nothing; a caller counting worktrees (the board's
// store line) takes them from the scan it already ran.
func (r Report) Worktrees() []SeenWorktree {
	out := []SeenWorktree{}
	for _, p := range r.Peers {
		if p.Source == SourceWorktree {
			out = append(out, SeenWorktree{Path: p.Path, Branch: p.Branch})
		}
	}
	for _, s := range r.Skipped {
		if s.Source == SourceWorktree && s.Path != "" {
			out = append(out, SeenWorktree{Path: s.Path, Branch: s.Branch, Spent: s.Reason == SkipMerged, Gone: s.Reason == SkipGone})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// HeldUnder counts the linked worktrees the scan saw whose real path lies
// beneath dir, and the spent ones among them: the board's worktrees line,
// with dir the repository's lane in the worktree store. A gone worktree is
// not counted, because it no longer occupies dir. Paths compare resolved, so
// a lane reached through a symlinked home still holds what git reports.
func (r Report) HeldUnder(dir string) (held, spent int) {
	base := realPath(dir)
	for _, w := range r.Worktrees() {
		if w.Gone {
			continue
		}
		rel, err := filepath.Rel(base, realPath(w.Path))
		if err != nil || rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			continue
		}
		held++
		if w.Spent {
			spent++
		}
	}
	return held, spent
}

// Read returns the peer picture for the checkout at root, which is a checkout
// root (gitutil.CheckoutRoot's answer), not an arbitrary directory under it.
//
// It fails only when this checkout itself cannot be read: git will not name its
// common dir, or its own record folders cannot be listed. Every fault in a PEER
// is that peer's NotRead, so one bad sibling never hides the others.
func Read(root string) (Report, error) { return read(root, true) }

// Scan is Read without the titles: the same peers, rows and locations, with
// no record file opened. It is for a caller that needs counts or a location —
// the board line, a not-found refusal — and should not pay a read per row.
func Scan(root string) (Report, error) { return read(root, false) }

func read(root string, titles bool) (Report, error) {
	rep := Report{Sources: append([]Source(nil), Sources...), Peers: []Peer{}, Skipped: []Skipped{}, root: root,
		bases: &baseCache{merged: map[string][]string{}}}
	common, err := gitutil.CommonDir(root)
	if err != nil {
		return rep, fmt.Errorf("peers: git could not name this checkout's common dir: %w", err)
	}
	here, _, err := scanDisk(root)
	if err != nil {
		return rep, fmt.Errorf("peers: reading this checkout's records: %w", err)
	}
	rep.here = here
	defaultRef := gitutil.DefaultRef(root)
	rep.DefaultRef = gitutil.ShortRef(defaultRef)
	merged := mergedBranches(root, defaultRef)

	wts, err := listWorktrees(root)
	if err != nil {
		return rep, fmt.Errorf("peers: listing this repository's worktrees: %w", err)
	}
	checkedOut := map[string]bool{}
	self := realPath(root)
	home, _ := os.UserHomeDir()
	for _, wt := range wts {
		if wt.bare || realPath(wt.path) == self {
			if wt.branch != "" {
				checkedOut[wt.branch] = true
			}
			continue
		}
		if embargoStore(home, wt.path) {
			// Private records live here: skipped unread, and counted with
			// neither its path nor its branch (adr-2610120558328615).
			if wt.branch != "" {
				checkedOut[wt.branch] = true
			}
			rep.Skipped = append(rep.Skipped, Skipped{Source: SourceWorktree, Reason: SkipEmbargo})
			continue
		}
		p, skip, viaBranch := readWorktree(root, common, wt, merged, defaultRef)
		// A worktree that is gone, or that git will not read, still leaves its
		// branch in this repository's object store; the branch source reads it
		// there rather than letting the dead worktree hide an unmerged commit.
		// A merged branch has nothing that could hide, and the worktree is
		// already counted, so it is not counted a second time as a branch.
		if wt.branch != "" && (!viaBranch || merged[wt.branch]) {
			checkedOut[wt.branch] = true
		}
		if skip != nil {
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
		rep.Peers[i].Rows = rep.diff(rep.Peers[i], titles)
	}
	return rep, nil
}

// embargoStore reports whether a worktree at p is an embargo store, by its
// path alone: worktrees/<key>/embargo in abcd's home, or that shape under
// any home (storepath.IsStorePath, storepath.HasStoreShape). It asks
// nothing of the store's lookup, so a lookup that fails or refuses to choose
// still leaves every store skipped.
func embargoStore(home, p string) bool {
	return storepath.IsStorePath(home, p) || storepath.HasStoreShape(p)
}

// readWorktree reads one linked worktree, or says why it is spent. viaBranch
// reports that its disk could not be read at all (gone, or refused by git), so
// its branch, if any, is left to the branch source.
func readWorktree(root, common string, wt worktree, merged map[string]bool, defaultRef string) (_ Peer, _ *Skipped, viaBranch bool) {
	p := Peer{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Rows: []Row{}, base: wt.head}
	if _, err := os.Lstat(wt.path); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return p, &Skipped{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Reason: SkipGone}, true
		}
		p.NotRead = "its directory cannot be read: " + err.Error()
		p.unjudged = true
		return p, nil, true
	}
	// The candidate's OWN answer, not this checkout's: a directory git lists can
	// have been replaced by another repository, and reading it would report that
	// repository's records as this one's.
	theirs, err := gitutil.CommonDir(wt.path)
	if err != nil {
		p.NotRead = NotReadGitRefused + firstLine(err.Error())
		// Where the git file points is read from the file, not left to git's
		// refusal: Linux's git names a broken git file's target "(null)", so
		// a reason relying on git's stderr names it on macOS alone.
		if target, ok := gitutil.GitfileTarget(wt.path); ok {
			p.NotRead += "; its git file points at " + target
		}
		p.unjudged = true
		return p, nil, true
	}
	if realPath(theirs) != realPath(common) {
		p.NotRead = "its own common dir is not this checkout's, so it belongs to another repository"
		return p, nil, true
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
		return p, &Skipped{Source: SourceWorktree, Branch: wt.branch, Path: wt.path, Reason: SkipMerged}, false
	}
	h, present, err := scanDisk(wt.path)
	if err != nil {
		p.NotRead = "its record folders cannot be read: " + err.Error()
		p.unjudged = true
		return p, nil, false
	}
	p.NotRead = judgeHoldings(h, present)
	p.unjudged = p.NotRead != "" && present
	if p.NotRead == "" {
		p.holdings = h
	}
	return p, nil, false
}

// readBranch reads one local branch from the common dir.
func readBranch(root, branch string) Peer {
	p := Peer{Source: SourceBranch, Branch: branch, Rows: []Row{}, base: "refs/heads/" + branch}
	h, present, err := scanTree(root, "refs/heads/"+branch)
	if err != nil {
		p.NotRead = "git could not list its records: " + firstLine(err.Error())
		p.unjudged = true
		return p
	}
	p.NotRead = judgeHoldings(h, present)
	p.unjudged = p.NotRead != "" && present
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
		return "it holds no records at the committed layout (" + recordid.IssuesRelDir + "/, " +
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
	return NotReadSplit + " (" + strings.Join(split, "; ") +
		"), which is no status at all; move or remove one of each so the ledger says which status the record is in"
}

// diff computes one peer's rows against this checkout's holdings.
func (r Report) diff(p Peer, titles bool) []Row {
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
		row := Row{ID: id, Kind: kind, Folder: h.folder}
		if titles {
			row.Title = r.title(p, id, h)
		}
		rows = append(rows, row)
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
			out = append(out, Location{Source: p.Source, Branch: p.Branch, Path: p.Path, Folder: h.folder,
				rel: h.rel, blob: h.blob, base: p.base})
		}
	}
	return out
}

// Holders returns the live, read peers that hold id themselves, as against
// this checkout holding it in folder: those holding it in another folder of
// its family (a lane that shipped or re-drafted it) whose copy differs from
// the one in its merge-base with this checkout's HEAD. A copy in folder is a
// copy, not a holding: every branch cut from the default branch carries one.
// So is a copy identical, at the same path, to the one in every merge-base
// tree of the peer and this checkout: the peer was cut before the record
// moved here and holds this checkout's older copy, which it never touched
// (iss-2610222008081888). It is measured against this checkout, not the
// default branch, so a peer cut after the default branch shipped what this
// checkout, being behind, still holds as planned holds it shipped. In a
// criss-cross history every merge-base must agree. A worktree peer is judged
// off its disk, against the merge-bases of its HEAD, so an uncommitted move
// or edit is a change.
//
// It fails closed: where this checkout's HEAD (unborn, or unreadable), the
// merge-bases, a merge-base tree's entry or the peer's copy cannot be read,
// the location holds it. The comparison is made only for a location in
// another folder, and the merge-bases once per peer for the Report's life.
//
// Locate keeps every location: a not-found path asks where a record is, and a
// stale copy is still where it is.
func (r Report) Holders(id, folder string) []Location {
	var out []Location
	for _, l := range r.Locate(id) {
		if l.Folder == folder || !r.changedSinceBase(l) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// changedSinceBase reports whether the peer at l changed its copy of the
// record since its merge-bases with this checkout's HEAD: whether any
// merge-base tree holds no regular file at l's path, or one whose content is
// not the peer's. Anything that cannot be read answers true.
func (r Report) changedSinceBase(l Location) bool {
	if l.base == "" || l.rel == "" {
		return true
	}
	bases := r.mergeBases(l.base)
	if len(bases) == 0 {
		return true
	}
	var disk []byte
	if l.Source == SourceWorktree {
		b, err := fsutil.ReadGuarded(filepath.Join(l.Path, filepath.FromSlash(l.rel)), maxListing)
		if err != nil {
			return true
		}
		disk = b
	}
	for _, mb := range bases {
		blob, ok := r.blobAt(mb, l.rel)
		if !ok {
			return true
		}
		switch l.Source {
		case SourceBranch:
			if l.blob == "" || l.blob != blob {
				return true
			}
		case SourceWorktree:
			content, err := gitutil.RunCappedBytes(r.root, maxListing, "cat-file", "blob", blob)
			if err != nil || !bytes.Equal(disk, content) {
				return true
			}
		default:
			return true
		}
	}
	return false
}

// mergeBases is every merge-base of base and this checkout's HEAD
// (`merge-base --all`), memoised per base in r.bases; nil when HEAD is unborn
// or unreadable, the two share no history, or git cannot answer.
func (r Report) mergeBases(base string) []string {
	c := r.bases
	if c == nil {
		c = &baseCache{merged: map[string][]string{}}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.headRead {
		c.headRead = true
		if head, err := gitutil.Run(r.root, "rev-parse", "--verify", "--quiet", "HEAD^{commit}"); err == nil && gitutil.IsFullSHA(head) {
			c.head = head
		}
	}
	if c.head == "" {
		return nil
	}
	if got, ok := c.merged[base]; ok {
		return got
	}
	var bases []string
	if out, err := gitutil.RunCapped(r.root, maxListing, "merge-base", "--all", base, c.head); err == nil {
		for _, ln := range strings.Split(out, "\n") {
			ln = strings.TrimSpace(ln)
			if !gitutil.IsFullSHA(ln) {
				bases = nil
				break
			}
			bases = append(bases, ln)
		}
	}
	c.merged[base] = bases
	return bases
}

// blobAt is the object name of the regular file at rel in commit's tree, and
// false when the tree holds none there, or anything else, or cannot be read.
func (r Report) blobAt(commit, rel string) (string, bool) {
	out, err := gitutil.RunCapped(r.root, maxListing, "ls-tree", "-z", "--full-tree", commit, "--", rel)
	if err != nil {
		return "", false
	}
	entry, rest, _ := strings.Cut(out, "\x00")
	meta, file, ok := strings.Cut(entry, "\t")
	f := strings.Fields(meta) // <mode> <type> <object>
	if !ok || file != rel || strings.Trim(rest, "\x00") != "" || len(f) != 3 ||
		f[0] != "100644" && f[0] != "100755" || f[1] != "blob" {
		return "", false
	}
	return f[2], true
}

// Unjudged returns the live peers that were named and not read for a reason
// that leaves what they hold unknown: git or the filesystem would not answer
// for them, or their ledger holds one id in two folders. Locate cannot see
// into these, so a caller that must not start past a peer's holding treats
// each as one. A peer of another repository, and one holding no records at
// the committed layout, hold nothing of this checkout's and are not returned.
func (r Report) Unjudged() []Peer {
	var out []Peer
	for _, p := range r.Peers {
		if p.unjudged {
			out = append(out, p)
		}
	}
	return out
}

// NotReadShown is p's not-read reason as a surface prints it, with no
// absolute local path in it: the peer's own directory as fsutil.DisplayPathsIn
// names it, HOME as ~, the checkout at repoRoot as ., and any other absolute
// path, such as where a broken git file points, by its last element alone
// (iss-2609281329007423). Every surface that prints a reason prints this
// form: the drain's refusal, the build's peers check and `abcd peers`.
func (p Peer) NotReadShown(repoRoot string) string {
	s := fsutil.DisplayPathsIn(p.NotRead, p.Path)
	if filepath.IsAbs(repoRoot) {
		s = fsutil.RedactRoot(s, repoRoot, ".")
		if real, err := filepath.EvalSymlinks(repoRoot); err == nil && real != repoRoot {
			s = fsutil.RedactRoot(s, real, ".")
		}
	}
	return displayAbsPaths(s)
}

// displayAbsPaths replaces each absolute path left in s, one starting where a
// path can (the start, or after a space, a quote, a bracket or an equals
// sign, so a URL's // is left alone) and running to the next space or quote, with
// fsutil.DisplayPath's form of it. A path in a message has no reliable end, so
// one holding a space keeps the words after it; what it never keeps is the
// directories above them.
func displayAbsPaths(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if s[i] != '/' || (i > 0 && !strings.ContainsRune(" \t\n'\"([=", rune(s[i-1]))) {
			b.WriteByte(s[i])
			i++
			continue
		}
		end := i + strings.IndexFunc(s[i:]+" ", func(r rune) bool { return strings.ContainsRune(" \t\n'\")]", r) })
		tok := strings.TrimRight(s[i:end], ".,;:")
		if tok == "/" {
			b.WriteString(tok)
		} else {
			b.WriteString(fsutil.DisplayPath(tok))
		}
		i += len(tok)
	}
	return b.String()
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
	// The block is the one frontmatter.Fields reads, a BOM at line 0 included
	// (iss-2608221126066379).
	end := frontmatter.Close(lines)
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

// RecordAt is one file a ref's tree holds a record in: its folder and the
// object name of its blob.
type RecordAt struct {
	Folder string
	Blob   string
}

// RecordsAt is the record ids ref's tree holds, each with every file that
// claims it (its folder: open, resolved, wontfix, or an intent bucket; and
// its blob), read from the object store by filename, with no checkout and no
// network. It is the peers reader's own tree read, exported so a reservation
// decides which folder a record is in, and which blob it is, by the same
// reader.
func RecordsAt(root, ref string) (map[string][]RecordAt, error) {
	h, _, err := scanTree(root, ref)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]RecordAt, len(h))
	for id, hs := range h {
		for _, x := range hs {
			out[id] = append(out[id], RecordAt{Folder: x.folder, Blob: x.blob})
		}
	}
	return out, nil
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

// recordsClean reports whether a worktree's record folders hold no uncommitted
// change, tracked or not. A git that cannot answer reads as not clean, so the
// peer is read rather than skipped.
func recordsClean(dir string) bool {
	var specs []string
	for _, fam := range families {
		specs = append(specs, fam.rel)
	}
	entries, err := gitutil.Status(dir, maxListing, gitutil.StatusOptions{Pathspecs: specs})
	return err == nil && len(entries) == 0
}

// listWorktrees is this repository's working trees, git's canonical listing
// (gitutil.ListWorktrees, which falls back to the newline form on a git older
// than 2.36) with each branch in its short name.
func listWorktrees(root string) ([]worktree, error) {
	listed, err := gitutil.ListWorktrees(root, maxListing)
	if err != nil {
		return nil, err
	}
	wts := make([]worktree, 0, len(listed))
	for _, w := range listed {
		wts = append(wts, worktree{path: w.Path, head: w.Head, branch: strings.TrimPrefix(w.Branch, "refs/heads/"), bare: w.Bare})
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
