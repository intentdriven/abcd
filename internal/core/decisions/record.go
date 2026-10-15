package decisions

// record.go is the WRITE side: Record, the one writer of a decision record
// (spc-2610110931381316, step 2). The verb's front door and, from step 3,
// ideate both call it, so a decision reaches the folder through one primitive.
//
// Recording writes one new file and touches no other path. That is the whole of
// the merge claim (A1, A4): two branches that each record a decision each add a
// file of their own, and git has nothing to reconcile, with no merge attribute
// and no driver.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// SharedTierRel is the shared working tier the folder sits in. Record creates
// the folder inside it and never creates the tier itself: once `ahoy install`
// has laid the tier it is abcd's namespace, and outside one there is nothing to
// commit a decision to (spec decision 10).
const SharedTierRel = ".abcd/work"

// ErrNoSharedTier is Record's refusal when the repository has no shared tier.
var ErrNoSharedTier = errors.New("this repository has no shared working tier (" + SharedTierRel +
	"/), so a decision has nowhere to be committed; run `abcd ahoy install` to lay it, then record the decision again")

// ErrNoSlug is Record's refusal for a text with nothing slug-able in it: the
// slug becomes the filename, so a text that yields none cannot be filed.
var ErrNoSlug = errors.New("the decision's text has no letters or digits to name its file by")

// minter is the dcn family's record-id mint seam (adr-45). The zero value is the
// production configuration; a test injects the clock and the entropy.
var minter recordid.Minter

// mintRetryBudget bounds how many fresh ids one record draws when a candidate
// already names a record in the folder: the same-second, same-suffix
// coincidence, redrawn rather than bumped, as the ADR store's mint does.
const mintRetryBudget = 8

// mintLockTimeout bounds how long Record waits for the folder's mint lock. A var
// so a test can shorten it.
var mintLockTimeout = 5 * time.Second

// Record writes text as a new decision record in the repository dir sits in,
// and returns it as the reader would read it. On any refusal nothing is
// written.
//
// It resolves the checkout root from dir and refuses outside a checkout,
// refuses a repository without the shared tier, and refuses a text with nothing
// slug-able in it. The text is collapsed to one line and passed through the
// canonical scanner, failing closed on a degraded one, because it becomes
// committed prose and, through the slug, a filename. The id comes from the adr-45
// mint, the date from the id's stamp, and the file is created exclusively under
// the folder's mint lock, so no path but the record's own is touched.
func Record(dir, text string) (Decision, error) {
	root, err := gitutil.CheckoutRoot(dir, "the decisions folder")
	if err != nil {
		return Decision{}, err
	}
	trimmed := strings.Join(strings.Fields(text), " ")
	if trimmed == "" {
		return Decision{}, errors.New("decisions: refusing to record a decision with no text")
	}
	redacted, err := redactText(root, trimmed)
	if err != nil {
		return Decision{}, err
	}
	slug := recordid.Slug(redacted, recordid.MaxSlugLen)
	if slug == "" {
		return Decision{}, fmt.Errorf("decisions: %w", ErrNoSlug)
	}
	if err := RequireSharedTier(root); err != nil {
		return Decision{}, err
	}
	if err := fsutil.EnsureRealDirAll(root, RelDir, 0o755); err != nil {
		return Decision{}, fmt.Errorf("decisions: creating %s: %w", RelDir, err)
	}

	var made Decision
	err = withMintLock(root, func() error {
		r, err := os.OpenRoot(root)
		if err != nil {
			return fmt.Errorf("decisions: opening the repository: %w", err)
		}
		defer r.Close()
		for attempt := 0; attempt < mintRetryBudget; attempt++ {
			id, err := minter.Mint(Family)
			if err != nil {
				return err
			}
			if present(root, id) {
				continue
			}
			stamp := strings.TrimPrefix(id, Family+"-")
			date := recordid.DateFromStamp(stamp)
			rel := RelDir + "/" + id + "-" + slug + ".md"
			body := date + " — " + redacted
			content := "---\nid: " + id + "\nslug: " + slug + "\ndate: " + date + "\n---\n" + body + "\n"
			if err := fsutil.CreateExclusiveIn(r, rel, []byte(content), 0o644); err != nil {
				if errors.Is(err, os.ErrExist) {
					continue
				}
				return fmt.Errorf("decisions: writing %s: %w", rel, err)
			}
			made = Decision{ID: id, Slug: slug, Date: date, Path: rel, Text: body}
			return nil
		}
		return fmt.Errorf("decisions: could not mint a free %s id after %d draws", Family, mintRetryBudget)
	})
	if err != nil {
		return Decision{}, err
	}
	return made, nil
}

// RequireSharedTier refuses a repository whose shared tier is absent, and one
// whose tier is not a real directory: a link there would carry the record out of
// the checkout. Record runs it first; a writer that must know before its own
// write whether a decision can land after it (ideate's verdict, whose pointer is
// a decision) runs it too, so the refusal comes before anything is written.
func RequireSharedTier(root string) error {
	abs := filepath.Join(root, filepath.FromSlash(SharedTierRel))
	fi, err := os.Lstat(abs)
	switch {
	case errors.Is(err, os.ErrNotExist) || errors.Is(err, os.ErrInvalid):
		return fmt.Errorf("decisions: %w", ErrNoSharedTier)
	case err != nil:
		return fmt.Errorf("decisions: reading %s: %w", SharedTierRel, err)
	case fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir():
		return fmt.Errorf("decisions: %s is not a directory of this checkout, so a decision is not written through it", SharedTierRel)
	}
	return nil
}

// present reports whether id already names a record in the folder.
func present(root, id string) bool {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(RelDir)))
	if err != nil {
		return false
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), id+"-") {
			return true
		}
	}
	return false
}

// withMintLock runs fn holding an exclusive advisory lock on the folder itself,
// which serialises the presence check and the write against another abcd
// process in the same checkout. The lock is on the directory, so it leaves no
// artefact: the record is the only path a recording touches.
func withMintLock(root string, fn func() error) error {
	dir := filepath.Join(root, filepath.FromSlash(RelDir))
	ran := false
	err := fsutil.WithDirLock(dir, mintLockTimeout, func() error {
		ran = true
		return fn()
	})
	switch {
	case ran || err == nil:
		return err
	case errors.Is(err, fsutil.ErrLockContention):
		return fmt.Errorf("decisions: could not acquire the mint lock on %s within %s", RelDir, mintLockTimeout)
	}
	return fmt.Errorf("decisions: opening the mint lock on %s: %w", RelDir, err)
}

// redactText passes the text through the one canonical scanner before anything
// is derived from it, failing closed when the scanner is unavailable or
// degraded: a decision is committed prose, and its slug is a filename.
func redactText(root, text string) (string, error) {
	sc, err := scanner.New(root)
	if err != nil {
		return "", fmt.Errorf("decisions: refusing to record text with an unavailable scanner: %w", err)
	}
	if unavail, reason := sc.Unavailable(); unavail {
		return "", fmt.Errorf("decisions: refusing to record text with a degraded scanner: %s", reason)
	}
	findings := sc.ScanText(text, "decide")
	if unavail, reason := sc.Unavailable(); unavail {
		return "", fmt.Errorf("decisions: refusing to record text with a degraded scanner: %s", reason)
	}
	if len(findings) == 0 {
		return text, nil
	}
	out, _ := scanner.Redact(text, findings)
	return out, nil
}

// frozenLogRel is a managed repository's frozen single log, kept as earlier
// history and never written (spec decision 7). The listing names it where it
// exists. Step 4 introduces the package's exported name for this path.
const frozenLogRel = SharedTierRel + "/DECISIONS.md"

// Listing is the folder read for the verb's bare form: every record newest
// first, the files the reader skipped, and whether a frozen log sits beside
// the folder.
type Listing struct {
	Decisions []Decision
	Skipped   []Skipped
	// Frozen is the frozen log's repo-relative path where one exists, and
	// empty otherwise.
	Frozen string
}

// ReadListing reads the listing for the repository dir sits in. It resolves the
// checkout root and refuses outside a checkout; it reads the folder and stats
// the frozen log, and writes nothing.
func ReadListing(dir string) (Listing, error) {
	root, err := gitutil.CheckoutRoot(dir, "the decisions folder")
	if err != nil {
		return Listing{}, err
	}
	ds, skipped, err := List(root)
	if err != nil {
		return Listing{}, err
	}
	// The record can hold years of decisions and the recent ones are the ones a
	// reader comes for, so the listing runs newest first (the product thinker's
	// ruling of 2026-10-15). List keeps id order for every other reader.
	for i, j := 0, len(ds)-1; i < j; i, j = i+1, j-1 {
		ds[i], ds[j] = ds[j], ds[i]
	}
	l := Listing{Decisions: ds, Skipped: skipped}
	if fi, err := os.Lstat(filepath.Join(root, filepath.FromSlash(frozenLogRel))); err == nil && fi.Mode().IsRegular() {
		l.Frozen = frozenLogRel
	}
	return l, nil
}

// Entries is the listing's JSON form: every record, newest first, as
// {id, date, slug, path, text}. An empty folder is an empty array, never null.
func (l Listing) Entries() []Decision {
	if l.Decisions == nil {
		return []Decision{}
	}
	return l.Decisions
}

// Text is the listing's human form: one block per record, the id and then the
// body, newest first; a line naming the frozen log as earlier history where one
// exists. A folder with no records says so and names how to record one.
func (l Listing) Text() string {
	var b strings.Builder
	if len(l.Decisions) == 0 {
		fmt.Fprintf(&b, "no decisions recorded in %s/ — record one with abcd decide \"<text>\"\n", RelDir)
	}
	for i, d := range l.Decisions {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(d.ID + "\n")
		b.WriteString(d.Text + "\n")
	}
	if l.Frozen != "" {
		fmt.Fprintf(&b, "\nearlier history: %s, a frozen log this listing does not list\n", l.Frozen)
	}
	return b.String()
}
