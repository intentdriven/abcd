package matrix

// checkcopy.go reads a check copy's findings into the run (spec step 7,
// decisions 22, 29 and 30). A check copy is a model's run of a check, such as
// doc fidelity, over the frozen material in its own clone; what it found is the
// model's output and is read as untrusted input: bounded, decoded strictly,
// every text made one inert line with every mention of where the copy ran
// withheld, and every position computed from the frozen material, never taken
// from the copy. A doc-fidelity copy's verdict is validated by the recorder's
// own decoder (docfidelity.ReportFindings) and never recorded where the release
// gates read; what it drafted as a correction is dropped, never applied.
//
// The findings reach the run's record before the copy's clone is deleted, and
// a copy whose findings cannot be recorded keeps its clone. Around the whole,
// the run proves it left the release gates' receipts, the review flags and the
// issue ledger as they were (snapshotGates): a check copy's findings are
// compared, never filed.

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/record/findingkey"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// CheckCopySchema is a check copy's record's schema version.
const CheckCopySchema = 1

// The bounds a check copy's answer is read within: the answer, its findings,
// a quoted sentence (the doc-fidelity recorder's own bound), a path, and the
// frozen material its positions are computed from.
const (
	maxCheckAnswerBytes = 1 << 20
	maxCheckFindings    = 200
	maxQuoteBytes       = 2048
	maxCheckPathBytes   = 512
	maxMaterialFiles    = 4096
	maxMaterialBytes    = 4 << 20
	maxMaterialTotal    = 64 << 20
	maxGateEntries      = 100000
)

// CheckForm is the shape a check copy answers in.
type CheckForm string

const (
	// CheckDocFidelity is the doc-fidelity review's verdict, as `docs
	// fidelity record` reads it.
	CheckDocFidelity CheckForm = "doc-fidelity"
	// CheckConsistency is the consistency pass's findings: each a class and
	// the two ends it sets against each other.
	CheckConsistency CheckForm = "consistency"
)

// LineSpan is where a quote stands in the frozen material, lines counted from
// one.
type LineSpan struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

func (a *LineSpan) overlaps(b *LineSpan) bool {
	return a != nil && b != nil && a.Start <= b.End && b.Start <= a.End
}

// FindingEnd is one end of a finding: the repository-relative path of the
// document, the quote collapsed to one line, and where the frozen material
// holds it, nil where it does not.
type FindingEnd struct {
	Path  string    `json:"path"`
	Quote string    `json:"quote"`
	Lines *LineSpan `json:"lines"`
}

// CheckFinding is one finding of a check copy. Key is its exact key, the one
// derivation (findingkey.Derive): for a consistency finding, the consistency
// finding key over its class and two ends; for a doc-fidelity finding, the
// same derivation over the document, the chapter and the quoted sentence.
type CheckFinding struct {
	Key      string       `json:"key"`
	Doc      string       `json:"doc,omitempty"`
	Class    string       `json:"class,omitempty"`
	Ends     []FindingEnd `json:"ends"`
	Evidence string       `json:"evidence"`
}

// CheckCopy is one check copy as the run records it. A copy whose runner
// failed, or whose answer was refused, is unanswered: its cell's failure, with
// nothing of the answer kept.
type CheckCopy struct {
	SchemaVersion int            `json:"schema_version"`
	Run           string         `json:"run"`
	Key           TrialKey       `json:"key"`
	Form          CheckForm      `json:"form"`
	Answered      bool           `json:"answered"`
	Reason        string         `json:"reason,omitempty"`
	Findings      []CheckFinding `json:"findings"`
	// Redacted counts the mentions of where the copy ran that were withheld
	// from its findings.
	Redacted int `json:"redacted"`
}

// checkCopyInputs is one check copy to read. RepoRoot is the run's own
// checkout and Store the run's clone store, each absolute; Material is the
// frozen material by path (readFrozenMaterial); Answer is the copy's runner's
// final message, and Failure the runner's failure, "" when it answered.
type checkCopyInputs struct {
	RepoRoot, Store, Run, Frozen string
	Key                          TrialKey
	Form                         CheckForm
	Material                     map[string]string
	Answer                       []byte
	Failure                      string
}

// checkCopyRel is copy key's findings record in its run's directory.
func checkCopyRel(run string, key TrialKey) string {
	return run + "/check/" + key.Task + "/copies/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".json"
}

// readCheckCopy reads one copy's answer. It never refuses: an answer that does
// not validate is the copy's failure, said in the record's reason.
func readCheckCopy(in checkCopyInputs) CheckCopy {
	c := CheckCopy{SchemaVersion: CheckCopySchema, Run: in.Run, Key: in.Key, Form: in.Form, Findings: []CheckFinding{}}
	red := newRedactor(reviewInputs{RepoRoot: in.RepoRoot, Store: in.Store, Dir: cloneDir(in.Store, in.Run, in.Key), Run: in.Run})
	reason := func(s string) string {
		s, _ = red.redact(termsafe.SanitizeLine(s))
		return clipTo(s, maxReasonBytes)
	}
	if in.Failure != "" {
		c.Reason = reason("the copy's runner gave no answer, so the copy failed: " + in.Failure)
		return c
	}
	fs, n, err := parseCheckAnswer(in.Form, in.Answer, in.Material, red)
	if err != nil {
		c.Reason = reason("the copy's answer is refused, so the copy failed: " + err.Error())
		return c
	}
	c.Answered, c.Findings, c.Redacted = true, fs, n
	return c
}

func parseCheckAnswer(form CheckForm, data []byte, material map[string]string, red redactor) ([]CheckFinding, int, error) {
	if len(data) > maxCheckAnswerBytes {
		return nil, 0, fmt.Errorf("it is %d bytes, past its bound of %d", len(data), maxCheckAnswerBytes)
	}
	switch form {
	case CheckDocFidelity:
		return docFidelityFindings(data, material, red)
	case CheckConsistency:
		return consistencyFindings(data, material, red)
	}
	return nil, 0, fmt.Errorf("%q is not a check form abcd reads", clip(string(form)))
}

// docFidelityFindings reads a doc-fidelity verdict through the recorder's own
// decoder, recording nothing.
func docFidelityFindings(data []byte, material map[string]string, red redactor) ([]CheckFinding, int, error) {
	_, sentences, err := docfidelity.ReportFindings(data)
	if err != nil {
		return nil, 0, err
	}
	if len(sentences) > maxCheckFindings {
		return nil, 0, fmt.Errorf("it names %d findings; a copy names at most %d", len(sentences), maxCheckFindings)
	}
	out := make([]CheckFinding, 0, len(sentences))
	redacted := 0
	for i, s := range sentences {
		p := s.Chapter
		if s.Doc == docfidelity.DocBrief {
			// The recorder's parse has made it a chapter's file name.
			p = docfidelity.ChaptersDir + "/" + s.Chapter
		}
		if err := checkPath(p); err != nil {
			return nil, 0, fmt.Errorf("finding %d: %w", i+1, err)
		}
		quote, n := inert(s.Sentence, red)
		evidence, m := inert(s.Evidence, red)
		redacted += n + m
		if len(evidence) > maxFindingBytes {
			return nil, 0, fmt.Errorf("finding %d's evidence is past its bound of %d bytes", i+1, maxFindingBytes)
		}
		out = append(out, CheckFinding{
			Key:  findingkey.Derive("doc-fidelity/"+s.Doc, findingkey.End{Path: p, Quote: quote}, findingkey.End{}),
			Doc:  s.Doc,
			Ends: []FindingEnd{{Path: p, Quote: quote, Lines: locate(material[p], quote)}}, Evidence: evidence,
		})
	}
	return out, redacted, nil
}

// consistencyAnswer is the consistency pass's findings as a check copy answers
// them.
type consistencyAnswer struct {
	SchemaVersion int                    `json:"schema_version"`
	Findings      []consistencyAnswerRow `json:"findings"`
}

type consistencyAnswerRow struct {
	Class       string `json:"class"`
	Severity    string `json:"severity,omitempty"`
	Summary     string `json:"summary"`
	Explanation string `json:"explanation,omitempty"`
	Ends        []struct {
		Path  string `json:"path"`
		Quote string `json:"quote"`
	} `json:"ends"`
}

// classRe is a consistency finding's class: one lower-case token.
var classRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{0,63}$`)

func consistencyFindings(data []byte, material map[string]string, red redactor) ([]CheckFinding, int, error) {
	var a consistencyAnswer
	if err := jsonstrict.Decode(data, &a); err != nil {
		return nil, 0, fmt.Errorf("it does not parse: %w", err)
	}
	if a.SchemaVersion != 1 {
		return nil, 0, fmt.Errorf("schema_version is %d; this abcd reads 1", a.SchemaVersion)
	}
	if len(a.Findings) > maxCheckFindings {
		return nil, 0, fmt.Errorf("it names %d findings; a copy names at most %d", len(a.Findings), maxCheckFindings)
	}
	out := make([]CheckFinding, 0, len(a.Findings))
	redacted := 0
	for i, f := range a.Findings {
		if !classRe.MatchString(f.Class) {
			return nil, 0, fmt.Errorf("finding %d's class is not one lower-case word", i+1)
		}
		if len(f.Ends) != 2 {
			return nil, 0, fmt.Errorf("finding %d names %d ends; a contradiction has exactly two", i+1, len(f.Ends))
		}
		summary, n := inert(f.Summary, red)
		redacted += n
		if summary == "" || len(summary) > maxFindingBytes {
			return nil, 0, fmt.Errorf("finding %d's summary is empty or past its bound of %d bytes", i+1, maxFindingBytes)
		}
		cf := CheckFinding{Class: f.Class, Evidence: summary}
		var ends [2]findingkey.End
		for j, e := range f.Ends {
			if err := checkPath(e.Path); err != nil {
				return nil, 0, fmt.Errorf("finding %d end %d: %w", i+1, j+1, err)
			}
			q, n := inert(e.Quote, red)
			redacted += n
			if q == "" || len(q) > maxQuoteBytes {
				return nil, 0, fmt.Errorf("finding %d end %d quotes nothing, or more than %d bytes", i+1, j+1, maxQuoteBytes)
			}
			ends[j] = findingkey.End{Path: e.Path, Quote: q}
			cf.Ends = append(cf.Ends, FindingEnd{Path: e.Path, Quote: q, Lines: locate(material[e.Path], q)})
		}
		cf.Key = findingkey.Derive(f.Class, ends[0], ends[1])
		out = append(out, cf)
	}
	return out, redacted, nil
}

// inert is untrusted text made one line: its whitespace collapsed, every
// terminal and direction control masked, and every mention of where the copy
// ran withheld. It returns how many mentions it withheld.
func inert(s string, red redactor) (string, int) {
	s = termsafe.SanitizeLine(strings.Join(strings.Fields(strings.ToValidUTF8(s, "?")), " "))
	return red.redact(s)
}

// checkPath refuses a finding's path unless it is a clean, slash-separated
// path inside the repository.
func checkPath(p string) error {
	switch {
	case p == "" || len(p) > maxCheckPathBytes || !utf8.ValidString(p):
		return fmt.Errorf("a path is empty or past its bound of %d bytes", maxCheckPathBytes)
	case strings.ContainsAny(p, "\\\x00") || termsafe.SanitizeLine(p) != p:
		return fmt.Errorf("the path %q carries a backslash or a control", clip(termsafe.SanitizeLine(p)))
	case path.IsAbs(p) || rooted(p) || path.Clean(p) != p || p == ".." || strings.HasPrefix(p, "../"):
		return fmt.Errorf("the path %q is not a clean path inside the repository", clip(p))
	}
	return nil
}

// locate is where text holds quote, whitespace collapsed on both sides, at its
// first occurrence: nil where it holds none.
func locate(text, quote string) *LineSpan {
	q := strings.Join(strings.Fields(quote), " ")
	if q == "" || text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	var b strings.Builder
	starts := make([]int, len(lines))
	for i, l := range lines {
		w := strings.Join(strings.Fields(l), " ")
		if b.Len() > 0 && w != "" {
			b.WriteByte(' ')
		}
		starts[i] = b.Len()
		b.WriteString(w)
	}
	at := strings.Index(b.String(), q)
	if at < 0 {
		return nil
	}
	lineOf := func(off int) int { return sort.Search(len(starts), func(i int) bool { return starts[i] > off }) }
	return &LineSpan{Start: lineOf(at), End: lineOf(at + len(q) - 1)}
}

// readFrozenMaterial reads every file of the frozen commit at or under each of
// material's paths from the run's own checkout, never from a copy's clone, so
// the positions a run matches findings by are the frozen commit's. A link is
// not read, and a path naming no file is refused.
func readFrozenMaterial(ctx context.Context, repoRoot, frozen string, material []string) (map[string]string, error) {
	tree, err := readFrozenTree(ctx, repoRoot, frozen)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(tree.files))
	for p := range tree.files {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	out := map[string]string{}
	total := 0
	for _, m := range material {
		m = strings.TrimSuffix(m, "/")
		if err := checkPath(m); err != nil {
			return nil, fmt.Errorf("the material: %w", err)
		}
		named := false
		for _, p := range paths {
			if p != m && !strings.HasPrefix(p, m+"/") {
				continue
			}
			named = true
			e := tree.files[p]
			if e.mode == "120000" {
				continue
			}
			if _, done := out[p]; done {
				continue
			}
			if len(out) >= maxMaterialFiles || e.size > maxMaterialBytes || total+int(e.size) > maxMaterialTotal {
				return nil, fmt.Errorf("the material under %q is past its bounds (%d files, %d bytes a file, %d in all)",
					clip(m), maxMaterialFiles, maxMaterialBytes, maxMaterialTotal)
			}
			data, err := readBlob(ctx, repoRoot, e.oid, e.size)
			if err != nil {
				return nil, err
			}
			total += len(data)
			out[p] = string(data)
		}
		if !named {
			return nil, fmt.Errorf("the material %q names no file of the frozen commit", clip(m))
		}
	}
	return out, nil
}

// gateRoots is what the release gates and the ledger read, and what a check
// run must leave as it found it: the doc-fidelity receipts, the review flags
// and the issue ledger.
var gateRoots = []string{docfidelity.ReceiptsDir, docfidelity.FlagsPath, recordid.IssuesRelDir}

// gateSnapshot is every entry under gateRoots, by repository-relative path:
// a regular file by its bytes' digest, a link by its target's, a directory as
// one.
type gateSnapshot map[string]string

func snapshotGates(repoRoot string) (gateSnapshot, error) {
	s := gateSnapshot{}
	for _, rel := range gateRoots {
		root := filepath.Join(repoRoot, filepath.FromSlash(rel))
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if p == root && errors.Is(err, fs.ErrNotExist) {
					return nil
				}
				return err
			}
			if len(s) >= maxGateEntries {
				return fmt.Errorf("more than %d entries under %s", maxGateEntries, rel)
			}
			k, err := filepath.Rel(repoRoot, p)
			if err != nil {
				return err
			}
			k = filepath.ToSlash(k)
			switch t := d.Type(); {
			case t.IsRegular():
				sum, err := fileDigest(p)
				if err != nil {
					return err
				}
				s[k] = "file:" + sum
			case t&fs.ModeSymlink != 0:
				target, err := os.Readlink(p)
				if err != nil {
					return err
				}
				sum := sha256.Sum256([]byte(target))
				s[k] = "link:" + hex.EncodeToString(sum[:])
			case t.IsDir():
				s[k] = "dir"
			default:
				s[k] = "other:" + t.String()
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("what the release gates read cannot be read: %w", err)
		}
	}
	return s, nil
}

func fileDigest(p string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// unchanged refuses a checkout whose gate entries differ from the snapshot's,
// naming the first of them.
func (s gateSnapshot) unchanged(repoRoot string) error {
	now, err := snapshotGates(repoRoot)
	if err != nil {
		return err
	}
	var changed []string
	for k, v := range s {
		if now[k] != v {
			changed = append(changed, k)
		}
	}
	for k := range now {
		if _, ok := s[k]; !ok {
			changed = append(changed, k)
		}
	}
	if len(changed) == 0 {
		return nil
	}
	sort.Strings(changed)
	if len(changed) > 5 {
		changed = append(changed[:5], "...")
	}
	return fmt.Errorf("a check run changed what the release gates and the ledger read, which it never does: %s",
		termsafe.Sanitize(strings.Join(changed, ", ")))
}

// encodeCheck is a check record as written: indented, strings as they are.
func encodeCheck(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// collectCheckCopy reads copy in.Key's answer into its run's record in lab
// labID, and only then deletes the copy's clone. A copy whose findings cannot
// be recorded (the screen refuses them, or a record of the copy is already
// written) keeps its clone, and so does a run that finds the release gates'
// receipts, the review flags or the ledger changed while it read the copy.
func collectCheckCopy(labID string, in checkCopyInputs) (CheckCopy, error) {
	if !runIDRe.MatchString(in.Run) {
		return CheckCopy{}, fmt.Errorf("%q is not a run id", clip(in.Run))
	}
	if err := in.Key.validate(); err != nil {
		return CheckCopy{}, err
	}
	if !filepath.IsAbs(in.RepoRoot) || !filepath.IsAbs(in.Store) {
		return CheckCopy{}, errors.New("the run's checkout and its clone store are named by absolute paths")
	}
	if in.Form != CheckDocFidelity && in.Form != CheckConsistency {
		return CheckCopy{}, fmt.Errorf("%q is not a check form abcd reads", clip(string(in.Form)))
	}
	before, err := snapshotGates(in.RepoRoot)
	if err != nil {
		return CheckCopy{}, err
	}
	proc, err := lab.OpenProcedure(in.RepoRoot, labID)
	if err != nil {
		return CheckCopy{}, err
	}
	defer proc.Close()
	if _, err := proc.ReadState(in.Run + "/run.json"); err != nil {
		return CheckCopy{}, fmt.Errorf("the copy is not collected: its run's record is not written: %w", err)
	}
	c := readCheckCopy(in)
	data, err := encodeCheck(c)
	if err != nil {
		return CheckCopy{}, err
	}
	rel := checkCopyRel(in.Run, in.Key)
	if err := screen(in.RepoRoot, []screened{{lab.StateFile{Rel: rel, Data: data}, data}}); err != nil {
		return CheckCopy{}, fmt.Errorf("copy %s keeps its clone: %w", in.Key, err)
	}
	if err := proc.WriteState(rel, data); err != nil {
		return CheckCopy{}, fmt.Errorf("copy %s keeps its clone: its findings are not recorded: %w", in.Key, err)
	}
	if back, err := proc.ReadState(rel); err != nil || !bytes.Equal(back, data) {
		return CheckCopy{}, fmt.Errorf("copy %s keeps its clone: its findings do not read back as written", in.Key)
	}
	if err := before.unchanged(in.RepoRoot); err != nil {
		return CheckCopy{}, fmt.Errorf("copy %s keeps its clone: %w", in.Key, err)
	}
	if err := os.RemoveAll(cloneDir(in.Store, in.Run, in.Key)); err != nil {
		return CheckCopy{}, fmt.Errorf("copy %s's findings are recorded, and its clone could not be deleted: %w", in.Key, err)
	}
	return c, nil
}
