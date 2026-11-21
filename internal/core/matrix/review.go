package matrix

// review.go assembles what one reviewer model is handed for one copy, and the
// run's own account of it (adr-2610161733400551, spc-2610161949559050 step 5).
//
// The request is the copy's normalised diff from the frozen tree and nothing
// else: each file the copy changed, as its unified diff (udiff.go), where the
// reading surface's include table admits the path (reading.ReviewKind),
// labelled with the table's material class and keyed by ordinal, under an
// opaque label whose mapping to the copy's builder the run holds. It is built
// from two trees, never from history: the frozen commit's tree read from the
// run's own checkout, and the copy's files read as plain data. So no commit,
// trailer, author, branch name or pull-request text can reach it, and the
// include table's structural deny keeps the record, where a copy's run state
// and decisions would name its model, out of it. What the copy wrote that
// names where it ran (its clone's path, the clone store's, the run checkout's,
// the account's home, the run's id) is withheld from the text and counted.
// Metadata is stripped; content is not, and the reviewer's guess at the
// author (reviewscore.go) measures what content still leaks.
//
// The manifest is never handed to the reviewer. It pins the include table the
// request was assembled through (reading.AssemblerVersion), the ask and the
// request by hash, each item's path, class, change and hash, and every changed
// path that did not travel with the reason why, so a review of part of a
// copy says which part.
//
// The copy controls everything in its clone, so it is read the way step 4's
// compliance check reads it: through an os.Root, every entry judged by Lstat
// before it is opened, a link never followed, a FIFO or device never opened
// or waited on (fsutil.ReadGuardedInRoot opens without blocking and judges the
// descriptor again), every read bounded, and the walk bounded in entries,
// bytes and by the caller's deadline. No git runs in the clone: the only git
// is the run checkout's ls-tree and cat-file, each under its own deadline. No
// refusal names a place on this machine.

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// The request's and the manifest's type tags and shape version.
const (
	ReviewRequestType  = "abcd.matrix.review-request"
	ReviewManifestType = "abcd.matrix.review-manifest"
	ReviewSchema       = 1
)

// reviewAsk is what every reviewer is asked, the same for every copy and
// every reviewer, and pinned by hash in the manifest. It names no model, no
// run and no builder, so the author guess it asks for after the verdict is
// the reviewer's own.
const reviewAsk = "You are reviewing one change to a software repository, handed to you as the unified diff of " +
	"each file it touches, each under an item key. You have nothing else: no repository, no tools and no " +
	"history. Judge whether the change carries a defect: a bug, a behaviour its own code implies that is broken " +
	"or missing, or a test that does not test what it claims. Answer with one JSON object and nothing else: " +
	`{"label": the label you were given, "verdict": "defect" or "clean", "findings": [{"item": an item key, ` +
	`"text": one finding}], "author_guess": the model you think wrote the change, or "" to decline}. ` +
	"Decide the verdict first: the author guess is asked only after it, and counts for nothing in it."

// What a changed path's item records as its change.
const (
	changeAdded    = "added"
	changeModified = "modified"
	changeDeleted  = "deleted"
)

// Why a changed path did not travel.
const (
	withheldByTable    = "the include table admits no such path"
	withheldLink       = "a symbolic link, never followed or shown"
	withheldNotRegular = "not a regular file, never opened"
	withheldTooLarge   = "larger than a review reads"
	withheldNotText    = "not text: it holds a NUL byte or is not valid UTF-8"
	withheldUnreadable = "could not be read"
	withheldOverBound  = "past the request's size bound"
	// withheldLineEndings: the diff reads CRLF as LF, so a change of line
	// endings alone shows nothing.
	withheldLineEndings = "a change of line endings alone, which the diff reads as none"
)

// redactedMark stands in the request's text where the copy wrote where it
// ran.
const redactedMark = "[run metadata]"

// The bounds a request is assembled within.
const (
	// maxReviewFileBytes is the largest file a review reads, on either side.
	maxReviewFileBytes = 1 << 20
	// maxReviewRequestBytes bounds the items' text together; a changed file
	// past it is withheld.
	maxReviewRequestBytes = 4 << 20
	// maxReviewEntries bounds the entries the copy's tree walk reads, and
	// maxReviewReadBytes the bytes it reads; a tree past either is refused.
	maxReviewEntries   = 200_000
	maxReviewReadBytes = 256 << 20
	// reviewDirBatch is how many entries of one directory are listed at a
	// time, so the entry bound holds inside a directory.
	reviewDirBatch = 256
	// maxWithheld bounds the withheld paths the manifest lists one by one;
	// past it they are counted.
	maxWithheld = 1000
	// maxWithheldPath bounds a withheld path as the manifest writes it.
	maxWithheldPath = 512
	// maxFrozenTreeBytes bounds the frozen tree's listing.
	maxFrozenTreeBytes = 64 << 20
	// reviewGitTimeout bounds each git call the assembly makes.
	reviewGitTimeout = 2 * time.Minute
)

// reviewLabelRe is an opaque copy label: it says nothing but that it is one.
var reviewLabelRe = regexp.MustCompile(`^copy-[0-9a-f]{8}$`)

// ReviewItem is one file of the copy's change as the reviewer receives it: an
// ordinal key, the include table's class for its path, and its diff.
type ReviewItem struct {
	Key  string       `json:"item_key"`
	Kind reading.Kind `json:"kind"`
	Text string       `json:"text"`
}

// ReviewRequest is the whole of what one reviewer is handed for one copy.
type ReviewRequest struct {
	Type          string       `json:"_type"`
	SchemaVersion int          `json:"schema_version"`
	Label         string       `json:"label"`
	Ask           string       `json:"ask"`
	Items         []ReviewItem `json:"items"`
}

// ReviewManifestItem is the run's account of one item: the path it was read
// from in the copy's tree, its class, its change, and its text's size and
// hash.
type ReviewManifestItem struct {
	Key    string       `json:"item_key"`
	Path   string       `json:"path"`
	Kind   reading.Kind `json:"kind"`
	Change string       `json:"change"`
	Bytes  int          `json:"bytes"`
	SHA256 string       `json:"sha256"`
}

// WithheldPath is a path the copy changed, or may have, that did not travel,
// and why.
type WithheldPath struct {
	Path   string `json:"path"`
	Reason string `json:"reason"`
}

// ReviewManifest is the run's account of one request. It stays with the run
// and is never handed to a reviewer.
type ReviewManifest struct {
	Type          string               `json:"_type"`
	SchemaVersion int                  `json:"schema_version"`
	Label         string               `json:"label"`
	Assembler     string               `json:"assembler"`
	AskSHA256     string               `json:"ask_sha256"`
	RequestSHA256 string               `json:"request_sha256"`
	Items         []ReviewManifestItem `json:"items"`
	Withheld      []WithheldPath       `json:"withheld"`
	WithheldMore  int                  `json:"withheld_more"`
	Redactions    int                  `json:"redactions"`
}

// reviewInputs is what one request is assembled from.
type reviewInputs struct {
	// RepoRoot is the run's own checkout, which holds Frozen; Dir is the
	// copy's tree as its builder left it; Store is the run's clone store.
	// Each is an absolute path.
	RepoRoot, Frozen, Dir, Store string
	// Run is the run's id, and Label the opaque label the request goes under.
	Run, Label string
}

// mintLabels mints n distinct opaque labels.
func mintLabels(n int) ([]string, error) {
	seen := make(map[string]bool, n)
	out := make([]string, 0, n)
	for len(out) < n {
		var b [4]byte
		if _, err := rand.Read(b[:]); err != nil {
			return nil, err
		}
		l := "copy-" + hex.EncodeToString(b[:])
		if !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return out, nil
}

// encodeReviewRequest is the request as it is sent and hashed.
func encodeReviewRequest(r ReviewRequest) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

// assembleReview assembles one copy's request and its manifest. A copy that
// changed nothing the include table admits is refused, so no reviewer is
// asked to judge an empty request.
func assembleReview(ctx context.Context, in reviewInputs) (ReviewRequest, ReviewManifest, error) {
	if err := in.validate(); err != nil {
		return ReviewRequest{}, ReviewManifest{}, err
	}
	frozen, err := readFrozenTree(ctx, in.RepoRoot, in.Frozen)
	if err != nil {
		return ReviewRequest{}, ReviewManifest{}, err
	}
	copied, err := readCopyTree(ctx, in.Dir, frozen)
	if err != nil {
		return ReviewRequest{}, ReviewManifest{}, err
	}
	red := newRedactor(in)
	req := ReviewRequest{Type: ReviewRequestType, SchemaVersion: ReviewSchema, Label: in.Label, Ask: reviewAsk, Items: []ReviewItem{}}
	man := ReviewManifest{Type: ReviewManifestType, SchemaVersion: ReviewSchema, Label: in.Label,
		Assembler: reading.AssemblerVersion(), AskSHA256: sha256Hex([]byte(reviewAsk)),
		Items: []ReviewManifestItem{}, Withheld: []WithheldPath{}}
	withhold := func(p, reason string) {
		if len(man.Withheld) >= maxWithheld {
			man.WithheldMore++
			return
		}
		man.Withheld = append(man.Withheld, WithheldPath{Path: clip512(p), Reason: reason})
	}
	total := 0
	for _, p := range changedPaths(frozen, copied) {
		if err := ctx.Err(); err != nil {
			return ReviewRequest{}, ReviewManifest{}, errors.New("the review's request was not assembled before its deadline")
		}
		fe, inFrozen := frozen[p]
		ce, inCopy := copied[p]
		if inCopy && ce.withheld != "" {
			withhold(p, ce.withheld)
			continue
		}
		if !inCopy && underUnreadable(copied, p) {
			// Not seen, not shown deleted: what is under the directory is
			// unknown.
			withhold(p, withheldUnreadable)
			continue
		}
		kind, ok := reading.ReviewKind(p)
		if !ok {
			withhold(p, withheldByTable)
			continue
		}
		if inFrozen && fe.mode == "120000" {
			withhold(p, withheldLink)
			continue
		}
		var old []byte
		if inFrozen {
			if fe.size > maxReviewFileBytes {
				withhold(p, withheldTooLarge)
				continue
			}
			old, err = readBlob(ctx, in.RepoRoot, fe.oid, fe.size)
			if err != nil {
				return ReviewRequest{}, ReviewManifest{}, err
			}
		}
		var now []byte
		if inCopy {
			now = ce.data
		}
		// A side that exists is never nil, which unifiedDiff reads as no
		// file: an empty file is a file.
		if inFrozen && old == nil {
			old = []byte{}
		}
		if inCopy && now == nil {
			now = []byte{}
		}
		if !diffable(old) || !diffable(now) {
			withhold(p, withheldNotText)
			continue
		}
		change := changeModified
		switch {
		case !inFrozen:
			change, old = changeAdded, nil
		case !inCopy:
			change, now = changeDeleted, nil
		}
		text := unifiedDiff(p, old, now)
		if inFrozen && inCopy && fe.mode != ce.mode {
			if text == "" {
				text = "--- a/" + p + "\n+++ b/" + p + "\n"
			}
			text = "old mode " + fe.mode + "\nnew mode " + ce.mode + "\n" + text
		}
		if text == "" {
			switch change {
			case changeAdded:
				// An empty file added: its header is the whole of its diff.
				text = "--- /dev/null\n+++ b/" + p + "\n"
			case changeDeleted:
				text = "--- a/" + p + "\n+++ /dev/null\n"
			default:
				// The same text once line endings are normalised.
				withhold(p, withheldLineEndings)
				continue
			}
		}
		text, n := red.redact(text)
		if total+len(text) > maxReviewRequestBytes {
			withhold(p, withheldOverBound)
			continue
		}
		total += len(text)
		man.Redactions += n
		key := fmt.Sprintf("item-%03d", len(req.Items)+1)
		req.Items = append(req.Items, ReviewItem{Key: key, Kind: kind, Text: text})
		man.Items = append(man.Items, ReviewManifestItem{Key: key, Path: p, Kind: kind, Change: change,
			Bytes: len(text), SHA256: sha256Hex([]byte(text))})
	}
	if len(req.Items) == 0 {
		return ReviewRequest{}, man, errors.New("the copy changed nothing the include table admits, so there is nothing to review")
	}
	data, err := encodeReviewRequest(req)
	if err != nil {
		return ReviewRequest{}, ReviewManifest{}, err
	}
	man.RequestSHA256 = sha256Hex(data)
	return req, man, nil
}

func (in reviewInputs) validate() error {
	if !filepath.IsAbs(in.RepoRoot) || !filepath.IsAbs(in.Dir) || !filepath.IsAbs(in.Store) {
		return errors.New("a review reads the run's checkout, the copy's tree and the clone store by absolute path")
	}
	if !gitutil.IsFullSHA(in.Frozen) {
		return fmt.Errorf("%q is not a full commit id, so it is no frozen commit", clip(in.Frozen))
	}
	if !runIDRe.MatchString(in.Run) {
		return fmt.Errorf("%q is not a run id", clip(in.Run))
	}
	if !reviewLabelRe.MatchString(in.Label) {
		return fmt.Errorf("%q is not an opaque copy label: a label is copy- and eight hex digits, and says nothing else", clip(in.Label))
	}
	return nil
}

// treeEntry is one file of a tree: its git mode, its object, its size and,
// for a copy's file read in full, its bytes. withheld is set where the copy's
// entry was not read, and says why.
type treeEntry struct {
	mode     string
	oid      string
	size     int64
	data     []byte
	withheld string
}

// readFrozenTree lists the frozen commit's files from the run's own checkout:
// regular files and links, each with its mode, object and size. A gitlink
// holds no file and is left out.
func readFrozenTree(ctx context.Context, repoRoot, frozen string) (map[string]treeEntry, error) {
	out, err := reviewGit(ctx, repoRoot, maxFrozenTreeBytes, "ls-tree", "-r", "-z", "-l", "--full-tree", frozen)
	if err != nil {
		return nil, fmt.Errorf("the frozen commit's tree cannot be read: %w", err)
	}
	tree := map[string]treeEntry{}
	for _, rec := range bytes.Split(out, []byte{0}) {
		if len(rec) == 0 {
			continue
		}
		meta, p, ok := bytes.Cut(rec, []byte{'\t'})
		f := strings.Fields(string(meta))
		if !ok || len(f) != 4 {
			return nil, errors.New("the frozen commit's tree listing is not in the shape git documents")
		}
		if f[1] != "blob" {
			continue
		}
		size, err := strconv.ParseInt(f[3], 10, 64)
		if err != nil || size < 0 || !gitutil.IsFullSHA(f[2]) {
			return nil, errors.New("the frozen commit's tree listing is not in the shape git documents")
		}
		tree[string(p)] = treeEntry{mode: f[0], oid: f[2], size: size}
	}
	return tree, nil
}

// readBlob reads one blob of the frozen tree, of the size its listing named.
func readBlob(ctx context.Context, repoRoot, oid string, size int64) ([]byte, error) {
	out, err := reviewGit(ctx, repoRoot, int(size), "cat-file", "blob", oid)
	if err != nil {
		return nil, fmt.Errorf("a file of the frozen commit cannot be read: %w", err)
	}
	if int64(len(out)) != size {
		return nil, errors.New("a file of the frozen commit is not the size its tree names")
	}
	return out, nil
}

// readCopyTree reads the copy's tree at dir as plain data, through an
// os.Root: every regular file within maxReviewFileBytes in full, with its
// object computed in the frozen tree's object format, and every other entry
// marked withheld with the reason, unread. A directory named .git, at any
// depth and in any letter case, is never entered, and a file of that name
// never read.
func readCopyTree(ctx context.Context, dir string, frozen map[string]treeEntry) (map[string]treeEntry, error) {
	tree, _, err := walkCopyTree(ctx, dir, frozen, maxReviewEntries, reviewDirBatch)
	return tree, err
}

// copyWalk is one bounded read of a copy's tree. A directory is listed
// reviewDirBatch entries at a time and every entry is counted as it is
// listed, so the entry bound holds inside one directory as well as across
// them: a directory of millions of entries is refused at the bound, never
// listed whole first, and the caller's deadline is judged between batches.
type copyWalk struct {
	ctx     context.Context
	root    *os.Root
	newHash func() hash.Hash
	bound   int
	batch   int
	listed  int
	read    int64
	tree    map[string]treeEntry
}

// walkCopyTree is readCopyTree within bound entries, listing batch at a
// time, and how many entries it listed.
func walkCopyTree(ctx context.Context, dir string, frozen map[string]treeEntry, bound, batch int) (map[string]treeEntry, int, error) {
	if fi, err := os.Lstat(dir); err != nil || !fi.IsDir() {
		return nil, 0, errors.New("the copy's tree is not a directory that can be read")
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, 0, errors.New("the copy's tree is not a directory that can be read")
	}
	defer root.Close()
	w := &copyWalk{ctx: ctx, root: root, newHash: sha1.New, bound: bound, batch: max(batch, 1), tree: map[string]treeEntry{}}
	for _, e := range frozen {
		if len(e.oid) == 64 {
			w.newHash = sha256.New
		}
		break
	}
	if err := w.walk(); err != nil {
		return nil, w.listed, err
	}
	return w.tree, w.listed, nil
}

// walk reads the tree depth first from its root. A directory that cannot be
// opened or listed, the root aside, is marked unreadable: what is under it is
// unknown.
func (w *copyWalk) walk() error {
	dirs := []string{"."}
	for len(dirs) > 0 {
		d := dirs[len(dirs)-1]
		dirs = dirs[:len(dirs)-1]
		sub, err := w.listDir(d)
		if err != nil {
			return err
		}
		dirs = append(dirs, sub...)
	}
	return nil
}

// listDir lists the directory d in batches, reading each entry as it comes,
// and returns its subdirectories to walk.
func (w *copyWalk) listDir(d string) ([]string, error) {
	f, err := openDirInRoot(w.root, d)
	if err != nil {
		if d == "." {
			return nil, errors.New("the copy's tree cannot be listed")
		}
		w.tree[d] = treeEntry{withheld: withheldUnreadable}
		return nil, nil
	}
	defer f.Close()
	var sub []string
	for {
		if err := w.ctx.Err(); err != nil {
			return nil, errors.New("the copy's tree was not read before the review's deadline")
		}
		ents, err := f.ReadDir(w.batch)
		for _, de := range ents {
			if w.listed++; w.listed > w.bound {
				return nil, fmt.Errorf("the copy's tree holds more than %d entries, more than a review reads", w.bound)
			}
			p := path.Join(d, de.Name())
			isDir, err := w.entry(p, de)
			if err != nil {
				return nil, err
			}
			if isDir {
				sub = append(sub, p)
			}
		}
		switch {
		case errors.Is(err, io.EOF):
			return sub, nil
		case err != nil:
			// Listed in part: what was not listed is unknown.
			if d == "." {
				return nil, errors.New("the copy's tree cannot be listed")
			}
			w.tree[d] = treeEntry{withheld: withheldUnreadable}
			return sub, nil
		case len(ents) == 0:
			return sub, nil
		}
	}
}

// entry reads one listed entry at p, and reports whether it is a directory to
// walk.
func (w *copyWalk) entry(p string, de fs.DirEntry) (bool, error) {
	if err := w.ctx.Err(); err != nil {
		return false, errors.New("the copy's tree was not read before the review's deadline")
	}
	t := de.Type()
	if strings.EqualFold(path.Base(p), ".git") {
		return false, nil
	}
	switch {
	case t.IsDir():
		return true, nil
	case t&fs.ModeSymlink != 0:
		w.tree[p] = linkEntry(w.root, p, w.newHash)
		return false, nil
	case !t.IsRegular():
		w.tree[p] = treeEntry{withheld: withheldNotRegular}
		return false, nil
	}
	data, err := fsutil.ReadGuardedInRoot(w.root, filepath.FromSlash(p), maxReviewFileBytes)
	switch {
	case errors.Is(err, fsutil.ErrTooBig):
		w.tree[p] = treeEntry{withheld: withheldTooLarge}
		return false, nil
	case errors.Is(err, fsutil.ErrNotRegular):
		w.tree[p] = treeEntry{withheld: withheldNotRegular}
		return false, nil
	case err != nil:
		w.tree[p] = treeEntry{withheld: withheldUnreadable}
		return false, nil
	}
	if w.read += int64(len(data)); w.read > maxReviewReadBytes {
		return false, fmt.Errorf("the copy's tree holds more than %d bytes, more than a review reads", maxReviewReadBytes)
	}
	mode := "100644"
	if fi, err := w.root.Lstat(filepath.FromSlash(p)); err == nil && fi.Mode().Perm()&0o111 != 0 {
		mode = "100755"
	}
	w.tree[p] = treeEntry{mode: mode, oid: blobOID(w.newHash, data), size: int64(len(data)), data: data}
	return false, nil
}

// openDirInRoot opens the directory d of root to list it, judged by Lstat
// before it is opened, opened without blocking, and judged again by its
// descriptor, so a link is never followed and an entry swapped for a FIFO
// after it was listed is never waited on.
func openDirInRoot(root *os.Root, d string) (*os.File, error) {
	name := filepath.FromSlash(d)
	fi, err := root.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, errors.New("not a directory")
	}
	f, err := root.OpenFile(name, os.O_RDONLY|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil || !st.IsDir() || !os.SameFile(fi, st) {
		f.Close()
		return nil, errors.New("the directory was replaced between its listing and its opening")
	}
	return f, nil
}

// linkEntry is a link in the copy's tree: its target is read only to tell
// whether it is the frozen tree's link, and is never followed or shown.
func linkEntry(root *os.Root, p string, newHash func() hash.Hash) treeEntry {
	target, err := root.Readlink(filepath.FromSlash(p))
	if err != nil || len(target) > maxReviewFileBytes {
		return treeEntry{withheld: withheldLink}
	}
	return treeEntry{mode: "120000", oid: blobOID(newHash, []byte(target)), withheld: withheldLink}
}

// blobOID is data's git blob object name under newHash.
func blobOID(newHash func() hash.Hash, data []byte) string {
	h := newHash()
	fmt.Fprintf(h, "blob %d\x00", len(data))
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

// changedPaths is every path the copy changed against the frozen tree, sorted:
// added, deleted, or held by both with another object or mode. A copy's entry
// that was not read is changed unless the frozen tree's is provably the same
// (a link to the same target).
func changedPaths(frozen, copied map[string]treeEntry) []string {
	var out []string
	for p, fe := range frozen {
		ce, ok := copied[p]
		if !ok || ce.oid != fe.oid || ce.mode != fe.mode {
			out = append(out, p)
		}
	}
	for p := range copied {
		if _, ok := frozen[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// underUnreadable reports whether p lies under a directory of the copy's tree
// that could not be listed.
func underUnreadable(copied map[string]treeEntry, p string) bool {
	for d := path.Dir(p); d != "." && d != "/"; d = path.Dir(d) {
		if e, ok := copied[d]; ok && e.withheld == withheldUnreadable {
			return true
		}
	}
	return false
}

// diffable reports whether data is text a diff can show: valid UTF-8 holding
// no NUL byte. No file (nil) is diffable.
func diffable(data []byte) bool {
	return bytes.IndexByte(data, 0) < 0 && utf8.Valid(data)
}

// redactor withholds from a request's text every mention of where the copy
// ran: its clone's path, the clone store's, the run checkout's and the
// account's home, each also as its links resolve, and the run's id, longest
// first so a path is withheld whole.
type redactor struct{ tokens []string }

func newRedactor(in reviewInputs) redactor {
	var raw []string
	for _, p := range []string{in.Dir, in.Store, in.RepoRoot} {
		raw = append(raw, p)
		if r, err := filepath.EvalSymlinks(p); err == nil {
			raw = append(raw, r)
		}
	}
	if home, err := os.UserHomeDir(); err == nil && filepath.IsAbs(home) {
		raw = append(raw, home)
		if r, err := filepath.EvalSymlinks(home); err == nil {
			raw = append(raw, r)
		}
	}
	raw = append(raw, in.Run)
	seen := map[string]bool{}
	var tokens []string
	for _, t := range raw {
		t = strings.TrimRight(t, string(filepath.Separator))
		// A token this short would withhold ordinary text, not a place.
		if len(t) < 4 || seen[t] {
			continue
		}
		seen[t] = true
		tokens = append(tokens, t)
	}
	sort.Slice(tokens, func(i, j int) bool { return len(tokens[i]) > len(tokens[j]) })
	return redactor{tokens: tokens}
}

// redact returns s with every token withheld, and how many it withheld.
func (r redactor) redact(s string) (string, int) {
	n := 0
	for _, t := range r.tokens {
		if c := strings.Count(s, t); c > 0 {
			n += c
			s = strings.ReplaceAll(s, t, redactedMark)
		}
	}
	return s, n
}

// reviewGit runs one isolated git in the run's own checkout under
// reviewGitTimeout, reading at most limit bytes of its output. Its refusal
// names the subcommand and how it failed, never git's own words, which can
// name a path on this machine.
func reviewGit(parent context.Context, dir string, limit int, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(parent, reviewGitTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitutil.IsolatedEnv()
	cmd.WaitDelay = time.Second
	out := &boundedBuffer{limit: limit}
	cmd.Stdout = out
	err := cmd.Run()
	switch {
	case parent.Err() != nil:
		return nil, fmt.Errorf("the review's deadline passed while git %s ran", args[0])
	case ctx.Err() != nil:
		return nil, fmt.Errorf("git %s did not answer within %s", args[0], reviewGitTimeout)
	case out.over:
		return nil, fmt.Errorf("git %s answered more than the %d bytes a review reads", args[0], limit)
	case err != nil:
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("git %s exited %d", args[0], ee.ExitCode())
		}
		return nil, fmt.Errorf("git %s could not be run", args[0])
	}
	return out.buf.Bytes(), nil
}

// boundedBuffer keeps at most limit bytes written to it and marks itself over
// past them, never failing the write, so git is not killed by a broken pipe
// before its exit can be read.
type boundedBuffer struct {
	buf   bytes.Buffer
	limit int
	over  bool
}

func (b *boundedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.buf.Len(); len(p) > room {
		b.over = true
		if room > 0 {
			b.buf.Write(p[:room])
		}
		return len(p), nil
	}
	b.buf.Write(p)
	return len(p), nil
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// clip512 bounds a path the manifest writes.
func clip512(p string) string {
	if len(p) <= maxWithheldPath {
		return p
	}
	return strings.ToValidUTF8(p[:maxWithheldPath], "") + "…"
}
