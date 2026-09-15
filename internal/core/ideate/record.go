package ideate

// record.go is the WRITE half of `abcd ideate record <idea-slug> --verdict-json`:
// it takes the host-composed verdict, proves it is a complete, honestly-cited
// three-leg run, and writes the durable record plus its pointer.
//
// The trust model is the release-ingest seam's, and so are the refusal semantics:
// WHOLE-DOCUMENT, never cite-or-be-dropped. A dropped synthesis entry costs an
// insight; a dropped leg, claim, or citation costs the record its meaning — a
// verdict record with a quietly-removed falsified claim, or a quietly-removed
// grill hit, is worse than no record at all, because a later session trusts it.
// So any fault refuses the document and writes nothing.
//
// The boundary between the host and this core is exact:
//
//	the host owns  — the research, the grill, the adversarial review, the wording
//	                 of every finding, and the verdict itself.
//	the core owns  — the DATE, the FILENAME, the record's structure, the citation
//	                 gate, the containment, the redaction of the host's prose
//	                 before it is committed (redact.go), and the pointer in the
//	                 decision log.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// SlugError is the refusal for an idea slug that cannot safely become a filename.
// It is typed because it is an OPERAND fault, not a payload fault: the fix is to
// re-run with a different slug, not to re-compose the verdict.
type SlugError struct {
	Slug   string
	Reason string
}

func (e *SlugError) Error() string {
	return fmt.Sprintf("%q is not a usable idea slug (%s); a slug is lower-case kebab-case, at most %d characters, "+
		"and carries no path separator or dot segment", termsafe.Sanitize(e.Slug), e.Reason, MaxSlugLen)
}

// CitationError is the record-grill citation gate's refusal: a hit names a record
// id that does not resolve in this repository, so the grill is reporting on
// something that is not there. It names every offending id, because a composer
// handed one id at a time re-runs the whole gauntlet once per mistake.
type CitationError struct {
	Unresolved []string
}

func (e *CitationError) Error() string {
	return "the record grill cites records that do not exist in this repository — nothing was written: " +
		strings.Join(e.Unresolved, ", ") +
		"\nevery grill hit is cited by id, and an id that names no record is a hit on nothing; re-run the grill against the live record"
}

// ExistsError is the no-overwrite refusal: a verdict record for this slug already
// exists under today's date. A second run is either a re-run of the same gauntlet
// (in which case the first record stands) or a different one (in which case it
// deserves its own slug); silently replacing the first would erase a recorded
// reason, which is the one thing this verb exists to preserve.
type ExistsError struct {
	Path string
}

func (e *ExistsError) Error() string {
	return "a verdict record already exists at " + e.Path +
		" — refusing to overwrite it; a re-run of the same idea keeps the first record, and a different run earns its own slug"
}

// MissingDecisionsError is the refusal for a repository with no decision log. The
// pointer is not decoration: the log is where a session looks for what was already
// decided, and a verdict record nothing points at is a record nobody finds.
type MissingDecisionsError struct {
	Path string
}

func (e *MissingDecisionsError) Error() string {
	return "this repository has no " + e.Path +
		" — the verdict record's dated pointer has nowhere to land, so nothing was written; create the decision log first"
}

// Record validates the host-composed verdict for slug and writes it.
//
// at is the clock, passed in rather than read: the date lands in a durable
// filename and in a durable log line, so it is an input a test can pin instead of
// a wall-clock read buried in a writer.
//
// Every refusal returns BEFORE the first write, with one exception that is handled
// rather than accepted: if appending the pointer fails after the record landed,
// the record is removed again, so a run never leaves half a decision behind.
//
// Both artefacts it writes are COMMITTED, so both go through the store-before-commit
// redactor (redact.go): the verdict's free text is redacted field by field on the
// way into the validated document, the rendered record and its decision-log
// pointer are re-scanned before either is written, and a scanner that cannot be
// trusted refuses the run rather than recording under it.
func Record(repoRoot, slug string, raw []byte, at time.Time) (Result, error) {
	if err := validateSlug(slug); err != nil {
		return Result{}, err
	}
	p, err := decodePayload(raw)
	if err != nil {
		return Result{}, err
	}
	// The store-before-commit redactor is built BEFORE the first field is cleaned
	// and long before the first byte is written, so a degraded scanner refuses the
	// run here rather than halfway through it (redact.go). The payload is decoded
	// first only so a malformed document is refused without paying for the
	// scanner's identity probe.
	red, err := newRecordRedactor(repoRoot)
	if err != nil {
		return Result{}, err
	}
	v, err := validate(repoRoot, red, p)
	if err != nil {
		return Result{}, err
	}
	v.slug = slug

	// ONE os.Root at the repository root, used for every path this verb touches.
	// It is the containment boundary: os.Root refuses symlink traversal at EVERY
	// component, not just the leaf, so a repository whose `.abcd/development` is a
	// symlink cannot redirect the write outside the tree. A leaf-only check would
	// pass such a tree, because the kernel resolves the ancestors before the check
	// ever sees them.
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()

	// The decision log is read BEFORE the record is written, so the commonest
	// reason the pointer cannot land is a refusal rather than a rollback. The
	// through-the-Root Lstat is what refuses a symlinked ANCESTOR; ReadGuarded's
	// O_NOFOLLOW refuses a symlinked leaf. The gap between them is the benign
	// TOCTOU of the trusted-worktree model.
	if _, err := root.Lstat(DecisionsRelDir); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Result{}, &MissingDecisionsError{Path: DecisionsRelDir}
		}
		return Result{}, fmt.Errorf("reading %s: %w", DecisionsRelDir, err)
	}
	decisionsAbs := filepath.Join(repoRoot, filepath.FromSlash(DecisionsRelDir))
	name := recordDate(at) + "-ideate-" + slug + ".md"
	rel := path.Join(ResearchRelDir, name)
	res := v.result(slug, rel)

	// BOTH committed artefacts are composed and re-scanned here, before either is
	// written: the record body, and the one dated line that will be appended to the
	// decision log. Stage two is fail-closed, so a blocking span that survived the
	// field pass refuses the whole run — and because the record write is a one-shot
	// exclusive create, a refusal at this point cannot leave a partial document
	// behind. There is nothing to roll back because nothing was begun.
	body := render(v, at)
	pointer := pointerLine(res, at)
	if err := red.verify(body, pointer); err != nil {
		return Result{}, err
	}

	// Read the log, write the record, and append the pointer under ONE exclusive
	// lock. The decision log is a read-modify-write over a shared file, which is
	// the house rule's case: without the lock two concurrent runs both read the
	// same log, both append, and the second write erases the first's pointer —
	// leaving a verdict record that nothing points at, the exact state the
	// rollback below exists to prevent.
	err = fsutil.WithFileLock(filepath.Join(repoRoot, filepath.FromSlash(decisionsLockRel)), decisionsLockTimeout, func() error {
		log, err := fsutil.ReadGuarded(decisionsAbs, maxDecisionsBytes)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return &MissingDecisionsError{Path: DecisionsRelDir}
			}
			return fmt.Errorf("reading %s: %w", DecisionsRelDir, err)
		}
		// The research directory is CREATED when absent rather than demanded.
		// Nothing in abcd establishes it and no convention check requires it, so
		// refusing here would fail the first run in any repository — after the three
		// host legs have already been paid for, and with the verdict unrecoverable
		// when it arrived on stdin. Through the Root, a symlinked component still
		// refuses.
		if err := root.MkdirAll(ResearchRelDir, 0o755); err != nil {
			return fmt.Errorf("preparing %s: %w", ResearchRelDir, err)
		}
		// The EXCLUSIVE create is the no-overwrite guarantee, and it is one syscall
		// rather than a check followed by a write, so it holds even against a writer
		// that is not taking this lock.
		if err := fsutil.CreateExclusiveIn(root, rel, []byte(body), 0o644); err != nil {
			if errors.Is(err, os.ErrExist) {
				return &ExistsError{Path: rel}
			}
			return err
		}
		if err := appendPointer(decisionsAbs, log, pointer); err != nil {
			// The record landed but its pointer did not. Roll the record back rather
			// than leave a verdict no session will find, and say the rollback failed
			// if it does, because that is the one case a human must fix by hand.
			if rmErr := root.Remove(rel); rmErr != nil {
				return fmt.Errorf("%w; THE ROLLBACK FAILED — remove %s by hand", err, rel)
			}
			return err
		}
		return nil
	})
	if err != nil {
		return Result{}, err
	}
	return res, nil
}

// validateSlug is the lexical containment layer. It runs FIRST, before the payload
// is even decoded, so an operand that could steer a write never travels further
// into the verb than this line.
func validateSlug(slug string) error {
	switch {
	case slug == "":
		return &SlugError{Slug: slug, Reason: "it is empty"}
	case len(slug) > MaxSlugLen:
		return &SlugError{Slug: slug[:MaxSlugLen], Reason: fmt.Sprintf("it is %d bytes", len(slug))}
	case !slugRe.MatchString(slug):
		return &SlugError{Slug: slug, Reason: "it is not lower-case kebab-case"}
	}
	return nil
}

// decodePayload reads the untrusted document behind the shared guards: a size cap,
// an unknown-field refusal (an invented key means the composer and this core
// disagree about the contract), a trailing-data refusal, the three-branch schema
// gate, and the prompt_version stamp.
func decodePayload(raw []byte) (Payload, error) {
	if len(raw) > MaxPayloadBytes {
		return Payload{}, fmt.Errorf("verdict payload exceeds the %d-byte cap", MaxPayloadBytes)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var p Payload
	if err := dec.Decode(&p); err != nil {
		return Payload{}, fmt.Errorf("malformed verdict JSON: %v", err)
	}
	if dec.More() {
		return Payload{}, errors.New("the verdict payload carries trailing data after the JSON document")
	}
	switch {
	case p.SchemaVersion == 0:
		return Payload{}, errors.New("verdict payload is missing schema_version")
	case p.SchemaVersion > SchemaVersion:
		return Payload{}, update.TooNew("verdict",
			p.SchemaVersion, SchemaVersion)
	case p.SchemaVersion != SchemaVersion:
		return Payload{}, fmt.Errorf("unsupported verdict schema_version %d", p.SchemaVersion)
	}
	if !promptVersionRe.MatchString(p.PromptVersion) {
		return Payload{}, errors.New("verdict payload is missing a semver prompt_version")
	}
	return p, nil
}

// verdictDoc is the validated, cleaned document — the only shape the renderer and
// the result builder ever see, so neither can reach a raw untrusted field.
type verdictDoc struct {
	slug         string
	idea         string
	verdict      Verdict
	claims       []Claim
	hits         []GrillHit
	kills        []KillAttempt
	rejected     []RejectedAlternative
	noneRejected bool
	cited        []string
	// redactions counts the secret/PII spans stage one rewrote out of the free
	// text above, so the surface can say what happened instead of the record
	// being quietly rewritten under its composer (loud-staging).
	redactions int
}

// validate runs every gate over the decoded payload and returns the cleaned
// document. The order is deliberate: cheap structural checks first, and the
// citation gate — the only one that touches the filesystem — last, so a malformed
// document never costs a record scan.
//
// red is stage one of the store-before-commit gate (redact.go). Every free-text
// field is redacted on the way in, ahead of termsafe's cleaning and well ahead of
// the renderer's escaping: the escape must stay the LAST transformation applied,
// or a placeholder dropped into rendered markdown reopens the structure hazards
// render.go closes. The enum-constrained members — a verdict, a claim status, a
// relation, a kill outcome, a cited record id — are validated shapes with nothing
// to redact, and running a matcher over them would corrupt a value the record is
// keyed on rather than protect anything.
func validate(repoRoot string, red *recordRedactor, p Payload) (verdictDoc, error) {
	var v verdictDoc

	idea, n := red.field(p.Idea)
	v.redactions += n
	v.idea = termsafe.CleanProse(idea, maxIdeaBytes)
	if v.idea == "" {
		return v, errors.New("the verdict payload carries no idea text; the record is about an idea, so the idea is required")
	}
	if !p.Verdict.Valid() {
		return v, fmt.Errorf("out-of-enum verdict %q; a verdict is one of %s|%s|%s",
			termsafe.Sanitize(string(p.Verdict)), VerdictSurvives, VerdictKilled, VerdictReframed)
	}
	v.verdict = p.Verdict

	if len(p.Legs) != len(legOrder) {
		return v, fmt.Errorf("the verdict carries %d legs; the gauntlet is exactly %d, in order (%s)",
			len(p.Legs), len(legOrder), legList())
	}
	for i, leg := range p.Legs {
		if leg.Kind != legOrder[i] {
			return v, fmt.Errorf("leg %d is %q; the legs run in order (%s), and the order is what each leg is looking at",
				i+1, termsafe.Sanitize(string(leg.Kind)), legList())
		}
	}
	var err error
	var n1, n2, n3, n4 int
	if v.claims, n1, err = validateResearch(red, p.Legs[0]); err != nil {
		return v, err
	}
	if v.hits, n2, err = validateGrill(red, p.Legs[1]); err != nil {
		return v, err
	}
	if v.kills, n3, err = validateAdversarial(red, p.Legs[2]); err != nil {
		return v, err
	}
	if v.rejected, v.noneRejected, n4, err = validateRejected(red, p); err != nil {
		return v, err
	}
	v.redactions += n1 + n2 + n3 + n4
	if v.cited, err = resolveCitations(repoRoot, v.hits); err != nil {
		return v, err
	}
	return v, nil
}

// validateResearch checks the first leg: at least one claim, each with prose, a
// named primary source, and a registered status.
func validateResearch(red *recordRedactor, leg Leg) ([]Claim, int, error) {
	if len(leg.Hits) > 0 || len(leg.KillAttempts) > 0 {
		return nil, 0, errors.New("the research leg carries another leg's evidence; each leg carries only its own")
	}
	if len(leg.Claims) == 0 {
		return nil, 0, errors.New("the research leg carries no claims; a leg that checked nothing is not a leg")
	}
	if len(leg.Claims) > maxClaims {
		return nil, 0, fmt.Errorf("the research leg carries %d claims (max %d)", len(leg.Claims), maxClaims)
	}
	redactions := 0
	out := make([]Claim, 0, len(leg.Claims))
	for i, c := range leg.Claims {
		at := i + 1
		// A primary source is free text too: the field holds a URL by convention,
		// but nothing constrains it, and a file:// path to the researcher's own
		// working copy is the commonest thing a research leg reaches for.
		claim, n := red.field(c.Claim)
		source, m := red.field(c.PrimarySource)
		redactions += n + m
		text := termsafe.CleanProseLine(claim, maxProseBytes)
		if text == "" {
			return nil, 0, fmt.Errorf("claim %d has no text", at)
		}
		src := termsafe.CleanProseLine(source, maxSourceBytes)
		if src == "" {
			return nil, 0, fmt.Errorf("claim %d names no primary source; a claim checked against nothing is an assertion", at)
		}
		if !claimStatusEnum[c.Status] {
			return nil, 0, fmt.Errorf("claim %d has status %q; a claim is %s, %s, or %s",
				at, termsafe.Sanitize(string(c.Status)), ClaimVerified, ClaimFalsified, ClaimUnverifiable)
		}
		out = append(out, Claim{Claim: text, PrimarySource: src, Status: c.Status})
	}
	return out, redactions, nil
}

// validateGrill checks the second leg. An EMPTY hit list is valid and meaningful:
// "no entry in the record touches this idea" is a finding, and the commonest one
// for a genuinely new idea. What is not valid is a hit that cites nothing usable.
func validateGrill(red *recordRedactor, leg Leg) ([]GrillHit, int, error) {
	if len(leg.Claims) > 0 || len(leg.KillAttempts) > 0 {
		return nil, 0, errors.New("the record-grill leg carries another leg's evidence; each leg carries only its own")
	}
	if len(leg.Hits) > maxHits {
		return nil, 0, fmt.Errorf("the record grill carries %d hits (max %d)", len(leg.Hits), maxHits)
	}
	redactions := 0
	out := make([]GrillHit, 0, len(leg.Hits))
	for i, h := range leg.Hits {
		at := i + 1
		// Bounded BEFORE the id is matched or quoted, so an over-long one is
		// described rather than echoed.
		if len(h.Record) > maxRecordIDBytes {
			return nil, 0, fmt.Errorf("grill hit %d cites a %d-byte record id (max %d)", at, len(h.Record), maxRecordIDBytes)
		}
		if !recordid.CitedIDRe.MatchString(h.Record) {
			return nil, 0, fmt.Errorf("grill hit %d cites %q, which is not a record id (want adr-N, itd-N, iss-N, or spc-N)",
				at, termsafe.Sanitize(h.Record))
		}
		if !relationEnum[h.Relation] {
			return nil, 0, fmt.Errorf("grill hit %d (%s) has relation %q; a hit is %s, %s, or %s",
				at, h.Record, termsafe.Sanitize(string(h.Relation)), RelationCovered, RelationContradicted, RelationSuperseded)
		}
		// The cited id is NOT redacted: CitedIDRe has just proved it is a record
		// id, and rewriting a value the citation gate resolves against would break
		// the proof the record carries.
		note, n := red.field(h.Note)
		redactions += n
		out = append(out, GrillHit{
			Record:   h.Record,
			Relation: h.Relation,
			Note:     termsafe.CleanProseLine(note, maxProseBytes),
		})
	}
	return out, redactions, nil
}

// validateAdversarial checks the third leg: at least one attempt, each with prose
// and a registered outcome. The leg's fresh-context, off-policy, unknown-authorship
// conduct is the orchestrating prompt's obligation — the binary cannot observe how
// an agent was run, and pretending to check it would be theatre.
func validateAdversarial(red *recordRedactor, leg Leg) ([]KillAttempt, int, error) {
	if len(leg.Claims) > 0 || len(leg.Hits) > 0 {
		return nil, 0, errors.New("the adversarial-review leg carries another leg's evidence; each leg carries only its own")
	}
	if len(leg.KillAttempts) == 0 {
		return nil, 0, errors.New("the adversarial-review leg carries no kill attempts; an adversary that tried nothing reviewed nothing")
	}
	if len(leg.KillAttempts) > maxKillAttempts {
		return nil, 0, fmt.Errorf("the adversarial review carries %d attempts (max %d)", len(leg.KillAttempts), maxKillAttempts)
	}
	redactions := 0
	out := make([]KillAttempt, 0, len(leg.KillAttempts))
	for i, k := range leg.KillAttempts {
		at := i + 1
		attempt, n := red.field(k.Attempt)
		redactions += n
		text := termsafe.CleanProseLine(attempt, maxProseBytes)
		if text == "" {
			return nil, 0, fmt.Errorf("kill attempt %d has no text", at)
		}
		if !killOutcomeEnum[k.Outcome] {
			return nil, 0, fmt.Errorf("kill attempt %d has outcome %q; an attempt is %s, %s, or %s",
				at, termsafe.Sanitize(string(k.Outcome)), KillSurvived, KillPartial, KillFatal)
		}
		out = append(out, KillAttempt{Attempt: text, Outcome: k.Outcome})
	}
	return out, redactions, nil
}

// validateRejected enforces the explicit-empty-marker rule. Omission is refused
// because the whole point of the record is that a later session can see WHAT was
// turned down; a silently empty list is indistinguishable from a composer that
// forgot, and the cost of that ambiguity is the idea being re-proposed.
func validateRejected(red *recordRedactor, p Payload) ([]RejectedAlternative, bool, int, error) {
	if len(p.RejectedAlternatives) > maxRejected {
		return nil, false, 0, fmt.Errorf("the verdict carries %d rejected alternatives (max %d)",
			len(p.RejectedAlternatives), maxRejected)
	}
	if len(p.RejectedAlternatives) == 0 {
		if !p.NoRejectedAlternatives {
			return nil, false, 0, errors.New(
				"the verdict records no rejected alternatives and does not say so explicitly — nothing was written; " +
					"list them, or set \"no_rejected_alternatives\": true to declare on the record that none were considered " +
					"(the record exists so the idea is not re-litigated, and silence cannot carry that)")
		}
		return nil, true, 0, nil
	}
	if p.NoRejectedAlternatives {
		return nil, false, 0, errors.New(
			"the verdict sets \"no_rejected_alternatives\" beside a populated list; the two contradict each other")
	}
	redactions := 0
	out := make([]RejectedAlternative, 0, len(p.RejectedAlternatives))
	for i, r := range p.RejectedAlternatives {
		at := i + 1
		alternative, n := red.field(r.Alternative)
		why, m := red.field(r.WhyRejected)
		redactions += n + m
		alt := termsafe.CleanProseLine(alternative, maxProseBytes)
		if alt == "" {
			return nil, false, 0, fmt.Errorf("rejected alternative %d has no text", at)
		}
		reason := termsafe.CleanProseLine(why, maxProseBytes)
		if reason == "" {
			return nil, false, 0, fmt.Errorf("rejected alternative %d (%s) records no reason; the reason is the half that stops the re-litigation", at, alt)
		}
		out = append(out, RejectedAlternative{Alternative: alt, WhyRejected: reason})
	}
	return out, false, redactions, nil
}

// resolveCitations is AC 4's gate: every record id the grill cited must name a
// record that exists in this repository. It returns the sorted, deduped set as the
// proof the result carries.
//
// It builds ONE resolver for the whole document, so every citation is judged
// against the same view of the record.
func resolveCitations(repoRoot string, hits []GrillHit) ([]string, error) {
	if len(hits) == 0 {
		return nil, nil
	}
	r, err := recordid.NewResolver(repoRoot)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var cited, unresolved []string
	for _, h := range hits {
		if seen[h.Record] {
			continue
		}
		seen[h.Record] = true
		if _, ok := r.Lookup(h.Record); !ok {
			unresolved = append(unresolved, h.Record)
			continue
		}
		cited = append(cited, h.Record)
	}
	if len(unresolved) > 0 {
		sort.Strings(unresolved)
		return nil, &CitationError{Unresolved: unresolved}
	}
	sort.Strings(cited)
	return cited, nil
}

// result builds the transport-agnostic outcome from the validated document.
func (v verdictDoc) result(slug, rel string) Result {
	res := Result{
		Slug:                 slug,
		Idea:                 v.idea,
		Verdict:              v.verdict,
		Graduates:            v.verdict == VerdictSurvives,
		Path:                 rel,
		DecisionPath:         DecisionsRelDir,
		GrillHits:            len(v.hits),
		CitedRecords:         v.cited,
		KillAttempts:         len(v.kills),
		RejectedAlternatives: len(v.rejected),
		Redactions:           v.redactions,
	}
	for _, c := range v.claims {
		switch c.Status {
		case ClaimVerified:
			res.Claims.Verified++
		case ClaimFalsified:
			res.Claims.Falsified++
		case ClaimUnverifiable:
			res.Claims.Unverifiable++
		}
	}
	return res
}

// pointerLine composes the one dated line that makes the verdict findable. It is
// a single line by design: the decision log is a scannable index, and the record
// it points at is where the reasoning lives.
//
// Only core-owned values reach the line: the date, the validated slug, the
// registered verdict, and the path this core built. No untrusted prose lands in
// the decision log at all — the idea's own words live in the record the line
// points at, where the renderer's escaping applies.
//
// That is a property of today's format string, not a guarantee, which is why the
// line is composed HERE rather than inside the append: composing it early is what
// lets stage two re-scan it alongside the record body, before either is written
// (redact.go). The values it does carry are not rewritten — a redacted path would
// name a record that does not exist — so a span surviving in this line is a
// refusal, and the free text that a later field might pull in from Result was
// already redacted at the source by stage one.
func pointerLine(res Result, at time.Time) string {
	return fmt.Sprintf("- %s — ideate: %s — verdict %s. The idea, the three legs, and the rejected alternatives: %s\n",
		recordDate(at), res.Slug, res.Verdict, res.Path)
}

// appendPointer appends the already-composed, already-verified pointer line.
func appendPointer(decisionsAbs string, existing []byte, line string) error {
	body := string(existing)
	if body != "" && !strings.HasSuffix(body, "\n") {
		body += "\n"
	}
	return fsutil.WriteFileAtomicPreserveMode(decisionsAbs, []byte(body+line))
}

// legList renders the required leg order for an error message, written from
// legOrder so the message can never fall out of step with the enum.
func legList() string {
	names := make([]string, 0, len(legOrder))
	for _, k := range legOrder {
		names = append(names, string(k))
	}
	return strings.Join(names, " -> ")
}
