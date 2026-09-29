package spec

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/provenance"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// mintLockTimeout bounds how long a spec writer waits for the spec store's
// lock (withStoreLock). A var (not const) so a test can shorten it to exercise
// contention.
var mintLockTimeout = 5 * time.Second

// beforeStoreLock, when set, runs as withStoreLock is entered, before the store
// is opened for its lock. It is a test seam: a test removes the store there to
// stand in for a concurrent deletion landing between a writer's decision to
// lock and the lock itself. Production never sets it.
var beforeStoreLock func()

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
		Bundle: nullToUnset(fields["bundle"].Value),
	}
	if f, ok := fields["intents"]; ok && !frontmatter.IsNull(f.Value) {
		sp.Intents = frontmatter.StringList(f.Value)
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
// filename), as is the production mode — a spec's text is written directly
// rather than derived from another record or a reading item, so its arrival
// path is researcher-authored (the route, not who ran the command) and is
// derived here rather than asked for. An empty mode takes the vocabulary's default. The write is
// atomic.
func Create(repoRoot, intentID, slug, productionMode string) (Spec, error) {
	return CreateWithSteps(repoRoot, intentID, slug, productionMode, nil)
}

// CreateWithSteps is Create with the minted spec's `## Steps` section seeded
// from steps — the unlanded steps a remainder carries forward from the spec it
// splits from (itd-2609212103565953). No steps seeds the empty placeholder.
func CreateWithSteps(repoRoot, intentID, slug, productionMode string, steps []Step) (Spec, error) {
	if !recordid.ValidIntentID(intentID) {
		return Spec{}, fmt.Errorf("spec: intent id %q must match ^itd-[0-9]+$", intentID)
	}
	return create(repoRoot, intentID, nil, "", slug, productionMode, steps)
}

// CreateBundle mints ONE spec realising every member of a bundle (itd-34): its
// `intent:` names the first member, its `intents:` list names all of them in
// order, and its `bundle:` carries the name the members' own `bundle:` fields
// hold, so the close can ship them together. The bundle name is the spec's slug
// too. A bundle has at least two members, each named once (canonically), and
// every id and the name are validated before the mint.
func CreateBundle(repoRoot string, members []string, bundle, productionMode string) (Spec, error) {
	if len(members) < 2 {
		return Spec{}, fmt.Errorf("spec: a bundle has at least two members (got %d)", len(members))
	}
	for i, m := range members {
		if !recordid.ValidIntentID(m) {
			return Spec{}, fmt.Errorf("spec: intent id %q must match ^itd-[0-9]+$", m)
		}
		for _, prev := range members[:i] {
			if recordid.SameID(prev, m) {
				return Spec{}, fmt.Errorf("spec: %s is named twice in one bundle", m)
			}
		}
	}
	return create(repoRoot, members[0], append([]string(nil), members...), bundle, bundle, productionMode, nil)
}

// create is the one mint-and-write both constructors share: intents and bundle
// are empty for an ordinary spec.
func create(repoRoot, intentID string, intents []string, bundle, slug, productionMode string, steps []Step) (Spec, error) {
	if !slugRe.MatchString(slug) {
		return Spec{}, fmt.Errorf("spec: slug %q must be kebab-case", slug)
	}
	stamp, err := provenance.NewStamp(provenance.KindResearcherAuthored, productionMode)
	if err != nil {
		return Spec{}, fmt.Errorf("spec: %w", err)
	}
	// Mint and write under the store's exclusive lock: the presence check inside
	// mintSpecID and the write of spc-N-<slug>.md are one critical section, so
	// two concurrent plans in this checkout that draw the same id — the
	// same-second, same-suffix coincidence — cannot both write it. The filenames
	// differ by slug, so neither the atomic write nor a clobber guard would
	// notice on its own.
	var sp Spec
	err = withStoreLock(repoRoot, mintLockTimeout, createStore, func() error {
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
		if err := fsutil.WriteFileAtomic(filepath.Join(openDir, name), []byte(renderSpecRecord(id, slug, intentID, intents, bundle, stamp, steps)), 0o644); err != nil {
			return fmt.Errorf("spec: writing %s: %w", filepath.Join(SpecsRelDir, StatusOpen, name), err)
		}
		sp = Spec{
			ID:      id,
			Slug:    slug,
			Intent:  intentID,
			Status:  StatusOpen,
			Path:    filepath.Join(SpecsRelDir, StatusOpen, name),
			Intents: intents,
			Bundle:  bundle,
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
// Called under the store's lock so the check and the caller's write are atomic
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

// ErrStoreLockBusy is the spec store's lock not granted within a writer's
// budget. The three-lock acquisition in the intent package retries on it, with
// every earlier lock released between attempts.
var ErrStoreLockBusy = errors.New("spec: could not acquire the spec store's lock")

// withStoreLock runs fn while holding the spec store's one lock, an exclusive
// advisory flock on the specs/ directory itself, so no lock artifact is left in
// the committed record tree; O_NOFOLLOW refuses a symlinked specs/. It creates
// the store when absent, which only the mint may do.
//
// Every writer of a spec record takes it (iss-2609262218309668): the mint
// (Create and its siblings), Close, Discard, and — through WithStoreLock, from
// the intent package's three-lock acquisition — every link repoint and the
// lifeboat embark, the writers outside this package that rewrite or create a
// spec. Unlocked, a close renaming a spec open/ -> closed/ while a repoint that
// had read it at open/ wrote it back there left one record in both status
// folders, and an edit landing between a repoint's read and its write was
// erased. The mint's own clash — two plans in this checkout drawing the same
// id in the same second (spc-33 ruling 2) — is one more thing it arbitrates.
// It cannot see a sibling checkout and does not need to: the mint reads no
// maximum, so two checkouts never share the state it protects.
//
// Lock order: the issue ledger's lock, THEN the intent store's, THEN this one.
// It is the innermost of the three. Plan mints its spec inside the intent
// store's lock, and the three-lock acquisition takes it last; close, discard
// and a remainder's mint take it alone. This package imports neither the
// ledger's package nor the intent store's, so nothing run under this lock can
// request an earlier one, and it may never be taken the other way round.
//
// It is NOT reentrant — a second flock on another descriptor in the same
// process blocks until the budget runs out — so a caller holding it must not
// call a writer of this package, every one of which takes it.
//
// mode says what an absent store means. createStore plants it, which only the
// mint and the three-lock acquisition's path may do. storeMustExist refuses
// with errStoreAbsent instead, and the open that takes the lock is the check:
// a writer with nothing to act on in an absent store (Close, Discard) decides
// on the store the lock actually holds, so a store removed after an earlier
// look is never re-planted empty (iss-2609262342345159).
func withStoreLock(repoRoot string, timeout time.Duration, mode storeLockMode, fn func() error) error {
	if beforeStoreLock != nil {
		beforeStoreLock()
	}
	specsDir := filepath.Join(repoRoot, SpecsRelDir)
	if mode == createStore {
		if err := ensureDir(specsDir, SpecsRelDir); err != nil {
			return err
		}
	} else if di, err := os.Lstat(specsDir); err == nil && di.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("spec: %s is a symlink (refusing to follow)", SpecsRelDir)
	}
	fd, err := syscall.Open(specsDir, syscall.O_RDONLY|syscall.O_DIRECTORY|syscall.O_NOFOLLOW, 0)
	if mode == storeMustExist && errors.Is(err, syscall.ENOENT) {
		return errStoreAbsent
	}
	if err != nil {
		return fmt.Errorf("spec: opening the store lock on %s: %w", SpecsRelDir, err)
	}
	defer syscall.Close(fd)

	deadline := time.Now().Add(timeout)
	for {
		lockErr := syscall.Flock(fd, syscall.LOCK_EX|syscall.LOCK_NB)
		if lockErr == nil {
			break
		}
		if lockErr != syscall.EWOULDBLOCK {
			return fmt.Errorf("spec: acquiring the store lock: %w", lockErr)
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("%w within %s", ErrStoreLockBusy, timeout)
		}
		time.Sleep(10 * time.Millisecond)
	}
	defer syscall.Flock(fd, syscall.LOCK_UN)

	return fn()
}

// storeLockMode says what withStoreLock does with an absent store.
type storeLockMode int

const (
	// createStore plants an absent store and locks it.
	createStore storeLockMode = iota
	// storeMustExist refuses an absent store with errStoreAbsent and plants
	// nothing.
	storeMustExist
)

// errStoreAbsent is withStoreLock's refusal, in storeMustExist mode, of a tree
// whose spec store is absent when the lock is taken; fn has not run.
var errStoreAbsent = errors.New("spec: the spec store is absent")

// WithStoreLock runs fn holding the spec store's lock — the one every spec
// writer takes, not a second one — for a caller outside this package that
// writes a spec record. A caller that also writes ledger or intent records
// takes it through intent.WithLedgerThenMintLock, which takes the three in
// order and never holds an earlier lock while it waits for a later one.
//
// A tree with no spec store runs fn WITHOUT the lock: taking it creates the
// store, and a verb that writes no spec must not plant an empty one. With no
// store there is no spec record for fn to race.
func WithStoreLock(repoRoot string, fn func() error) error {
	return WithStoreLockWithin(repoRoot, mintLockTimeout, fn)
}

// WithStoreLockWithin is WithStoreLock with its own acquisition budget; a lock
// not granted within it is ErrStoreLockBusy, and fn has not run.
func WithStoreLockWithin(repoRoot string, timeout time.Duration, fn func() error) error {
	if !storeExists(repoRoot) {
		return fn()
	}
	return withStoreLock(repoRoot, timeout, createStore, fn)
}

// storeExists reports whether the tree has a spec store. Only an absent one
// reads as false: any other Lstat failure is left for the lock's own open to
// report.
func storeExists(repoRoot string) bool {
	_, err := os.Lstat(filepath.Join(repoRoot, SpecsRelDir))
	return !errors.Is(err, fs.ErrNotExist)
}

// Close moves a spec file open/ -> closed/ via os.Rename (atomic on one
// filesystem) and returns the updated Spec. It fails closed if the spec is
// missing or already closed. The linked intent is deliberately left untouched:
// moving it is a later reconcile concern that consumes Spec.Intent. The read
// and the rename are one critical section under the store's lock, so a writer
// holding it — a repoint between its read and its write — finishes before the
// spec moves. A tree with no spec store holds no spec to close, so the lock is
// taken in storeMustExist mode: the id is refused as not found and nothing is
// planted, even when the store is removed a moment before the lock.
func Close(repoRoot, specID string) (Spec, error) {
	if !recordid.ValidSpecID(specID) {
		return Spec{}, fmt.Errorf("spec: id %q must match ^spc-[0-9]+$", specID)
	}
	var sp Spec
	err := withStoreLock(repoRoot, mintLockTimeout, storeMustExist, func() error {
		var err error
		sp, err = closeLocked(repoRoot, specID)
		return err
	})
	if errors.Is(err, errStoreAbsent) {
		return Spec{}, fmt.Errorf("spec: %s not found", specID)
	}
	if err != nil {
		return Spec{}, err
	}
	return sp, nil
}

// closeLocked is Close's body, run under the store's lock.
func closeLocked(repoRoot, specID string) (Spec, error) {
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
	// Clobber guard: os.Rename would silently overwrite the destination, so
	// refuse when it already exists. Every abcd writer of the store holds the
	// store's lock across this check and the rename; a hand edit does not, and
	// under the trusted-worktree model (only the developer/agent mutates the
	// store) the atomic same-filesystem rename is preferred over a non-atomic
	// no-clobber link+remove that a crash could leave half-done.
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

// Discard takes back a spec a refused operation minted a moment ago, under the
// store's lock, so the removal cannot interleave with another writer's read and
// write of the same file — a repoint writing it back would resurrect it. Only a
// spec in open/, named as the mint names one, is removed; any other path is
// refused before anything is touched. A spec already gone is not an error, and
// a tree with no spec store has none to remove: the lock is taken in
// storeMustExist mode, so nothing is planted there.
func Discard(repoRoot string, sp Spec) error {
	rel := filepath.ToSlash(sp.Path)
	name := path.Base(rel)
	if rel != path.Join(filepath.ToSlash(SpecsRelDir), StatusOpen, name) || !specFileRe.MatchString(name) {
		return fmt.Errorf("spec: refusing to discard %q, which is not a spec in %s", sp.Path, filepath.Join(SpecsRelDir, StatusOpen))
	}
	err := withStoreLock(repoRoot, mintLockTimeout, storeMustExist, func() error {
		if err := os.Remove(filepath.Join(repoRoot, filepath.FromSlash(rel))); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("spec: discarding %s: %w", sp.Path, err)
		}
		return nil
	})
	if errors.Is(err, errStoreAbsent) {
		return nil
	}
	return err
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
