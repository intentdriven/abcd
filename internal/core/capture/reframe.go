package capture

// The reframe verb (itd-2609020625402518, spc-2609020626048705).
//
// A reframe occasioned by a reading is recorded as a reframe: one record under
// reframes/rfm-N.md naming what occasioned it, the SHA-256 of each of the
// frame's three committed surfaces before and after the rewrite, which of them
// changed, and the grounds. The frame is the framing as it presently stands,
// which adr-55 enumerates as three committed surfaces and adr-2609021016288378
// fixes: the framing chapter's construal section, the committed glossary terms
// and the committed scope. The record carries nothing of any surface's text —
// the abandoned framing stays on the local side — so it can show that a reframe
// happened, when, why and where, without committing what it replaced.
//
// The verb reads the surfaces itself, at HEAD, in the working tree and in the
// surfaces' history, so the operator supplies no hash. Written after the
// rewrite's commit it is one write; written before it, it is two: `--open`
// records the before half, and `--complete rfm-N` finishes it once the rewrite
// is committed. The join to the occasion is the operator's assertion, checked
// in one respect only: the occasion predates the rewrite.

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/readingitem"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/site"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// FrameSurface is one of the frame's committed surfaces: its name in the
// record, its repo-relative path, and, for a surface that is one section of a
// chapter, the H2 heading that section is titled.
type FrameSurface struct {
	Name    string
	Path    string
	Heading string
}

// FrameSurfaces is the frame, as ONE table (the intent's first scope
// condition): the paths and the heading are constants, and a repository whose
// frame lives elsewhere is outside this verb's scope. The order is the
// record's, and the names are issueschema.FrameSurfaceNames.
var FrameSurfaces = []FrameSurface{
	{Name: "construal", Path: ".abcd/development/brief/01-product/06-framing.md", Heading: "Construal"},
	{Name: "glossary", Path: ".abcd/development/brief/glossary"},
	{Name: "scope", Path: ".abcd/development/brief/01-product/04-scope.md"},
}

// Frame is one frame state: the fingerprint of each surface.
type Frame struct {
	Construal string `json:"construal"`
	Glossary  string `json:"glossary"`
	Scope     string `json:"scope"`
}

// of returns the fingerprint of the named surface.
func (f Frame) of(name string) string {
	switch name {
	case "construal":
		return f.Construal
	case "glossary":
		return f.Glossary
	case "scope":
		return f.Scope
	}
	return ""
}

// moved names the surfaces whose fingerprints differ between f and g, in the
// record's order.
func (f Frame) moved(g Frame) []string {
	var out []string
	for _, n := range issueschema.FrameSurfaceNames {
		if f.of(n) != g.of(n) {
			out = append(out, n)
		}
	}
	return out
}

// String renders the triple for a refusal that has to name it.
func (f Frame) String() string {
	return fmt.Sprintf("construal %s, glossary %s, scope %s", f.Construal, f.Glossary, f.Scope)
}

// The three halves a reframe write can be.
const (
	ReframeHalfWhole     = "whole"
	ReframeHalfOpen      = "open"
	ReframeHalfCompleted = "completed"
)

// frameHistoryBound is how many commits touching any of the three surfaces a
// walk reads back from HEAD. A search that reaches it without finding what it
// looks for is refused rather than extended. A variable so a test can reach it
// with a short history.
var frameHistoryBound = 64

// ReframeRequest records one reframe, or completes an open one.
type ReframeRequest struct {
	RepoRoot   string
	IssuesRoot string
	// OccasionedBy is the reading item, disposition or surprise that occasioned
	// the reframe (rdi-N, dsp-N or srp-N), and nothing else.
	OccasionedBy string
	// Grounds is why the frame moved, held to the grounds floor.
	Grounds string
	// Open records the first half, before the rewrite is committed.
	Open bool
	// Complete names the open record (rfm-N) to finish after the commit. It
	// takes no occasion, ground or --open: those are the first half's.
	Complete string
}

// ReframeResult is the outcome of a successful Reframe.
type ReframeResult struct {
	ID           string   `json:"id"`
	Path         string   `json:"path"`
	OccasionedBy string   `json:"occasioned_by"`
	Before       Frame    `json:"before"`
	After        Frame    `json:"after"`
	Changed      []string `json:"changed"`
	// Half says which half this invocation wrote: whole, open or completed.
	Half string `json:"half"`
	// Commits is how many commits touching the frame the walk crossed between
	// the before and the after state.
	Commits  int    `json:"commits"`
	Redacted int    `json:"redacted,omitempty"`
	Degraded string `json:"redaction_degraded,omitempty"`
}

// --- the fingerprints ---

// normaliseSurface is the one normalisation every fingerprint applies: line
// endings to LF, trailing whitespace at the end of the text removed, and
// leading blank lines dropped. The trailing trim is what makes the three
// readers agree: git's output reaches this package with its trailing
// whitespace trimmed, and a file read from the working tree keeps it.
func normaliseSurface(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.TrimRight(s, " \t\n")
	for {
		nl := strings.IndexByte(s, '\n')
		if nl < 0 || strings.TrimSpace(s[:nl]) != "" {
			break
		}
		s = s[nl+1:]
	}
	if strings.TrimSpace(s) == "" {
		return ""
	}
	return s
}

func sumHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// ConstrualFingerprint fingerprints the framing chapter's construal section:
// the body under the one H2 titled `Construal`, down to the next heading of
// level two or shallower or the end of the file, line endings normalised and
// blank edges trimmed. The heading itself is a constant and is not hashed. A
// not-yet-real marker the section opens with is part of the section: a change
// to it is a change to what the construal states about itself.
//
// A chapter with no such section, or with more than one, is refused by name.
func ConstrualFingerprint(doc string) (string, error) {
	heading := FrameSurfaces[0].Heading
	body, _ := site.StripFrontmatter(strings.ReplaceAll(strings.ReplaceAll(doc, "\r\n", "\n"), "\r", "\n"))
	secs, err := site.Sections(FrameSurfaces[0].Path, body, 0)
	if err != nil {
		return "", fmt.Errorf("the framing chapter cannot be read into sections: %w", err)
	}
	at := -1
	count := 0
	for i, s := range secs {
		if s.Level == 2 && s.Title == heading {
			count++
			at = i
		}
	}
	switch count {
	case 0:
		return "", fmt.Errorf("the framing chapter %s carries no H2 section titled %q; the construal is that section, so there is nothing to fingerprint",
			FrameSurfaces[0].Path, heading)
	case 1:
	default:
		return "", fmt.Errorf("the framing chapter %s carries %d H2 sections titled %q; the construal is one section, and which of them is the frame is not a question this verb answers",
			FrameSurfaces[0].Path, count, heading)
	}
	lines := strings.Split(body, "\n")
	start := secs[at].Line // the line after the heading, 0-based
	end := len(lines)
	for _, s := range secs[at+1:] {
		if s.Level >= 1 && s.Level <= 2 {
			end = s.Line - 1
			break
		}
	}
	return sumHex([]byte(normaliseSurface(strings.Join(lines[start:end], "\n")))), nil
}

// GlossaryFingerprint fingerprints the committed glossary terms: every `.md`
// file handed to it except each README.md (the index is a render of the terms,
// held by core/glossary) and the _template.md scaffold, sorted by path, as the
// SHA-256 over path, NUL, normalised content, NUL for each. A term added,
// removed, renamed or edited moves it; an index regeneration does not. Paths
// are repo-relative and slash-separated, so every reader keys them alike.
func GlossaryFingerprint(files map[string][]byte) string {
	paths := make([]string, 0, len(files))
	for p := range files {
		if isGlossaryTerm(p) {
			paths = append(paths, p)
		}
	}
	sort.Strings(paths)
	h := sha256.New()
	for _, p := range paths {
		h.Write([]byte(p))
		h.Write([]byte{0})
		h.Write([]byte(normaliseSurface(string(files[p]))))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

// isGlossaryTerm reports whether a glossary path is a term rather than an
// index or the scaffold.
func isGlossaryTerm(p string) bool {
	base := path.Base(p)
	return strings.HasSuffix(base, ".md") && base != "README.md" && base != "_template.md"
}

// ScopeFingerprint fingerprints the committed scope chapter whole, after its
// frontmatter is stripped and its line endings normalised.
func ScopeFingerprint(doc string) string {
	body, _ := site.StripFrontmatter(strings.ReplaceAll(strings.ReplaceAll(doc, "\r\n", "\n"), "\r", "\n"))
	return sumHex([]byte(normaliseSurface(body)))
}

// --- the readers ---

// frameSurfacePaths are the three paths a history walk is limited to.
func frameSurfacePaths() []string {
	out := make([]string, 0, len(FrameSurfaces))
	for _, s := range FrameSurfaces {
		out = append(out, s.Path)
	}
	return out
}

// fingerprintFrame composes the triple from the three surfaces' content.
func fingerprintFrame(framing, scope string, glossary map[string][]byte) (Frame, error) {
	c, err := ConstrualFingerprint(framing)
	if err != nil {
		return Frame{}, err
	}
	return Frame{Construal: c, Glossary: GlossaryFingerprint(glossary), Scope: ScopeFingerprint(scope)}, nil
}

// blobCache holds blob contents by object id, so a walk over many commits
// reads each distinct blob once.
type blobCache map[string]string

// frameAtCommit fingerprints the three surfaces as rev holds them, reading the
// tree listing once and each blob through the cache.
func frameAtCommit(repoRoot, rev string, cache blobCache) (Frame, error) {
	args := append([]string{"ls-tree", "-r", "-z", rev, "--"}, frameSurfacePaths()...)
	out, err := gitutil.RunCapped(repoRoot, maxStatusBytes, args...)
	if err != nil {
		return Frame{}, fmt.Errorf("cannot list the frame at %s: %w", rev, err)
	}
	blobs := map[string]string{}
	for _, rec := range strings.Split(out, "\x00") {
		meta, p, ok := strings.Cut(rec, "\t")
		if !ok {
			continue
		}
		f := strings.Fields(meta)
		if len(f) != 3 || f[1] != "blob" {
			continue
		}
		blobs[p] = f[2]
	}
	read := func(oid string) (string, error) {
		if v, ok := cache[oid]; ok {
			return v, nil
		}
		v, err := gitutil.RunCapped(repoRoot, issueschema.RecordReadLimit, "cat-file", "blob", oid)
		if err != nil {
			return "", err
		}
		cache[oid] = v
		return v, nil
	}
	chapter := func(s FrameSurface) (string, error) {
		oid, ok := blobs[s.Path]
		if !ok {
			return "", fmt.Errorf("the %s surface %s is absent at %s", s.Name, s.Path, shortRev(rev))
		}
		return read(oid)
	}
	framing, err := chapter(FrameSurfaces[0])
	if err != nil {
		return Frame{}, err
	}
	scope, err := chapter(FrameSurfaces[2])
	if err != nil {
		return Frame{}, err
	}
	glossary := map[string][]byte{}
	prefix := FrameSurfaces[1].Path + "/"
	for p, oid := range blobs {
		if !strings.HasPrefix(p, prefix) || !isGlossaryTerm(p) {
			continue
		}
		v, err := read(oid)
		if err != nil {
			return Frame{}, err
		}
		glossary[p] = []byte(v)
	}
	f, err := fingerprintFrame(framing, scope, glossary)
	if err != nil {
		return Frame{}, fmt.Errorf("at %s: %w", shortRev(rev), err)
	}
	return f, nil
}

// frameInWorkingTree fingerprints the three surfaces as the working tree holds
// them, through the guarded read the ledger uses. The glossary is every file
// git tracks under its directory plus every untracked one it does not ignore,
// so a term added and not yet committed is a change, and a tracked term deleted
// from the tree is absent.
func frameInWorkingTree(repoRoot string) (Frame, error) {
	chapter := func(s FrameSurface) (string, error) {
		return readRecordGuarded(filepath.Join(repoRoot, filepath.FromSlash(s.Path)))
	}
	framing, err := chapter(FrameSurfaces[0])
	if err != nil {
		return Frame{}, fmt.Errorf("the %s surface cannot be read from the working tree: %w", FrameSurfaces[0].Name, err)
	}
	scope, err := chapter(FrameSurfaces[2])
	if err != nil {
		return Frame{}, fmt.Errorf("the %s surface cannot be read from the working tree: %w", FrameSurfaces[2].Name, err)
	}
	out, err := gitutil.RunCapped(repoRoot, maxStatusBytes,
		"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--", FrameSurfaces[1].Path)
	if err != nil {
		return Frame{}, fmt.Errorf("cannot list the glossary in the working tree: %w", err)
	}
	glossary := map[string][]byte{}
	for _, p := range strings.Split(out, "\x00") {
		if p == "" || !isGlossaryTerm(p) {
			continue
		}
		v, err := readRecordGuarded(filepath.Join(repoRoot, filepath.FromSlash(p)))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return Frame{}, fmt.Errorf("the glossary term %s cannot be read: %w", p, err)
		}
		glossary[p] = []byte(v)
	}
	return fingerprintFrame(framing, scope, glossary)
}

// frameWalk is the surfaces' bounded history from HEAD backwards: the commits
// touching any of the three surfaces, newest first, each fingerprinted on
// demand. A frame state is the triple after that commit.
type frameWalk struct {
	repoRoot string
	commits  []string
	frames   map[int]Frame
	errs     map[int]error
	cache    blobCache
}

// The two shapes a walk reads the history in.
const (
	// walkMainline follows first parents only, so across a merge the state
	// before is the one the line HEAD stands on held, whichever line holds the
	// newer commits: a rewrite brought in by a --no-ff merge is recorded as a
	// squash or a rebase of the same branch would record it, and the outcome is
	// fixed by topology, never by timestamps. The whole write reads this.
	walkMainline = true
	// walkEveryLine reads every line of history, so a before triple held only
	// on a merged branch is still found. The completion reads this: it looks
	// for one known triple, and which line holds it does not matter.
	walkEveryLine = false
)

// frameHistory reads the bounded list of commits touching the frame, along
// first parents only when mainline is set.
func frameHistory(repoRoot string, cache blobCache, mainline bool) (*frameWalk, error) {
	args := []string{"log", "--format=%H", "-n", fmt.Sprint(frameHistoryBound)}
	if mainline {
		args = append(args, "--first-parent")
	}
	args = append(append(args, "HEAD", "--"), frameSurfacePaths()...)
	out, err := gitutil.RunCapped(repoRoot, maxStatusBytes, args...)
	if err != nil {
		return nil, fmt.Errorf("cannot read the frame's history: %w", err)
	}
	w := &frameWalk{repoRoot: repoRoot, frames: map[int]Frame{}, errs: map[int]error{}, cache: cache}
	for _, ln := range strings.Split(out, "\n") {
		if ln = strings.TrimSpace(ln); ln != "" {
			w.commits = append(w.commits, ln)
		}
	}
	return w, nil
}

// at fingerprints the frame after commit i.
func (w *frameWalk) at(i int) (Frame, error) {
	if f, ok := w.frames[i]; ok {
		return f, nil
	}
	if err, ok := w.errs[i]; ok {
		return Frame{}, err
	}
	f, err := frameAtCommit(w.repoRoot, w.commits[i], w.cache)
	if err != nil {
		w.errs[i] = err
		return Frame{}, err
	}
	w.frames[i] = f
	return f, nil
}

// full reports whether the walk read as many commits as the bound allows, so a
// search that found nothing may have stopped short of what it sought.
func (w *frameWalk) full() bool { return len(w.commits) >= frameHistoryBound }

// searched renders how far a fruitless walk went, for its refusal.
func (w *frameWalk) searched() string {
	if w.full() {
		return fmt.Sprintf("the walk reached its bound of %d commits touching the frame", frameHistoryBound)
	}
	return fmt.Sprintf("the walk read the whole history, %d commit(s) touching the frame", len(w.commits))
}

func shortRev(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// --- the verb ---

// Reframe writes one reframe record, its first half, or its second half.
//
// The whole write (no flag) requires the working tree to match HEAD, walks
// the surfaces' history to the previous distinct committed state, and writes
// both halves at once. `Open` writes the before half from HEAD's triple.
// `Complete` finishes an open record once HEAD's triple differs from its before
// triple, walking back until it finds that triple within the bound. Every
// refusal writes nothing.
func Reframe(req ReframeRequest) (ReframeResult, error) {
	repoRoot, issuesRoot, err := resolveRoots(req.RepoRoot, req.IssuesRoot)
	if err != nil {
		return ReframeResult{}, err
	}
	if req.Complete != "" {
		return completeReframe(repoRoot, issuesRoot, req)
	}
	occasion := req.OccasionedBy
	if !issueschema.ValidReframeOccasion(occasion) {
		return ReframeResult{}, fmt.Errorf("%w: --occasioned-by %q is not a handle of %s; a reframe is keyed to the reading record that occasioned it, never to prose (nothing written)",
			ErrMalformedFrontmatter, occasion, reframeOccasionList())
	}
	ground, redacted, degraded, err := requireFreeGrounds(repoRoot, "reframe", req.Grounds)
	if err != nil {
		return ReframeResult{}, err
	}
	occPath, err := resolveReframeOccasion(repoRoot, occasion)
	if err != nil {
		return ReframeResult{}, err
	}
	occCommit, err := occasionCommit(repoRoot, occasion, occPath)
	if err != nil {
		return ReframeResult{}, err
	}

	cache := blobCache{}
	head, err := frameAtCommit(repoRoot, "HEAD", cache)
	if err != nil {
		return ReframeResult{}, fmt.Errorf("%w: the frame at HEAD cannot be fingerprinted: %v (nothing written)", ErrInvariantViolation, err)
	}
	result := ReframeResult{OccasionedBy: occasion, Redacted: redacted, Degraded: degraded}
	if req.Open {
		// The first half: the rewrite is not yet committed, so the occasion
		// being committed at HEAD — which occasionCommit established by finding
		// it in HEAD's history — is the whole of the predate check.
		result.Half, result.Before = ReframeHalfOpen, head
	} else {
		if err := requireWorkingTreeAtHead(repoRoot, head, "commit the rewrite, or record the first half with --open"); err != nil {
			return ReframeResult{}, err
		}
		walk, err := frameHistory(repoRoot, cache, walkMainline)
		if err != nil {
			return ReframeResult{}, err
		}
		i := 0
		for ; i < len(walk.commits); i++ {
			f, err := walk.at(i)
			if err != nil {
				return ReframeResult{}, fmt.Errorf("%w: the frame's previous state cannot be fingerprinted: %v; there is no prior committed state to record against (nothing written)",
					ErrInvariantViolation, err)
			}
			if f != head {
				break
			}
		}
		if i == len(walk.commits) || i == 0 {
			return ReframeResult{}, fmt.Errorf("%w: the frame at HEAD matches no prior committed state, so there is no reframe to record (%s; nothing written)",
				ErrInvariantViolation, walk.searched())
		}
		before, _ := walk.at(i)
		if err := requirePredates(repoRoot, occasion, occCommit, walk.commits[i-1]); err != nil {
			return ReframeResult{}, err
		}
		result.Half, result.Before, result.After = ReframeHalfWhole, before, head
		result.Changed, result.Commits = before.moved(head), i
	}

	if err := mutationPreamble(repoRoot, issuesRoot); err != nil {
		return ReframeResult{}, err
	}
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		// The occasion is resolved again where the write is decided.
		if _, err := resolveReframeOccasion(repoRoot, occasion); err != nil {
			return err
		}
		if req.Open {
			open, err := openReframes(issuesRoot)
			if err != nil {
				return err
			}
			if len(open) > 0 {
				return fmt.Errorf("%w: %s is open; a second open record could be completed against the wrong rewrite, so commit the rewrite and run `abcd capture reframe --complete %s` first (nothing written)",
					ErrInvariantViolation, strings.Join(open, ", "), open[0])
			}
		}
		id, err := minter.Mint(issueschema.ReframeFamily)
		if err != nil {
			return err
		}
		fields, fm := reframeFields(id, occasion, ground, result)
		if err := validateReframeStrict(fm); err != nil {
			return err
		}
		content, err := buildIssueText(fields, "")
		if err != nil {
			return err
		}
		dir := filepath.Join(issuesRoot, issueschema.ReframesDir)
		if err := safeMkdirLeaf(dir); err != nil {
			return err
		}
		p := filepath.Join(dir, id+".md")
		if err := refuseExistingRecord(p, id); err != nil {
			return err
		}
		if err := writeReadingRecord(ledgerBase(repoRoot, issuesRoot), p, []byte(content)); err != nil {
			return err
		}
		result.ID, result.Path = id, fsutil.RepoRel(repoRoot, p)
		return nil
	})
	if err != nil {
		return ReframeResult{}, err
	}
	return result, nil
}

// completeReframe is the second half: it finishes an open record once the
// rewrite is committed.
func completeReframe(repoRoot, issuesRoot string, req ReframeRequest) (ReframeResult, error) {
	id := req.Complete
	if req.OccasionedBy != "" || strings.TrimSpace(req.Grounds) != "" || req.Open {
		return ReframeResult{}, fmt.Errorf("%w: --complete takes the record id alone; the occasion and the ground are the first half's, and the record already carries them (nothing written)",
			ErrMalformedFrontmatter)
	}
	if !recordid.ValidReframeID(id) {
		return ReframeResult{}, fmt.Errorf("%w: --complete %q does not match ^%s-[0-9]+$ (nothing written)", ErrMalformedFrontmatter, id, issueschema.ReframeFamily)
	}
	recPath, fm, _, err := readReframeRecord(issuesRoot, id)
	if err != nil {
		return ReframeResult{}, err
	}
	before, err := openBefore(id, fm)
	if err != nil {
		return ReframeResult{}, err
	}
	occasion := asString(fm["occasioned_by"])
	occPath, err := resolveReframeOccasion(repoRoot, occasion)
	if err != nil {
		return ReframeResult{}, err
	}
	occCommit, err := occasionCommit(repoRoot, occasion, occPath)
	if err != nil {
		return ReframeResult{}, err
	}

	cache := blobCache{}
	head, err := frameAtCommit(repoRoot, "HEAD", cache)
	if err != nil {
		return ReframeResult{}, fmt.Errorf("%w: the frame at HEAD cannot be fingerprinted: %v (nothing written)", ErrInvariantViolation, err)
	}
	if err := requireWorkingTreeAtHead(repoRoot, head, "commit the rewrite, then complete "+id); err != nil {
		return ReframeResult{}, err
	}
	if head == before {
		return ReframeResult{}, fmt.Errorf("%w: the frame at HEAD is still the state the record opened against; nothing was rewritten, so commit the rewrite before completing %s (nothing written)",
			ErrInvariantViolation, id)
	}
	walk, err := frameHistory(repoRoot, cache, walkEveryLine)
	if err != nil {
		return ReframeResult{}, err
	}
	found := -1
	for i := range walk.commits {
		// A state that cannot be fingerprinted is not the one sought; the walk
		// goes on past it, and a walk that finds nothing refuses below.
		if f, err := walk.at(i); err == nil && f == before {
			found = i
			break
		}
	}
	if found < 1 {
		return ReframeResult{}, fmt.Errorf("%w: the surfaces' history no longer contains the state %s opened against, so the rewrite cannot be paired with it: before %s; HEAD %s (%s; nothing written)",
			ErrInvariantViolation, id, before, head, walk.searched())
	}
	if err := requirePredates(repoRoot, occasion, occCommit, walk.commits[found-1]); err != nil {
		return ReframeResult{}, err
	}

	result := ReframeResult{
		ID: id, Path: fsutil.RepoRel(repoRoot, recPath), OccasionedBy: occasion,
		Before: before, After: head, Changed: before.moved(head),
		Half: ReframeHalfCompleted, Commits: found,
	}
	err = withLedgerLock(repoRoot, issuesRoot, func() error {
		// Read again where the write is decided: the record must still be open,
		// against the same before triple the walk paired.
		_, fm, content, err := readReframeRecord(issuesRoot, id)
		if err != nil {
			return err
		}
		again, err := openBefore(id, fm)
		if err != nil {
			return err
		}
		if again != before {
			return fmt.Errorf("%w: %s changed while it was being completed (nothing written)", ErrInvariantViolation, id)
		}
		for _, kv := range []struct{ key, val string }{
			{"construal_after", head.Construal}, {"glossary_after", head.Glossary}, {"scope_after", head.Scope},
		} {
			if content, err = setScalarField(content, kv.key, kv.val); err != nil {
				return err
			}
		}
		if content, err = setListField(content, "changed", result.Changed); err != nil {
			return err
		}
		done, _, err := parseFrontmatterAndBody(content)
		if err != nil {
			return err
		}
		if err := validateReframeStrict(done); err != nil {
			return err
		}
		return writeReadingRecord(ledgerBase(repoRoot, issuesRoot), recPath, []byte(content))
	})
	if err != nil {
		return ReframeResult{}, err
	}
	return result, nil
}

// requireWorkingTreeAtHead refuses a working tree whose surfaces differ from
// HEAD's, naming the first one that does.
func requireWorkingTreeAtHead(repoRoot string, head Frame, remedy string) error {
	wt, err := frameInWorkingTree(repoRoot)
	if err != nil {
		return fmt.Errorf("%w: %v (nothing written)", ErrInvariantViolation, err)
	}
	if moved := head.moved(wt); len(moved) > 0 {
		verb := "has"
		if len(moved) > 1 {
			verb = "have"
		}
		return fmt.Errorf("%w: the %s %s uncommitted changes; %s (nothing written)",
			ErrInvariantViolation, strings.Join(moved, " and the "), verb, remedy)
	}
	return nil
}

// resolveReframeOccasion resolves the occasion through the one occasion
// resolver the reading chain shares.
func resolveReframeOccasion(repoRoot, occasion string) (string, error) {
	fams := make([]readingitem.Family, 0, len(issueschema.ReframeOccasionFamilies))
	for _, f := range issueschema.ReframeOccasionFamilies {
		fams = append(fams, readingitem.Family(f))
	}
	p, err := readingitem.ResolveOccasion(repoRoot, occasion, fams...)
	if err != nil {
		return "", fmt.Errorf("--occasioned-by %s does not resolve: %w (nothing written)", occasion, wrapLocatorErr(err))
	}
	return p, nil
}

// occasionCommit is the commit that added the occasion's record, found in
// HEAD's history. An occasion no commit added is refused: a reframe cannot be
// occasioned by a record that does not yet exist in history.
func occasionCommit(repoRoot, occasion, abs string) (string, error) {
	rel := fsutil.RepoRel(repoRoot, abs)
	out, err := gitutil.RunCapped(repoRoot, 4096, "log", "--diff-filter=A", "--format=%H", "-n", "1", "HEAD", "--", rel)
	if err != nil {
		return "", fmt.Errorf("cannot read the history of the occasion %s: %w (nothing written)", occasion, err)
	}
	if c := strings.TrimSpace(out); c != "" {
		return c, nil
	}
	return "", fmt.Errorf("%w: the occasion %s is not committed; a reframe cannot be occasioned by a record that does not yet exist in history (nothing written)",
		ErrInvariantViolation, occasion)
}

// requirePredates holds the one check the join carries: the commit that added
// the occasion is a strict ancestor of the rewrite. It is a floor, not a proof
// that the occasion caused the rewrite.
func requirePredates(repoRoot, occasion, occCommit, rewrite string) error {
	if occCommit != rewrite {
		ok, err := gitutil.IsAncestor(repoRoot, occCommit, rewrite)
		if err != nil {
			return fmt.Errorf("cannot order the occasion %s against the rewrite: %w (nothing written)", occasion, err)
		}
		if ok {
			return nil
		}
	}
	return fmt.Errorf("%w: the occasion %s was committed in %s, which does not precede the rewrite %s; a reframe cannot be occasioned by what came later (nothing written)",
		ErrInvariantViolation, occasion, shortRev(occCommit), shortRev(rewrite))
}

// readReframeRecord reads one reframe record by id from the flat store,
// refusing a symlinked store or leaf, and returns its path, parsed frontmatter
// and raw content.
func readReframeRecord(issuesRoot, id string) (string, map[string]any, string, error) {
	dir := filepath.Join(issuesRoot, issueschema.ReframesDir)
	if err := refuseSymlinkedDir(dir); err != nil {
		return "", nil, "", err
	}
	p := filepath.Join(dir, id+".md")
	content, err := readRecordGuarded(p)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, "", fmt.Errorf("%w: %s is not a reframe this ledger holds (nothing written)", ErrUnknownIssueID, id)
		}
		return "", nil, "", err
	}
	fm, _, err := parseFrontmatterAndBody(content)
	if err != nil {
		return "", nil, "", fmt.Errorf("%w: %s does not parse: %v (nothing written)", ErrMalformedFrontmatter, id, err)
	}
	if err := validateReframeStrict(fm); err != nil {
		return "", nil, "", fmt.Errorf("%s: %w (nothing written)", id, err)
	}
	return p, fm, content, nil
}

// openBefore returns an open record's before triple, refusing a complete one.
func openBefore(id string, fm map[string]any) (Frame, error) {
	if _, done := fm["construal_after"]; done {
		return Frame{}, fmt.Errorf("%w: %s is already complete; a reframe record is completed once (nothing written)", ErrInvariantViolation, id)
	}
	return Frame{
		Construal: asString(fm["construal_before"]),
		Glossary:  asString(fm["glossary_before"]),
		Scope:     asString(fm["scope_before"]),
	}, nil
}

// openReframes lists the ids of the open reframe records: those carrying no
// after half. A record that cannot be read is refused by name rather than
// counted either way.
func openReframes(issuesRoot string) ([]string, error) {
	dir := filepath.Join(issuesRoot, issueschema.ReframesDir)
	if err := refuseSymlinkedDir(dir); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var open []string
	for _, e := range entries {
		id := strings.TrimSuffix(e.Name(), ".md")
		if id == e.Name() || !recordid.ValidReframeID(id) {
			continue
		}
		_, fm, _, err := readReframeRecord(issuesRoot, id)
		if err != nil {
			return nil, fmt.Errorf("cannot tell whether %s is open: %w", id, err)
		}
		if _, done := fm["construal_after"]; !done {
			open = append(open, id)
		}
	}
	sort.Strings(open)
	return open, nil
}

// reframeOccasionList renders the admitted families for a refusal.
func reframeOccasionList() string {
	names := make([]string, 0, len(issueschema.ReframeOccasionFamilies))
	for _, f := range issueschema.ReframeOccasionFamilies {
		names = append(names, f+"-N")
	}
	return strings.Join(names, ", ")
}

// reframeFields assembles one reframe's frontmatter in the schema's order: the
// required set, then the after half where the write carries it — the order a
// completion leaves an opened record in, so the two spell one record alike.
func reframeFields(id, occasion, ground string, r ReframeResult) ([]kv, map[string]any) {
	fields := []kv{
		{"schema_version", 1},
		{"id", id},
		{"occasioned_by", occasion},
		{"construal_before", r.Before.Construal},
		{"glossary_before", r.Before.Glossary},
		{"scope_before", r.Before.Scope},
		{"grounds", ground},
	}
	if r.Half == ReframeHalfWhole {
		fields = append(fields,
			kv{"construal_after", r.After.Construal},
			kv{"glossary_after", r.After.Glossary},
			kv{"scope_after", r.After.Scope},
			kv{"changed", r.Changed},
		)
	}
	fm := map[string]any{}
	for _, f := range fields {
		fm[f.key] = f.val
	}
	return fields, fm
}

// validateReframeStrict holds a reframe to issueschema's declaration of the
// family: its closed key set, every required key present, the id and occasion
// well-formed, every fingerprint a 64-hex SHA-256, and the after half either
// wholly absent (open) or wholly present (complete) with `changed` naming
// exactly the surfaces whose fingerprints differ — never none of them.
func validateReframeStrict(fm map[string]any) error {
	if err := requireSchemaVersion(fm); err != nil {
		return err
	}
	for k := range fm {
		if !issueschema.ReframeKnown[k] {
			return fmt.Errorf("%w: unknown property %q on a reframe", ErrMalformedFrontmatter, k)
		}
	}
	for _, key := range issueschema.ReframeRequired[1:] {
		if err := requireNonBlankString(fm, key); err != nil {
			return err
		}
	}
	if id := asString(fm["id"]); !recordid.ValidReframeID(id) {
		return fmt.Errorf("%w: id %q does not match ^%s-[0-9]+$", ErrMalformedFrontmatter, id, issueschema.ReframeFamily)
	}
	if occ := asString(fm["occasioned_by"]); !issueschema.ValidReframeOccasion(occ) {
		return fmt.Errorf("%w: occasioned_by %q is not a handle of %s", ErrMalformedFrontmatter, occ, reframeOccasionList())
	}
	var before, after Frame
	present := 0
	for _, n := range issueschema.FrameSurfaceNames {
		b := asString(fm[n+"_before"])
		if !issueschema.ValidFingerprint(b) {
			return fmt.Errorf("%w: %s_before %q is not a 64-hex SHA-256 fingerprint", ErrMalformedFrontmatter, n, b)
		}
		setFrame(&before, n, b)
		if v, ok := fm[n+"_after"]; ok {
			present++
			a, isStr := v.(string)
			if !isStr || !issueschema.ValidFingerprint(a) {
				return fmt.Errorf("%w: %s_after %v is not a 64-hex SHA-256 fingerprint", ErrMalformedFrontmatter, n, v)
			}
			setFrame(&after, n, a)
		}
	}
	changedRaw, hasChanged := fm["changed"]
	switch {
	case present == 0 && !hasChanged:
		return nil // an open record
	case present != len(issueschema.FrameSurfaceNames) || !hasChanged:
		return fmt.Errorf("%w: a reframe's after half is the three after fingerprints and `changed` together, or none of them", ErrMalformedFrontmatter)
	}
	changed, ok := changedRaw.([]string)
	if !ok {
		return fmt.Errorf("%w: changed must be a list of surface names", ErrMalformedFrontmatter)
	}
	for _, c := range changed {
		if !issueschema.ValidFrameSurface(c) {
			return fmt.Errorf("%w: changed names %q, which is not one of %s", ErrMalformedFrontmatter, c, strings.Join(issueschema.FrameSurfaceNames, ", "))
		}
	}
	want := before.moved(after)
	if len(want) == 0 {
		return fmt.Errorf("%w: a completed reframe in which no surface moved records no reframe", ErrInvariantViolation)
	}
	if strings.Join(changed, ",") != strings.Join(want, ",") {
		return fmt.Errorf("%w: changed %v does not name the surfaces whose fingerprints differ (%v)", ErrMalformedFrontmatter, changed, want)
	}
	return nil
}

func setFrame(f *Frame, name, v string) {
	switch name {
	case "construal":
		f.Construal = v
	case "glossary":
		f.Glossary = v
	case "scope":
		f.Scope = v
	}
}
