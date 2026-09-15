package spec

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// mintLockTimeout bounds how long Create waits for the spec-store mint lock. A
// var (not const) so a test can shorten it to exercise contention.
var mintLockTimeout = 5 * time.Second

// specFamily is the spec store's id prefix, the family tag the mint splices
// into every native spc id.
const specFamily = "spc"

// minter is the spec family's record-id mint seam (adr-45; per-family adoption
// as configuration, ruling 3). The zero value is the production configuration —
// real clock, crypto entropy; tests inject both so a same-instant case is
// deterministic.
var minter recordid.Minter

// mintRetryBudget bounds how many fresh ids one spec mint draws when a candidate
// already names a spec in this checkout — the same-second, same-suffix
// coincidence, which is redrawn rather than bumped (spc-33 ruling 2). It mirrors
// the capture ledger's placeholder retry budget.
const mintRetryBudget = 8

// Load discovers spec files under both buckets, parses their frontmatter, and
// returns the in-memory Store. A missing specs/ directory yields an empty store
// (soft, mirroring lint's missing-dir behaviour). A present-but-malformed spec
// file is a hard, loud error.
func Load(repoRoot string) (Store, error) {
	// Seed Specs non-nil so an empty store marshals as [] in --json, not bare
	// null (every --json collection is an empty list, never null).
	store := Store{Specs: []Spec{}}
	for _, bucket := range []string{StatusOpen, StatusClosed} {
		specs, err := loadBucket(repoRoot, bucket)
		if err != nil {
			return Store{}, err
		}
		store.Specs = append(store.Specs, specs...)
	}
	return store, nil
}

// loadBucket reads one bucket directory. A missing directory is soft (nil, nil).
func loadBucket(repoRoot, bucket string) ([]Spec, error) {
	dir := filepath.Join(repoRoot, SpecsRelDir, bucket)
	di, err := os.Lstat(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("spec: stat %s: %w", filepath.Join(SpecsRelDir, bucket), err)
	}
	if di.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("spec: %s is a symlink (refusing to follow)", filepath.Join(SpecsRelDir, bucket))
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("spec: reading %s: %w", filepath.Join(SpecsRelDir, bucket), err)
	}
	var specs []Spec
	for _, e := range entries {
		if e.IsDir() || !specFileRe.MatchString(e.Name()) {
			continue
		}
		rel := filepath.Join(SpecsRelDir, bucket, e.Name())
		data, err := readRepoFile(filepath.Join(dir, e.Name()), rel)
		if err != nil {
			return nil, err
		}
		sp, err := parseSpec(rel, string(data), bucket)
		if err != nil {
			return nil, err
		}
		specs = append(specs, sp)
	}
	return specs, nil
}

// parseSpec builds a Spec from a file's content and validates it. A file whose
// frontmatter lacks a well-formed id or intent is malformed and rejected.
func parseSpec(relPath, content, bucket string) (Spec, error) {
	fields := frontmatter.Fields(strings.Split(content, "\n"))
	// A YAML null (`id: NULL`, `~`, `null`, …) is an UNSET field, not a malformed
	// value. Without this normalisation Validate saw the literal "NULL" and quoted
	// it back as a bad id — a "malformed shape" diagnosis — while record-lint, which
	// gates on frontmatter.IsNull first, called the same field unset. Two diagnoses
	// for one field. Routing every null spelling to the empty string here makes the
	// two gates agree, the iss-286 direction (iss-2608270908332975).
	sp := Spec{
		ID:     nullToUnset(fields["id"].Value),
		Slug:   nullToUnset(fields["slug"].Value),
		Intent: nullToUnset(fields["intent"].Value),
		Status: bucket,
		Path:   relPath,
	}
	if err := Validate(sp); err != nil {
		return Spec{}, fmt.Errorf("spec: malformed %s: %w", relPath, err)
	}
	return sp, nil
}

// nullToUnset maps a YAML null scalar to the empty (unset) string via the one
// canonical null predicate, so a null frontmatter field reads as unset rather
// than as a malformed literal value. The empty string is itself null, so a
// genuinely absent field passes through unchanged.
func nullToUnset(v string) string {
	if frontmatter.IsNull(v) {
		return ""
	}
	return v
}

// Create mints a native timestamp-numeric spc id through the shared recordid
// seam and writes specs/open/spc-N-<slug>.md with the intent link and the
// origin/production_mode disclosure pair in frontmatter. Both the intent id and
// the slug are validated before any path is built (the slug becomes a
// filename), as is the production mode — a spec is minted by a verb a person
// invoked, so its arrival path is researcher-authored and is derived here rather
// than asked for. An empty mode takes the vocabulary's default. The write is
// atomic.
func Create(repoRoot, intentID, slug, productionMode string) (Spec, error) {
	if !recordid.ValidIntentID(intentID) {
		return Spec{}, fmt.Errorf("spec: intent id %q must match ^itd-[0-9]+$", intentID)
	}
	if !slugRe.MatchString(slug) {
		return Spec{}, fmt.Errorf("spec: slug %q must be kebab-case", slug)
	}
	stamp, err := provenance.NewStamp(provenance.KindResearcherAuthored, productionMode)
	if err != nil {
		return Spec{}, fmt.Errorf("spec: %w", err)
	}
	// Mint and write under the exclusive mint lock: the presence check inside
	// mintSpecID and the write of spc-N-<slug>.md are one critical section, so
	// two concurrent plans in this checkout that draw the same id — the
	// same-second, same-suffix coincidence — cannot both write it. The filenames
	// differ by slug, so neither the atomic write nor a clobber guard would
	// notice on its own.
	var sp Spec
	err = withMintLock(repoRoot, func() error {
		store, err := Load(repoRoot)
		if err != nil {
			return err
		}
		id, err := mintSpecID(store)
		if err != nil {
			return err
		}
		openDir := filepath.Join(repoRoot, SpecsRelDir, StatusOpen)
		if err := ensureDir(openDir, filepath.Join(SpecsRelDir, StatusOpen)); err != nil {
			return err
		}
		name := fmt.Sprintf("%s-%s.md", id, slug)
		// 0o644 matches the intent-side markdown writer — both write committed design-record files.
		if err := fsutil.WriteFileAtomic(filepath.Join(openDir, name), []byte(renderSpec(id, slug, intentID, stamp)), 0o644); err != nil {
			return fmt.Errorf("spec: writing %s: %w", filepath.Join(SpecsRelDir, StatusOpen, name), err)
		}
		sp = Spec{
			ID:     id,
			Slug:   slug,
			Intent: intentID,
			Status: StatusOpen,
			Path:   filepath.Join(SpecsRelDir, StatusOpen, name),
		}
		return nil
	})
	if err != nil {
		return Spec{}, err
	}
	return sp, Validate(sp)
}

// mintSpecID draws a native spc id that names no spec in the loaded store. It
// reads no maximum (adr-45 ruling 2) — not the store's, not the intents'
// spec_id reservations, not the refs' — so a sibling checkout, which no lock
// here can see, needs no coordination to stay distinct: the clock orders the
// ids and the entropy separates two minters in the same second. A candidate
// already present is redrawn, never bumped: a bump would re-derive the next id
// from the store's occupancy, a miniature maximum-plus-one (spc-33 ruling 2).
// Called under the mint lock so the check and the caller's write are atomic
// within the checkout.
func mintSpecID(store Store) (string, error) {
	for attempt := 0; attempt < mintRetryBudget; attempt++ {
		id, err := minter.Mint(specFamily)
		if err != nil {
			return "", err
		}
		if _, present := store.Lookup(id); !present {
			return id, nil
		}
	}
	return "", fmt.Errorf("spec: could not mint a free spc id after %d draws", mintRetryBudget)
}

// withMintLock runs fn while holding an exclusive advisory lock over the spec
// store. It serializes the presence check and the write of one mint against
// concurrent abcd processes in the SAME checkout (two agent sessions, or a hook
// firing beside a manual command), which is the one clash — same second, same
// suffix, one directory — that time and entropy leave to the store to arbitrate
// (spc-33 ruling 2). It cannot see a sibling checkout and does not need to: the
// mint reads no maximum, so two checkouts never share the state a lock would
// have to protect. It flocks the specs/ directory file descriptor itself, so no
// lock artifact is left in the committed record tree. O_NOFOLLOW refuses a
// symlinked specs/.
func withMintLock(repoRoot string, fn func() error) error {
	specsDir := filepath.Join(repoRoot, SpecsRelDir)
	if err := ensureDir(specsDir, SpecsRelDir); err != nil {
		return err
	}
	fd, err := syscall.Open(specsDir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if err != nil {
		return fmt.Errorf("spec: opening mint lock on %s: %w", SpecsRelDir, err)
	}
	defer syscall.Close(fd)

	deadline := time.Now().Add(mintLockTimeout)
	for {
		lockErr := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if lockErr == nil {
			break
		}
		if lockErr != syscall.EWOULDBLOCK {
			return fmt.Errorf("spec: acquiring mint lock: %w", lockErr)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("spec: could not acquire mint lock within %s", mintLockTimeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	return fn()
}

// Close moves a spec file open/ -> closed/ via os.Rename (atomic on one
// filesystem) and returns the updated Spec. It fails closed if the spec is
// missing or already closed. The linked intent is deliberately left untouched:
// moving it is a later reconcile concern that consumes Spec.Intent.
func Close(repoRoot, specID string) (Spec, error) {
	if !recordid.ValidSpecID(specID) {
		return Spec{}, fmt.Errorf("spec: id %q must match ^spc-[0-9]+$", specID)
	}
	store, err := Load(repoRoot)
	if err != nil {
		return Spec{}, err
	}
	sp, ok := store.Lookup(specID)
	if !ok {
		return Spec{}, fmt.Errorf("spec: %s not found", specID)
	}
	if sp.Status == StatusClosed {
		return Spec{}, fmt.Errorf("spec: %s is already closed", specID)
	}
	name := filepath.Base(sp.Path)
	closedDir := filepath.Join(repoRoot, SpecsRelDir, StatusClosed)
	if err := ensureDir(closedDir, filepath.Join(SpecsRelDir, StatusClosed)); err != nil {
		return Spec{}, err
	}
	dstRel := filepath.Join(SpecsRelDir, StatusClosed, name)
	// Best-effort clobber guard: os.Rename would silently overwrite the destination,
	// so refuse when it already exists. This Lstat→Rename check is racy against a
	// file appearing in the window — accepted under the trusted-worktree model (only
	// the developer/agent mutates the store; there is no concurrent adversary), where
	// the atomic same-filesystem rename is preferred over a non-atomic no-clobber
	// link+remove that a crash could leave half-done.
	if _, err := os.Lstat(filepath.Join(closedDir, name)); err == nil {
		return Spec{}, fmt.Errorf("spec: refusing to overwrite existing %s", dstRel)
	}
	if err := os.Rename(filepath.Join(repoRoot, sp.Path), filepath.Join(closedDir, name)); err != nil {
		return Spec{}, fmt.Errorf("spec: closing %s: %w", specID, err)
	}
	sp.Status = StatusClosed
	sp.Path = filepath.Join(SpecsRelDir, StatusClosed, name)
	return sp, nil
}

// readRepoFile reads a repo file behind the trust-boundary guards. It opens ONCE
// with O_NOFOLLOW (refuse a symlinked leaf) and O_NONBLOCK (a FIFO/device leaf
// returns immediately instead of blocking the open), then validates the SAME file
// descriptor (regular file, size cap) before reading — so a symlink swap between
// stat and read cannot redirect it.
func readRepoFile(abs, rel string) ([]byte, error) {
	f, err := os.OpenFile(abs, os.O_RDONLY|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if err != nil {
		if errors.Is(err, syscall.ELOOP) {
			return nil, fmt.Errorf("spec: %s is a symlink (refusing to follow)", rel)
		}
		return nil, fmt.Errorf("spec: opening %s: %w", rel, err)
	}
	defer f.Close()
	fi, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("spec: stat %s: %w", rel, err)
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("spec: %s is not a regular file", rel)
	}
	if fi.Size() > maxSpecFileBytes {
		return nil, fmt.Errorf("spec: %s exceeds the %d-byte cap", rel, maxSpecFileBytes)
	}
	data, err := io.ReadAll(io.LimitReader(f, maxSpecFileBytes+1))
	if err != nil {
		return nil, fmt.Errorf("spec: reading %s: %w", rel, err)
	}
	if int64(len(data)) > maxSpecFileBytes {
		return nil, fmt.Errorf("spec: %s exceeds the %d-byte cap", rel, maxSpecFileBytes)
	}
	return data, nil
}

// ensureDir creates dir if absent, refusing a symlinked leaf directory.
// NOTE: a symlinked ANCESTOR (e.g. a symlinked specs/) is not caught here — a
// low-severity follow-up under the trusted-worktree model (planting one needs
// write access equal to editing the record directly).
func ensureDir(dir, rel string) error {
	if di, err := os.Lstat(dir); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("spec: %s is a symlink (refusing to follow)", rel)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("spec: creating %s: %w", rel, err)
	}
	return nil
}
