// Package decide is the WRITE side of the decision record: `abcd decide
// "<title>"` mints an ADR id and lays the skeleton the store's existing files
// carry, under `.abcd/development/decisions/adrs/`.
//
// It exists because ADRs were the last record family allocating by hand. Two
// branches minted `0055` and `0056` on the same day for different decisions —
// the add/add collision the timestamp mint removed for captures, intents and
// specs — and the 2026-09-01 ruling (`.abcd/work/DECISIONS.md`, that date) took
// the family onto the same seam: an ADR filed from then on is
// `adr-<yymmddHHMMSS><rrrr>`, in a file the stamp orders. The hand-numbered
// ordinals `0001`–`0058` keep their ids and their filenames; nothing is
// renumbered, and every reader admits both vintages through one derivation
// (`recordid.CanonADRID` / `recordid.ADRFileID`).
//
// The boundary is the same one the intent and spec stores draw: the core owns
// the ID, the DATE, the FILENAME and the record's structure; a human owns every
// word of the decision. The skeleton this package writes states nothing — it
// carries the four sections the store's README specifies, each with the question
// it answers, and `status: proposed`, because a freshly minted file is a draft
// whose decision is not yet locked and only its author can say otherwise.
package decide

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// ADRsRelDir is the decision store, repo-relative and slash-separated.
const ADRsRelDir = ".abcd/development/decisions/adrs"

// adrFamily is the store's id prefix, the family tag the mint splices into every
// native adr id.
const adrFamily = "adr"

// minter is the ADR family's record-id mint seam (adr-45; per-family adoption as
// configuration, ruling 3). The zero value is the production configuration —
// real clock, crypto entropy; tests inject both so a same-instant case is
// deterministic.
var minter recordid.Minter

// mintRetryBudget bounds how many fresh ids one mint draws when a candidate
// already names a record in this checkout — the same-second, same-suffix
// coincidence, redrawn rather than bumped (spc-33 ruling 2). It mirrors the
// intent and spec stores' budget.
const mintRetryBudget = 8

// maxSlugLen caps the derived slug so a pathological title cannot produce an
// unwieldy filename. Mirrors the intent-side derivation budget.
const maxSlugLen = 60

// mintLockTimeout bounds how long Create waits for the store mint lock. A var
// (not const) so a test can shorten it to exercise contention.
var mintLockTimeout = 5 * time.Second

// slugNonAlnumRe collapses every run of non-slug characters to one hyphen.
var slugNonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

// slugRe is the kebab-case shape a derived slug must land in before it may
// become a filename.
var slugRe = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Decision is one minted decision record: what it is called, where it landed,
// and the id every citation of it resolves through.
type Decision struct {
	ID    string `json:"id"`
	Slug  string `json:"slug"`
	Title string `json:"title"`
	Date  string `json:"date"`
	Path  string `json:"path"`
}

// Create mints a decision record for title and writes its skeleton, returning
// what it minted. On any refusal nothing is written.
//
// The id is drawn through the shared record-id seam and reads no maximum (adr-45
// ruling 2): the clock orders the ids and the entropy separates two minters in
// the same second, so a sibling checkout — which no lock here can see — needs no
// coordination to stay distinct. That is the whole point of the change: the
// hand-numbered ordinal had exactly the cross-branch collision no convention can
// close.
func Create(repoRoot, title string) (Decision, error) {
	trimmed := strings.Join(strings.Fields(title), " ")
	if trimmed == "" {
		return Decision{}, fmt.Errorf("decide: refusing to mint a decision with an empty title")
	}
	// Redact the caller's text through the one canonical scanner BEFORE anything
	// derived from it is built. The slug becomes the committed filename and is
	// derived straight from this title, so a secret or a home path in the prose
	// would otherwise reach the path as well as the body — the leak the intent and
	// capture stores already close at their own mint.
	redacted, err := redactDecisionText(repoRoot, trimmed)
	if err != nil {
		return Decision{}, err
	}
	slug, err := deriveSlug(redacted)
	if err != nil {
		return Decision{}, err
	}

	var created Decision
	err = withMintLock(repoRoot, func() error {
		// Minted under the lock, so the presence check and the write below are one
		// critical section: no decision in this checkout can take the id between
		// the two, and the filename the id forms is free by construction.
		id, err := mintADRID(repoRoot)
		if err != nil {
			return err
		}
		stamp := strings.TrimPrefix(id, adrFamily+"-")
		name := stamp + "-" + slug + ".md"
		rel := filepath.Join(ADRsRelDir, name)
		created = Decision{
			ID:    id,
			Slug:  slug,
			Title: redacted,
			// The date comes off the minted stamp rather than a second clock
			// reading: one id, one instant, and a record whose frontmatter date can
			// never disagree with the file it is written in.
			Date: dateFromStamp(stamp),
			Path: rel,
		}
		body := renderSkeleton(created)
		if err := fsutil.WriteFileAtomic(filepath.Join(repoRoot, filepath.FromSlash(rel)), []byte(body), 0o644); err != nil {
			return fmt.Errorf("decide: writing %s: %w", rel, err)
		}
		return nil
	})
	if err != nil {
		return Decision{}, err
	}
	return created, nil
}

// deriveSlug lowercases the title, collapses non-[a-z0-9] runs to a single
// hyphen, trims the ends, caps the length, and insists the result is kebab-case
// and non-empty — the slug becomes a filename, so it is validated before any
// path is built.
func deriveSlug(title string) (string, error) {
	collapsed := strings.Trim(slugNonAlnumRe.ReplaceAllString(strings.ToLower(title), "-"), "-")
	if len(collapsed) > maxSlugLen {
		collapsed = strings.Trim(collapsed[:maxSlugLen], "-")
	}
	if collapsed == "" {
		return "", fmt.Errorf("decide: title %q has no slug-able characters", title)
	}
	if !slugRe.MatchString(collapsed) {
		return "", fmt.Errorf("decide: derived slug %q is not kebab-case", collapsed)
	}
	return collapsed, nil
}

// dateFromStamp renders the mint stamp's yymmdd head as the ISO date the record
// frontmatter carries. The stamp is UTC and fixed-width by construction, so the
// slice is total for every id the mint produces.
func dateFromStamp(stamp string) string {
	return "20" + stamp[0:2] + "-" + stamp[2:4] + "-" + stamp[4:6]
}

// mintADRID draws a native adr id that names no decision in this checkout. A
// candidate already present is the same-second, same-suffix coincidence inside
// one checkout, and it is redrawn, never bumped: a bump would re-derive the next
// id from the store's occupancy, a miniature maximum-plus-one (spc-33 ruling 2).
// Called under the mint lock so the check and the caller's write are atomic
// within the checkout.
func mintADRID(repoRoot string) (string, error) {
	for attempt := 0; attempt < mintRetryBudget; attempt++ {
		id, err := minter.Mint(adrFamily)
		if err != nil {
			return "", err
		}
		if !adrPresent(repoRoot, id) {
			return id, nil
		}
	}
	return "", fmt.Errorf("decide: could not mint a free adr id after %d draws", mintRetryBudget)
}

// adrPresent reports whether id names a file in the decision store, judged by
// the ONE shared filename derivation the read-side resolver uses. It sees both
// vintages — an ordinal file and a stamped one — because that derivation does,
// so the mint can never re-issue an id some citation already resolves.
func adrPresent(repoRoot, id string) bool {
	entries, err := os.ReadDir(filepath.Join(repoRoot, filepath.FromSlash(ADRsRelDir)))
	if err != nil {
		return false // absent store is soft; the mint creates it
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if recordid.ADRFileID(e.Name()) == id {
			return true
		}
	}
	return false
}

// renderSkeleton lays out the record shape every file in the store carries: the
// nine frontmatter keys, the `ADR-<id>: <Title>` H1, and the four body sections
// the store README specifies, each carrying the question it answers rather than
// an answer nobody has given yet.
func renderSkeleton(d Decision) string {
	var b strings.Builder
	b.WriteString("---\n")
	b.WriteString("id: " + d.ID + "\n")
	b.WriteString("slug: " + d.Slug + "\n")
	// proposed, not accepted: the binary knows an id and a date, and cannot know
	// that a decision is in force. The author sets `accepted` in the change that
	// states the decision.
	b.WriteString("status: proposed\n")
	b.WriteString("date: " + d.Date + "\n")
	b.WriteString("supersedes: null\n")
	b.WriteString("superseded_by: null\n")
	b.WriteString("related_intents: []\n")
	b.WriteString("related_rfcs: []\n")
	b.WriteString("related_adrs: []\n")
	b.WriteString("---\n\n")
	b.WriteString("# ADR-" + strings.TrimPrefix(d.ID, adrFamily+"-") + ": " + d.Title + "\n\n")
	b.WriteString("## Context\n\n")
	b.WriteString("_What forced the decision? What constraints were already locked?_\n\n")
	b.WriteString("## Decision\n\n")
	b.WriteString("_The decision as a positive declaration: \"We will X.\" Set `status: accepted` once it is in force._\n\n")
	b.WriteString("## Alternatives Considered\n\n")
	b.WriteString("_2–4 options laid out fairly, the chosen one among them; for each, why it was rejected or chosen._\n\n")
	b.WriteString("## Consequences\n\n")
	b.WriteString("_What follows — what is now easier, what is now harder, what new obligations this creates._\n")
	return b.String()
}

// redactDecisionText sanitises the caller's title through the ONE canonical
// detector — the same scanner the transcript store, the capture ledger, the
// intent store and the launch bundler use.
//
// It FAILS CLOSED on a degraded scanner, taking the intent store's stance rather
// than capture's redact-and-report one: an ADR is durable committed prose, and a
// broken detector must never let an unredacted secret reach it under a false
// "clean" signal. It reuses scanner.Redact, the single masking primitive, so no
// second scanner or masking rule is introduced.
func redactDecisionText(repoRoot, text string) (string, error) {
	sc, err := scanner.New(repoRoot)
	if err != nil {
		return "", fmt.Errorf("decide: refusing to persist text with an unavailable scanner: %w", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return "", fmt.Errorf("decide: refusing to persist text with a degraded scanner: %s", reason)
	}
	findings := sc.ScanText(text, "decide")
	if len(findings) == 0 {
		return text, nil
	}
	out, _ := scanner.Redact(text, findings)
	return out, nil
}

// withMintLock runs fn while holding an exclusive advisory lock over the
// decision store. It serialises the presence check and the write of one mint
// against concurrent abcd processes in the SAME checkout, which is the one clash
// — same second, same suffix, one directory — that time and entropy leave to the
// store to arbitrate (spc-33 ruling 2). It cannot see a sibling checkout and does
// not need to: the mint reads no maximum, so two checkouts never share the state
// a lock would have to protect. It flocks the store's own directory file
// descriptor, so no lock artefact is left in the committed record tree, and
// O_NOFOLLOW refuses a symlinked store.
func withMintLock(repoRoot string, fn func() error) error {
	dir := filepath.Join(repoRoot, filepath.FromSlash(ADRsRelDir))
	if di, err := os.Lstat(dir); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("decide: %s is a symlink (refusing to follow)", ADRsRelDir)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("decide: creating %s: %w", ADRsRelDir, err)
	}
	fd, err := syscall.Open(dir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("decide: opening mint lock on %s: %w", ADRsRelDir, err)
	}
	defer syscall.Close(fd)

	deadline := time.Now().Add(mintLockTimeout)
	for {
		lockErr := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if lockErr == nil {
			break
		}
		if lockErr != syscall.EWOULDBLOCK {
			return fmt.Errorf("decide: acquiring mint lock: %w", lockErr)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("decide: could not acquire mint lock within %s", mintLockTimeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	return fn()
}
