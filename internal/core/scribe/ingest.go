package scribe

// ingest.go is the scribe's OUTPUT contract, on the idiom `reading ingest`
// carries: the scribe emits JSON, this verb validates it, and the capture verbs
// write the records.
//
// This is a trust boundary. The payload is an agent's output, read behind the
// guarded reader with a byte cap, decoded strictly with every key checked
// against the closed shapes, and no payload string is joined into a path before
// its grammar is checked: the run id is matched against recordid first. Every
// payload string quoted into a message is neutralised and capped.
//
// The one validation this verb adds is the one only it can make: that the scribe
// AUTHORED NOTHING. The scribe reformats; it never adds a word. So every item it
// answers must be named in the researcher's supplied text, and every ground,
// exit condition and surprise it carries must stand there verbatim once
// whitespace is folded. Everything else — the state vocabulary, the substance
// floor, the ordering gate, the one-ground rule, redaction — is the capture
// verbs' own, inherited whole by calling them.

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// IngestRequest is one scribe ingest.
type IngestRequest struct {
	RepoRoot string
	// ScribeJSONPath is the payload the scribe returned.
	ScribeJSONPath string
	// ContextPath is the context the session was handed; empty means the local-
	// tier default for the payload's run. The manifest is read from beside it.
	ContextPath string
	// DispositionsPath is the researcher's dispositions text, the file assemble
	// was handed. It is required: the parked pair sits where a scribe session
	// with tools can rewrite it, so the verbatim checks read the researcher's
	// own file, and the parked copy and the manifest's hash are held to it.
	DispositionsPath string
}

// OutDisposition is one disposition the scribe transcribed.
type OutDisposition struct {
	Item          string   `json:"item"`
	State         string   `json:"state"`
	Grounds       string   `json:"grounds"`
	ExitCondition string   `json:"exit_condition"`
	Supersedes    string   `json:"supersedes"`
	Recurs        []string `json:"recurs"`
}

// OutAdmission is one admission the scribe transcribed.
type OutAdmission struct {
	Item    string `json:"item"`
	Grounds string `json:"grounds"`
}

// OutSurprise is one surprise the scribe transcribed.
type OutSurprise struct {
	OccasionedBy string `json:"occasioned_by"`
	Text         string `json:"text"`
}

// FidelityFlag names two pieces of material that disagree, and stops there.
type FidelityFlag struct {
	First  string `json:"first"`
	Second string `json:"second"`
}

// Refusal is something the scribe refused, with its reason.
type Refusal struct {
	Subject string `json:"subject"`
	Reason  string `json:"reason"`
}

// Output is `abcd.scribe.output/1`, the scribe's whole return.
type Output struct {
	Type          string           `json:"_type"`
	Run           string           `json:"run"`
	ContextSHA256 string           `json:"context_sha256"`
	Dispositions  []OutDisposition `json:"dispositions"`
	Admissions    []OutAdmission   `json:"admissions"`
	Surprises     []OutSurprise    `json:"surprises"`
	FidelityFlags []FidelityFlag   `json:"fidelity_flags"`
	Outstanding   []string         `json:"outstanding"`
	Refusals      []Refusal        `json:"refusals"`
}

// The closed key sets, per level. A key outside them is a field the scribe may
// not author, refused by name.
var (
	topKeys         = keySet("_type", "run", "context_sha256", "dispositions", "admissions", "surprises", "fidelity_flags", "outstanding", "refusals")
	dispositionKeys = keySet("item", "state", "grounds", "exit_condition", "supersedes", "recurs")
	admissionKeys   = keySet("item", "grounds")
	surpriseKeys    = keySet("occasioned_by", "text")
	flagKeys        = keySet("first", "second")
	refusalKeys     = keySet("subject", "reason")
	requiredTopKeys = []string{"_type", "run", "context_sha256"}
)

func keySet(keys ...string) map[string]bool {
	out := make(map[string]bool, len(keys))
	for _, k := range keys {
		out[k] = true
	}
	return out
}

// IngestResult is what one ingest landed and what it carries back unresolved.
type IngestResult struct {
	Run           string                      `json:"run"`
	ContextSHA256 string                      `json:"context_sha256"`
	Dispositions  []capture.DispositionResult `json:"dispositions"`
	Admissions    []capture.AdmitResult       `json:"admissions"`
	Surprises     []capture.SurpriseResult    `json:"surprises"`
	Outstanding   []string                    `json:"outstanding"`
	// FidelityFlags and Refusals are carried to the researcher UNRESOLVED and
	// never into a record, which is what the definition promises.
	FidelityFlags []FidelityFlag `json:"fidelity_flags"`
	Refusals      []Refusal      `json:"refusals"`
	// Manifest is the promoted manifest's repository-relative path, set only
	// once every write has landed, and only when at least one record did.
	Manifest string `json:"manifest,omitempty"`
}

// Landed names every record this ingest wrote, in order.
func (r IngestResult) Landed() []string {
	var out []string
	for _, d := range r.Dispositions {
		out = append(out, d.ID)
	}
	for _, a := range r.Admissions {
		if a.DispositionWritten {
			out = append(out, a.Disposition)
		}
		out = append(out, a.Admission)
	}
	for _, s := range r.Surprises {
		out = append(out, s.ID)
	}
	return out
}

// Ingest validates one scribe payload and writes what it transcribed.
func Ingest(req IngestRequest) (IngestResult, error) {
	if strings.TrimSpace(req.RepoRoot) == "" {
		return IngestResult{}, errors.New("scribe: no repository root given")
	}
	if strings.TrimSpace(req.ScribeJSONPath) == "" {
		return IngestResult{}, errors.New("scribe: no scribe output named")
	}
	if strings.TrimSpace(req.DispositionsPath) == "" {
		return IngestResult{}, errors.New("scribe: no dispositions supplied; the ingest holds every word the " +
			"scribe carries to the researcher's own dispositions text, the file assemble was handed, and the " +
			"copy parked beside the context is no witness to it")
	}
	raw, err := fsutil.ReadGuarded(req.ScribeJSONPath, reading.MaxFileBytes)
	if err != nil {
		return IngestResult{}, fmt.Errorf("scribe: reading the scribe output: %w", err)
	}
	out, err := decodeOutput(raw)
	if err != nil {
		return IngestResult{}, err
	}
	if out.Type != OutputType {
		return IngestResult{}, fmt.Errorf("scribe: the output's _type is %q, want %q", echo(out.Type), OutputType)
	}
	if !recordid.ValidReadingRunID(out.Run) {
		return IngestResult{}, fmt.Errorf("scribe: the output names run %q, which is not a reading run id (rdg-N)", echo(out.Run))
	}

	// The run's identity, proven before anything is written: the context on
	// disk must hash to the parked manifest's context hash, and the payload must
	// cite that same hash.
	ctx, m, err := proveContext(req, out)
	if err != nil {
		return IngestResult{}, err
	}
	supplied, err := proveSupplied(req, ctx, m)
	if err != nil {
		return IngestResult{}, err
	}
	if err := requireCommittedRun(req.RepoRoot, out.Run); err != nil {
		return IngestResult{}, err
	}
	if err := refusePromoted(req.RepoRoot, out.Run); err != nil {
		return IngestResult{}, err
	}
	items, err := runItems(req.RepoRoot, out.Run)
	if err != nil {
		return IngestResult{}, err
	}
	if err := refuseAuthored(out, supplied, items); err != nil {
		return IngestResult{}, err
	}

	res := IngestResult{
		Run: out.Run, ContextSHA256: m.ContextSHA256,
		Dispositions: []capture.DispositionResult{}, Admissions: []capture.AdmitResult{},
		Surprises: []capture.SurpriseResult{}, Outstanding: nonNil(out.Outstanding),
		FidelityFlags: nonNilFlags(out.FidelityFlags), Refusals: nonNilRefusals(out.Refusals),
	}

	// The writes, through the verbs' own functions, in payload order. Each takes
	// the ledger lock for itself and applies its own redaction and refusals; the
	// first refusal stops the ingest, and the error names what landed before it.
	//
	// A ground and an exit condition are frontmatter scalars, which hold one
	// line, so they are handed over whitespace-folded: the same folding the
	// verbatim check reads them under, and a change of layout, never of words.
	for i, d := range out.Dispositions {
		r, err := capture.Disposition(capture.DispositionRequest{
			RepoRoot: req.RepoRoot, Item: d.Item, State: d.State, Grounds: fold(d.Grounds),
			ExitCondition: fold(d.ExitCondition), Supersedes: d.Supersedes, Recurs: d.Recurs,
		})
		if err != nil {
			return res, stopped(res, fmt.Sprintf("dispositions[%d] (%s)", i, d.Item), err)
		}
		res.Dispositions = append(res.Dispositions, r)
	}
	for i, a := range out.Admissions {
		r, err := capture.Admit(capture.AdmitRequest{RepoRoot: req.RepoRoot, Item: a.Item, Grounds: fold(a.Grounds)})
		if err != nil {
			return res, stopped(res, fmt.Sprintf("admissions[%d] (%s)", i, a.Item), err)
		}
		res.Admissions = append(res.Admissions, r)
	}
	for i, s := range out.Surprises {
		r, err := capture.Surprise(capture.SurpriseRequest{RepoRoot: req.RepoRoot, OccasionedBy: s.OccasionedBy, Text: s.Text})
		if err != nil {
			return res, stopped(res, fmt.Sprintf("surprises[%d] (%s)", i, s.OccasionedBy), err)
		}
		res.Surprises = append(res.Surprises, r)
	}

	// Promotion comes LAST, so a refused ingest leaves the manifest parked and a
	// rerun re-proves the same context. The run directory is denied to every
	// assembly by the exclusion floor, so the next reading cannot see it.
	//
	// An ingest that landed nothing promotes nothing. The promoted manifest is
	// write-once and locks the run against every later scribe session, so it is
	// the evidence of a session that wrote records; a payload of outstanding
	// items and refusals alone wrote none, and the researcher who answers next
	// week must still be able to use the scribe for it.
	if len(res.Landed()) == 0 {
		return res, nil
	}
	rel, err := reading.WriteRunArtefact(req.RepoRoot, out.Run, ManifestFileName, m)
	if err != nil {
		return res, fmt.Errorf("scribe: every record landed (%s) and promoting the manifest failed: %w",
			landedList(res), err)
	}
	res.Manifest = rel
	return res, nil
}

// stopped states a refusal from a write, and what landed before it.
func stopped(res IngestResult, at string, err error) error {
	return fmt.Errorf("scribe: %s was refused, so the ingest stopped; landed before it: %s; the manifest "+
		"stays parked, and a rerun must drop what landed: %w", at, landedList(res), err)
}

func landedList(res IngestResult) string {
	if l := res.Landed(); len(l) > 0 {
		return strings.Join(l, ", ")
	}
	return "nothing"
}

// decodeOutput checks every key at every level against the closed shapes, then
// decodes strictly. The key walk comes first so a refusal names the field the
// scribe authored and the entry it sat on, which the decoder's own message does
// not.
func decodeOutput(raw []byte) (Output, error) {
	var top map[string]json.RawMessage
	if err := json.Unmarshal(raw, &top); err != nil {
		return Output{}, fmt.Errorf("scribe: the output is not a JSON object: %w", err)
	}
	if err := refuseKeys("the output", top, topKeys); err != nil {
		return Output{}, err
	}
	for _, k := range requiredTopKeys {
		if _, ok := top[k]; !ok {
			return Output{}, fmt.Errorf("scribe: the output carries no %q", k)
		}
	}
	for list, allowed := range map[string]map[string]bool{
		"dispositions": dispositionKeys, "admissions": admissionKeys, "surprises": surpriseKeys,
		"fidelity_flags": flagKeys, "refusals": refusalKeys,
	} {
		rawList, ok := top[list]
		if !ok || string(rawList) == "null" {
			continue
		}
		var entries []map[string]json.RawMessage
		if err := json.Unmarshal(rawList, &entries); err != nil {
			return Output{}, fmt.Errorf("scribe: %q is not a list of objects: %w", list, err)
		}
		for i, e := range entries {
			where := fmt.Sprintf("%s[%d]", list, i)
			var subject string
			for _, k := range []string{"item", "occasioned_by", "subject"} {
				if v, ok := e[k]; ok {
					_ = json.Unmarshal(v, &subject)
					break
				}
			}
			if subject != "" {
				where += " (" + echo(subject) + ")"
			}
			if err := refuseKeys(where, e, allowed); err != nil {
				return Output{}, err
			}
		}
	}
	var out Output
	if err := decodeStrict(raw, &out, "the scribe output"); err != nil {
		return Output{}, err
	}
	return out, nil
}

// refuseKeys refuses the first key, in sorted order, that is not in allowed.
func refuseKeys(where string, obj map[string]json.RawMessage, allowed map[string]bool) error {
	keys := make([]string, 0, len(obj))
	for k := range obj {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		if !allowed[k] {
			return fmt.Errorf("scribe: %s carries %q, a field the scribe may not author; the scribe transcribes "+
				"the declared shapes and authors nothing, so the payload is refused and nothing is written",
				where, echo(k))
		}
	}
	return nil
}

// proveContext reads the parked manifest beside the context and proves the
// context against it, and the payload against both.
func proveContext(req IngestRequest, out Output) (Context, Manifest, error) {
	ctxPath := req.ContextPath
	if ctxPath == "" {
		ctxPath = DefaultRunDir + "/" + out.Run + "/" + ContextFileName
	}
	if !filepath.IsAbs(ctxPath) {
		ctxPath = filepath.Join(req.RepoRoot, filepath.FromSlash(ctxPath))
	}
	manifestPath := filepath.Join(filepath.Dir(ctxPath), ManifestFileName)
	mRaw, err := fsutil.ReadGuarded(manifestPath, reading.MaxFileBytes)
	if err != nil {
		return Context{}, Manifest{}, fmt.Errorf("scribe: reading the parked manifest for %s: %w; assemble the "+
			"session first, and ingest against the context it parked", out.Run, err)
	}
	m, err := DecodeManifest(mRaw)
	if err != nil {
		return Context{}, Manifest{}, err
	}
	cRaw, err := fsutil.ReadGuarded(ctxPath, reading.MaxFileBytes)
	if err != nil {
		return Context{}, Manifest{}, fmt.Errorf("scribe: reading the context: %w", err)
	}
	if got := sha256Hex(cRaw); got != m.ContextSHA256 {
		return Context{}, Manifest{}, fmt.Errorf("scribe: the context on disk hashes to %s and its manifest "+
			"records %s, so it is not the context the session was handed; nothing is written", got, m.ContextSHA256)
	}
	if out.ContextSHA256 != m.ContextSHA256 {
		return Context{}, Manifest{}, fmt.Errorf("scribe: the output cites context %s and the parked context "+
			"is %s, so the output is not from this session; nothing is written", echo(out.ContextSHA256), m.ContextSHA256)
	}
	ctx, err := decodeContext(cRaw)
	if err != nil {
		return Context{}, Manifest{}, err
	}
	if ctx.Run != out.Run || m.Run != out.Run || ctx.ContextStamp != m.ContextStamp {
		return Context{}, Manifest{}, fmt.Errorf("scribe: the output names run %s, the context %s and the "+
			"manifest %s; one session is over one run", echo(out.Run), ctx.Run, m.Run)
	}
	return ctx, m, nil
}

// proveSupplied authenticates the parked pair against the researcher's own
// text. The context and the manifest are parked in the local tier, where a
// scribe session granted tools can rewrite both and recompute every hash that
// binds them, so their agreement proves nothing about what the researcher
// wrote. The dispositions file the operator handed assemble is the one witness
// the session was never given: it is re-read here, scrubbed exactly as assemble
// scrubbed it, and the manifest's supplied hash and the context's supplied copy
// must both equal it. What the verbatim checks then read is that text.
func proveSupplied(req IngestRequest, ctx Context, m Manifest) (string, error) {
	raw, err := fsutil.ReadGuarded(req.DispositionsPath, reading.MaxFileBytes)
	if err != nil {
		return "", fmt.Errorf("scribe: reading the supplied dispositions: %w", err)
	}
	supplied := scrub(req.RepoRoot, string(raw))
	if got := sha256Hex([]byte(supplied)); got != m.Supplied.DispositionsSHA256 {
		return "", fmt.Errorf("scribe: the supplied dispositions hash to %s and the parked manifest records %s, "+
			"so the session was not assembled over this text, or the parked pair was rewritten; nothing is "+
			"written", got, echo(m.Supplied.DispositionsSHA256))
	}
	if ctx.Supplied.Dispositions != supplied {
		return "", fmt.Errorf("scribe: the context's copy of the supplied dispositions is not the researcher's " +
			"text, so the parked pair was rewritten after assembly; nothing is written")
	}
	return supplied, nil
}

// refusePromoted refuses a run whose manifest is already beside it: the durable
// tier is write-once, so a second session over the run would land its records
// and then fail to promote. It is refused here, before anything lands.
func refusePromoted(repoRoot, run string) error {
	rel := issueschema.ReadingsRecordDir + "/" + run + "/" + ManifestFileName
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("scribe: opening the repository root: %w", err)
	}
	defer root.Close()
	switch _, err := root.Lstat(rel); {
	case err == nil:
		return fmt.Errorf("scribe: %s already exists, so a scribe session over %s was ingested; the durable "+
			"tier is write-once, and a later answer is written with the capture verbs", rel, run)
	case !os.IsNotExist(err):
		return fmt.Errorf("scribe: probing %s: %w", rel, err)
	}
	return nil
}

// runItems lists the run's reading items as the store holds them, each with
// whether a disposition already stands over it.
func runItems(repoRoot, run string) (map[string]bool, error) {
	// The listing is a plain path read, so the directories above it are judged
	// first by the same rule the assembly applies, and so are the two it lists
	// through, the readings directory and the run's own: os.ReadDir follows a
	// symlinked leaf.
	if err := refuseRedirectedLedger(repoRoot, issueschema.ReadingsDir, run); err != nil {
		return nil, err
	}
	dir := filepath.Join(repoRoot, filepath.FromSlash(runRecordsDir(run)))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("scribe: listing the reading records of %s: %w", run, err)
	}
	items := map[string]bool{}
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".md")
		if !ok || !recordid.ValidReadingItemID(id) || !e.Type().IsRegular() {
			continue
		}
		fate, err := capture.ItemFate(repoRoot, run, id)
		if err != nil {
			return nil, fmt.Errorf("scribe: reading the fate of %s: %w", id, err)
		}
		items[id] = len(fate.Dispositions) > 0 || fate.Cyclic
	}
	return items, nil
}

// refuseAuthored is the authoring refusal: every item answered is the run's and
// is named in the supplied text; every free text is the researcher's own words;
// the outstanding list is the run's unanswered items; and no item of the run
// that stands unanswered is passed over in silence.
func refuseAuthored(out Output, supplied string, items map[string]bool) error {
	folded := fold(supplied)
	named := func(id string) bool { return mentions(supplied, id) }
	verbatim := func(where, field, text string) error {
		if text == "" || strings.Contains(folded, fold(text)) {
			return nil
		}
		return fmt.Errorf("scribe: %s carries a %s the researcher did not write (%q does not stand in the "+
			"supplied dispositions); the scribe reformats and never adds a word, so the payload is refused and "+
			"nothing is written", where, field, echo(text))
	}
	ofRun := func(where, id string) error {
		if _, ok := items[id]; !ok {
			return fmt.Errorf("scribe: %s names %s, which is not an item of %s; a scribe session transcribes "+
				"one run", where, echo(id), out.Run)
		}
		return nil
	}

	answered := map[string]string{}
	answer := func(where, id string) error {
		if prior, dup := answered[id]; dup {
			return fmt.Errorf("scribe: %s answers %s, which %s already answers; one item takes one answer in "+
				"one payload", where, echo(id), prior)
		}
		answered[id] = where
		return nil
	}
	for i, d := range out.Dispositions {
		where := fmt.Sprintf("dispositions[%d] (%s)", i, echo(d.Item))
		if err := ofRun(where, d.Item); err != nil {
			return err
		}
		if !named(d.Item) {
			return fmt.Errorf("scribe: %s is a disposition the researcher did not supply: the supplied "+
				"dispositions never name %s", where, echo(d.Item))
		}
		// The state is the ruling itself, so it is held to the supplied text as
		// the grounds are, and more tightly: it must stand whole-word on a line
		// that names the item, because a state another item's line carries is not
		// the researcher's answer to this one.
		if !lineCarries(supplied, d.Item, d.State) {
			return fmt.Errorf("scribe: %s carries state %q, and no line of the supplied dispositions that names "+
				"%s carries it; the state is the researcher's ruling and the scribe never supplies one, so the "+
				"payload is refused and nothing is written", where, echo(d.State), echo(d.Item))
		}
		for _, id := range append([]string{d.Supersedes}, d.Recurs...) {
			if id != "" && !named(id) {
				return fmt.Errorf("scribe: %s cites %s, which the supplied dispositions never name", where, echo(id))
			}
		}
		if err := verbatim(where, "grounds", d.Grounds); err != nil {
			return err
		}
		if err := verbatim(where, "exit_condition", d.ExitCondition); err != nil {
			return err
		}
		if err := answer(where, d.Item); err != nil {
			return err
		}
	}
	for i, a := range out.Admissions {
		where := fmt.Sprintf("admissions[%d] (%s)", i, echo(a.Item))
		if err := ofRun(where, a.Item); err != nil {
			return err
		}
		if !named(a.Item) {
			return fmt.Errorf("scribe: %s is an admission the researcher did not supply: the supplied "+
				"dispositions never name %s", where, echo(a.Item))
		}
		// An admission writes an accepted disposition, so it is a state too, held
		// by the same rule: the item's own line admits or accepts the proposal.
		if !lineCarries(supplied, a.Item, admissionTokens...) {
			return fmt.Errorf("scribe: %s is an admission, and no line of the supplied dispositions that names "+
				"%s admits or accepts it (%s); an admission writes an acceptance, which is the researcher's "+
				"ruling and never the scribe's, so the payload is refused and nothing is written",
				where, echo(a.Item), strings.Join(admissionTokens, ", "))
		}
		if err := verbatim(where, "grounds", a.Grounds); err != nil {
			return err
		}
		// An admission and a disposition of one item are one act when their
		// grounds agree, which Admit holds; a second admission is not.
		if prior, dup := answered[a.Item]; dup && strings.HasPrefix(prior, "admissions") {
			return fmt.Errorf("scribe: %s admits %s, which %s already admits", where, echo(a.Item), prior)
		}
		if _, dup := answered[a.Item]; !dup {
			answered[a.Item] = where
		}
	}
	for i, s := range out.Surprises {
		where := fmt.Sprintf("surprises[%d] (%s)", i, echo(s.OccasionedBy))
		if !named(s.OccasionedBy) {
			return fmt.Errorf("scribe: %s is keyed to %s, which the supplied dispositions never name",
				where, echo(s.OccasionedBy))
		}
		if strings.TrimSpace(s.Text) == "" {
			return fmt.Errorf("scribe: %s carries no text", where)
		}
		if err := verbatim(where, "text", s.Text); err != nil {
			return err
		}
	}
	for i, id := range out.Outstanding {
		where := fmt.Sprintf("outstanding[%d]", i)
		if err := ofRun(where, id); err != nil {
			return err
		}
		if prior, dup := answered[id]; dup {
			return fmt.Errorf("scribe: %s lists %s as outstanding and %s answers it; an outstanding item is "+
				"one given no disposition", where, echo(id), prior)
		}
		answered[id] = where
	}
	for i, r := range out.Refusals {
		if _, ok := items[r.Subject]; ok {
			if _, dup := answered[r.Subject]; !dup {
				answered[r.Subject] = fmt.Sprintf("refusals[%d]", i)
			}
		}
	}

	// Silence. An item of the run with no standing disposition that the payload
	// neither answers, lists as outstanding nor refuses is refused: an item the
	// scribe says nothing about reads as one nobody raised. An item already
	// answered in the ledger is not owed again, which is what lets a rerun after
	// a partial ingest drop what landed.
	var silent []string
	for id, standing := range items {
		if _, ok := answered[id]; !ok && !standing {
			silent = append(silent, id)
		}
	}
	sort.Strings(silent)
	if len(silent) > 0 {
		return fmt.Errorf("scribe: the output says nothing about %s of %s; silence is not one of the scribe's "+
			"options, so every item is answered, listed as outstanding, or named in a refusal",
			strings.Join(silent, ", "), out.Run)
	}
	return nil
}

// admissionTokens are the words that carry an admission on an item's line. At
// the widening position acceptance IS admission, so the state's own name counts
// beside the verb's forms.
var admissionTokens = []string{issueschema.DispositionAccepted, "admit", "admits", "admitted"}

// lineCarries reports whether some line of supplied that names id carries one of
// tokens as a whole word, ignoring case. It is the mechanical form of "the
// researcher gave this item this ruling": the token must sit on the item's own
// line, and a word that merely contains it ("unaccepted") does not carry it. It
// reads words, not sense, so a line that names a state to negate it still
// carries it; that residue is the chapter's to disclose.
//
// A line ends at any terminator a researcher's editor writes (lineBreak): split
// on LF alone, a text whose lines end in CR or a Unicode separator is one line,
// and the per-line check collapses to a whole-text one. Within a line, the token
// must sit in the item's own part of it (itemParts), so a line that names two
// items does not grant one item's ruling to both.
func lineCarries(supplied, id string, tokens ...string) bool {
	for _, line := range strings.FieldsFunc(supplied, lineBreak) {
		for _, part := range itemParts(line, id) {
			for _, tok := range tokens {
				if tok != "" && wholeWord(part, tok) {
					return true
				}
			}
		}
	}
	return false
}

// itemParts is the text of line that belongs to id. A line that names id and no
// other item is the item's own, whole, so a ruling written ahead of the id still
// counts for it. On a line that names more than one item, each mention of id
// owns only the text from its end to the next item id the line names: the
// ruling a line gives one item is not granted to another it mentions in passing
// ("rdi-2: accepted, unlike rdi-1" accepts rdi-2 and gives rdi-1 nothing), and a
// ruling written ahead of every id on such a line belongs to none of them, which
// refuses rather than guesses. A line that does not name id gives it nothing.
func itemParts(line, id string) []string {
	spans := itemIDs(line)
	own, other := false, false
	for _, m := range spans {
		if line[m[0]:m[1]] == id {
			own = true
		} else {
			other = true
		}
	}
	switch {
	case !own:
		return nil
	case !other:
		return []string{line}
	}
	var parts []string
	for i, m := range spans {
		if line[m[0]:m[1]] != id {
			continue
		}
		end := len(line)
		if i+1 < len(spans) {
			end = spans[i+1][0]
		}
		parts = append(parts, line[m[1]:end])
	}
	return parts
}

// itemIDPattern matches a reading-item id; itemIDs keeps the matches that stand
// as whole tokens.
var itemIDPattern = regexp.MustCompile(regexp.QuoteMeta(issueschema.ReadingItemFamily) + `-[0-9]+`)

// itemIDs lists, in order, the byte spans of every reading-item id line names as
// a whole token, by the boundary mentions applies: no letter, digit or hyphen
// before it, and no letter or digit after it.
func itemIDs(line string) [][]int {
	var out [][]int
	for _, m := range itemIDPattern.FindAllStringIndex(line, -1) {
		if m[0] > 0 && idByte(line[m[0]-1], true) {
			continue
		}
		if m[1] < len(line) && idByte(line[m[1]], false) {
			continue
		}
		out = append(out, m)
	}
	return out
}

// idByte reports whether c continues an id token: an ASCII letter or digit, or,
// where hyphen is set, a hyphen.
func idByte(c byte, hyphen bool) bool {
	return c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || hyphen && c == '-'
}

// lineBreak reports whether r ends a line: LF, CR (so CRLF too, the empty field
// between them dropped), U+2028 LINE SEPARATOR and U+2029 PARAGRAPH SEPARATOR.
// The code points are written as numbers so no layer between the author and
// the compiler can decode an escape into the wrong byte.
func lineBreak(r rune) bool {
	switch r {
	case 0x0a, 0x0d, 0x2028, 0x2029:
		return true
	}
	return false
}

// wholeWord reports whether text holds word, ignoring case, bounded on each side
// by the text's edge or a character that is neither a letter nor a digit.
func wholeWord(text, word string) bool {
	re, err := regexp.Compile(`(?i)(^|[^\pL\pN])` + regexp.QuoteMeta(word) + `($|[^\pL\pN])`)
	if err != nil {
		return false
	}
	return re.MatchString(text)
}

// fold collapses every run of whitespace to one space and trims the ends, so a
// re-wrapped sentence is the same words.
func fold(s string) string { return strings.Join(strings.Fields(s), " ") }

// mentions reports whether text names id as a whole token: rdi-12 is not named
// by a text that says rdi-123.
func mentions(text, id string) bool {
	if id == "" {
		return false
	}
	re, err := regexp.Compile(`(^|[^A-Za-z0-9-])` + regexp.QuoteMeta(id) + `($|[^A-Za-z0-9])`)
	if err != nil {
		return false
	}
	return re.MatchString(text)
}

func nonNil(in []string) []string {
	if in == nil {
		return []string{}
	}
	return in
}

func nonNilFlags(in []FidelityFlag) []FidelityFlag {
	if in == nil {
		return []FidelityFlag{}
	}
	return in
}

func nonNilRefusals(in []Refusal) []Refusal {
	if in == nil {
		return []Refusal{}
	}
	return in
}
