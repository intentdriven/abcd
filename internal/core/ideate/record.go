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
//	                 before it is committed (redact.go), and the decision record
//	                 that points at a public verdict.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/decisions"
	"github.com/intentdriven/abcd/internal/core/homeignore"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/source"
	"github.com/intentdriven/abcd/internal/core/update"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
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

// CitationError is the citation gate's refusal: the verdict names a record id
// that does not resolve in this repository, in a grill hit or anywhere else in
// its legs, so it is reporting on something that is not there. It names every
// offending id and where each was written, because a composer handed one id at
// a time re-runs the whole gauntlet once per mistake.
type CitationError struct {
	// Unresolved is the sorted, deduped set of ids that name no record.
	Unresolved []string
	// At is each unresolved id with the part of the verdict it was written in,
	// in the order the verdict carries them.
	At []UnresolvedCitation
}

// UnresolvedCitation is one id that names no record, and the leg (or the other
// part of the verdict) it was written in.
type UnresolvedCitation struct {
	Where string
	ID    string
}

// newCitationError builds the refusal from the citations that did not resolve.
func newCitationError(at []UnresolvedCitation) *CitationError {
	seen := map[string]bool{}
	var ids []string
	for _, c := range at {
		if !seen[c.ID] {
			seen[c.ID] = true
			ids = append(ids, c.ID)
		}
	}
	sort.Strings(ids)
	return &CitationError{Unresolved: ids, At: at}
}

func (e *CitationError) Error() string {
	parts := make([]string, len(e.At))
	for i, c := range e.At {
		parts[i] = c.Where + " cites " + c.ID
	}
	return "the verdict cites records that do not exist in this repository — nothing was written: " +
		strings.Join(parts, "; ") +
		"\nevery record id in the verdict must name a record, and an id that names none cites nothing; check each against the live record"
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

// VisibilityError is the refusal for a verdict with no answer to "is this idea
// private or public?" (iss-2610090740139804). It is typed because it is an
// OPERAND fault: the fix is to ask the person and re-run with the answer, never
// to re-compose the verdict, and never to pick a store on their behalf.
type VisibilityError struct {
	Got Visibility
}

func (e *VisibilityError) Error() string {
	return "the verdict's store is not answered (" + termsafe.DescribeRefused(string(e.Got)) + ") — nothing was written; " +
		"ask the person whether the idea is private or public, and record it with that answer"
}

// Record validates the host-composed verdict for slug and writes it.
//
// at is the clock, passed in rather than read: the date lands in a durable
// filename, so it is an input a test can pin instead of a wall-clock read buried
// in a writer. The decision record that points at the verdict takes its date
// from its own minted id, as every decision does.
//
// Every refusal returns BEFORE the first write, with one exception that is handled
// rather than accepted: if recording the pointer fails after the verdict landed,
// the verdict is removed again, so a run never leaves half a decision behind.
//
// store carries the person's answer, private or public, and the root commit
// both stores are keyed on (iss-2610090740139804); it is checked beside the
// slug, before the payload is decoded, so a run without the answer writes
// nothing. A private verdict goes to recordPrivate and never touches the
// repository.
//
// A public verdict's artefacts are COMMITTED, so both go through the
// store-before-commit redactor (redact.go): the verdict's free text is redacted
// field by field on the way into the validated document, the rendered record and
// the text of the decision that points at it are re-scanned before either is
// written, and a scanner that cannot be trusted refuses the run rather than
// recording under it. A private verdict takes the same gate.
func Record(repoRoot, slug string, store Store, raw []byte, at time.Time) (Result, error) {
	if err := validateSlug(slug); err != nil {
		return Result{}, err
	}
	if err := validateStore(store); err != nil {
		return Result{}, err
	}
	p, err := decodePayload(repoRoot, raw)
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
	v, err := validate(repoRoot, store.SourcesDir, red, p)
	if err != nil {
		return Result{}, err
	}
	v.slug = slug
	name := recordDate(at) + "-ideate-" + slug + ".md"
	if store.Visibility == VisibilityPrivate {
		return recordPrivate(v, red, store, name, at)
	}

	// ONE os.Root at the repository root, used for every path this function
	// writes (the decision writer contains its own write the same way).
	// It is the containment boundary: os.Root resolves every component, ancestors
	// included, and refuses any path that would ESCAPE the root, so a repository
	// whose `.abcd/development` is a symlink pointing outside the tree cannot
	// redirect the write there. A leaf-only check would pass such a tree, because
	// the kernel resolves the ancestors before the check ever sees them. os.Root
	// does NOT refuse every symlink: a link that stays inside the tree is followed
	// and the write lands at its in-tree target. Containment is the only guarantee
	// this writer rests on, so that is correct, and a repository's own in-tree
	// arrangement is left as it set it (iss-2610050259237102).
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return Result{}, err
	}
	defer root.Close()

	// The shared tier the pointer is recorded in is checked BEFORE the verdict is
	// written, so the commonest reason the pointer cannot land (a repository
	// with no shared tier, which `ahoy install` lays) is a refusal rather than a
	// rollback. The refusal is the decision writer's own (spc-2610110931381316,
	// step 3): ideate keeps no separate missing-log error, because there is no
	// log to be missing.
	if err := decisions.RequireSharedTier(repoRoot); err != nil {
		return Result{}, err
	}
	dir := path.Join(ResearchRelDir, store.RootSHA)
	rel := path.Join(dir, name)
	res := v.result(slug, rel)
	res.Visibility = VisibilityPublic

	// BOTH committed artefacts are composed and re-scanned here, before either is
	// written: the record body, and the text of the decision that will point at
	// it. Stage two is fail-closed, so a blocking span that survived the field
	// pass refuses the whole run — and because the record write is a one-shot
	// exclusive create, a refusal at this point cannot leave a partial document
	// behind. There is nothing to roll back because nothing was begun.
	body := render(v, at)
	pointer := pointerText(res)
	if err := red.verify(body, pointer); err != nil {
		return Result{}, err
	}

	// The research directory is CREATED when absent rather than demanded.
	// Nothing in abcd establishes it and no convention check requires it, so
	// refusing here would fail the first run in any repository — after the three
	// host legs have already been paid for, and with the verdict unrecoverable
	// when it arrived on stdin. Through the Root, a component symlinked out of
	// the tree still refuses.
	if err := root.MkdirAll(dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("preparing %s: %w", dir, err)
	}
	// The EXCLUSIVE create is the no-overwrite guarantee, and it is one syscall
	// rather than a check followed by a write, so it holds against a concurrent
	// run for the same slug without a lock. Nothing else here is shared: the
	// pointer is a new file of its own, so two runs never contend for one.
	if err := fsutil.CreateExclusiveIn(root, rel, []byte(body), 0o644); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Result{}, &ExistsError{Path: rel}
		}
		return Result{}, err
	}
	d, err := decisions.Record(repoRoot, pointer)
	if err != nil {
		// The verdict landed but its pointer did not. Roll the verdict back rather
		// than leave one no session will find, and say the rollback failed if it
		// does, because that is the one case a human must fix by hand.
		if rmErr := root.Remove(rel); rmErr != nil {
			return Result{}, fmt.Errorf("%w; THE ROLLBACK FAILED — remove %s by hand", err, rel)
		}
		return Result{}, err
	}
	res.DecisionPath = d.Path
	return res, nil
}

// recordPrivate writes a private verdict into the person's home store,
// ~/.abcd.noindex/ideas/<root-sha>/, and nothing else: no decision points at
// it, because the decisions folder is committed and the pointer would publish
// the idea's slug.
//
// The folder is created and opened through the home-scope walk every writer
// into abcd's home uses (fsutil.EnsureHomeScope), so a level symlinked out of
// the home is refused rather than written through, and each level it creates is
// owner-only. The record is created exclusively, owner-only, so a re-run keeps
// the first record exactly as the public store does.
//
// The record still goes through the redactor, stage two included, although it
// is never committed: a credential in a private note is still a credential on
// disk, and one redaction path for both stores means the private record a
// person later publishes by hand was already scanned.
func recordPrivate(v verdictDoc, red *recordRedactor, store Store, name string, at time.Time) (Result, error) {
	rel := path.Join(PrivateRel, store.RootSHA)
	shown := abcdhome.Display("ideas", store.RootSHA, name)
	res := v.result(v.slug, shown)
	res.Visibility = VisibilityPrivate
	body := render(v, at)
	if err := red.verify(body); err != nil {
		return Result{}, err
	}
	dir, err := fsutil.EnsureHomeScope(store.Home, rel, abcdhome.DirMode)
	if err != nil {
		return Result{}, fmt.Errorf("preparing %s: %w", abcdhome.Display("ideas", store.RootSHA), err)
	}
	defer dir.Close()
	// The ignore file is laid here, before the verdict, rather than left to the
	// front door: that lay is skipped after a failure and silent under --json, and
	// the account's own home is let through validateStore only because abcd's
	// home ignores itself there. A home it cannot be laid in takes no verdict.
	if _, err := homeignore.Lay(store.Home); err != nil {
		return Result{}, fmt.Errorf("%s could not be laid, so a git repository in the home folder could stage a private verdict — nothing was written: %w",
			abcdhome.Display(homeignore.IgnoreName), err)
	}
	if err := fsutil.CreateExclusiveIn(dir, name, []byte(body), abcdhome.FileMode); err != nil {
		if errors.Is(err, os.ErrExist) {
			return Result{}, &ExistsError{Path: shown}
		}
		return Result{}, err
	}
	return res, nil
}

// validateStore is the store's operand check, run beside the slug's and before
// the payload is decoded: the answer must be given, the key must be a git
// object name before it reaches a path, and a private verdict's home must be
// one a commit cannot carry.
func validateStore(store Store) error {
	if store.Visibility != VisibilityPrivate && store.Visibility != VisibilityPublic {
		return &VisibilityError{Got: store.Visibility}
	}
	if !gitutil.IsFullSHA(store.RootSHA) {
		return fmt.Errorf("the store key %s is not a repository root commit — nothing was written; "+
			"a verdict is filed under the root commit of the repository it is about, so a repository with no commit has no store",
			termsafe.DescribeRefused(store.RootSHA))
	}
	if store.Visibility == VisibilityPublic {
		return nil
	}
	if store.Home == "" || !filepath.IsAbs(store.Home) {
		return errors.New("no absolute home directory is resolved (HOME is unset or relative), so a private verdict has no " +
			abcdhome.Display("ideas") + " to go to — nothing was written")
	}
	// The one rule for whether a home may hold abcd's machine-scoped records
	// (gitutil.HomeInsideWorkingTree): a home inside a git working tree that is
	// not the account's own is one a commit could carry, and a private verdict
	// written there is published by the next `git add -A`. The account's own
	// home is let through even when it is a dotfiles repository, because abcd's
	// home ignores itself there (internal/core/homeignore,
	// itd-2610140858048004); a person who deleted that ignore file chose to
	// let their repository track it.
	if gitutil.HomeInsideWorkingTree(store.Home) != "" {
		return errors.New("the home directory lies inside a git working tree, where a commit could carry a private verdict — nothing was written; " +
			"run abcd with your own HOME")
	}
	return nil
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
//
// The decoder's message names an undeclared field by the payload's own key, the
// one value the composer needs to find the fault, so it is redacted through the
// canonical scanner for repoRoot rather than returned raw (iss-2609290218032954).
func decodePayload(repoRoot string, raw []byte) (Payload, error) {
	if len(raw) > MaxPayloadBytes {
		return Payload{}, fmt.Errorf("verdict payload exceeds the %d-byte cap", MaxPayloadBytes)
	}
	// An older shape is refused before the strict decode, which would stop at
	// the first field version 2 moved (a version-1 adversarial leg's
	// kill_attempts) and never name the version to compose.
	var peek struct {
		SchemaVersion int `json:"schema_version"`
	}
	if json.Unmarshal(raw, &peek) == nil && peek.SchemaVersion > 0 && peek.SchemaVersion < SchemaVersion {
		return Payload{}, olderSchemaError(peek.SchemaVersion)
	}
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	var p Payload
	if err := dec.Decode(&p); err != nil {
		return Payload{}, fmt.Errorf("malformed verdict JSON: %s", scanner.RedactRefusal(repoRoot, err.Error()))
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
	case p.SchemaVersion < 0:
		return Payload{}, fmt.Errorf("unsupported verdict schema_version %d", p.SchemaVersion)
	case p.SchemaVersion != SchemaVersion:
		return Payload{}, olderSchemaError(p.SchemaVersion)
	}
	if !promptVersionRe.MatchString(p.PromptVersion) {
		return Payload{}, errors.New("verdict payload is missing a semver prompt_version")
	}
	return p, nil
}

// olderSchemaError refuses an older shape, which is never upgraded: version 1
// had nowhere to put a search, a second reviewer or a reframing, so reading it
// as version 2 would invent what each leg proved. The message names the
// version to compose, because the host that sent it is reading an older
// command page.
func olderSchemaError(got int) error {
	return fmt.Errorf("the verdict payload is schema_version %d, which this abcd no longer records — nothing was written; "+
		"compose it as version %d, which adds origin_sources, the grill's searches, the adversarial leg's reviewers and ruling, "+
		"and a reframing for a reframed verdict (the ideate command page carries the shape)", got, SchemaVersion)
}

// verdictDoc is the validated, cleaned document — the only shape the renderer and
// the result builder ever see, so neither can reach a raw untrusted field.
type verdictDoc struct {
	slug         string
	idea         string
	verdict      Verdict
	origins      []string
	claims       []Claim
	hits         []GrillHit
	searches     []Search
	reviewers    []Reviewer
	ruling       *Ruling
	rejected     []RejectedAlternative
	noneRejected bool
	reframing    string
	cited        []string
	// redactions counts the secret/PII spans stage one rewrote out of the free
	// text above, so the surface can say what happened instead of the record
	// being quietly rewritten under its composer (loud-staging).
	redactions int
}

// validate runs every gate over the decoded payload and returns the cleaned
// document. The order is deliberate: cheap structural checks first, and the
// citation gate and the sources check — the only ones that touch the
// filesystem — last, so a malformed document never costs a record scan.
//
// red is stage one of the store-before-commit gate (redact.go). Every free-text
// field is redacted on the way in, ahead of termsafe's cleaning and well ahead of
// the renderer's escaping: the escape must stay the LAST transformation applied,
// or a placeholder dropped into rendered markdown reopens the structure hazards
// render.go closes. The enum-constrained members — a verdict, a claim status, a
// relation, a kill outcome, a cited record id — are validated shapes with nothing
// to redact, and running a matcher over them would corrupt a value the record is
// keyed on rather than protect anything.
//
// sourcesDir is the corpus a `source:<key>` is resolved against (Store.SourcesDir).
func validate(repoRoot, sourcesDir string, red *recordRedactor, p Payload) (verdictDoc, error) {
	var v verdictDoc

	idea, n := red.field(p.Idea)
	v.redactions += n
	v.idea = termsafe.CleanProse(idea, maxIdeaBytes)
	if v.idea == "" {
		return v, errors.New("the verdict payload carries no idea text; the record is about an idea, so the idea is required")
	}
	if !p.Verdict.Valid() {
		return v, fmt.Errorf("out-of-enum verdict (%s); a verdict is one of %s|%s|%s",
			termsafe.DescribeRefused(string(p.Verdict)), VerdictSurvives, VerdictKilled, VerdictReframed)
	}
	v.verdict = p.Verdict

	if len(p.Legs) != len(legOrder) {
		return v, fmt.Errorf("the verdict carries %d legs; the gauntlet is exactly %d, in order (%s)",
			len(p.Legs), len(legOrder), legList())
	}
	for i, leg := range p.Legs {
		if leg.Kind != legOrder[i] {
			return v, fmt.Errorf("leg %d is not %s (its kind is %s); the legs run in order (%s), and the order is what each leg is looking at",
				i+1, legOrder[i], termsafe.DescribeRefused(string(leg.Kind)), legList())
		}
	}
	var err error
	var n0, n1, n2, n3, n4, n5 int
	if v.origins, n0, err = validateOrigins(red, p.OriginSources); err != nil {
		return v, err
	}
	if v.claims, n1, err = validateResearch(red, p.Legs[0], v.origins); err != nil {
		return v, err
	}
	if v.hits, v.searches, n2, err = validateGrill(red, p.Legs[1]); err != nil {
		return v, err
	}
	if v.reviewers, v.ruling, n3, err = validateAdversarial(red, p.Legs[2]); err != nil {
		return v, err
	}
	if v.rejected, v.noneRejected, n4, err = validateRejected(red, p); err != nil {
		return v, err
	}
	if v.reframing, n5, err = validateReframing(red, p); err != nil {
		return v, err
	}
	v.redactions += n0 + n1 + n2 + n3 + n4 + n5
	if v.cited, err = resolveCitations(repoRoot, v); err != nil {
		return v, err
	}
	if err := resolveSourceKeys(sourcesDir, v); err != nil {
		return v, err
	}
	return v, nil
}

// validateOrigins checks the sources the idea came from. The list is required
// but may be empty: the self-quotation check (validateResearch) compares every
// verified claim against it, and a list that could be left out would let that
// check be skipped by omission, the failure the rejected-alternatives marker
// exists for. An origin is free text, because an idea can come from a
// conversation as well as a page; it is never a primary source, so its shape is
// not one.
func validateOrigins(red *recordRedactor, origins []string) ([]string, int, error) {
	if origins == nil {
		return nil, 0, errors.New("the verdict does not say where the idea came from — nothing was written; " +
			"list the sources it came from in \"origin_sources\", or [] for an idea with no source behind it " +
			"(a claim checked only against the idea's own source is quoted, not verified, and that check reads the list)")
	}
	if len(origins) > maxOriginSources {
		return nil, 0, fmt.Errorf("the verdict carries %d origin sources (max %d)", len(origins), maxOriginSources)
	}
	redactions := 0
	out := make([]string, 0, len(origins))
	for i, o := range origins {
		text, n := red.field(o)
		redactions += n
		clean := termsafe.CleanProseLine(text, maxSourceBytes)
		if clean == "" {
			return nil, 0, fmt.Errorf("origin source %d is empty", i+1)
		}
		out = append(out, clean)
	}
	return out, redactions, nil
}

// validateResearch checks the first leg: at least one claim, each with prose, a
// primary source of a known shape, and a registered status. A claim marked
// verified whose primary source is one of the idea's own origins is refused:
// checked against the source the idea came from, it was quoted, not verified.
func validateResearch(red *recordRedactor, leg Leg, origins []string) ([]Claim, int, error) {
	if len(leg.Hits) > 0 || len(leg.Searches) > 0 || len(leg.Reviewers) > 0 || leg.Ruling != nil {
		return nil, 0, errors.New("the research leg carries another leg's evidence; each leg carries only its own")
	}
	if len(leg.Claims) == 0 {
		return nil, 0, errors.New("the research leg carries no claims; a leg that checked nothing is not a leg")
	}
	if len(leg.Claims) > maxClaims {
		return nil, 0, fmt.Errorf("the research leg carries %d claims (max %d)", len(leg.Claims), maxClaims)
	}
	origin := originKeys(origins)
	redactions := 0
	out := make([]Claim, 0, len(leg.Claims))
	for i, c := range leg.Claims {
		at := i + 1
		// A primary source is free text until its shape is checked, and a
		// file:// path to the researcher's own working copy is the commonest
		// thing a research leg reaches for, so it is redacted like any prose.
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
		if !sourceShaped(src) {
			return nil, 0, fmt.Errorf("claim %d names a primary source of no known shape (%s); %s",
				at, termsafe.DescribeRefused(src), sourceShapes)
		}
		if !claimStatusEnum[c.Status] {
			return nil, 0, fmt.Errorf("claim %d has an out-of-enum status (%s); a claim is %s, %s, %s, or %s",
				at, termsafe.DescribeRefused(string(c.Status)), ClaimVerified, ClaimQuoted, ClaimFalsified, ClaimUnverifiable)
		}
		if c.Status == ClaimVerified && origin[sourceKey(src)] {
			return nil, 0, fmt.Errorf("claim %d is marked %s, but its primary source is one of the idea's own origin sources — nothing was written; "+
				"a claim checked only against the source the idea came from is %s, not %s", at, ClaimVerified, ClaimQuoted, ClaimVerified)
		}
		out = append(out, Claim{Claim: text, PrimarySource: src, Status: c.Status})
	}
	return out, redactions, nil
}

// sourceShapes says in words what sourceShaped accepts, for every refusal of a
// source to quote, so the refusals cannot drift from the check.
var sourceShapes = "a source is an https:// address, source:<key> for a source filed in the sources corpus, or a record id (" +
	recordid.CitedIDForms() + ")"

// sourceShaped reports whether a cleaned source is one of the three shapes a
// source may take: an https:// address with a host and no whitespace, a
// `source:<key>` whose key has the corpus's key grammar, or a record id. The
// address is not fetched and the record id is resolved with every other
// citation (resolveCitations); this is the shape alone, so a remark heard in
// passing cannot stand where a source does.
func sourceShaped(src string) bool {
	switch {
	case strings.HasPrefix(src, "https://"):
		if strings.IndexFunc(src, unicode.IsSpace) >= 0 {
			return false
		}
		u, err := url.Parse(src)
		return err == nil && u.Host != ""
	case strings.HasPrefix(src, sourceKeyPrefix):
		return source.ValidKey(strings.TrimPrefix(src, sourceKeyPrefix))
	}
	return recordid.CitedIDRe.MatchString(src)
}

// sourceKeyPrefix marks a source filed in the sources corpus.
const sourceKeyPrefix = "source:"

// sourceKey is the spelling the self-quotation check compares sources by, so
// one source written two ways is still one source: an address loses its
// fragment and a trailing slash, its host is lower-cased and http and https
// meet, and a record id is folded as the resolver folds it (`ADR-0035` is
// adr-35). Anything else is compared as written.
func sourceKey(s string) string {
	if lower := strings.ToLower(s); strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") {
		if u, err := url.Parse(s); err == nil && u.Host != "" {
			k := "https://" + strings.ToLower(u.Host) + strings.TrimSuffix(u.EscapedPath(), "/")
			if u.RawQuery != "" {
				k += "?" + u.RawQuery
			}
			return k
		}
	}
	if c := recordid.CanonCitedID(s); c != "" {
		return c
	}
	return s
}

// originKeys is every source the idea's origins name, by sourceKey. An origin
// is free text, so besides the whole entry each word of it that is an address
// or a source key counts, as written and with surrounding punctuation trimmed
// ("https://… the post the idea came from" names the post, and so does
// "(https://…)"). A record id counts only as a whole entry: one mentioned in
// passing is not where the idea came from, and checking it is a primary check.
func originKeys(origins []string) map[string]bool {
	keys := make(map[string]bool, len(origins))
	for _, o := range origins {
		keys[sourceKey(o)] = true
		for _, w := range strings.Fields(o) {
			for _, s := range []string{w, strings.Trim(w, "()<>[]{},.;\"'")} {
				if lower := strings.ToLower(s); strings.HasPrefix(lower, "https://") || strings.HasPrefix(lower, "http://") ||
					strings.HasPrefix(s, sourceKeyPrefix) {
					keys[sourceKey(s)] = true
				}
			}
		}
	}
	return keys
}

// addressRe matches a web address written in running text, up to the next
// space, quote, bracket, comma or semicolon, so an id written straight after
// one (`…/thread;itd-9`) is still read as a citation.
var addressRe = regexp.MustCompile(`(?i)\bhttps?://[^\s,;<>"'()\[\]]+`)

// withoutAddresses blanks every web address in text, so an id-shaped segment
// of a path (`…/adr-0007-use-markdown.md`) is read as part of the address it
// sits in, not as a citation of a record in this repository.
func withoutAddresses(text string) string {
	return addressRe.ReplaceAllString(text, " ")
}

// validateGrill checks the second leg. An EMPTY hit list is valid and meaningful:
// "no entry in the record touches this idea" is a finding, and the commonest one
// for a genuinely new idea — but only when the grill says what it searched. A
// grill with no hits and no searches is indistinguishable from one whose search
// never ran, so it is refused. What is never valid is a hit that cites nothing
// usable, or a search that cannot say what it searched or what came back.
func validateGrill(red *recordRedactor, leg Leg) ([]GrillHit, []Search, int, error) {
	if len(leg.Claims) > 0 || len(leg.Reviewers) > 0 || leg.Ruling != nil {
		return nil, nil, 0, errors.New("the record-grill leg carries another leg's evidence; each leg carries only its own")
	}
	if len(leg.Hits) > maxHits {
		return nil, nil, 0, fmt.Errorf("the record grill carries %d hits (max %d)", len(leg.Hits), maxHits)
	}
	if len(leg.Hits) == 0 && len(leg.Searches) == 0 {
		return nil, nil, 0, errors.New("the record grill found no hits and records no searches — nothing was written; " +
			"an empty grill says what it searched: list each search under \"searches\" as {terms, places, count}, " +
			"so a grill that found nothing can be told from one that never looked")
	}
	redactions := 0
	out := make([]GrillHit, 0, len(leg.Hits))
	for i, h := range leg.Hits {
		at := i + 1
		// Bounded BEFORE the id is matched or quoted, so an over-long one is
		// described rather than echoed.
		if len(h.Record) > maxRecordIDBytes {
			return nil, nil, 0, fmt.Errorf("grill hit %d cites a %d-byte record id (max %d)", at, len(h.Record), maxRecordIDBytes)
		}
		if !recordid.CitedIDRe.MatchString(h.Record) {
			return nil, nil, 0, fmt.Errorf("grill hit %d cites %s, which is not a record id (want %s)",
				at, termsafe.DescribeRefused(h.Record), recordid.CitedIDForms())
		}
		if !relationEnum[h.Relation] {
			return nil, nil, 0, fmt.Errorf("grill hit %d (%s) has an out-of-enum relation (%s); a hit is %s, %s, or %s",
				at, h.Record, termsafe.DescribeRefused(string(h.Relation)), RelationCovered, RelationContradicted, RelationSuperseded)
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
	searches, n, err := validateSearches(red, leg.Searches)
	if err != nil {
		return nil, nil, 0, err
	}
	return out, searches, redactions + n, nil
}

// validateSearches checks the grill's search record: each search names at least
// one term and one place, and says how many entries came back. A count of zero
// is the finding an empty grill rests on; an omitted count is refused, because
// read as zero it would report a search that never returned as one that found
// nothing.
func validateSearches(red *recordRedactor, in []Search) ([]Search, int, error) {
	if len(in) > maxSearches {
		return nil, 0, fmt.Errorf("the record grill carries %d searches (max %d)", len(in), maxSearches)
	}
	redactions := 0
	out := make([]Search, 0, len(in))
	for i, s := range in {
		at := i + 1
		if s.Count == nil {
			return nil, 0, fmt.Errorf("grill search %d records no count; a search that cannot say what came back records nothing", at)
		}
		if *s.Count < 0 {
			return nil, 0, fmt.Errorf("grill search %d records a negative count", at)
		}
		terms, n, err := cleanList(red, s.Terms, maxSearchTerms, fmt.Sprintf("grill search %d", at), "term")
		if err != nil {
			return nil, 0, err
		}
		places, m, err := cleanList(red, s.Places, maxSearchPlaces, fmt.Sprintf("grill search %d", at), "place")
		if err != nil {
			return nil, 0, err
		}
		redactions += n + m
		count := *s.Count
		out = append(out, Search{Terms: terms, Places: places, Count: &count})
	}
	return out, redactions, nil
}

// cleanList redacts and line-cleans a required, bounded list of short values,
// refusing an empty list or an empty member. owner and noun name the list in a
// refusal ("grill search 2 names no place").
func cleanList(red *recordRedactor, in []string, max int, owner, noun string) ([]string, int, error) {
	if len(in) == 0 {
		return nil, 0, fmt.Errorf("%s names no %s", owner, noun)
	}
	if len(in) > max {
		return nil, 0, fmt.Errorf("%s names %d %ss (max %d)", owner, len(in), noun, max)
	}
	redactions := 0
	out := make([]string, 0, len(in))
	for j, raw := range in {
		text, n := red.field(raw)
		redactions += n
		clean := termsafe.CleanProseLine(text, maxSourceBytes)
		if clean == "" {
			return nil, 0, fmt.Errorf("%s %s %d is empty", owner, noun, j+1)
		}
		out = append(out, clean)
	}
	return out, redactions, nil
}

// validateAdversarial checks the third leg: at least one reviewer, each under a
// label of its own with at least one kill attempt, a registered verdict, and the
// sources it consulted; and, where the reviewers' verdicts differ, the ruling
// that settled the split. The leg's fresh-context, off-policy, unknown-authorship
// conduct is the orchestrating prompt's obligation — the binary cannot observe
// how an agent was run, and pretending to check it would be theatre.
func validateAdversarial(red *recordRedactor, leg Leg) ([]Reviewer, *Ruling, int, error) {
	if len(leg.Claims) > 0 || len(leg.Hits) > 0 || len(leg.Searches) > 0 {
		return nil, nil, 0, errors.New("the adversarial-review leg carries another leg's evidence; each leg carries only its own")
	}
	if len(leg.Reviewers) == 0 {
		return nil, nil, 0, errors.New("the adversarial-review leg carries no reviewers; an adversary that was never run reviewed nothing")
	}
	if len(leg.Reviewers) > maxReviewers {
		return nil, nil, 0, fmt.Errorf("the adversarial review carries %d reviewers (max %d)", len(leg.Reviewers), maxReviewers)
	}
	redactions := 0
	labels := map[string]int{}
	out := make([]Reviewer, 0, len(leg.Reviewers))
	for i, r := range leg.Reviewers {
		at := i + 1
		raw, n := red.field(r.Label)
		redactions += n
		label := termsafe.CleanProseLine(raw, maxLabelBytes)
		if label == "" {
			return nil, nil, 0, fmt.Errorf("reviewer %d has no label; each reviewer's findings are kept under its label", at)
		}
		if first, dup := labels[label]; dup {
			return nil, nil, 0, fmt.Errorf("reviewer %d repeats the label of reviewer %d; each reviewer is kept under a label of its own", at, first)
		}
		labels[label] = at
		if !r.Verdict.Valid() {
			return nil, nil, 0, fmt.Errorf("reviewer %d has an out-of-enum verdict (%s); a reviewer's verdict is one of %s|%s|%s",
				at, termsafe.DescribeRefused(string(r.Verdict)), VerdictSurvives, VerdictKilled, VerdictReframed)
		}
		kills, n, err := validateKillAttempts(red, at, r.KillAttempts)
		if err != nil {
			return nil, nil, 0, err
		}
		redactions += n
		if len(r.Sources) > maxReviewerSources {
			return nil, nil, 0, fmt.Errorf("reviewer %d names %d sources (max %d)", at, len(r.Sources), maxReviewerSources)
		}
		sources := make([]string, 0, len(r.Sources))
		for j, s := range r.Sources {
			text, n := red.field(s)
			redactions += n
			src := termsafe.CleanProseLine(text, maxSourceBytes)
			if !sourceShaped(src) {
				return nil, nil, 0, fmt.Errorf("reviewer %d source %d is of no known shape (%s); %s",
					at, j+1, termsafe.DescribeRefused(src), sourceShapes)
			}
			sources = append(sources, src)
		}
		out = append(out, Reviewer{Label: label, KillAttempts: kills, Verdict: r.Verdict, Sources: sources})
	}
	ruling, n, err := validateRuling(red, leg.Ruling, out)
	if err != nil {
		return nil, nil, 0, err
	}
	return out, ruling, redactions + n, nil
}

// validateKillAttempts checks one reviewer's attempts: at least one, each with
// prose and a registered outcome.
func validateKillAttempts(red *recordRedactor, reviewer int, in []KillAttempt) ([]KillAttempt, int, error) {
	if len(in) == 0 {
		return nil, 0, fmt.Errorf("reviewer %d carries no kill attempts; an adversary that tried nothing reviewed nothing", reviewer)
	}
	if len(in) > maxKillAttempts {
		return nil, 0, fmt.Errorf("reviewer %d carries %d attempts (max %d)", reviewer, len(in), maxKillAttempts)
	}
	redactions := 0
	out := make([]KillAttempt, 0, len(in))
	for i, k := range in {
		at := i + 1
		attempt, n := red.field(k.Attempt)
		redactions += n
		text := termsafe.CleanProseLine(attempt, maxProseBytes)
		if text == "" {
			return nil, 0, fmt.Errorf("reviewer %d kill attempt %d has no text", reviewer, at)
		}
		if !killOutcomeEnum[k.Outcome] {
			return nil, 0, fmt.Errorf("reviewer %d kill attempt %d has an out-of-enum outcome (%s); an attempt is %s, %s, or %s",
				reviewer, at, termsafe.DescribeRefused(string(k.Outcome)), KillSurvived, KillPartial, KillFatal)
		}
		out = append(out, KillAttempt{Attempt: text, Outcome: k.Outcome})
	}
	return out, redactions, nil
}

// validateRuling checks the ruling against the reviewers it settles. It is
// required when their verdicts differ, and optional otherwise; given, it names
// who ruled, a registered verdict, and the reason. Whether the ruling was
// right is the person's judgement, and the binary never checks it against the
// verdict the whole gauntlet reached.
func validateRuling(red *recordRedactor, in *Ruling, reviewers []Reviewer) (*Ruling, int, error) {
	if in == nil {
		for _, r := range reviewers[1:] {
			if r.Verdict != reviewers[0].Verdict {
				return nil, 0, fmt.Errorf("the reviewers' verdicts differ (%s) and the adversarial-review leg records no ruling — nothing was written; "+
					"a split between reviewers is recorded with the ruling that settled it: \"ruling\": {by, verdict, reason}",
					reviewerVerdicts(reviewers))
			}
		}
		return nil, 0, nil
	}
	by, n := red.field(in.By)
	reason, m := red.field(in.Reason)
	out := Ruling{
		By:      termsafe.CleanProseLine(by, maxLabelBytes),
		Verdict: in.Verdict,
		Reason:  termsafe.CleanProseLine(reason, maxProseBytes),
	}
	switch {
	case out.By == "":
		return nil, 0, errors.New("the ruling does not say who ruled")
	case !out.Verdict.Valid():
		return nil, 0, fmt.Errorf("the ruling has an out-of-enum verdict (%s); a ruling is one of %s|%s|%s",
			termsafe.DescribeRefused(string(in.Verdict)), VerdictSurvives, VerdictKilled, VerdictReframed)
	case out.Reason == "":
		return nil, 0, errors.New("the ruling records no reason; the reason is what a later reader weighs the split by")
	}
	return &out, n + m, nil
}

// reviewerVerdicts lists each reviewer's verdict by its position, for the
// split refusal. Only core-owned values reach it: a position and a registered
// verdict, never a label.
func reviewerVerdicts(reviewers []Reviewer) string {
	parts := make([]string, len(reviewers))
	for i, r := range reviewers {
		parts[i] = fmt.Sprintf("reviewer %d %s", i+1, r.Verdict)
	}
	return strings.Join(parts, ", ")
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

// validateReframing checks the reframing against the verdict: a reframed
// verdict carries a non-empty one, and any other verdict carries none, since a
// reframing beside a surviving or a killed idea leaves a reader to guess which
// of the two the record means.
func validateReframing(red *recordRedactor, p Payload) (string, int, error) {
	text, n := red.field(p.Reframing)
	reframing := termsafe.CleanProse(text, maxIdeaBytes)
	switch {
	case p.Verdict == VerdictReframed && reframing == "":
		return "", 0, errors.New("the verdict is reframed and carries no reframing — nothing was written; " +
			"a reframed verdict records the idea's surviving shape in \"reframing\", because that, not the original wording, is what may graduate")
	case p.Verdict != VerdictReframed && reframing != "":
		return "", 0, fmt.Errorf("the verdict carries a reframing beside a %s verdict; only a reframed verdict has one, and the two contradict each other", p.Verdict)
	}
	return reframing, n, nil
}

// citedText is one place in the verdict a record id may be written, and the
// part of the verdict it belongs to, which a refusal names.
type citedText struct {
	where string
	text  string
}

// citationPlaces lists every place in the cleaned document a record id may be
// written, each with the part of the verdict it belongs to: every leg's prose
// and sources, the rejected alternatives, the origin sources and the
// reframing. The idea's own words are not among them — it is the subject of
// the record, in the words it entered with — and nor are the grill's search
// terms and places, where a searched-for id that names nothing is the finding.
// A grill hit's record is not here either: it is resolved on its own, exactly,
// because it is the grill's citation proof.
func citationPlaces(v verdictDoc) []citedText {
	const (
		research    = "the research leg"
		grill       = "the record-grill leg"
		adversarial = "the adversarial-review leg"
	)
	var out []citedText
	add := func(where string, texts ...string) {
		for _, t := range texts {
			out = append(out, citedText{where: where, text: t})
		}
	}
	for _, c := range v.claims {
		add(research, c.Claim, c.PrimarySource)
	}
	for _, h := range v.hits {
		add(grill, h.Note)
	}
	for _, r := range v.reviewers {
		add(adversarial, r.Label)
		for _, k := range r.KillAttempts {
			add(adversarial, k.Attempt)
		}
		add(adversarial, r.Sources...)
	}
	if v.ruling != nil {
		add(adversarial, v.ruling.By, v.ruling.Reason)
	}
	for _, r := range v.rejected {
		add("the rejected alternatives", r.Alternative, r.WhyRejected)
	}
	add("the origin sources", v.origins...)
	add("the reframing", v.reframing)
	return out
}

// proseIDRe finds a record id of a resolvable family anywhere in running text,
// in any case and with any zero padding. Its families are recordid's cited
// families, so prose is held to exactly the ids a grill hit may cite. The token
// boundaries are checked in code (idsIn), because Go's regexp has no lookbehind.
var proseIDRe = regexp.MustCompile(`(?i)(?:` + citedAlternation() + `)-[0-9]+`)

// citedAlternation is recordid.CitedFamilies as an alternation.
func citedAlternation() string {
	fams := recordid.CitedFamilies()
	for i, f := range fams {
		fams[i] = regexp.QuoteMeta(f)
	}
	return strings.Join(fams, "|")
}

// idsIn returns every record id written in text as a whole token: not preceded
// by a letter, digit or hyphen (`xitd-1`, `pre-itd-1` name nothing), and not
// followed by a letter or digit. A hyphen after the number does not stop it
// being a citation, so `itd-104-ideate` names itd-104, as record-lint reads
// the same shape.
func idsIn(text string) []string {
	var out []string
	for _, loc := range proseIDRe.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && isIDByte(text[loc[0]-1], true) {
			continue
		}
		if loc[1] < len(text) && isIDByte(text[loc[1]], false) {
			continue
		}
		out = append(out, text[loc[0]:loc[1]])
	}
	return out
}

// isIDByte reports whether b would make a matched id part of a longer token: a
// letter or digit on either side, and a hyphen before it.
func isIDByte(b byte, before bool) bool {
	switch {
	case b >= 'a' && b <= 'z', b >= 'A' && b <= 'Z', b >= '0' && b <= '9':
		return true
	case b == '-':
		return before
	}
	return false
}

// resolveCitations is criterion 1's gate (and AC 4's before it): every record id
// the verdict cites, in any leg, must name a record that exists in this
// repository. It returns the sorted, deduped set of the GRILL's citations as the
// proof the result carries; every other id it resolved is proved by the record
// having been written at all.
//
// It builds ONE resolver for the whole document, so every citation is judged
// against the same view of the record, and only when the document cites
// something: a verdict that names no id costs no scan.
//
// A prose id is folded to the spelling the resolver keys on (recordid.CanonCitedID)
// before it is looked up, so `ADR-0035` in a kill attempt names adr-35; an id
// that folds to nothing (an all-zero number) names no record and is refused.
func resolveCitations(repoRoot string, v verdictDoc) ([]string, error) {
	type found struct{ where, id string }
	var prose []found
	for _, place := range citationPlaces(v) {
		for _, raw := range idsIn(withoutAddresses(place.text)) {
			prose = append(prose, found{where: place.where, id: raw})
		}
	}
	if len(v.hits) == 0 && len(prose) == 0 {
		return nil, nil
	}
	r, err := recordid.NewResolver(repoRoot)
	if err != nil {
		return nil, err
	}
	var missing []UnresolvedCitation
	seen := map[UnresolvedCitation]bool{}
	miss := func(where, id string) {
		c := UnresolvedCitation{Where: where, ID: id}
		if !seen[c] {
			seen[c] = true
			missing = append(missing, c)
		}
	}
	cited := map[string]bool{}
	for _, h := range v.hits {
		if !r.Has(h.Record) {
			miss("the record-grill leg", h.Record)
			continue
		}
		cited[h.Record] = true
	}
	for _, f := range prose {
		canon := recordid.CanonCitedID(f.id)
		switch {
		case len(f.id) > maxRecordIDBytes:
			// Bounded before it is echoed: the prose cap admits an id of
			// thousands of digits, and the refusal describes it instead.
			miss(f.where, termsafe.DescribeRefused(f.id))
		case canon == "":
			miss(f.where, strings.ToLower(f.id))
		case !r.Has(canon):
			miss(f.where, canon)
		}
	}
	if len(missing) > 0 {
		return nil, newCitationError(missing)
	}
	out := make([]string, 0, len(cited))
	for id := range cited {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, nil
}

// resolveSourceKeys checks every `source:<key>` the verdict cites — a claim's
// primary source or a reviewer's — against the sources corpus at dir. Where
// there is no corpus (dir is empty, or names a directory with none in it) there
// is nothing to resolve against, and the key's shape, already checked, is all
// that can be proved; a corpus that exists and cannot be read refuses the run,
// because a key it might not carry would otherwise be recorded as filed.
func resolveSourceKeys(dir string, v verdictDoc) error {
	type cite struct{ where, key string }
	var keys []cite
	for i, c := range v.claims {
		if k, ok := strings.CutPrefix(c.PrimarySource, sourceKeyPrefix); ok {
			keys = append(keys, cite{where: fmt.Sprintf("claim %d", i+1), key: k})
		}
	}
	for i, r := range v.reviewers {
		for _, s := range r.Sources {
			if k, ok := strings.CutPrefix(s, sourceKeyPrefix); ok {
				keys = append(keys, cite{where: fmt.Sprintf("reviewer %d", i+1), key: k})
			}
		}
	}
	if len(keys) == 0 || dir == "" {
		return nil
	}
	want := make([]string, len(keys))
	for i, k := range keys {
		want[i] = k.key
	}
	unfiled, err := source.Unfiled(dir, want)
	switch {
	case errors.Is(err, source.ErrNoCorpus):
		return nil
	case err != nil:
		return fmt.Errorf("the verdict cites a filed source, and the sources corpus cannot be read to resolve it — nothing was written: %w", err)
	}
	if len(unfiled) == 0 {
		return nil
	}
	missing := map[string]bool{}
	for _, k := range unfiled {
		missing[k] = true
	}
	var parts []string
	for _, k := range keys {
		if missing[k.key] {
			// The key has the corpus's key grammar (sourceShaped), so it is
			// safe to quote, and it is the one handle a refusal names a source by.
			parts = append(parts, k.where+" cites "+sourceKeyPrefix+k.key)
		}
	}
	return fmt.Errorf("the verdict cites sources the sources corpus does not carry — nothing was written: %s; "+
		"file each with `abcd library add`, or cite it by its https:// address", strings.Join(parts, ", "))
}

// result builds the transport-agnostic outcome from the validated document.
func (v verdictDoc) result(slug, rel string) Result {
	res := Result{
		Slug:                 slug,
		Idea:                 v.idea,
		Verdict:              v.verdict,
		Graduates:            v.verdict == VerdictSurvives,
		Reframing:            v.reframing,
		Path:                 rel,
		GrillHits:            len(v.hits),
		CitedRecords:         v.cited,
		Searches:             len(v.searches),
		Reviewers:            len(v.reviewers),
		RejectedAlternatives: len(v.rejected),
		Redactions:           v.redactions,
	}
	for _, r := range v.reviewers {
		res.KillAttempts += len(r.KillAttempts)
	}
	for _, c := range v.claims {
		switch c.Status {
		case ClaimVerified:
			res.Claims.Verified++
		case ClaimQuoted:
			res.Claims.Quoted++
		case ClaimFalsified:
			res.Claims.Falsified++
		case ClaimUnverifiable:
			res.Claims.Unverifiable++
		}
	}
	return res
}

// pointerText composes the text of the decision that makes a public verdict
// findable. It is one line by design: a decision is a scannable entry, and the
// verdict it points at is where the reasoning lives. decisions.Record opens it
// with the decision's date, so it carries none of its own.
//
// Only core-owned values reach the text: the validated slug, the registered
// verdict, and the path this core built. No untrusted prose lands in the
// decision at all — the idea's own words live in the record it points at, where
// the renderer's escaping applies.
//
// That is a property of today's format string, not a guarantee, which is why the
// text is composed HERE rather than inside the write: composing it early is what
// lets stage two re-scan it alongside the record body, before either is written
// (redact.go). The values it does carry are not rewritten — a redacted path would
// name a record that does not exist — so a span surviving in this text is a
// refusal, and the free text that a later field might pull in from Result was
// already redacted at the source by stage one.
func pointerText(res Result) string {
	return fmt.Sprintf("ideate: %s — verdict %s. The idea, the three legs, and the rejected alternatives: %s",
		res.Slug, res.Verdict, res.Path)
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
