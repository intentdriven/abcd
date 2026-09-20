package intent

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/changelog"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// mintLockTimeout bounds how long CreateFromText waits for the intent-store mint
// lock. A var (not const) so a test can shorten it to exercise contention.
var mintLockTimeout = 5 * time.Second

// intentFamily is the intent store's id prefix, the family tag the mint splices
// into every native itd id.
const intentFamily = "itd"

// minter is the intent family's record-id mint seam (adr-45; per-family
// adoption as configuration, ruling 3). The zero value is the production
// configuration — real clock, crypto entropy; tests inject both so a
// same-instant case is deterministic.
var minter recordid.Minter

// mintRetryBudget bounds how many fresh ids one draft mint draws when a
// candidate already names a record in this checkout — the same-second,
// same-suffix coincidence, which is redrawn rather than bumped (spc-33 ruling
// 2). It mirrors the capture ledger's placeholder retry budget.
const mintRetryBudget = 8

// maxSlugLen caps a derived slug so a pathological free-text line cannot produce
// an unwieldy filename. Mirrors the capture-side derivation budget.
const maxSlugLen = 60

// CreateFromText files a new draft intent seeded from free-form text, mirroring
// the capture engine's create shape: it derives a filename-safe slug, mints a
// native timestamp-numeric itd id under the store mint lock, and atomically
// writes drafts/itd-N-<slug>.md with the canonical draft frontmatter set and a
// minimal, honest body skeleton. Empty/whitespace text is refused and nothing
// is written.
//
// The text IS the press release: the whole of it seeds `## Press Release` as
// prose, and the H1 is its first sentence (deriveTitle), or opts.Title when the
// caller gives one. Neither is pasted a second time under `## Why This
// Matters`, which takes a prompt for its own content instead
// (iss-2609170726360399: a paragraph as a heading is unreadable, the slug
// truncates it, and a draft whose one press-release-first section is a
// placeholder reads as malformed on sight).
//
// The seeded record is lint-valid (intent_lifecycle accepts a draft whose kind is
// null and whose spec_id is null) and passes Validate; a human expands it, then
// `abcd intent plan` schedules it. This is the quoted-text create path itd-46
// delivers — the create half of what spc-6 AC3 (promote) needs.
func CreateFromText(repoRoot, text string, opts TextOptions) (Intent, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return Intent{}, fmt.Errorf("intent: refusing to create from empty text")
	}
	// An explicit title is held to the text's own bar before anything is
	// derived: non-empty once trimmed, and one line — it becomes the H1, where a
	// second line would open a new block. It is redacted below through the same
	// scanner as the text. A title is given when the caller says so (TitleSet)
	// or supplies any text at all, so `--title ""` and `--title " "` are refused
	// alike rather than one of them silently falling back to the derived H1.
	title := strings.TrimSpace(opts.Title)
	if (opts.TitleSet || opts.Title != "") && title == "" {
		return Intent{}, fmt.Errorf("intent: refusing an empty --title (nothing written)")
	}
	if strings.ContainsAny(title, "\r\n") {
		return Intent{}, fmt.Errorf("intent: --title must be a single line (nothing written)")
	}
	// Redact the caller's text through the one canonical scanner BEFORE anything
	// derived from it is built (gh-486). The slug becomes the filename and is
	// derived straight from this text, so a secret/home-path in the prose would
	// otherwise reach the committed filename too (the capture-slug leak shape):
	// redacting first and deriving the slug from the redacted text is what keeps a
	// finding out of the path as well as the body. CreateDraft redacts title and
	// body again as its own boundary guard; that second pass is idempotent here.
	redacted, _, err := redactIntentText(repoRoot, trimmed)
	if err != nil {
		return Intent{}, err
	}
	slug, err := deriveIntentSlug(redacted)
	if err != nil {
		return Intent{}, err
	}
	if title == "" {
		title = deriveTitle(redacted)
	} else {
		// The same boundary the text crossed: the title is free prose too, and
		// CreateDraft's own pass is idempotent on it.
		if title, _, err = redactIntentText(repoRoot, title); err != nil {
			return Intent{}, err
		}
	}
	return CreateDraft(repoRoot, DraftOptions{
		Slug:         slug,
		Title:        title,
		PressRelease: redacted,
		Impact:       opts.Impact,
		// A quoted-text create is a person at a keyboard, so the arrival path is
		// the default; the production mode is the operator's declared one, or the
		// repo's default resolved by the surface.
		ProductionMode: opts.ProductionMode,
	})
}

// TextOptions parameterizes CreateFromText beyond the text itself: an explicit
// H1 title in place of the derived first sentence, the optional impact
// judgement and the production mode. Impact and mode are stamped as CreateDraft
// validates them; the title is validated and redacted by CreateFromText.
//
// TitleSet says the caller gave a title at all, so that an explicit empty one
// (`--title ""`) is refused like a blank one rather than read as no title: the
// surface passes whether the flag was set, and a non-empty Title counts as set
// on its own.
type TextOptions struct {
	Title          string
	TitleSet       bool
	Impact         string
	ProductionMode string
}

// sentenceEndRe finds a sentence terminator that is followed by whitespace or
// ends the text. A terminator glued to the next token (`v0.9`, `e.g.x`) is not
// a boundary; there is no abbreviation list, deliberately — the split is simple
// and the caller can override the result with an explicit title.
var sentenceEndRe = regexp.MustCompile(`[.!?](\s|$)`)

// deriveTitle is the H1 a quoted-text create derives: the text's first sentence
// with its terminator dropped (no intent in the record ends its H1 with one),
// whitespace collapsed, and — when the sentence is longer than the slug cap —
// cut on a word boundary at or before maxSlugLen runes, so the title stays a
// heading and never a paragraph. A single unbroken token longer than the cap is
// cut at the cap. It never returns an empty string for non-empty input: a text
// that is only terminators keeps its first sentence whole.
func deriveTitle(text string) string {
	first := text
	if loc := sentenceEndRe.FindStringIndex(text); loc != nil {
		// Trim the run, not just the match, so an ellipsis leaves no stub.
		first = strings.TrimRight(text[:loc[0]], ".!?")
	}
	first = titleLine(first)
	if first == "" {
		first = titleLine(text)
	}
	runes := []rune(first)
	if len(runes) <= maxSlugLen {
		return first
	}
	cut := string(runes[:maxSlugLen])
	if i := strings.LastIndexAny(cut, " "); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimSpace(cut)
}

// DraftOptions parameterises CreateDraft: the explicit slug and title, the
// prose that seeds the two narrative sections, an optional impact judgement,
// and — on the promote path (spc-24) — the iss-N the draft graduated from,
// written as the promoted_from back-edge (an iss-N, or the rdi-N of a
// dispositioned reading item).
//
// The two prose members are per-section, and each is optional on its own:
// PressRelease seeds `## Press Release` as prose (the quoted-text route, whose
// text is the press release) and, when empty, that section takes the route's
// seed note; SeedBody seeds `## Why This Matters` (the promote route's by-id
// pointer) and, when empty, that section takes WhyThisMattersPrompt. A draft
// with neither carries no prose at all and is refused.
type DraftOptions struct {
	Slug         string
	Title        string
	PressRelease string
	SeedBody     string
	Impact       string
	PromotedFrom string
	// Origin is the draft's arrival path (itd-178), PARSED rather than named: a
	// caller declaring the reading kind hands over the run and the item in the
	// same field, so the pointer cannot be forgotten at a call site. It is DERIVED
	// from which command ran, never carried as free text — the zero value means
	// the default (a verb a person invoked), the issue route of capture.Promote
	// passes extracted-from-record, and its reading route passes
	// contributed-by-reading with the pair it read out of the readings store.
	Origin provenance.Origin
	// ProductionMode is how the seed text was produced. Empty takes
	// provenance.DefaultMode, so a draft written through a command carries the key
	// whatever the caller says.
	ProductionMode string
}

// promotedFromRe constrains the promote back-edge to a captured id before it is
// written into frontmatter. Two families graduate into an intent: an issue
// somebody noticed (iss-N) and a reading item an instrument returned and a
// researcher dispositioned (rdi-N). Both are one act — an observation earning an
// admission — so both write the same back edge, and the forward stamp on the
// source record is what closes the join.
var promotedFromRe = regexp.MustCompile(`^(iss|rdi)-[0-9]+$`)

// CreateDraft is the one canonical draft-mint primitive: both the quoted-text
// create (CreateFromText) and the capture-promote path (capture.Promote, which
// supplies an explicit slug and seed body) route through it, so a draft can
// never be minted outside the store mint lock. It validates every option at the
// boundary — the slug becomes a filename — mints a native timestamp-numeric itd
// id through the shared recordid seam, and atomically writes
// drafts/itd-N-<slug>.md. On any refusal nothing is written.
func CreateDraft(repoRoot string, opts DraftOptions) (Intent, error) {
	if !slugRe.MatchString(opts.Slug) {
		return Intent{}, fmt.Errorf("intent: slug %q is not kebab-case", opts.Slug)
	}
	if strings.TrimSpace(opts.Title) == "" {
		return Intent{}, fmt.Errorf("intent: refusing to create a draft with an empty title")
	}
	if strings.TrimSpace(opts.SeedBody) == "" && strings.TrimSpace(opts.PressRelease) == "" {
		return Intent{}, fmt.Errorf("intent: refusing to create a draft with neither a press release nor a seed body")
	}
	if opts.PromotedFrom != "" && !promotedFromRe.MatchString(opts.PromotedFrom) {
		return Intent{}, fmt.Errorf("intent: promoted_from %q must match ^(iss|rdi)-[0-9]+$", opts.PromotedFrom)
	}
	// impact is optional on a draft (intent_impact_valid gates the move into
	// shipped/, not the seed), but when set it must be a legal, non-internal
	// judgement — the same bar the gate applies at shipped/ — so the value the
	// tool stamps travels unchanged to shipped/ and passes the blocker there. An
	// invalid or internal impact is refused up front, never absorbed.
	if opts.Impact != "" {
		imp, err := changelog.ParseImpact(opts.Impact)
		if err != nil {
			return Intent{}, fmt.Errorf("intent: %w", err)
		}
		if imp == changelog.ImpactInternal {
			return Intent{}, fmt.Errorf("intent: impact must not be internal on an intent — a press-release-first intent is user-facing by definition; declare one of additive|breaking|fix, or record the work as an issue instead")
		}
	}

	// The disclosure pair is validated HERE, at the one canonical draft-mint
	// primitive, so every caller stamps both keys and none of them can mint a
	// value outside the vocabulary. An unset origin means the default arrival
	// path; an unset mode means provenance.DefaultMode.
	stamp, err := draftStamp(opts)
	if err != nil {
		return Intent{}, fmt.Errorf("intent: %w", err)
	}

	// Boundary redaction (gh-486): CreateDraft is the ONE canonical draft-mint
	// primitive, so the "no unredacted secret/home-path in a committed intent"
	// invariant lives HERE, where every caller (quoted-text create AND
	// capture-promote) passes through it — never once per caller, where a future
	// caller would reopen the leak. Title and SeedBody are the two free-text
	// members; the structural fields (slug, ids, impact) are validated/enum-
	// constrained above and carry nothing to redact. Fail-closed on a degraded
	// scanner; the pass is idempotent for text a caller already redacted.
	rTitle, _, err := redactIntentText(repoRoot, opts.Title)
	if err != nil {
		return Intent{}, err
	}
	rPress, _, err := redactIntentText(repoRoot, opts.PressRelease)
	if err != nil {
		return Intent{}, err
	}
	rBody, _, err := redactIntentText(repoRoot, opts.SeedBody)
	if err != nil {
		return Intent{}, err
	}
	opts.Title = rTitle
	opts.PressRelease = rPress
	opts.SeedBody = rBody

	var created Intent
	err = withIntentMintLock(repoRoot, func() error {
		draftsDirAbs := filepath.Join(repoRoot, IntentsRelDir, BucketDrafts)
		if err := ensureRecordDir(repoRoot, filepath.Join(IntentsRelDir, BucketDrafts)); err != nil {
			return err
		}
		// Minted under the lock, so the presence check inside mintIntentID and
		// the write below are one critical section: no draft in this checkout can
		// take the id between the two. The id names no existing record in any
		// bucket, so the filename it forms is free by construction.
		id, err := mintIntentID(repoRoot)
		if err != nil {
			return err
		}
		name := id + "-" + opts.Slug + ".md"
		rel := filepath.Join(IntentsRelDir, BucketDrafts, name)
		abs := filepath.Join(draftsDirAbs, name)
		content := seedDraft(id, opts, stamp)
		if err := writeIntentFile(abs, rel, content); err != nil {
			return err
		}
		created = Intent{
			ID:           id,
			Slug:         opts.Slug,
			Kind:         "null",
			SpecID:       "null",
			Bucket:       BucketDrafts,
			Path:         rel,
			PromotedFrom: opts.PromotedFrom,
		}
		return nil
	})
	if err != nil {
		return Intent{}, err
	}
	return created, Validate(created)
}

// draftStamp resolves the draft's disclosure pair from the mint options: the
// arrival path, the reading pointer when the path is a reading contribution, and
// the production mode.
//
// The reading kind has one extra bar, and it is here rather than in the
// provenance leaf because only this primitive holds both halves of the join: the
// origin's item and the promoted_from back-edge are ONE join written twice, so a
// draft carrying them in disagreement is a state no command produced. It is
// refused before anything is written rather than reconciled by picking one.
func draftStamp(opts DraftOptions) (provenance.Stamp, error) {
	switch opts.Origin.Kind {
	case "":
		return provenance.NewStamp(provenance.KindResearcherAuthored, opts.ProductionMode)
	case provenance.KindContributedByReading:
		if opts.PromotedFrom != opts.Origin.Item {
			return provenance.Stamp{}, fmt.Errorf(
				"a reading origin names item %s but the back-edge names %q; the two are one join",
				opts.Origin.Item, opts.PromotedFrom)
		}
		return provenance.NewReadingStamp(opts.Origin.Run, opts.Origin.Item, opts.ProductionMode)
	default:
		return provenance.NewStamp(opts.Origin.Kind, opts.ProductionMode)
	}
}

// deriveIntentSlug lowercases the text, collapses non-[a-z0-9] runs to a single
// hyphen, trims leading/trailing hyphens, caps the length, and insists the result
// is kebab-case and non-empty — the slug becomes a filename, so it is validated
// before any path is built (path-traversal / filename-safety defence).
func deriveIntentSlug(text string) (string, error) {
	lowered := strings.ToLower(text)
	collapsed := strings.Trim(slugNonAlnumRe.ReplaceAllString(lowered, "-"), "-")
	if len(collapsed) > maxSlugLen {
		collapsed = strings.Trim(collapsed[:maxSlugLen], "-")
	}
	if collapsed == "" {
		return "", fmt.Errorf("intent: text %q has no slug-able characters", text)
	}
	if !slugRe.MatchString(collapsed) {
		return "", fmt.Errorf("intent: derived slug %q is not kebab-case", collapsed)
	}
	return collapsed, nil
}

var slugNonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// mintIntentID draws a native itd id that names no record in any bucket of this
// checkout. It reads no maximum (adr-45 ruling 2): the clock orders the ids and
// the entropy separates two minters in the same second, so a sibling checkout —
// which no lock here can see — needs no coordination to stay distinct. A
// candidate already present is the same-second, same-suffix coincidence inside
// one checkout, and it is redrawn, never bumped: a bump would re-derive the next
// id from the store's occupancy, a miniature maximum-plus-one (spc-33 ruling 2).
// Called under the mint lock so the check and the caller's write are atomic
// within the checkout.
func mintIntentID(repoRoot string) (string, error) {
	for attempt := 0; attempt < mintRetryBudget; attempt++ {
		id, err := minter.Mint(intentFamily)
		if err != nil {
			return "", err
		}
		if !intentPresent(repoRoot, id) {
			return id, nil
		}
	}
	return "", fmt.Errorf("intent: could not mint a free itd id after %d draws", mintRetryBudget)
}

// intentPresent reports whether id names a file in any intent bucket, judged by
// the one canonical filename grammar (recordid.FilenameNumRe). It walks Buckets
// rather than a literal, so a bucket this scan does not visit is not one the
// mint could re-issue an id into.
func intentPresent(repoRoot, id string) bool {
	for _, bucket := range Buckets {
		entries, err := os.ReadDir(filepath.Join(repoRoot, IntentsRelDir, bucket))
		if err != nil {
			continue // absent bucket is soft
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			m := intentFileNumRe.FindStringSubmatch(e.Name())
			if m != nil && intentFamily+"-"+m[1] == id {
				return true
			}
		}
	}
	return false
}

// intentFileNumRe is the intent store's filename grammar, shared with the
// read-side resolver and record-lint so the presence check judges exactly the
// files those two resolve.
var intentFileNumRe = recordid.FilenameNumRe(intentFamily)

// seedDraft renders the canonical draft skeleton: the full draft frontmatter set
// (id, slug, spec_id: null, kind: null, suggested_kind: null,
// reclassification_history: [], builds_on: [], severity: minor, plus the
// promoted_from back-edge when the draft graduated from an issue, plus the
// origin/production_mode disclosure pair) and an honest, minimal body: the
// Press Release prose or the route's seed note, the Why This Matters seed body
// or its prompt, and the itd-1 discipline's Acceptance Criteria section left as
// a placeholder for the human to fill before planning.
func seedDraft(id string, opts DraftOptions, stamp provenance.Stamp) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("id: " + id + "\n")
	b.WriteString("slug: " + opts.Slug + "\n")
	b.WriteString("spec_id: null\n")
	b.WriteString("kind: null\n")
	b.WriteString("suggested_kind: null\n")
	b.WriteString("reclassification_history: []\n")
	b.WriteString("builds_on: []\n")
	b.WriteString("severity: minor\n")
	// The promote back-edge (spc-24): bare, like every id field in this store.
	// Absent on a quoted-text draft — the field exists only when the draft
	// graduated from an issue.
	if opts.PromotedFrom != "" {
		b.WriteString("promoted_from: " + opts.PromotedFrom + "\n")
	}
	// impact is written only when the caller declared one (validated in
	// CreateDraft). It is bare — the machine-read enum the shipped-intent gate
	// compares byte-for-byte — and travels unchanged to shipped/. An unset impact
	// writes no line: a draft is "not judged yet", exactly like the null fields.
	if opts.Impact != "" {
		b.WriteString("impact: " + opts.Impact + "\n")
	}
	// The disclosure pair (itd-178), bare like every other machine-read scalar in
	// this block. Both are written together — a lone key is a state no write path
	// produces, which is what makes the lint able to see a hand edit.
	b.WriteString(provenance.KeyOrigin + ": " + stamp.OriginValue() + "\n")
	b.WriteString(provenance.KeyProductionMode + ": " + stamp.ModeValue() + "\n")
	b.WriteString("---\n\n")
	b.WriteString("# " + opts.Title + "\n\n")
	b.WriteString("## Press Release\n\n")
	if strings.TrimSpace(opts.PressRelease) != "" {
		// Prose, in the blockquote every written press release in the record
		// uses — never the seed note, which would make a press release the
		// caller wrote read as a placeholder nobody filled.
		b.WriteString(blockquote(opts.PressRelease) + "\n\n")
	} else {
		b.WriteString("> " + seedNote(opts) + "\n\n")
	}
	b.WriteString("## Why This Matters\n\n")
	if strings.TrimSpace(opts.SeedBody) != "" {
		b.WriteString(opts.SeedBody + "\n\n")
	} else {
		b.WriteString(WhyThisMattersPrompt + "\n\n")
	}
	// The two claim sections the claim-recording gradient prompts for, each with
	// its one-line contract. They sit above the criteria, matching the record
	// template, and they arrive as a PROMPT: a seeded nullity token would record a
	// decline nobody made, which is precisely the collapse the gradient forbids.
	b.WriteString("## Mechanism\n\n")
	b.WriteString(MechanismPrompt + "\n\n")
	b.WriteString("## Scope Conditions\n\n")
	b.WriteString(ScopeConditionsPrompt + "\n\n")
	b.WriteString("## Acceptance Criteria\n\n")
	b.WriteString("> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for \"shipped\" before this draft can be planned._\n\n")
	b.WriteString("## Open Questions\n\n")
	b.WriteString("_None recorded yet._\n\n")
	b.WriteString("## Audit Notes\n\n")
	b.WriteString("_Empty. Populated by intent-auditor when intent moves to shipped/._\n")
	return b.String()
}

// blockquote renders prose as a Markdown blockquote, one `> ` per line so a
// multi-paragraph text stays one quote.
func blockquote(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			lines[i] = ">"
			continue
		}
		lines[i] = "> " + line
	}
	return strings.Join(lines, "\n")
}

// WhyThisMattersPrompt is what `## Why This Matters` carries on a draft whose
// route seeded no body for it — the quoted-text route, whose text is the press
// release and is not pasted here a second time. Same register as the claim
// prompts below: a contract the human replaces, not content anybody wrote. No
// gate reads the section today, so nothing needs to tell the prompt from prose;
// a reader that comes to need that compares against this constant, for the
// reason IsClaimPrompt gives.
const WhyThisMattersPrompt = "> _Why this matters to the user — replace before planning._"

// The two claim-section prompts this package seeds. They are the contract a
// human replaces, not a claim a human made, so a gate that reads one has to be
// able to say which it is looking at.
const (
	MechanismPrompt       = "> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable \"we expect X because Y\" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._"
	ScopeConditionsPrompt = "> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._"
)

// IsClaimPrompt reports whether a claim section's body is still one of the
// prompts above, rather than something somebody wrote.
//
// It lives HERE, with the templates, for the reason IsSeedNote does: a consumer
// comparing against its own copy of the sentence would keep matching the old
// wording the moment these are reworded, and the symptom would be an unanswered
// prompt reported to a reader as a recorded claim.
func IsClaimPrompt(body string) bool {
	trimmed := strings.TrimSpace(body)
	return trimmed == MechanismPrompt || trimmed == ScopeConditionsPrompt
}

// The Press Release placeholders this package mints, in their parts: a per-path
// opening clause and the instruction all of them close with. The capture form
// is no longer minted — the quoted-text route seeds the section with its text —
// but drafts already in the record carry it, so it stays recognized.
const (
	captureSeedOpening   = "Seeded from a quoted-text intent capture."
	promotionSeedOpening = "Seeded by promotion from "
	seedNoteTail         = "Expand into the full press-release narrative before planning."
	// readingSeedSource is what the reading route names in place of the item id.
	// The Press Release is the FIRST field the intent projection carries, and a
	// draft is admitted at the entailment position — so an rdi-N here would put a
	// prior reading's output inside the object of the next one, which the readings
	// companion forbids (companion 8.3). The join lives in `origin` and
	// `promoted_from`, two frontmatter keys no projection names.
	readingSeedSource = "a reading item"
)

// CaptureSeedNote is the placeholder a quoted-text capture once minted, whole:
// recognized on the drafts that carry it, written by no route today.
const CaptureSeedNote = captureSeedOpening + " " + seedNoteTail

// IsSeedNote reports whether a press-release body is still one of the templates
// this package writes, rather than something somebody wrote.
//
// It lives HERE, with the templates, because it is the only place that can stay
// true: a consumer comparing against its own copy of the sentence would keep
// matching the old wording the moment these are reworded, and the symptom would
// be a placeholder quoted at a reader as a testimonial. Both minted forms are
// covered and nothing else is — the promotion form's source id is the only part
// allowed to vary, and a body with a single written sentence in it fails.
//
// The caller passes the body already reduced to its words: quote markers,
// emphasis markers and whitespace runs removed.
func IsSeedNote(text string) bool {
	if text == CaptureSeedNote {
		return true
	}
	return strings.HasPrefix(text, promotionSeedOpening) && strings.HasSuffix(text, seedNoteTail)
}

// seedNote is the standard Press Release placeholder for a draft whose route
// supplied no press-release prose, honest about which create path seeded it.
//
// The reading route names the KIND of source rather than the source: same
// opening, same tail, so IsSeedNote goes on matching both forms by prefix and
// suffix, and the issue route keeps its iss-N — an issue is something a person
// noticed, not a reading's output.
func seedNote(opts DraftOptions) string {
	if opts.Origin.Kind == provenance.KindContributedByReading {
		return "_" + promotionSeedOpening + readingSeedSource + ". " + seedNoteTail + "_"
	}
	if opts.PromotedFrom != "" {
		return "_" + promotionSeedOpening + opts.PromotedFrom + ". " + seedNoteTail + "_"
	}
	return "_" + CaptureSeedNote + "_"
}

// titleLine collapses internal whitespace and trims a text into a single
// heading line (a multi-word free-text line becomes one clean title).
func titleLine(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

// withIntentMintLock runs fn while holding an exclusive advisory lock over the
// intent store. It serializes the presence check and the write of one mint
// against concurrent abcd processes in the SAME checkout (two agent sessions, a
// hook firing beside a manual command), which is the one clash — same second,
// same suffix, one directory — that time and entropy leave to the store to
// arbitrate (spc-33 ruling 2). It cannot see a sibling checkout and does not
// need to: the mint reads no maximum, so two checkouts never share the state a
// lock would have to protect. It flocks the intents/ directory file descriptor
// itself, so no lock artifact is left in the committed record tree (mirroring
// the spec store's mint lock). O_NOFOLLOW refuses a symlinked intents/.
func withIntentMintLock(repoRoot string, fn func() error) error {
	intentsDir := filepath.Join(repoRoot, IntentsRelDir)
	if err := ensureRecordDir(repoRoot, IntentsRelDir); err != nil {
		return err
	}
	fd, err := syscall.Open(intentsDir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("intent: opening mint lock on %s: %w", IntentsRelDir, err)
	}
	defer syscall.Close(fd)

	deadline := time.Now().Add(mintLockTimeout)
	for {
		lockErr := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if lockErr == nil {
			break
		}
		if lockErr != syscall.EWOULDBLOCK {
			return fmt.Errorf("intent: acquiring mint lock: %w", lockErr)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("intent: could not acquire mint lock within %s", mintLockTimeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	return fn()
}
