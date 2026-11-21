package matrix

// checkcopy.go reads a check copy's findings into the run (spec step 7,
// decisions 22, 29 and 30). A check copy is a model's run of a check, such as
// doc fidelity, over the frozen material in its own clone; what it found is the
// model's output and is read as untrusted input: bounded, decoded strictly,
// every text made one inert line with every mention of where the copy ran
// withheld. A check copy is never shown its task's or its cell's id: its clone
// is made under an alias minted at random when it starts (makeCopyClone), and
// every name in the run that could identify a copy (its run, task, cell,
// route, runner and pins, and its alias) and every name of a model or of its
// maker is withheld from the copy's own words wherever it stands, inside a
// longer word too, so a copy cannot sign what it found in front of the person,
// the matcher or the outside judge (identityScrub); a quote the frozen
// material holds is the material's own and is kept whole. Every position is
// computed from the frozen material, never taken from the copy. A doc-fidelity copy's verdict is validated by the recorder's
// own decoder (docfidelity.ReportFindings) and never recorded where the release
// gates read; what it drafted as a correction is dropped, never applied.
//
// The findings reach the run's record before the copy's clone is deleted, and
// a copy whose findings cannot be recorded keeps its clone; a collection that
// failed after the record was written can be tried again. Around the copy's
// whole run, from a snapshot taken before the copy starts to the check after
// its findings are recorded, the run proves it left the release gates'
// receipts, the review flags and the issue ledger as they were
// (snapshotGates): a check copy's findings are compared, never filed.

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
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/docfidelity"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/lab"
	"github.com/intentdriven/abcd/internal/core/record/findingkey"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/termsafe"
	"golang.org/x/text/unicode/norm"
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
	// Redacted counts the mentions withheld from its findings: of where the
	// copy ran, of what in the run identifies it, and of models and makers.
	Redacted int `json:"redacted"`
}

// checkCopyInputs is one check copy to read. RepoRoot is the run's own
// checkout and Store the run's clone store, each absolute; Material is the
// frozen material by path (readFrozenMaterial); Answer is the copy's runner's
// final message, and Failure the runner's failure, "" when it answered. Gates
// is what the release gates read, snapshotted before the copy started
// (snapshotGates), and Identity what in the run identifies a copy, which
// collectCheckCopy reads from the run's record, its declaration and the
// copy's alias record, replacing a caller's.
type checkCopyInputs struct {
	RepoRoot, Store, Run, Frozen string
	Key                          TrialKey
	Form                         CheckForm
	Material                     map[string]string
	Answer                       []byte
	Failure                      string
	Gates                        gateSnapshot
	Identity                     runIdentity
}

// runIdentity is what in a run identifies one of its copies to whoever judges
// what the copy found: the run's id, every task's id, every cell the run pins,
// and the copy's alias (newIdentityScrub reads its names from them).
type runIdentity struct {
	Tasks []string
	Cells []CellPin
	Alias string
}

// checkCopyRel is copy key's findings record in its run's directory.
func checkCopyRel(run string, key TrialKey) string {
	return run + "/check/" + key.Task + "/copies/" + key.Cell + "/" + strconv.Itoa(key.Repeat) + ".json"
}

// readCheckCopy reads one copy's answer. It never refuses: an answer that does
// not validate is the copy's failure, said in the record's reason.
func readCheckCopy(in checkCopyInputs) CheckCopy {
	c := CheckCopy{SchemaVersion: CheckCopySchema, Run: in.Run, Key: in.Key, Form: in.Form, Findings: []CheckFinding{}}
	red := checkRedactor{
		place: newRedactor(reviewInputs{RepoRoot: in.RepoRoot, Store: in.Store, Dir: cloneDir(in.Store, in.Run, in.Identity.Alias), Run: in.Run}),
		model: newIdentityScrub(in.Run, in.Key, in.Identity),
	}
	reason := func(s string) string {
		s, _ = red.text(termsafe.SanitizeLine(s))
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

func parseCheckAnswer(form CheckForm, data []byte, material map[string]string, red checkRedactor) ([]CheckFinding, int, error) {
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
func docFidelityFindings(data []byte, material map[string]string, red checkRedactor) ([]CheckFinding, int, error) {
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
		if err := red.path(p, material); err != nil {
			return nil, 0, fmt.Errorf("finding %d: %w", i+1, err)
		}
		end, n := red.end(p, s.Sentence, material)
		evidence, m := red.inert(s.Evidence)
		redacted += n + m
		if len(evidence) > maxFindingBytes {
			return nil, 0, fmt.Errorf("finding %d's evidence is past its bound of %d bytes", i+1, maxFindingBytes)
		}
		out = append(out, CheckFinding{
			Key:  findingkey.Derive("doc-fidelity/"+s.Doc, findingkey.End{Path: p, Quote: end.Quote}, findingkey.End{}),
			Doc:  s.Doc,
			Ends: []FindingEnd{end}, Evidence: evidence,
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

func consistencyFindings(data []byte, material map[string]string, red checkRedactor) ([]CheckFinding, int, error) {
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
		summary, n := red.inert(f.Summary)
		redacted += n
		if summary == "" || len(summary) > maxFindingBytes {
			return nil, 0, fmt.Errorf("finding %d's summary is empty or past its bound of %d bytes", i+1, maxFindingBytes)
		}
		class := f.Class
		if red.model.names(class) {
			class = withheldClass
			redacted++
		}
		cf := CheckFinding{Class: class, Evidence: summary}
		var ends [2]findingkey.End
		for j, e := range f.Ends {
			if err := red.path(e.Path, material); err != nil {
				return nil, 0, fmt.Errorf("finding %d end %d: %w", i+1, j+1, err)
			}
			end, n := red.end(e.Path, e.Quote, material)
			redacted += n
			if end.Quote == "" || len(end.Quote) > maxQuoteBytes {
				return nil, 0, fmt.Errorf("finding %d end %d quotes nothing, or more than %d bytes", i+1, j+1, maxQuoteBytes)
			}
			ends[j] = findingkey.End{Path: e.Path, Quote: end.Quote}
			cf.Ends = append(cf.Ends, end)
		}
		cf.Key = findingkey.Derive(class, ends[0], ends[1])
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

// checkRedactor is what a check copy's text is read through: the review's
// redactor for where the copy ran, then the identity scrub for which copy it
// is and who its model is.
type checkRedactor struct {
	place redactor
	model identityScrub
}

// text withholds both from s, and says how many mentions it withheld.
func (r checkRedactor) text(s string) (string, int) {
	s, n := r.place.redact(s)
	s, m := r.model.scrub(s)
	return s, n + m
}

// inert is s made one inert line with both withheld.
func (r checkRedactor) inert(s string) (string, int) {
	s, n := inert(s, r.place)
	s, m := r.model.scrub(s)
	return s, n + m
}

// path refuses a finding's path unless it is a clean path inside the
// repository, and, where the frozen material holds no file at it, unless it
// names nothing of the run's identity and no model or maker: a path the
// material holds is the repository's own, while one it does not is the copy's
// own words, and a path is the finding's place, never withheld in part.
func (r checkRedactor) path(p string, material map[string]string) error {
	if err := checkPath(p); err != nil {
		return err
	}
	if _, ok := material[p]; !ok && r.model.names(p) {
		return errors.New("a path the frozen material does not hold names the run, a task, a cell, its route, or a model or its maker")
	}
	return nil
}

// end is one end of a finding: the quote made inert, and where the frozen
// material at p holds it. A quote the material holds is the material's own
// words, the same for every copy that quotes it, so it signs no copy and is
// kept whole (dcn-2611210903322839). One it does not hold is the copy's own,
// and has everything the identity scrub withholds withheld.
func (r checkRedactor) end(p, quote string, material map[string]string) (FindingEnd, int) {
	q, n := inert(quote, r.place)
	lines := locate(material[p], q)
	var m int
	if lines == nil {
		q, m = r.model.scrub(q)
	}
	return FindingEnd{Path: p, Quote: q, Lines: lines}, n + m
}

// withheldMark stands in for what is withheld from a check copy's text by the
// identity scrub: a name of the run, a task, a cell, its route, runner or pin,
// the copy's alias, or a model or its maker.
const withheldMark = "[withheld]"

// withheldClass is the class a consistency finding takes when its own names
// anything the identity scrub withholds (dcn-2611210834192091).
const withheldClass = "withheld"

// identityScrub withholds from a check copy's own words everything that would
// tell a judge which copy wrote them (dcn-2611210903322839), so a copy cannot
// sign its findings. Each name is matched in any case and as a substring,
// wherever it stands, fused to a letter or a digit or joined by any mark, so
// no way of writing an ordinary id escapes a word boundary, and in the text
// as scrubFold folds it, so no respelling does either: compatibility forms
// read as their plain letters, combining marks and format characters (a
// zero-width space or joiner, a soft hyphen) dropped, every dash read as a
// hyphen, and Cyrillic and Greek letters that look Latin read as the Latin
// ones. A withheld name takes the rest of its model id token with it
// (versionEnd), and a joiner between two withheld spans is withheld with
// them, so "claude-opus-5-5" goes under one mark:
//
//   - every name in the run that could identify a copy: the run's id; every
//     task's id; and each cell's id, source, route whole and by its parts (its
//     runner, provider and model), runner, model whole and by each segment,
//     harness version, and the names its provider or local pin records
//     (identityNames); and the copy's alias;
//   - every name of the comparison's model families (modelFamilies) and of
//     their developers (developers), and every AI name the attribution gate
//     refuses (identity.AINameSpans), the lists being those packages' own;
//   - an o-series name: an `o` no letter stands before, written straight
//     before a digit.
//
// A name shorter than minNameBytes would withhold ordinary text wherever its
// letters fall, so it is not matched: a task's or a cell's id that short is
// kept from the copy instead, which is never shown its task's or its cell's id
// (cloneDir), only its alias.
type identityScrub struct {
	// withheld is every name withheld, lower-cased.
	withheld []string
}

// minNameBytes is the shortest name the scrub matches.
const minNameBytes = 3

func newIdentityScrub(run string, key TrialKey, id runIdentity) identityScrub {
	var s identityScrub
	seen := map[string]bool{}
	add := func(n string) {
		if n, _, _ = scrubFold(strings.TrimSpace(n)); len(n) >= minNameBytes && !seen[n] {
			seen[n] = true
			s.withheld = append(s.withheld, n)
		}
	}
	for _, n := range identityNames(run, key, id) {
		add(n)
	}
	for _, n := range modelFamilies {
		add(n)
	}
	for _, d := range developers {
		add(d)
	}
	sort.Strings(s.withheld)
	return s
}

// identityNames is every name in the run that could identify a copy: the
// run's and the copy's own key's ids always, so a copy read on its own is
// scrubbed of them too, and its alias. A cell's role is not one (every copy of
// a check shares it), nor are its sampling settings or its pin's
// data-collection policy, which are values, not names.
//
// Every task's id joined to every cell's by a slash is one too: how a key is
// written (<task>/<cell>/<repeat>) and how every record of a copy is filed
// (trials/, aliases/ and the check's copies/ under <task>/<cell>/), so an
// error naming a copy's key or a record's path, which a runner may echo, is
// withheld whole however short the cell's own id.
func identityNames(run string, key TrialKey, id runIdentity) []string {
	out := append([]string{run, key.Task, key.Cell, id.Alias}, id.Tasks...)
	tasks := append([]string{key.Task}, id.Tasks...)
	cells := []string{key.Cell}
	for _, c := range id.Cells {
		cells = append(cells, c.Cell)
	}
	for _, t := range tasks {
		for _, c := range cells {
			out = append(out, t+"/"+c)
		}
	}
	for _, c := range id.Cells {
		out = append(out, c.Cell, string(c.Source), c.Runner, c.HarnessVersion)
		out = append(out, routeNames(c.Route)...)
		out = append(out, modelNames(c.Model)...)
		if p := c.ProviderPin; p != nil {
			out = append(out, p.CanonicalSlug)
			out = append(out, p.Order...)
			out = append(out, p.Quantizations...)
			out = append(out, modelNames(p.CanonicalSlug)...)
		}
		if p := c.LocalPin; p != nil {
			out = append(out, p.Runtime, p.RuntimeVersion, p.WeightsSHA256, p.WeightsFile, p.Quantisation, p.KVCacheType)
		}
	}
	return out
}

// modelNames is a model's id whole and each of its segments after a slash.
func modelNames(m string) []string {
	return append([]string{m}, strings.Split(m, "/")...)
}

// routeNames is a cell's route whole and by its parts:
// `<role>=runner:<name>[@<provider>/<model>]` is its runner's name, and its
// provider and model whole and by segment; the grammar's own words (a role,
// `runner`) are no cell's.
func routeNames(route string) []string {
	out := []string{route}
	rest := route
	if _, r, ok := strings.Cut(rest, "="); ok {
		rest = r
	}
	rest = strings.TrimPrefix(rest, "runner:")
	name, model, _ := strings.Cut(rest, "@")
	out = append(out, name)
	if model != "" {
		out = append(out, modelNames(model)...)
	}
	return out
}

// asciiLower is s with its ASCII letters lower-cased and every other byte as
// it is, so offsets into it are offsets into s.
func asciiLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 'a' - 'A'
		}
	}
	return string(b)
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// scrub is s with every place it names anything the scrub withholds
// withheld, each run of withheld bytes under one mark, and how many marks it
// wrote.
func (m identityScrub) scrub(s string) (string, int) {
	held := m.held(s)
	var b strings.Builder
	n := 0
	for i := 0; i < len(s); i++ {
		switch {
		case !held[i]:
			b.WriteByte(s[i])
		case i == 0 || !held[i-1]:
			b.WriteString(withheldMark)
			n++
		}
	}
	return b.String(), n
}

// held marks each byte of s that is part of a name the scrub withholds. The
// names are matched in s as scrubFold folds it, and each match holds the
// bytes of s it was folded from, the format characters inside it included.
func (m identityScrub) held(s string) []bool {
	low, from, to := scrubFold(s)
	held := make([]bool, len(s))
	hold := func(i, j int) {
		for ; i < j; i++ {
			held[i] = true
		}
	}
	// holdFolded holds low[i:j] as the bytes of s it came from.
	holdFolded := func(i, j int) {
		if i < j {
			hold(from[i], to[j-1])
		}
	}
	for _, name := range m.withheld {
		for at := 0; at < len(low); {
			i := strings.Index(low[at:], name)
			if i < 0 {
				break
			}
			holdFolded(at+i, versionEnd(low, at+i+len(name), false))
			at += i + 1
		}
	}
	for _, sp := range identity.AINameSpans(s) {
		hold(sp[0], sp[1])
	}
	for _, sp := range identity.AINameSpans(low) {
		holdFolded(sp[0], versionEnd(low, sp[1], false))
	}
	for i := 0; i+1 < len(low); i++ {
		if low[i] == oSeries[0] && (i == 0 || !isLetter(low[i-1])) && isDigit(low[i+1]) {
			j := i + 1
			for j < len(low) && isDigit(low[j]) {
				j++
			}
			holdFolded(i, versionEnd(low, j, true))
		}
	}
	// A joiner between two withheld spans joins one name: "claude-opus".
	for i := 1; i+1 < len(s); i++ {
		if !held[i] && held[i-1] && held[i+1] && strings.IndexByte(tokenJoiners, s[i]) >= 0 {
			held[i] = true
		}
	}
	return held
}

// tokenJoiners join the parts of a model id token.
const tokenJoiners = "-._:/"

// versionEnd is where the model id token a withheld name ends at j in folded
// text low goes on to: a version after it, joined by a joiner or a space or
// written straight on ("-5-5", " 5.5", "4o"), each part led by a digit, and
// once a version has begun (versioned), every part joined to it by a joiner
// ("-mini", "-coder-32b"). A name fused to a letter, or followed by a word,
// takes nothing more.
func versionEnd(low string, j int, versioned bool) int {
	end := j
	for i := j; ; {
		k := i
		sep := byte(0)
		if k < len(low) && (low[k] == ' ' || strings.IndexByte(tokenJoiners, low[k]) >= 0) {
			sep = low[k]
			k++
		}
		if k >= len(low) || !isAlnum(low[k]) {
			return end
		}
		digit := isDigit(low[k])
		if !digit && (sep == 0 || sep == ' ' || !versioned) {
			return end
		}
		for k < len(low) && isAlnum(low[k]) {
			k++
		}
		i, end, versioned = k, k, versioned || digit
	}
}

func isAlnum(c byte) bool { return isLetter(c) || isDigit(c) }

// scrubFold is s as the identity scrub matches it, and, for each byte of the
// result, the span of s it was folded from (from[i] to to[i]). It reads s
// through NFKD, so a compatibility form becomes its plain letters (as NFKC
// would) and a combining mark comes apart from its letter; drops every
// combining mark and every format character (a zero-width space, joiner or
// non-joiner, a word joiner, a soft hyphen, a direction mark); reads every
// dash and the minus sign as a hyphen; and folds the Cyrillic and Greek
// letters that look Latin (scrubConfusables) to the Latin ones, and every
// other letter to lower case.
func scrubFold(s string) (string, []int, []int) {
	var b strings.Builder
	from := make([]int, 0, len(s))
	to := make([]int, 0, len(s))
	var it norm.Iter
	it.InitString(norm.NFKD, s)
	for !it.Done() {
		pos := it.Pos()
		seg := string(it.Next())
		end := it.Pos()
		exact := seg == s[pos:end]
		for k := 0; k < len(seg); {
			r, w := utf8.DecodeRuneInString(seg[k:])
			lo, hi := pos, end
			if exact {
				lo, hi = pos+k, pos+k+w
			}
			k += w
			switch {
			case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Cf, r):
				continue
			case unicode.Is(unicode.Pd, r) || r == '\u2212':
				r = '-'
			default:
				if c, ok := scrubConfusables[r]; ok {
					r = c
				} else {
					r = unicode.ToLower(r)
				}
			}
			n := b.Len()
			b.WriteRune(r)
			for ; n < b.Len(); n++ {
				from = append(from, lo)
				to = append(to, hi)
			}
		}
	}
	return b.String(), from, to
}

// scrubConfusables are the Cyrillic and Greek letters, in both cases, that
// look like a Latin letter, each to the lower-case Latin letter it looks
// like: the common ones of Unicode's confusables (TR39) for these scripts.
var scrubConfusables = map[rune]rune{
	// Cyrillic capitals and smalls.
	'А': 'a', 'а': 'a', 'В': 'b', 'в': 'b', 'С': 'c', 'с': 'c', 'Ԁ': 'd', 'ԁ': 'd',
	'Е': 'e', 'е': 'e', 'Һ': 'h', 'һ': 'h', 'Н': 'h', 'н': 'h', 'І': 'i', 'і': 'i',
	'Ј': 'j', 'ј': 'j', 'К': 'k', 'к': 'k', 'Ӏ': 'l', 'ӏ': 'l', 'М': 'm', 'м': 'm',
	'О': 'o', 'о': 'o', 'Р': 'p', 'р': 'p', 'Ԛ': 'q', 'ԛ': 'q', 'Ѕ': 's', 'ѕ': 's',
	'Т': 't', 'т': 't', 'Ү': 'y', 'ү': 'y', 'У': 'y', 'у': 'y', 'Ԝ': 'w', 'ԝ': 'w',
	'Х': 'x', 'х': 'x', 'Ь': 'b', 'ь': 'b',
	// Greek capitals and smalls.
	'Α': 'a', 'α': 'a', 'Β': 'b', 'β': 'b', 'Ε': 'e', 'ε': 'e', 'Ζ': 'z', 'Η': 'h',
	'η': 'n', 'Ι': 'i', 'ι': 'i', 'Κ': 'k', 'κ': 'k', 'Μ': 'm', 'Ν': 'n', 'ν': 'v',
	'Ο': 'o', 'ο': 'o', 'Ρ': 'p', 'ρ': 'p', 'Τ': 't', 'τ': 't', 'Υ': 'y', 'υ': 'u',
	'Χ': 'x', 'χ': 'x', 'ϲ': 'c', 'Ϲ': 'c', 'ϳ': 'j',
	// Latin letters that read as another.
	'ı': 'i', 'ɡ': 'g', 'ɑ': 'a',
}

// names reports whether s names anything the scrub withholds.
func (m identityScrub) names(s string) bool {
	_, n := m.scrub(s)
	return n > 0
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
// labID, and only then deletes the copy's clone. A copy whose findings the
// secret screen refuses is recorded as failed, with nothing of its answer
// kept. A copy whose findings cannot be recorded (another record of the copy
// is already written) keeps its clone, and so does a run that finds the
// release gates' receipts, the review flags or the ledger changed since
// in.Gates was snapshotted, before the copy started. A record already written
// with the same bytes counts as written, so a collection that failed after
// writing it can be tried again, and the retry deletes the clone.
func collectCheckCopy(labID string, in checkCopyInputs) (CheckCopy, error) {
	c, clone, err := recordCheckCopy(labID, in)
	if err != nil {
		return CheckCopy{}, err
	}
	if err := os.RemoveAll(clone); err != nil {
		return CheckCopy{}, fmt.Errorf("copy %s's findings are recorded, and its clone could not be deleted: %w", in.Key, err)
	}
	return c, nil
}

// recordCheckCopy is collectCheckCopy short of deleting the clone: it records
// copy in.Key's findings, proves the release gates' reads unchanged, and
// returns the clone to delete once the caller has written what else it must
// (a hand-back's trial), so nothing it refuses can leave a copy with no clone.
func recordCheckCopy(labID string, in checkCopyInputs) (CheckCopy, string, error) {
	if !runIDRe.MatchString(in.Run) {
		return CheckCopy{}, "", fmt.Errorf("%q is not a run id", clip(in.Run))
	}
	if err := in.Key.validate(); err != nil {
		return CheckCopy{}, "", err
	}
	if !filepath.IsAbs(in.RepoRoot) || !filepath.IsAbs(in.Store) {
		return CheckCopy{}, "", errors.New("the run's checkout and its clone store are named by absolute paths")
	}
	if in.Form != CheckDocFidelity && in.Form != CheckConsistency {
		return CheckCopy{}, "", fmt.Errorf("%q is not a check form abcd reads", clip(string(in.Form)))
	}
	if in.Gates == nil {
		return CheckCopy{}, "", errors.New("the copy is not collected: what the release gates read was not snapshotted before it ran")
	}
	proc, err := lab.OpenProcedure(in.RepoRoot, labID)
	if err != nil {
		return CheckCopy{}, "", err
	}
	defer proc.Close()
	raw, err := proc.ReadState(in.Run + "/run.json")
	if err != nil {
		return CheckCopy{}, "", fmt.Errorf("the copy is not collected: its run's record is not written: %w", err)
	}
	var rec RunRecord
	if err := jsonstrict.Decode(raw, &rec); err != nil || rec.RunID != in.Run {
		return CheckCopy{}, "", errors.New("the copy is not collected: its run's record does not read as the run's")
	}
	decl, err := proc.ReadState(in.Run + "/declaration.json")
	if err != nil || contentOf("declaration.json", decl) != rec.Declaration {
		return CheckCopy{}, "", errors.New("the copy is not collected: its run's declaration does not read as the one its record keeps")
	}
	d, err := parseDeclaration(decl)
	if err != nil {
		return CheckCopy{}, "", fmt.Errorf("the copy is not collected: %w", err)
	}
	alias, err := readAlias(proc, in.Run, in.Key)
	if err != nil {
		return CheckCopy{}, "", fmt.Errorf("the copy is not collected: its clone's alias is not recorded: %w", err)
	}
	in.Identity = runIdentity{Cells: rec.Cells, Alias: alias}
	for _, t := range d.Tasks {
		in.Identity.Tasks = append(in.Identity.Tasks, t.ID)
	}
	rel := checkCopyRel(in.Run, in.Key)
	c, data, err := screenedCheckCopy(in, rel)
	if err != nil {
		return CheckCopy{}, "", err
	}
	werr := proc.WriteState(rel, data)
	if back, err := proc.ReadState(rel); err != nil || !bytes.Equal(back, data) {
		if werr != nil {
			return CheckCopy{}, "", fmt.Errorf("copy %s keeps its clone: its findings are not recorded: %w", in.Key, werr)
		}
		return CheckCopy{}, "", fmt.Errorf("copy %s keeps its clone: its findings do not read back as written", in.Key)
	}
	if err := in.Gates.unchanged(in.RepoRoot); err != nil {
		return CheckCopy{}, "", fmt.Errorf("copy %s keeps its clone: %w", in.Key, err)
	}
	return c, cloneDir(in.Store, in.Run, alias), nil
}

// screenRefusedReason is a copy's failure when the secret screen refuses its
// findings: fixed, so nothing of what it refused is kept.
const screenRefusedReason = "the copy's findings carry what the secret screen refuses, so the copy failed and nothing of its answer is kept"

// screenedCheckCopy reads copy in.Key and encodes its record at rel, passed
// through the secret screen. A record the screen refuses becomes the copy's
// failure; one the screen refuses even so (the screen cannot run) is refused.
func screenedCheckCopy(in checkCopyInputs, rel string) (CheckCopy, []byte, error) {
	c := readCheckCopy(in)
	for {
		data, err := encodeCheck(c)
		if err != nil {
			return CheckCopy{}, nil, err
		}
		serr := screen(in.RepoRoot, []screened{{lab.StateFile{Rel: rel, Data: data}, data}})
		if serr == nil {
			return c, data, nil
		}
		if !c.Answered && c.Reason == screenRefusedReason {
			return CheckCopy{}, nil, fmt.Errorf("copy %s keeps its clone: %w", in.Key, serr)
		}
		c = CheckCopy{SchemaVersion: CheckCopySchema, Run: in.Run, Key: in.Key, Form: in.Form, Findings: []CheckFinding{}, Reason: screenRefusedReason}
	}
}
