package intent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/mdrecord"
	"github.com/intentdriven/abcd/internal/core/record/findingkey"
	"github.com/intentdriven/abcd/internal/core/record/match"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// consistency.go — Role 2 of the intent-auditor: the cross-document
// consistency pass (itd-48, spc-2609211921272106).
//
// The opponent is the other documents. One pass reads the brief and every live
// intent together and names the places two of them cannot both be right:
// terminology drift, premise contradictions, scope leakage, sequencing
// impossibilities and naming conflicts. The judgement rides the host; this file
// is the binary's half, on the same request/ingest seam as Role 1's audit:
//
//   - EMIT (EmitConsistency): assembles the corpus — every brief page and every
//     intent's press release, scope, decisions and rule — into one input file
//     under the local tier, and writes a request beside it carrying the classes,
//     the rubric and the host-computed provenance (rubric_hash, prompt_hash) the
//     Role 1 request carries, plus the commit the tree stood at.
//   - INGEST (IngestConsistency): reads the untrusted findings JSON, validates
//     it FAIL-CLOSED against the issued request and the corpus as it stands, and
//     only then writes: one capture per finding through the caller's filer (or a
//     link to the open record that already holds it) and one dated report on the
//     reviews shelf. A payload that does not validate is refused with nothing
//     written anywhere.
//
// Read-only over the record: neither half writes a brief page or an intent.
// The emit writes only the local tier; the ingest writes only the report and
// the ledger.
//
// Receipt. receipt_id = "rcp-" + first-12-hex of sha256("consistency" | scope |
// corpus digest), where the corpus digest covers every assembled document's path
// and content. It is deterministic, so a re-emit over an unchanged corpus reuses
// the receipt, and a corpus that moved between emit and ingest recomputes to a
// different one, which refuses the ingest rather than validating quotes against
// text the reviewer never read.

// ConsistencyType is the only _type the consistency ingest accepts.
const ConsistencyType = "abcd/intent-consistency-findings/v1"

// consistencyRubricID names the judging contract rubric_hash is taken over. Bump
// it when the rubric's shape changes; the hash tracks its content by itself.
const consistencyRubricID = "abcd/intent-consistency-rubric/v1"

// ConsistencyScopeCorpus is the scope of a bare run: the whole corpus.
const ConsistencyScopeCorpus = "corpus"

// ReviewsShelfRelDir is the reviews shelf the report is filed on — the
// committed working tier's `reviews/` directory, under its charter.
const ReviewsShelfRelDir = ".abcd/work/reviews"

// briefRelDir is the brief the corpus reads, every page of it.
const briefRelDir = ".abcd/development/brief"

// maxConsistencyFindings caps one payload. A pass over the corpus that returns
// more than this is not a list a person can act on, and each one files a record.
const maxConsistencyFindings = 100

// minQuoteChars is the shortest quote an end may carry, after whitespace is
// collapsed. An end is located by its quote, and the ledger search that links a
// finding to an open record matches on it, so a quote short enough to occur
// everywhere locates nothing.
const minQuoteChars = 12

// ConsistencyClasses is the closed set of judgement classes, in report order.
var ConsistencyClasses = []string{
	"terminology_drift",
	"premise_contradiction",
	"scope_leakage",
	"sequencing_impossibility",
	"naming_conflict",
}

// consistencyClassText is what each class means, stated once: the request
// hands it to the reviewer and the report heads its rows with the label.
var consistencyClassText = map[string][2]string{
	"terminology_drift":        {"terminology drift", "a term used against the glossary, or used in different senses across documents"},
	"premise_contradiction":    {"premise contradiction", "two documents asserting incompatible facts or assumptions about the same surface"},
	"scope_leakage":            {"scope leakage", "two documents claiming the same ground, so it is covered twice or covered in contradictory ways"},
	"sequencing_impossibility": {"sequencing impossibility", "a document depending on another whose scope, as written, cannot satisfy the dependency"},
	"naming_conflict":          {"naming conflict", "one name used for two concepts, or two names for one concept"},
}

// consistencyRubricRules is the canonical statement of what the ingest enforces.
// Every line names a check validateConsistency runs; the hash over it is what a
// report attests to.
var consistencyRubricRules = []string{
	"findings: each finding names exactly two ends, and each end is a document in the corpus manifest, by its path",
	"quotes: each end quotes its document verbatim, at least 12 characters once whitespace is collapsed; the quote must occur in the document as the corpus presents it",
	"ends: the two ends of one finding differ, and no two findings share a class and the same pair of ends",
	"scoped run: when the scope is one intent, every finding has at least one end in that intent",
	"fields: summary and explanation are stated for every finding; nothing else is accepted",
	"count: at most 100 findings; an empty list is a pass that found nothing",
}

// consistencyHeadingRe selects the intent sections the corpus carries: the press
// release, the scope (in and out), the decisions, and a discipline's rule. It
// matches a level-two heading only; the section runs to the next heading of
// level one or two, so its sub-headings travel with it.
var consistencyHeadingRe = regexp.MustCompile(`(?i)^##\s+(press release|what[\x{2019}']s in scope\b.*|what[\x{2019}']s out of scope\b.*|decisions\b.*|(the )?rule)\s*$`)

// h2Re and h12Re bound a level-two section.
var (
	h2Re  = regexp.MustCompile(`^##\s`)
	h12Re = regexp.MustCompile(`^#{1,2}\s`)
	h1Re  = regexp.MustCompile(`^#\s+\S`)
)

// consistencyDoc is one assembled document.
type consistencyDoc struct {
	Path     string // repo-relative, slash-separated
	IntentID string // the intent's id; empty for a brief page
	Text     string // exactly what the corpus presents for this document
	Digest   string // sha256:<hex> over Text
}

// consistencyCorpus is the assembled input for one scope.
type consistencyCorpus struct {
	Scope     string // ConsistencyScopeCorpus or an itd-N
	ScopePath string // the scoped intent's path; empty for the corpus
	Docs      []consistencyDoc
	Digest    string // sha256:<hex> over every document's path and digest
	Brief     int
	Intents   int
	// Chunk is set when the corpus is one chunk of an overflowing emit; Docs,
	// Digest and the counts are then that chunk's own.
	Chunk *consistencyChunkInfo
}

func (c consistencyCorpus) doc(path string) (consistencyDoc, bool) {
	for _, d := range c.Docs {
		if d.Path == path {
			return d, true
		}
	}
	return consistencyDoc{}, false
}

// ConsistencyEmitOptions carries what a front door adds to the request.
type ConsistencyEmitOptions struct {
	// RoutingSection is the request's routing section, as Role 1's request
	// carries it: after the provenance block, outside the hashed prompt.
	RoutingSection string
	// MaxRequestBytes is the request limit the corpus is measured against
	// (consistency.max_request_bytes, which LoadConsistencyConfig resolves); 0
	// is the bundled default.
	MaxRequestBytes int
	// MaxChunks, when positive, refuses an overflowing plan of more chunks
	// than it before anything is written, with a *ConsistencySendCapError
	// naming the count and the bytes they hold. A front door sets it on a
	// provider route, where every chunk is a request sent and paid for on its
	// own; 0 leaves only MaxConsistencyChunks.
	MaxChunks int
}

// ConsistencySendCapError is an overflowing plan refused for holding more
// chunks than ConsistencyEmitOptions.MaxChunks: Chunks is its count, Bytes the
// request and corpus bytes they would have written (and a provider route
// sent), and Cap the limit it was held to. Nothing was written.
type ConsistencySendCapError struct {
	Chunks, Bytes, Cap int
}

func (e *ConsistencySendCapError) Error() string {
	return fmt.Sprintf("intent: the corpus overflows into %d chunks, %d bytes of requests and corpus files in all, more than the %d allowed; nothing was written",
		e.Chunks, e.Bytes, e.Cap)
}

// ConsistencyOverflow is what an emit reports when the corpus is larger than
// the request limit: the corpus's size, the limit, and the chunk plan.
type ConsistencyOverflow struct {
	CorpusBytes int                     `json:"corpus_bytes"`
	LimitBytes  int                     `json:"limit_bytes"`
	Groups      int                     `json:"groups"`
	Chunks      int                     `json:"chunks"`
	Unreviewed  []ConsistencyUnreviewed `json:"unreviewed"`
}

// ConsistencyUnreviewed is a document too large for any chunk: Bytes is its
// size as the corpus frames it.
type ConsistencyUnreviewed struct {
	Path  string `json:"path"`
	Bytes int    `json:"bytes"`
}

// ConsistencyChunk is one chunk of an overflowing corpus: an ordinary request
// and corpus pair.
type ConsistencyChunk struct {
	Index       int    `json:"index"`
	ReceiptID   string `json:"receipt_id"`
	RequestPath string `json:"request_path"`
	CorpusPath  string `json:"corpus_path"`
	Documents   int    `json:"documents"`
	// Bytes is what the chunk's request and corpus file hold together, as the
	// emit wrote them; the ingest's listing leaves it out.
	Bytes int `json:"bytes,omitempty"`
}

// ConsistencyEmitResult reports one emit.
type ConsistencyEmitResult struct {
	Status          string `json:"status"` // issued
	ReceiptID       string `json:"receipt_id"`
	Scope           string `json:"scope"`
	RequestPath     string `json:"request_path"`
	CorpusPath      string `json:"corpus_path"`
	ReviewOfCommit  string `json:"review_of_commit"`
	CorpusDigest    string `json:"corpus_digest"`
	Documents       int    `json:"documents"`
	BriefDocuments  int    `json:"brief_documents"`
	IntentDocuments int    `json:"intent_documents"`
	// Dirty is true when a corpus document differs from ReviewOfCommit: the pass
	// reads the working tree, so the report is marked rather than refused
	// (itd-28's dirty-tree policy for a review pin). DirtyPaths names them.
	Dirty      bool     `json:"dirty"`
	DirtyPaths []string `json:"dirty_paths"`
	// Overflow is set, and Chunks lists every chunk, only when the corpus is
	// larger than the request limit.
	Overflow *ConsistencyOverflow `json:"overflow,omitempty"`
	Chunks   []ConsistencyChunk   `json:"chunks,omitempty"`
	// ChunkBytes is the bytes every chunk's request and corpus file hold
	// together: what a provider route sends.
	ChunkBytes int `json:"chunk_bytes,omitempty"`
}

// EmitConsistency assembles the corpus for one scope — the whole corpus when
// intentID is empty, else that intent against it — and writes the request and
// the assembled input under the local tier. It writes nothing else.
//
// The assembled corpus is measured against opts.MaxRequestBytes. Within it,
// the emit is one request and one corpus file. Over it, the emit writes one
// request and corpus pair per chunk (consistency_chunk.go), states the overflow
// and lists the chunks, and leaves RequestPath and CorpusPath empty: the whole
// corpus is not written, because nothing can send it.
func EmitConsistency(repoRoot, intentID string, opts ConsistencyEmitOptions) (ConsistencyEmitResult, error) {
	limit := opts.MaxRequestBytes
	if limit == 0 {
		limit = DefaultConsistencyRequestBytes
	}
	if limit < 0 {
		return ConsistencyEmitResult{}, fmt.Errorf("intent: the request limit %d is not a byte count", limit)
	}
	scope := ConsistencyScopeCorpus
	if intentID != "" {
		if !recordid.ValidIntentID(intentID) {
			return ConsistencyEmitResult{}, fmt.Errorf("intent: id %q must match ^itd-[0-9]+$", intentID)
		}
		scope = intentID
	}
	commit, err := headCommit(repoRoot)
	if err != nil {
		return ConsistencyEmitResult{}, err
	}
	c, err := assembleConsistency(repoRoot, scope)
	if err != nil {
		return ConsistencyEmitResult{}, err
	}
	dirty, err := dirtyCorpusPaths(repoRoot, c, commit)
	if err != nil {
		return ConsistencyEmitResult{}, err
	}
	rcp := consistencyReceipt(scope, c.Digest)
	res := ConsistencyEmitResult{
		Status: "issued", ReceiptID: rcp, Scope: scope,
		ReviewOfCommit: commit, Dirty: len(dirty) > 0, DirtyPaths: dirty, CorpusDigest: c.Digest,
		Documents: len(c.Docs), BriefDocuments: c.Brief, IntentDocuments: c.Intents,
	}
	whole := consistencyCorpusText(c, rcp)
	if len(whole) <= limit {
		if err := writeConsistencyPair(repoRoot, c, rcp, whole, commit, dirty, opts.RoutingSection); err != nil {
			return ConsistencyEmitResult{}, err
		}
		res.RequestPath = filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".request.md"))
		res.CorpusPath = filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".corpus.md"))
		return res, nil
	}

	plan, err := planConsistencyChunks(c, rcp, len(whole), limit)
	if err != nil {
		return ConsistencyEmitResult{}, err
	}
	// Every chunk is rendered and measured before the first is written: a
	// chunk over the limit is a fault in the plan, and it is never sent. The
	// measure keeps no chunk's text, and the write below renders each again
	// and writes it before rendering the next, so the emit holds one chunk at
	// a time, never the whole set.
	chunks := plan.chunkList()
	for i, ch := range plan.Chunks {
		corpusLen := len(consistencyCorpusText(ch, plan.Receipts[i]))
		if corpusLen > limit {
			return ConsistencyEmitResult{}, fmt.Errorf("intent: chunk %d of %d renders to %d bytes, over the request limit of %d; "+
				"the chunk plan is at fault, and nothing was written", i+1, len(plan.Chunks), corpusLen, limit)
		}
		chunks[i].Bytes = corpusLen + len(consistencyRequestDoc(ch, plan.Receipts[i], commit, dirty, opts.RoutingSection))
		res.ChunkBytes += chunks[i].Bytes
	}
	if opts.MaxChunks > 0 && len(plan.Chunks) > opts.MaxChunks {
		return ConsistencyEmitResult{}, &ConsistencySendCapError{Chunks: len(plan.Chunks), Bytes: res.ChunkBytes, Cap: opts.MaxChunks}
	}
	for i, ch := range plan.Chunks {
		if err := writeConsistencyPair(repoRoot, ch, plan.Receipts[i], consistencyCorpusText(ch, plan.Receipts[i]), commit, dirty, opts.RoutingSection); err != nil {
			return ConsistencyEmitResult{}, err
		}
	}
	res.Overflow, res.Chunks = plan.overflow(), chunks
	return res, nil
}

// writeConsistencyPair writes one request and the corpus file it names under
// the local tier, the corpus first: a request on disk always has its input
// beside it.
func writeConsistencyPair(repoRoot string, c consistencyCorpus, rcp, corpusText, commit string, dirty []string, routing string) error {
	if err := ensureRecordDir(repoRoot, reviewsRelDir); err != nil {
		return err
	}
	dir := filepath.Join(repoRoot, reviewsRelDir)
	corpusRel := filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".corpus.md"))
	requestRel := filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".request.md"))
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, rcp+".corpus.md"), []byte(corpusText), 0o644); err != nil {
		return fmt.Errorf("intent: writing consistency corpus %s: %w", corpusRel, err)
	}
	doc := consistencyRequestDoc(c, rcp, commit, dirty, routing)
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, rcp+".request.md"), []byte(doc), 0o644); err != nil {
		return fmt.Errorf("intent: writing consistency request %s: %w", requestRel, err)
	}
	return nil
}

// consistencyRequestDoc is the request file for one corpus: the prompt, the
// provenance block and the routing section.
func consistencyRequestDoc(c consistencyCorpus, rcp, commit string, dirty []string, routing string) string {
	body := consistencyPromptBody(c, rcp)
	doc := body + consistencyProvenanceBlock(consistencyPolicyFor(body), commit, dirty)
	if routing != "" {
		doc += "\n## Routing\n\n" + routing
	}
	return doc
}

// dirtyCorpusPaths names the corpus paths whose working-tree text is not the
// text the given commit holds: an edited or untracked corpus document, or one
// deleted (or renamed away) since it, which that commit holds and the pass did not
// read. A change under the corpus roots that the corpus skips — a brief
// template, a superseded intent still on disk — is not the pass's input, so it
// is not named. Dirtiness elsewhere in the tree is not the pass's business.
//
// The emit asks it against HEAD, the commit it pins; the ingest asks it again
// against the commit the request pins, so a mark edited out of the request, or
// a tree committed since the emit, cannot unmark a read that commit does not
// hold. The working tree is diffed against commit itself, not HEAD, for that
// second reading: the index and HEAD may both have moved since the pin.
// --no-renames names a rename as its source and its target; the untracked
// listing names every file, never a collapsed directory, so a new page inside
// an untracked directory is named.
//
// The repository's content filters are blanked (gitutil.FilterOverrides): over
// a corpus whose index stat no longer matches, git re-hashes each document
// through filter.<name>.clean, a program the repository names
// (iss-2610090821548169). A filter git still insists on fails the read rather
// than reading as clean. --ignore-submodules=dirty keeps the diff from
// starting a status inside a checked-out submodule, under the submodule's own
// config (iss-2610091935327982).
func dirtyCorpusPaths(repoRoot string, c consistencyCorpus, commit string) ([]string, error) {
	roots := []string{"--", briefRelDir, filepath.ToSlash(IntentsRelDir)}
	filters, err := gitutil.FilterOverrides(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("intent: reading how the corpus differs from %s: %w", commit, err)
	}
	diff := append(filters, "diff", "--name-only", "-z", "--no-renames", "--ignore-submodules=dirty", commit)
	changed, err := gitutil.RunCapped(repoRoot, 8<<20, append(diff, roots...)...)
	if err != nil {
		return nil, fmt.Errorf("intent: reading how the corpus differs from %s: %w", commit, err)
	}
	untracked, err := gitutil.RunCapped(repoRoot, 8<<20, append([]string{"ls-files", "-z", "--others", "--exclude-standard"}, roots...)...)
	if err != nil {
		return nil, fmt.Errorf("intent: reading the untracked corpus paths: %w", err)
	}
	seen := map[string]bool{}
	dirty := []string{}
	for _, p := range strings.Split(changed+"\x00"+untracked, "\x00") {
		if p == "" || seen[p] {
			continue
		}
		if _, inCorpus := c.doc(p); !inCorpus {
			if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(p))); !errors.Is(err, fs.ErrNotExist) {
				continue
			}
		}
		seen[p] = true
		dirty = append(dirty, p)
	}
	sort.Strings(dirty)
	return dirty, nil
}

// headCommit is the commit the tree stands at: the report names it as the
// commit the pass read (the reviews charter's review_of_commit pin).
func headCommit(repoRoot string) (string, error) {
	out, err := gitutil.Run(repoRoot, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil || !gitutil.IsFullSHA(out) {
		return "", fmt.Errorf("intent: the consistency report names the commit it read, and this tree has no commit git can name (run it in a checkout with at least one commit)")
	}
	return out, nil
}

// assembleConsistency reads the corpus for one scope. The brief is every page
// under the brief directory except a template (a name starting with `_`); the
// intents are every record outside superseded/, each reduced to its title and
// the sections consistencyHeadingRe selects. Documents are ordered by path, so
// the assembly — and the digest over it — is deterministic.
func assembleConsistency(repoRoot, scope string) (consistencyCorpus, error) {
	c := consistencyCorpus{Scope: scope}
	briefDocs, err := assembleBrief(repoRoot)
	if err != nil {
		return consistencyCorpus{}, err
	}
	corpus, err := Load(repoRoot)
	if err != nil {
		return consistencyCorpus{}, err
	}
	var intentDocs []consistencyDoc
	for _, it := range corpus.Intents {
		if it.Bucket == BucketSuperseded {
			continue
		}
		data, err := readRepoFile(filepath.Join(repoRoot, it.Path), it.Path)
		if err != nil {
			return consistencyCorpus{}, err
		}
		intentDocs = append(intentDocs, newConsistencyDoc(filepath.ToSlash(it.Path), it.ID, intentConsistencyText(string(data))))
	}
	sort.Slice(intentDocs, func(i, j int) bool { return intentDocs[i].Path < intentDocs[j].Path })
	c.Docs = append(briefDocs, intentDocs...)
	c.Brief, c.Intents = len(briefDocs), len(intentDocs)

	if scope != ConsistencyScopeCorpus {
		it, ok := corpus.Lookup(scope)
		if !ok {
			return consistencyCorpus{}, fmt.Errorf("intent: %s not found in any bucket", scope)
		}
		if it.Bucket == BucketSuperseded {
			return consistencyCorpus{}, fmt.Errorf("intent: %s is superseded; a retired record is not part of the corpus the pass reads", scope)
		}
		c.ScopePath = filepath.ToSlash(it.Path)
	}

	c.Digest = corpusDigest(c.Docs)
	return c, nil
}

// assembleBrief reads every brief page, whole, in path order. A symlink anywhere
// in the brief is refused rather than followed.
func assembleBrief(repoRoot string) ([]consistencyDoc, error) {
	root := filepath.Join(repoRoot, briefRelDir)
	if _, err := os.Lstat(root); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	var docs []consistencyDoc
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, rerr := filepath.Rel(repoRoot, p)
		if rerr != nil {
			return rerr
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return fmt.Errorf("intent: %s is a symlink (refusing to follow)", filepath.ToSlash(rel))
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") || strings.HasPrefix(d.Name(), "_") {
			return nil
		}
		data, err := readRepoFile(p, rel)
		if err != nil {
			return err
		}
		docs = append(docs, newConsistencyDoc(filepath.ToSlash(rel), "", string(data)))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("intent: reading the brief: %w", err)
	}
	sort.Slice(docs, func(i, j int) bool { return docs[i].Path < docs[j].Path })
	return docs, nil
}

func newConsistencyDoc(path, intentID, text string) consistencyDoc {
	return consistencyDoc{Path: path, IntentID: intentID, Text: text, Digest: sha256Field(text)}
}

// intentConsistencyText reduces an intent to what the pass compares: its title
// line and the selected level-two sections, each with its sub-headings, in file
// order. A heading inside a fence or a comment neither opens nor closes one.
func intentConsistencyText(content string) string {
	lines := strings.Split(content, "\n")
	mask := mdrecord.Mask(lines)
	masked := func(i int) bool { return i < len(mask) && mask[i] != 0 }
	var out []string
	for i, ln := range lines {
		if !masked(i) && h1Re.MatchString(ln) {
			out = append(out, strings.TrimRight(ln, "\r"), "")
			break
		}
	}
	for i := 0; i < len(lines); i++ {
		ln := strings.TrimRight(lines[i], "\r")
		if masked(i) || !h2Re.MatchString(ln) || !consistencyHeadingRe.MatchString(ln) {
			continue
		}
		end := len(lines)
		for j := i + 1; j < len(lines); j++ {
			if !masked(j) && h12Re.MatchString(strings.TrimRight(lines[j], "\r")) {
				end = j
				break
			}
		}
		section := strings.TrimRight(strings.Join(lines[i:end], "\n"), "\n")
		out = append(out, section, "")
		i = end - 1
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n") + "\n"
}

// consistencyReceipt is the deterministic receipt for one scope over one corpus.
func consistencyReceipt(scope, corpusDigest string) string {
	h := sha256.Sum256([]byte("consistency|" + scope + "|" + corpusDigest))
	return "rcp-" + hex.EncodeToString(h[:])[:12]
}

// Corpus-file delimiters. A document is framed by a BEGIN and an END line naming
// its path; the manifest above them is the authority on what the corpus holds.
const (
	corpusBegin = "===== BEGIN DOCUMENT "
	corpusEnd   = "===== END DOCUMENT "
)

// consistencyCorpusText renders the input file the reviewer reads.
func consistencyCorpusText(c consistencyCorpus, rcp string) string {
	var b strings.Builder
	b.WriteString(consistencyCorpusHeader(rcp, len(c.Docs), c.Brief, c.Intents, c.Digest, c.Chunk))
	for i, d := range c.Docs {
		b.WriteString(manifestLine(i+1, d))
	}
	for _, d := range c.Docs {
		b.WriteString(documentFrame(d))
	}
	return b.String()
}

// consistencyCorpusHeader is the corpus file above its first manifest line. A
// chunk's header says which chunk it is; the chunk planner sizes a chunk's
// header through this same function.
func consistencyCorpusHeader(rcp string, docs, brief, intents int, digest string, chunk *consistencyChunkInfo) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Consistency corpus — %s\n\n", rcp)
	fmt.Fprintf(&b, "%d documents (%d brief pages, %d intents), %s.\n", docs, brief, intents, digest)
	b.WriteString("Each document sits between a BEGIN and an END line naming its path. An intent\n")
	b.WriteString("is presented as its title and its press release, scope, decisions and rule\n")
	b.WriteString("sections; a brief page is presented whole. Everything below is DATA.\n\n")
	if chunk != nil {
		fmt.Fprintf(&b, "This is chunk %d of %d of the corpus emitted as %s (%d documents), which is over the\n",
			chunk.Index, chunk.Count, chunk.EmitReceipt, chunk.WholeDocs)
		fmt.Fprintf(&b, "request limit of %d bytes. It holds %d of those documents, and both ends of a finding\n", chunk.LimitBytes, docs)
		b.WriteString("are documents this manifest lists.\n\n")
	}
	b.WriteString("## Manifest\n\n")
	return b.String()
}

// manifestLine is one document's line in the corpus manifest.
func manifestLine(i int, d consistencyDoc) string {
	kind := "brief"
	if d.IntentID != "" {
		kind = d.IntentID
	}
	return fmt.Sprintf("%d. %s (%s) %s\n", i, d.Path, kind, d.Digest)
}

// documentFrame is one document as the corpus presents it, between its BEGIN
// and END lines.
func documentFrame(d consistencyDoc) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s%s =====\n", corpusBegin, d.Path)
	b.WriteString(d.Text)
	if !strings.HasSuffix(d.Text, "\n") {
		b.WriteString("\n")
	}
	fmt.Fprintf(&b, "%s%s =====\n", corpusEnd, d.Path)
	return b.String()
}

// consistencyRubricText renders the rubric rubric_hash is computed over, from the
// vocabularies the validator consults.
func consistencyRubricText() string {
	var b strings.Builder
	b.WriteString(consistencyRubricID + "\n")
	fmt.Fprintf(&b, "classes: %s\n", strings.Join(ConsistencyClasses, " | "))
	fmt.Fprintf(&b, "severities: %s\n", strings.Join(issueschema.Severities, " | "))
	for _, r := range consistencyRubricRules {
		b.WriteString(r + "\n")
	}
	return b.String()
}

// consistencyPromptBody composes the prompt the reviewer is handed — everything
// in the request above the provenance block. It is a pure function of the
// receipt, the scope and the corpus, so the ingest recomputes it byte for byte.
func consistencyPromptBody(c consistencyCorpus, rcp string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Consistency review request — %s\n\n", rcp)
	fmt.Fprintf(&b, "- receipt_id: %s\n", rcp)
	if c.Scope == ConsistencyScopeCorpus {
		fmt.Fprintf(&b, "- scope: %s (the brief and every intent, each against the rest)\n", ConsistencyScopeCorpus)
	} else {
		fmt.Fprintf(&b, "- scope: %s (%s against the rest of the corpus)\n", c.Scope, c.ScopePath)
	}
	if ch := c.Chunk; ch != nil {
		fmt.Fprintf(&b, "- chunk: %d of %d\n", ch.Index, ch.Count)
		fmt.Fprintf(&b, "- emit_receipt: %s\n", ch.EmitReceipt)
		fmt.Fprintf(&b, "- limit_bytes: %d\n", ch.LimitBytes)
	}
	fmt.Fprintf(&b, "- corpus: %s (%d documents, %s)\n\n",
		filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".corpus.md")), len(c.Docs), c.Digest)
	if ch := c.Chunk; ch != nil {
		fmt.Fprintf(&b, "The corpus emitted as %s (%d documents) is over the request limit of %d bytes, so it\n",
			ch.EmitReceipt, ch.WholeDocs, ch.LimitBytes)
		fmt.Fprintf(&b, "is read in %d chunks chosen so that every pair of its documents shares one. This request is\n", ch.Count)
		b.WriteString("one of them: its corpus file holds this chunk's documents only, and both ends of a\n")
		b.WriteString("finding are documents its manifest lists.\n\n")
		if len(ch.NotOwned) > 0 {
			b.WriteString("Another chunk reports any contradiction whose two ends sit in one of these sets, so report none here:")
			for _, set := range ch.NotOwned {
				fmt.Fprintf(&b, " [%s]", strings.Join(set, ", "))
			}
			b.WriteString("\n\n")
		}
	}
	b.WriteString("## What to find (authority; one class per finding)\n\n")
	for _, k := range ConsistencyClasses {
		t := consistencyClassText[k]
		fmt.Fprintf(&b, "- `%s` — %s: %s\n", k, t[0], t[1])
	}
	b.WriteString("\n## Rubric (authority; the contract the ingest enforces)\n\n")
	b.WriteString(consistencyRubricText())
	b.WriteString("\n## Findings shape (authority; the fields the ingest decodes, and no other)\n\n")
	b.WriteString(consistencyShape(rcp))
	b.WriteString("\nRun the intent-auditor agent's Role 2 over the corpus file; its findings\n")
	if ch := c.Chunk; ch != nil {
		b.WriteString("JSON takes the shape above. Run every chunk, then ingest all of their findings together,\n")
		b.WriteString("one file per chunk (the ingest refuses a set missing any chunk):\n\n")
		fmt.Fprintf(&b, "    abcd intent consistency ingest --findings-json <chunk 1> ... --findings-json <chunk %d>   # emit receipt %s\n",
			ch.Count, ch.EmitReceipt)
		return b.String()
	}
	b.WriteString("JSON takes the shape above. Ingest it with:\n\n")
	fmt.Fprintf(&b, "    abcd intent consistency ingest --findings-json <path>   # receipt %s\n", rcp)
	return b.String()
}

// consistencyShapeHints are the placeholders the stated findings shape shows,
// keyed by JSON name; the class and severity vocabularies are the ones the
// rubric states and the ingest checks.
func consistencyShapeHints(rcp string) map[string]string {
	return map[string]string{
		"_type":       ConsistencyType,
		"receipt_id":  rcp,
		"rubric_hash": "sha256:<the Provenance block's rubric_hash>",
		"prompt_hash": "sha256:<the Provenance block's prompt_hash>",
		"class":       strings.Join(ConsistencyClasses, " | "),
		"severity":    strings.Join(issueschema.Severities, " | "),
		"path":        "<a path the corpus manifest lists>",
		"quote":       "<verbatim from that document, at least 12 characters>",
	}
}

// consistencyShape renders the findings payload the consistency ingest decodes
// as one example object (iss-2609262011046013), from the struct itself, as the
// fidelity request states its verdict. A finding shows both of its ends,
// because the ingest requires exactly two.
func consistencyShape(rcp string) string {
	return renderShape(reflect.TypeOf(consistencyPayload{}), shapeSpec{
		hints: consistencyShapeHints(rcp),
		lens:  map[string]int{"ends": 2},
	})
}

// consistencyPolicyFor computes the provenance the host issues for one request.
func consistencyPolicyFor(promptBody string) auditPolicy {
	return auditPolicy{
		RubricHash: sha256Field(consistencyRubricText()),
		PromptHash: sha256Field(promptBody),
	}
}

// consistencyProvenanceBlock renders the block appended to the request. The
// commit and the dirty mark sit here rather than in the prompt: they are facts
// about the tree, not about the corpus, so they must not move prompt_hash.
func consistencyProvenanceBlock(p auditPolicy, commit string, dirty []string) string {
	var b strings.Builder
	b.WriteString("\n## Provenance (host-computed — echo both hashes verbatim into `policy`)\n\n")
	fmt.Fprintf(&b, "- rubric_hash: %s\n", p.RubricHash)
	fmt.Fprintf(&b, "- prompt_hash: %s\n", p.PromptHash)
	fmt.Fprintf(&b, "- review_of_commit: %s\n", commit)
	fmt.Fprintf(&b, "- dirty: %t\n", len(dirty) > 0)
	for _, d := range dirty {
		fmt.Fprintf(&b, "- dirty_path: %s\n", oneLine(d))
	}
	b.WriteString("\nDo not compute these yourself. `abcd intent consistency ingest` recomputes\n")
	b.WriteString("both hashes and refuses findings carrying any other value.\n")
	return b.String()
}

// ---------------------------------------------------------------------------
// Ingest
// ---------------------------------------------------------------------------

type consistencyPayload struct {
	Type      string                   `json:"_type"`
	ReceiptID string                   `json:"receipt_id"`
	Verifier  verdictVerifier          `json:"verifier"`
	Policy    verdictPolicy            `json:"policy"`
	Findings  []consistencyFindingJSON `json:"findings"`
}

type consistencyFindingJSON struct {
	Class       string               `json:"class"`
	Severity    string               `json:"severity"`
	Summary     string               `json:"summary"`
	Explanation string               `json:"explanation"`
	Ends        []consistencyEndJSON `json:"ends"`
}

type consistencyEndJSON struct {
	Path  string `json:"path"`
	Quote string `json:"quote"`
}

// ConsistencyEnd is one located end of a finding. Quote is the reviewer's
// quotation made single-line and inert; Line is where the binary found it.
type ConsistencyEnd struct {
	Path     string `json:"path"`
	Line     int    `json:"line"`
	Quote    string `json:"quote"`
	IntentID string `json:"intent_id,omitempty"`
}

// ConsistencyFinding is one validated finding. Every text field is the
// reviewer's prose made single-line and inert (oneLine); it is not yet
// redacted — each writer redacts on its own write.
//
// Key is the finding's stable key (findingkey.Derive over the class and the two
// ends), the same for the same contradiction on every run: the ledger joins a
// finding found again to the record that already holds it by this key
// (itd-2609291923559186, G1).
type ConsistencyFinding struct {
	Number      int               `json:"number"`
	Key         string            `json:"key"`
	Class       string            `json:"class"`
	Severity    string            `json:"severity"`
	Summary     string            `json:"summary"`
	Explanation string            `json:"explanation"`
	Ends        [2]ConsistencyEnd `json:"ends"`
}

// ClassLabel is the class as prose.
func (f ConsistencyFinding) ClassLabel() string { return consistencyClassText[f.Class][0] }

// IntentIDs is the intents the finding's ends sit in, deduplicated, in end order.
func (f ConsistencyFinding) IntentIDs() []string {
	var out []string
	for _, e := range f.Ends {
		if e.IntentID != "" && (len(out) == 0 || out[0] != e.IntentID) {
			out = append(out, e.IntentID)
		}
	}
	return out
}

// consistencyReview is a validated payload with everything the writers need.
type consistencyReview struct {
	ReceiptID     string
	Scope         string
	ScopePath     string
	Commit        string
	DirtyPaths    []string // non-empty when the corpus read differed from Commit
	CorpusDigest  string
	Documents     int
	Verifier      verdictVerifier
	PayloadDigest string
	Findings      []ConsistencyFinding
	// Chunk is set when the payload answers one chunk of an overflowing
	// emit; on the merged review of a chunk set, ReceiptID is the emit's.
	Chunk *consistencyReviewChunk
	// Verifiers is every distinct verifier of a merged chunk set, in chunk
	// order, when more than one ran the chunks.
	Verifiers []verdictVerifier
	// SetAside counts a merged chunk set's findings on a pair another chunk
	// owns, which are not filed.
	SetAside int
}

// consistencyReviewChunk is the chunk a payload answers, and the plan it is
// one chunk of.
type consistencyReviewChunk struct {
	Index, Count int
	EmitReceipt  string
	Overflow     *ConsistencyOverflow
	Chunks       []ConsistencyChunk
	Owners       consistencyOwners
}

// ConsistencyFiling is the ledger's answer for one finding: the record filed
// for it, the open record that already held it (Linked), or the resolved or
// won't-fix record that already answered it (Addressed). IssueID names the
// record in every case.
type ConsistencyFiling struct {
	IssueID string `json:"issue_id"`
	Linked  bool   `json:"linked"`
	// Addressed is true when a resolved or won't-fix record carries the
	// finding's key: nothing is filed, and nothing is written to that record.
	// Folder is that record's status folder.
	Addressed bool   `json:"addressed"`
	Folder    string `json:"folder,omitempty"`
	// Match is the filing-time match's outcome (itd-2609212137116617) on a
	// record the ledger filed, when the ingest asked for one.
	Match *match.Outcome `json:"match,omitempty"`
}

// ConsistencyFiler files one finding in the ledger, or names the open record
// that already holds it. reportRel is the report the finding is evidenced by.
// The intent store cannot reach the ledger's core (the ledger reads intents), so
// the caller supplies it.
type ConsistencyFiler func(f ConsistencyFinding, reportRel string) (ConsistencyFiling, error)

// ConsistencyCheck holds one finding, as the filer would write it, to the
// record gate the filed record must pass, and refuses it with nothing written.
// It is asked of EVERY finding before the first is filed, so a refusal leaves
// the ledger as it was. The filer's caller supplies it beside the filer,
// because only the ledger knows the text a finding is filed as
// (iss-2609261835118276).
type ConsistencyCheck func(f ConsistencyFinding, reportRel string) error

// ConsistencyIngestRequest is one ingest.
type ConsistencyIngestRequest struct {
	RepoRoot string
	Payload  []byte
	// Payloads is one payload per chunk of an overflowing emit; Payload alone
	// is the single case.
	Payloads [][]byte
	// Date is the report's date, YYYY-MM-DD; empty is today in UTC.
	Date  string
	File  ConsistencyFiler
	Check ConsistencyCheck
}

// The states a report row is in: filed as a new record, linked to the open
// record that already held it, or addressed by a resolved or won't-fix record
// carrying its key.
const (
	ConsistencyStateFiled     = "filed"
	ConsistencyStateLinked    = "linked"
	ConsistencyStateAddressed = "addressed"
)

// ConsistencyAddress names the record that already answered a finding: its id
// and the status folder it sits in (resolved or wontfix).
type ConsistencyAddress struct {
	IssueID string `json:"issue_id"`
	Folder  string `json:"folder"`
}

// ConsistencyRow is one finding as the report and the result carry it. State
// says which ledger answer the row got; Linked stays beside it for the --json
// consumers that read it.
type ConsistencyRow struct {
	ConsistencyFinding
	IssueID     string              `json:"issue_id"`
	State       string              `json:"state"`
	Linked      bool                `json:"linked"`
	AddressedBy *ConsistencyAddress `json:"addressed_by,omitempty"`
	Match       *match.Outcome      `json:"match,omitempty"`
}

// newConsistencyRow is the row for a finding and the ledger's answer to it.
func newConsistencyRow(f ConsistencyFinding, filing ConsistencyFiling) ConsistencyRow {
	row := ConsistencyRow{ConsistencyFinding: f, IssueID: filing.IssueID, State: ConsistencyStateFiled, Linked: filing.Linked, Match: filing.Match}
	switch {
	case filing.Addressed:
		row.State, row.Linked = ConsistencyStateAddressed, false
		row.AddressedBy = &ConsistencyAddress{IssueID: filing.IssueID, Folder: filing.Folder}
	case filing.Linked:
		row.State = ConsistencyStateLinked
	}
	return row
}

// ConsistencyIngestResult reports one ingest.
type ConsistencyIngestResult struct {
	Status         string           `json:"status"` // ingested | noop
	ReceiptID      string           `json:"receipt_id"`
	Scope          string           `json:"scope"`
	ReportPath     string           `json:"report_path"`
	ReviewOfCommit string           `json:"review_of_commit"`
	Dirty          bool             `json:"dirty"`
	Findings       int              `json:"findings"`
	Filed          []string         `json:"filed"`
	Linked         []string         `json:"linked"`
	Addressed      []string         `json:"addressed"`
	Rows           []ConsistencyRow `json:"rows"`
	// Overflow and Chunks are set when the payloads were an overflowing
	// emit's chunks.
	Overflow *ConsistencyOverflow `json:"overflow,omitempty"`
	Chunks   []ConsistencyChunk   `json:"chunks,omitempty"`
	// SetAside counts the findings a chunk reported on a pair another chunk
	// owns: neither filed nor refused.
	SetAside int `json:"set_aside,omitempty"`
}

// ReadConsistencyFindings reads a findings file the way the Role 1 ingest reads
// a verdict (guarded, capped), for a front door that reports from the payload.
func ReadConsistencyFindings(path string) ([]byte, error) {
	return readVerdictFile(path)
}

// IngestConsistency validates the payload against the issued request and the
// corpus as it stands, then files the findings and writes the report. Nothing is
// written unless the whole payload validates. A payload already ingested — the
// same receipt and the same bytes, found on the shelf — is a noop naming the
// report that holds it.
func IngestConsistency(req ConsistencyIngestRequest) (ConsistencyIngestResult, error) {
	if req.File == nil {
		return ConsistencyIngestResult{}, fmt.Errorf("intent: consistency ingest has no ledger to file findings in")
	}
	if req.Check == nil {
		return ConsistencyIngestResult{}, fmt.Errorf("intent: consistency ingest has no prose-citation check for the records it files; refusing to ingest (nothing written)")
	}
	date := req.Date
	if date == "" {
		date = time.Now().UTC().Format(time.DateOnly)
	}
	if _, err := time.Parse(time.DateOnly, date); err != nil {
		return ConsistencyIngestResult{}, fmt.Errorf("intent: report date %q is not YYYY-MM-DD", date)
	}
	payloads := req.Payloads
	switch {
	case len(payloads) == 0:
		payloads = [][]byte{req.Payload}
	case req.Payload != nil:
		return ConsistencyIngestResult{}, fmt.Errorf("intent: consistency ingest was handed a payload and a chunk set at once; refusing to ingest (nothing written)")
	}
	// Every payload is validated whole before the set is judged, and the set
	// before anything is written.
	reviews := make([]consistencyReview, 0, len(payloads))
	for i, raw := range payloads {
		rv, err := validateConsistency(req.RepoRoot, raw)
		if err != nil {
			if len(payloads) > 1 {
				return ConsistencyIngestResult{}, fmt.Errorf("payload %d of %d: %w", i+1, len(payloads), err)
			}
			return ConsistencyIngestResult{}, err
		}
		reviews = append(reviews, rv)
	}
	rv, err := mergeConsistencyReviews(reviews)
	if err != nil {
		return ConsistencyIngestResult{}, err
	}
	res := ConsistencyIngestResult{
		ReceiptID: rv.ReceiptID, Scope: rv.Scope, ReviewOfCommit: rv.Commit, Dirty: len(rv.DirtyPaths) > 0,
		Findings: len(rv.Findings), Filed: []string{}, Linked: []string{}, Addressed: []string{}, Rows: []ConsistencyRow{},
	}
	if rv.Chunk != nil {
		res.Overflow, res.Chunks, res.SetAside = rv.Chunk.Overflow, rv.Chunk.Chunks, rv.SetAside
	}
	existing, err := findConsistencyReport(req.RepoRoot, rv.ReceiptID, rv.PayloadDigest)
	if err != nil {
		return ConsistencyIngestResult{}, err
	}
	if existing != "" {
		res.Status, res.ReportPath = "noop", existing
		return res, nil
	}
	// The free-text renderer is built before anything is written, so a degraded
	// detector stops the ingest before the first capture.
	free, degraded, err := newVerdictProse(req.RepoRoot)
	if err != nil {
		return ConsistencyIngestResult{}, err
	}
	dirName, err := nextConsistencyReportDir(req.RepoRoot, date, rv.Scope)
	if err != nil {
		return ConsistencyIngestResult{}, err
	}
	reportRel := ReviewsShelfRelDir + "/" + dirName + "/00-summary.md"

	// Every finding is held to the record gate before the first is filed: a
	// refusal part-way through the filing would leave the findings before it
	// in the ledger with no report to cite.
	for _, f := range rv.Findings {
		if err := req.Check(f, reportRel); err != nil {
			return ConsistencyIngestResult{}, fmt.Errorf("intent: finding %d of %d: %w", f.Number, len(rv.Findings), err)
		}
	}
	for _, f := range rv.Findings {
		filing, err := req.File(f, reportRel)
		if err != nil {
			return ConsistencyIngestResult{}, fmt.Errorf("intent: filing finding %d of %d: %w; filed before it: %s — "+
				"no report was written, and ingesting the same findings again links those records rather than filing them twice",
				f.Number, len(rv.Findings), err, orNone(res.Filed))
		}
		row := newConsistencyRow(f, filing)
		res.Rows = append(res.Rows, row)
		switch row.State {
		case ConsistencyStateAddressed:
			res.Addressed = append(res.Addressed, filing.IssueID)
		case ConsistencyStateLinked:
			res.Linked = append(res.Linked, filing.IssueID)
		default:
			res.Filed = append(res.Filed, filing.IssueID)
		}
	}

	// Every finding is in the ledger by now, each citing reportRel as its
	// evidence, so a failure from here on says which records those are.
	afterFiling := func(err error) error {
		return fmt.Errorf("%w; the findings were already filed as %s and linked to %s, each citing %s as its evidence, "+
			"which this ingest did not write — ingesting the same findings again links those records rather than filing them twice",
			err, orNone(res.Filed), orNone(res.Linked), reportRel)
	}
	report := renderConsistencyReport(rv, res.Rows, date, free)
	if err := degraded(); err != nil {
		return ConsistencyIngestResult{}, afterFiling(err)
	}
	if err := ensureRecordDir(req.RepoRoot, filepath.Join(ReviewsShelfRelDir, dirName)); err != nil {
		return ConsistencyIngestResult{}, afterFiling(err)
	}
	// Create-only: the shelf is append-only, so a report is never overwritten.
	fh, err := os.OpenFile(filepath.Join(req.RepoRoot, filepath.FromSlash(reportRel)), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return ConsistencyIngestResult{}, afterFiling(fmt.Errorf("intent: creating report %s: %w", reportRel, err))
	}
	if _, err := fh.WriteString(report); err != nil {
		fh.Close()
		return ConsistencyIngestResult{}, afterFiling(fmt.Errorf("intent: writing report %s: %w", reportRel, err))
	}
	if err := fh.Close(); err != nil {
		return ConsistencyIngestResult{}, afterFiling(fmt.Errorf("intent: writing report %s: %w", reportRel, err))
	}
	res.Status, res.ReportPath = "ingested", reportRel
	return res, nil
}

// mergeConsistencyReviews joins the validated payloads of one ingest into the
// review the report is written from. One unchunked payload is that review. A
// chunk set must hold exactly one payload for every chunk of one emit: a chunk
// missing or given twice, chunks of two emits, and an unchunked payload beside
// others are each refused. A finding on a pair another chunk owns is set aside,
// so findings of different chunks that share a key are one finding, the owning
// chunk's copy: a key names one pair, one pair has one owning chunk, and one
// payload never repeats a key. The merged findings are numbered in chunk order.
func mergeConsistencyReviews(reviews []consistencyReview) (consistencyReview, error) {
	if len(reviews) == 1 && reviews[0].Chunk == nil {
		return reviews[0], nil
	}
	for _, rv := range reviews {
		if rv.Chunk == nil {
			return consistencyReview{}, fmt.Errorf("intent: the findings for %s answer an unchunked request, which is ingested alone, "+
				"and %d payloads were given; refusing to ingest (nothing written)", rv.ReceiptID, len(reviews))
		}
	}
	first := reviews[0].Chunk
	byIndex := map[int]consistencyReview{}
	for _, rv := range reviews {
		ch := rv.Chunk
		if ch.EmitReceipt != first.EmitReceipt || ch.Count != first.Count || rv.Commit != reviews[0].Commit {
			return consistencyReview{}, fmt.Errorf("intent: the payloads answer two emits (chunk %d of %d of %s at %s, and chunk %d of %d of %s at %s); "+
				"ingest every chunk of one emit together; refusing to ingest (nothing written)",
				first.Index, first.Count, first.EmitReceipt, reviews[0].Commit, ch.Index, ch.Count, ch.EmitReceipt, rv.Commit)
		}
		if _, dup := byIndex[ch.Index]; dup {
			return consistencyReview{}, fmt.Errorf("intent: chunk %d of %d of %s is given twice; give each chunk's findings once; refusing to ingest (nothing written)",
				ch.Index, ch.Count, ch.EmitReceipt)
		}
		byIndex[ch.Index] = rv
	}
	var missing []string
	for i := 1; i <= first.Count; i++ {
		if _, ok := byIndex[i]; !ok {
			missing = append(missing, fmt.Sprintf("chunk %d of %d (%s)", i, first.Count, first.Chunks[i-1].ReceiptID))
		}
	}
	if len(missing) > 0 {
		return consistencyReview{}, fmt.Errorf("intent: the payloads hold %d of the %d chunks of %s; missing: %s. "+
			"The report covers every chunk, so every chunk's findings are ingested together; refusing to ingest (nothing written)",
			len(byIndex), first.Count, first.EmitReceipt, strings.Join(missing, ", "))
	}

	base := byIndex[1]
	out := consistencyReview{
		ReceiptID: first.EmitReceipt, Scope: base.Scope, ScopePath: base.ScopePath, Commit: base.Commit,
		CorpusDigest: base.CorpusDigest, Documents: base.Documents, Verifier: base.Verifier,
		Chunk:    &consistencyReviewChunk{Count: first.Count, EmitReceipt: first.EmitReceipt, Overflow: first.Overflow, Chunks: first.Chunks},
		Findings: []ConsistencyFinding{},
	}
	h := sha256.New()
	h.Write([]byte("abcd/intent-consistency-chunks/v1\n"))
	seenVerifier := map[verdictVerifier]bool{}
	for i := 1; i <= first.Count; i++ {
		rv := byIndex[i]
		fmt.Fprintf(h, "%d\x00%s\n", i, rv.PayloadDigest)
		out.DirtyPaths = unionSorted(out.DirtyPaths, rv.DirtyPaths)
		if !seenVerifier[rv.Verifier] {
			seenVerifier[rv.Verifier] = true
			out.Verifiers = append(out.Verifiers, rv.Verifier)
		}
		for _, f := range rv.Findings {
			if o := rv.Chunk.Owners.owner(f.Ends[0].Path, f.Ends[1].Path); o != 0 && o != i {
				out.SetAside++
				continue
			}
			f.Number = len(out.Findings) + 1
			out.Findings = append(out.Findings, f)
		}
	}
	if len(out.Verifiers) == 1 {
		out.Verifiers = nil
	}
	out.PayloadDigest = "sha256:" + hex.EncodeToString(h.Sum(nil))
	return out, nil
}

func orNone(ids []string) string {
	if len(ids) == 0 {
		return "none"
	}
	return strings.Join(ids, ", ")
}

// requestScopeRe, requestCommitRe and requestDirtyRe read the facts the ingest
// takes from the issued request. The scope is bound by the receipt
// recomputation and the commit by the object check; the dirty mark and its
// paths are the emit's reading of the tree, which the ingest reads again and
// widens to what the tree shows at ingest.
var (
	requestScopeRe     = regexp.MustCompile(`(?m)^- scope: (corpus|itd-[0-9]+) `)
	requestCommitRe    = regexp.MustCompile(`(?m)^- review_of_commit: ([0-9a-f]+)\s*$`)
	requestDirtyRe     = regexp.MustCompile(`(?m)^- dirty: (true|false)$`)
	requestDirtyPathRe = regexp.MustCompile(`(?m)^- dirty_path: (.+)$`)
	requestChunkRe     = regexp.MustCompile(`(?m)^- chunk: ([0-9]+) of ([0-9]+)$`)
	requestEmitRe      = regexp.MustCompile(`(?m)^- emit_receipt: (rcp-[0-9a-f]{12})$`)
	requestLimitRe     = regexp.MustCompile(`(?m)^- limit_bytes: ([0-9]+)$`)
)

// requestChunk reads the chunk a request names: nil for an unchunked
// request. The lines sit in the prompt, so prompt_hash binds them; a chunk
// line without its emit receipt and limit is refused.
func requestChunk(reqData []byte, requestRel string) (*consistencyChunkInfo, error) {
	cm := requestChunkRe.FindSubmatch(reqData)
	if cm == nil {
		return nil, nil
	}
	em, lm := requestEmitRe.FindSubmatch(reqData), requestLimitRe.FindSubmatch(reqData)
	if em == nil || lm == nil {
		return nil, fmt.Errorf("intent: request %s names a chunk but no emit_receipt or limit_bytes line; refusing to ingest", requestRel)
	}
	i, ierr := strconv.Atoi(string(cm[1]))
	n, nerr := strconv.Atoi(string(cm[2]))
	limit, lerr := strconv.Atoi(string(lm[1]))
	if ierr != nil || nerr != nil || lerr != nil || n < 1 || i < 1 || i > n || limit < 1 {
		return nil, fmt.Errorf("intent: request %s names chunk %s of %s under a limit of %s bytes, which is no chunk; refusing to ingest",
			requestRel, cm[1], cm[2], lm[1])
	}
	return &consistencyChunkInfo{Index: i, Count: n, EmitReceipt: string(em[1]), LimitBytes: limit}, nil
}

// validateConsistency parses and fully validates a findings payload. Every
// failure is a refusal with nothing written: there is no parked marker in a
// committed record to quarantine against, so a bad payload has no home.
func validateConsistency(repoRoot string, raw []byte) (consistencyReview, error) {
	var lenient struct {
		Type      string `json:"_type"`
		ReceiptID string `json:"receipt_id"`
	}
	if err := json.Unmarshal(raw, &lenient); err != nil {
		return consistencyReview{}, fmt.Errorf("intent: findings are not parseable JSON; refusing to ingest: %w", err)
	}
	if lenient.Type != ConsistencyType {
		return consistencyReview{}, fmt.Errorf("intent: findings _type is %s, which is not %q; refusing to ingest",
			termsafe.DescribeRefused(lenient.Type), ConsistencyType)
	}
	if !rcpIDRe.MatchString(lenient.ReceiptID) {
		return consistencyReview{}, fmt.Errorf("intent: findings carry no resolvable receipt_id (malformed or absent); refusing to ingest")
	}
	rcp := lenient.ReceiptID
	requestRel := filepath.ToSlash(filepath.Join(reviewsRelDir, rcp+".request.md"))
	reqData, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(requestRel)), requestRel)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return consistencyReview{}, fmt.Errorf("intent: no consistency request was issued for %s here (unsolicited, or its request was swept); "+
				"re-emit with `abcd intent consistency [<itd-N>]` and run the pass again", rcp)
		}
		return consistencyReview{}, err
	}
	sm := requestScopeRe.FindSubmatch(reqData)
	cm := requestCommitRe.FindSubmatch(reqData)
	if sm == nil || cm == nil {
		return consistencyReview{}, fmt.Errorf("intent: request %s is not a consistency request (no scope or review_of_commit line); refusing to ingest", requestRel)
	}
	scope, commit := string(sm[1]), string(cm[1])
	if !gitutil.IsFullSHA(commit) {
		return consistencyReview{}, fmt.Errorf("intent: request %s names review_of_commit %q, which is not a full sha", requestRel, commit)
	}
	if _, err := gitutil.Run(repoRoot, "cat-file", "-e", commit+"^{commit}"); err != nil {
		return consistencyReview{}, fmt.Errorf("intent: request %s names review_of_commit %s, which is no commit here", requestRel, commit)
	}
	dm := requestDirtyRe.FindSubmatch(reqData)
	if dm == nil {
		return consistencyReview{}, fmt.Errorf("intent: request %s does not say whether the corpus it read was committed (no dirty line); "+
			"re-emit with `abcd intent consistency%s` and run the pass again", requestRel, scopeArg(scope))
	}
	var dirtyPaths []string
	for _, m := range requestDirtyPathRe.FindAllSubmatch(reqData, -1) {
		dirtyPaths = append(dirtyPaths, string(m[1]))
	}
	if (string(dm[1]) == "true") != (len(dirtyPaths) > 0) {
		return consistencyReview{}, fmt.Errorf("intent: request %s says dirty: %s but names %d dirty path(s); refusing to ingest", requestRel, dm[1], len(dirtyPaths))
	}

	chunk, err := requestChunk(reqData, requestRel)
	if err != nil {
		return consistencyReview{}, err
	}
	emitRcp := rcp
	if chunk != nil {
		emitRcp = chunk.EmitReceipt
	}

	c, err := assembleConsistency(repoRoot, scope)
	if err != nil {
		return consistencyReview{}, err
	}
	if now := consistencyReceipt(scope, c.Digest); now != emitRcp {
		return consistencyReview{}, fmt.Errorf("intent: the corpus has moved since %s was issued (it now reads as %s), so the findings judge text the tree no longer holds; "+
			"re-emit with `abcd intent consistency%s` and run the pass again", emitRcp, now, scopeArg(scope))
	}
	// A chunk's findings are judged against the chunk, recomputed from the
	// corpus and the limit the request states: its manifest is the one the
	// reviewer read, so an end outside the chunk is refused.
	view := c
	var reviewChunk *consistencyReviewChunk
	if chunk != nil {
		whole := len(consistencyCorpusText(c, emitRcp))
		if whole <= chunk.LimitBytes {
			return consistencyReview{}, fmt.Errorf("intent: request %s names chunk %d of %d, but the corpus is %d bytes, within its limit of %d; "+
				"re-emit with `abcd intent consistency%s` and run the pass again", requestRel, chunk.Index, chunk.Count, whole, chunk.LimitBytes, scopeArg(scope))
		}
		plan, err := planConsistencyChunks(c, emitRcp, whole, chunk.LimitBytes)
		if err != nil {
			return consistencyReview{}, err
		}
		if len(plan.Chunks) != chunk.Count || plan.Receipts[chunk.Index-1] != rcp {
			return consistencyReview{}, fmt.Errorf("intent: request %s names chunk %d of %d, which the corpus no longer plans as %s; "+
				"re-emit with `abcd intent consistency%s` and run the pass again", requestRel, chunk.Index, chunk.Count, rcp, scopeArg(scope))
		}
		view = plan.Chunks[chunk.Index-1]
		reviewChunk = &consistencyReviewChunk{Index: chunk.Index, Count: chunk.Count, EmitReceipt: emitRcp,
			Overflow: plan.overflow(), Chunks: plan.chunkList(), Owners: plan.Owners}
	}
	// The request's mark is local-tier text: read the tree again against the
	// pinned commit and keep the union, so the report is marked with at least
	// what the tree shows now — marked, never unmarked, and not refused on a
	// disagreement (itd-28's dirty-tree policy is to mark, not block).
	now, err := dirtyCorpusPaths(repoRoot, c, commit)
	if err != nil {
		return consistencyReview{}, err
	}
	dirtyPaths = unionSorted(dirtyPaths, now)

	dec := json.NewDecoder(strings.NewReader(string(raw)))
	dec.DisallowUnknownFields()
	var p consistencyPayload
	if err := dec.Decode(&p); err != nil {
		return consistencyReview{}, fmt.Errorf("intent: malformed findings JSON: %s; refusing to ingest", redactRefused(repoRoot, err.Error()))
	}
	if dec.More() {
		return consistencyReview{}, fmt.Errorf("intent: findings JSON carries more than one value; refusing to ingest")
	}
	for _, h := range [][2]string{{"policy.rubric_hash", p.Policy.RubricHash}, {"policy.prompt_hash", p.Policy.PromptHash}} {
		if !sha256FieldRe.MatchString(h[1]) {
			return consistencyReview{}, fmt.Errorf("intent: %s is required as sha256:<64 lowercase hex>, not %s; refusing to ingest",
				h[0], termsafe.DescribeRefused(h[1]))
		}
	}
	want := consistencyPolicyFor(consistencyPromptBody(view, rcp))
	if p.Policy.RubricHash != want.RubricHash || p.Policy.PromptHash != want.PromptHash {
		return consistencyReview{}, issuedPolicyRefusal("findings payload", rcp, p.Policy, want, "abcd intent consistency"+scopeArg(scope))
	}
	if strings.TrimSpace(p.Verifier.ID) == "" {
		return consistencyReview{}, fmt.Errorf("intent: verifier.id is required; refusing to ingest")
	}
	if p.Findings == nil {
		return consistencyReview{}, fmt.Errorf("intent: findings is required (an empty list is a pass that found nothing); refusing to ingest")
	}
	if len(p.Findings) > maxConsistencyFindings {
		return consistencyReview{}, fmt.Errorf("intent: %d findings exceed the cap of %d; refusing to ingest", len(p.Findings), maxConsistencyFindings)
	}

	findings, err := validateConsistencyFindings(repoRoot, view, p.Findings)
	if err != nil {
		return consistencyReview{}, fmt.Errorf("intent: %v; refusing to ingest (nothing written)", err)
	}
	return consistencyReview{
		ReceiptID: rcp, Scope: scope, ScopePath: c.ScopePath, Commit: commit, DirtyPaths: dirtyPaths,
		CorpusDigest: c.Digest, Documents: len(c.Docs), Verifier: p.Verifier,
		PayloadDigest: sha256Field(string(raw)), Findings: findings, Chunk: reviewChunk,
	}, nil
}

// unionSorted is the sorted set of the paths either list names.
func unionSorted(a, b []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, p := range append(append([]string{}, a...), b...) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

func scopeArg(scope string) string {
	if scope == ConsistencyScopeCorpus {
		return ""
	}
	return " " + scope
}

// validateConsistencyFindings checks every finding against the corpus and
// locates its ends.
func validateConsistencyFindings(repoRoot string, c consistencyCorpus, in []consistencyFindingJSON) ([]ConsistencyFinding, error) {
	classes := map[string]bool{}
	for _, k := range ConsistencyClasses {
		classes[k] = true
	}
	severities := map[string]bool{}
	for _, s := range issueschema.Severities {
		severities[s] = true
	}
	seen := map[string]int{}
	out := make([]ConsistencyFinding, 0, len(in))
	for i, f := range in {
		n := i + 1
		if !classes[f.Class] {
			return nil, fmt.Errorf("finding %d has class %s, not one of %s", n, termsafe.DescribeRefused(f.Class), strings.Join(ConsistencyClasses, " | "))
		}
		if !severities[f.Severity] {
			return nil, fmt.Errorf("finding %d has severity %s, not one of %s", n, termsafe.DescribeRefused(f.Severity), strings.Join(issueschema.Severities, " | "))
		}
		if strings.TrimSpace(f.Summary) == "" || strings.TrimSpace(f.Explanation) == "" {
			return nil, fmt.Errorf("finding %d states no summary or no explanation", n)
		}
		if len(f.Ends) != 2 {
			return nil, fmt.Errorf("finding %d names %d ends; a contradiction has exactly two", n, len(f.Ends))
		}
		var ends [2]ConsistencyEnd
		for j, e := range f.Ends {
			end, err := locateEnd(repoRoot, c, e)
			if err != nil {
				return nil, fmt.Errorf("finding %d end %d: %v", n, j+1, err)
			}
			ends[j] = end
		}
		if endKey(f.Ends[0]) == endKey(f.Ends[1]) {
			return nil, fmt.Errorf("finding %d names the same end twice", n)
		}
		// The duplicate check and the row's key are one computation: two
		// findings repeat each other exactly when they carry one key.
		key := findingkey.Derive(f.Class,
			findingkey.End{Path: ends[0].Path, Quote: f.Ends[0].Quote},
			findingkey.End{Path: ends[1].Path, Quote: f.Ends[1].Quote})
		if prev, dup := seen[key]; dup {
			return nil, fmt.Errorf("finding %d repeats finding %d (same class, same two ends: key %s)", n, prev, key)
		}
		seen[key] = n
		if c.ScopePath != "" && ends[0].Path != c.ScopePath && ends[1].Path != c.ScopePath {
			return nil, fmt.Errorf("finding %d has no end in %s, and the run is scoped to it", n, c.Scope)
		}
		out = append(out, ConsistencyFinding{
			Number: n, Key: key, Class: f.Class, Severity: f.Severity,
			Summary: oneLine(f.Summary), Explanation: oneLine(f.Explanation), Ends: ends,
		})
	}
	return out, nil
}

func endKey(e consistencyEndJSON) string {
	return strings.TrimSpace(e.Path) + "\x00" + collapseSpace(e.Quote)
}

func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }

// locateEnd resolves one end against the corpus: its path must be a manifest
// document, and its quote must occur in the document as the corpus presents it.
// The line is where the quote begins in the file on disk.
//
// A refused path or quote is DESCRIBED, never quoted: the finding and end
// numbers the caller prefixes locate it in the payload, and the value itself
// reaches the terminal and the transcript with no redaction on the way
// (iss-2609290144116254).
func locateEnd(repoRoot string, c consistencyCorpus, e consistencyEndJSON) (ConsistencyEnd, error) {
	path := strings.TrimSpace(e.Path)
	d, ok := c.doc(path)
	if !ok {
		return ConsistencyEnd{}, fmt.Errorf("path is %s, which is not a document in the corpus manifest", termsafe.DescribeRefused(path))
	}
	q := collapseSpace(e.Quote)
	if len([]rune(q)) < minQuoteChars {
		return ConsistencyEnd{}, fmt.Errorf("the quote from %s is shorter than %d characters; quote enough to locate it", path, minQuoteChars)
	}
	if _, ok := findCollapsed(d.Text, q); !ok {
		return ConsistencyEnd{}, fmt.Errorf("the quote (%s) does not occur in %s as the corpus presents it", termsafe.DescribeRefused(q), path)
	}
	line := 0
	if data, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(path)), path); err == nil {
		if off, ok := findCollapsed(string(data), q); ok {
			line = 1 + strings.Count(string(data[:off]), "\n")
		}
	}
	return ConsistencyEnd{Path: path, Line: line, Quote: oneLine(q), IntentID: d.IntentID}, nil
}

// findCollapsed finds q (whitespace already collapsed) in text, treating every
// whitespace run in text as one space, and returns the byte offset in text where
// the match begins.
func findCollapsed(text, q string) (int, bool) {
	var b strings.Builder
	offsets := make([]int, 0, len(text))
	inSpace := false
	for i, r := range text {
		// unicode.IsSpace is strings.Fields' notion of whitespace, which is what
		// collapsed the quote, so a quote carrying a no-break space still matches.
		if unicode.IsSpace(r) {
			if !inSpace {
				b.WriteByte(' ')
				offsets = append(offsets, i)
				inSpace = true
			}
			continue
		}
		inSpace = false
		start := b.Len()
		b.WriteRune(r)
		for k := start; k < b.Len(); k++ {
			offsets = append(offsets, i)
		}
	}
	idx := strings.Index(b.String(), q)
	if idx < 0 {
		return 0, false
	}
	return offsets[idx], true
}

// consistencyReportDirRe is the charter's review-directory shape.
var consistencyReportDirRe = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}-[a-z0-9]+(-[a-z0-9]+)*$`)

// nextConsistencyReportDir picks the first free directory name for the report:
// `<date>-consistency[-<itd-N>]`, then `-2`, `-3`, … when a review of the same
// scope already sits on the shelf for that date. The shelf is append-only, so a
// second run the same day gets its own directory rather than an edit.
func nextConsistencyReportDir(repoRoot, date, scope string) (string, error) {
	base := date + "-consistency"
	if scope != ConsistencyScopeCorpus {
		base += "-" + scope
	}
	for n := 1; n < 100; n++ {
		name := base
		if n > 1 {
			name += "-" + strconv.Itoa(n)
		}
		if !consistencyReportDirRe.MatchString(name) {
			return "", fmt.Errorf("intent: report directory %q does not fit the reviews charter's <YYYY-MM-DD>-<scope> shape", name)
		}
		_, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(ReviewsShelfRelDir), name))
		if errors.Is(err, fs.ErrNotExist) {
			return name, nil
		}
		if err != nil {
			return "", fmt.Errorf("intent: checking %s/%s: %w", ReviewsShelfRelDir, name, err)
		}
	}
	return "", fmt.Errorf("intent: ninety-nine %s reviews already sit on the shelf for %s", base, date)
}

// findConsistencyReport returns the report on the shelf that already holds this
// receipt's ingest of these exact bytes, or "".
func findConsistencyReport(repoRoot, rcp, payloadDigest string) (string, error) {
	shelf := filepath.Join(repoRoot, filepath.FromSlash(ReviewsShelfRelDir))
	entries, err := os.ReadDir(shelf)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("intent: reading %s: %w", ReviewsShelfRelDir, err)
	}
	wantR, wantP := "\n- receipt: "+rcp+"\n", "\n- payload: "+payloadDigest+"\n"
	for _, e := range entries {
		if !e.IsDir() || !strings.Contains(e.Name(), "-consistency") {
			continue
		}
		rel := ReviewsShelfRelDir + "/" + e.Name() + "/00-summary.md"
		data, err := readRepoFile(filepath.Join(repoRoot, filepath.FromSlash(rel)), rel)
		if err != nil {
			continue
		}
		if strings.Contains(string(data), wantR) && strings.Contains(string(data), wantP) {
			return rel, nil
		}
	}
	return "", nil
}

// renderConsistencyReport renders 00-summary.md. The reviewer's prose goes
// through free (redaction, then oneLine); paths are manifest entries and ids
// are validated shapes.
func renderConsistencyReport(rv consistencyReview, rows []ConsistencyRow, date string, free proseField) string {
	var b strings.Builder
	fmt.Fprintf(&b, "---\nreview_of_commit: %s\n", rv.Commit)
	if len(rv.DirtyPaths) > 0 {
		b.WriteString("dirty: true\n")
	}
	b.WriteString("---\n")
	if rv.Scope == ConsistencyScopeCorpus {
		b.WriteString("# Consistency review — the brief and every intent\n\n")
	} else {
		fmt.Fprintf(&b, "# Consistency review — %s against the corpus\n\n", rv.Scope)
	}
	fmt.Fprintf(&b, "- date: %s\n", date)
	if rv.Scope == ConsistencyScopeCorpus {
		b.WriteString("- scope: the whole corpus — the brief and every intent outside `superseded/`, each against the rest\n")
	} else {
		fmt.Fprintf(&b, "- scope: %s (`%s`) against the rest of the corpus; every finding has an end in it\n", rv.Scope, rv.ScopePath)
	}
	fmt.Fprintf(&b, "- read: commit `%s`, corpus %s over %d documents\n", rv.Commit, rv.CorpusDigest, rv.Documents)
	if len(rv.DirtyPaths) > 0 {
		quoted := make([]string, len(rv.DirtyPaths))
		for i, p := range rv.DirtyPaths {
			quoted[i] = "`" + oneLine(p) + "`"
		}
		fmt.Fprintf(&b, "- dirty: the corpus was read from a working tree holding %d uncommitted corpus path(s), "+
			"so an end quoted from one of them may not be at that commit: %s\n", len(rv.DirtyPaths), strings.Join(quoted, ", "))
	}
	if rv.Chunk != nil {
		renderConsistencyOverflow(&b, rv)
	}
	fmt.Fprintf(&b, "- receipt: %s\n", rv.ReceiptID)
	fmt.Fprintf(&b, "- payload: %s\n", rv.PayloadDigest)
	if len(rv.Verifiers) > 1 {
		names := make([]string, len(rv.Verifiers))
		for i, v := range rv.Verifiers {
			names[i] = orFree(v.ID, free) + " " + orFree(v.Version, free)
		}
		fmt.Fprintf(&b, "- verifier: %s\n", strings.Join(names, ", "))
	} else {
		fmt.Fprintf(&b, "- verifier: %s %s\n", orFree(rv.Verifier.ID, free), orFree(rv.Verifier.Version, free))
	}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Class]++
	}
	var parts []string
	for _, k := range ConsistencyClasses {
		parts = append(parts, fmt.Sprintf("%s %d", consistencyClassText[k][0], counts[k]))
	}
	fmt.Fprintf(&b, "- findings: %d (%s)\n\n", len(rows), strings.Join(parts, " · "))
	b.WriteString("Produced by `abcd intent consistency`: the binary assembled the corpus and validated\n")
	b.WriteString("the findings against it, and the judgement rode the host. Each finding is filed in\n")
	b.WriteString("the issue ledger with this report as its evidence, linked to the open record that\n")
	b.WriteString("already held it, or marked addressed when a resolved or won't-fix record carries its\n")
	b.WriteString("key. The brief and the intents were not edited.\n\n")
	b.WriteString("## Findings\n\n")
	if len(rows) == 0 {
		b.WriteString("None: the pass found no contradiction across the corpus.\n")
		return b.String()
	}
	b.WriteString("| # | Class | Severity | Key | End A | End B | Ledger |\n|---|---|---|---|---|---|---|\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "| %d | %s | %s | `%s` | `%s` | `%s` | %s |\n", r.Number, r.ClassLabel(), r.Severity, r.Key,
			endLocation(r.Ends[0]), endLocation(r.Ends[1]), r.LedgerCell())
	}
	for _, r := range rows {
		fmt.Fprintf(&b, "\n### %d. %s\n\n", r.Number, free(r.Summary))
		fmt.Fprintf(&b, "- class: %s · severity: %s · finding_ref: %s · ledger: %s\n", r.ClassLabel(), r.Severity, r.Key, r.LedgerCell())
		for j, e := range r.Ends {
			fmt.Fprintf(&b, "- end %c: `%s` — \u201c%s\u201d\n", 'A'+j, endLocation(e), free(e.Quote))
		}
		fmt.Fprintf(&b, "\n%s\n", free(r.Explanation))
	}
	return b.String()
}

// renderConsistencyOverflow states a chunked review's overflow in the report
// header: the corpus's size against the limit, the plan, every document no
// chunk could hold, and every chunk's receipt.
func renderConsistencyOverflow(b *strings.Builder, rv consistencyReview) {
	ov := rv.Chunk.Overflow
	how := "every pair of reviewed documents shares a chunk"
	if rv.ScopePath != "" {
		how = "the scope sits in every chunk, beside each group of the rest"
	}
	fmt.Fprintf(b, "- overflow: the corpus is %d bytes, over the request limit of %d bytes, so it was read in %d chunks over %d groups (%s)\n",
		ov.CorpusBytes, ov.LimitBytes, ov.Chunks, ov.Groups, how)
	if len(ov.Unreviewed) == 0 {
		b.WriteString("- unreviewed: none\n")
	} else {
		parts := make([]string, len(ov.Unreviewed))
		for i, u := range ov.Unreviewed {
			parts[i] = fmt.Sprintf("`%s` (%d bytes)", oneLine(u.Path), u.Bytes)
		}
		fmt.Fprintf(b, "- unreviewed: %s, each larger than a chunk leaves it, so read in no chunk\n", strings.Join(parts, ", "))
	}
	for _, ch := range rv.Chunk.Chunks {
		fmt.Fprintf(b, "- chunk %d of %d: %s (%d documents)\n", ch.Index, ov.Chunks, ch.ReceiptID, ch.Documents)
	}
	if rv.SetAside > 0 {
		fmt.Fprintf(b, "- set aside: %d finding(s) a chunk reported on a pair another chunk owns\n", rv.SetAside)
	}
}

// LedgerCell is the row's ledger answer as the report and the front door name
// it: `iss-N (filed)`, `iss-N (already open)`, or `iss-N (addressed: resolved)`
// and `(addressed: wontfix)`.
func (r ConsistencyRow) LedgerCell() string {
	switch r.State {
	case ConsistencyStateAddressed:
		folder := ""
		if r.AddressedBy != nil {
			folder = r.AddressedBy.Folder
		}
		return r.IssueID + " (addressed: " + folder + ")"
	case ConsistencyStateLinked:
		return r.IssueID + " (already open)"
	}
	return r.IssueID + " (filed)"
}

// endLocation is `path:line`, or the path alone when the line was not found on
// disk.
func endLocation(e ConsistencyEnd) string {
	if e.Line > 0 {
		return e.Path + ":" + strconv.Itoa(e.Line)
	}
	return e.Path
}
