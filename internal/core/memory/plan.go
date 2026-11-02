package memory

// plan.go — the practice run of the memory store's two writing verbs that
// read a filled-in input (itd-2610301428390406, spc-2610301431131636 step 2):
// `library ingest --pages-json` and `library ask --file-back --page-json`.
//
// Every store mutation goes through the writer's locked sequence (skeleton,
// backfill, prune, reconcile, the pages, the registry, reconcile again, the
// log). A later step reads what an earlier one wrote, so a plan cannot be made
// by inspecting the disk once: the sequence itself runs, unchanged, against an
// in-memory overlay of its own writes over the store as it stands, and the
// overlay is then reported as the plan. The real write runs the same sequence
// against the disk, so the two share every step by construction.

import (
	"os"
	"path"
	"path/filepath"
	"sort"
	"time"

	"github.com/intentdriven/abcd/internal/core/dryrun"
)

// storeFS is the file operations the writer's locked sequence makes. The
// disk is one; a plan's overlay is the other.
type storeFS interface {
	// read is triStateRead: (content, present, error).
	read(path string) (string, bool, error)
	write(path, content string) error
	remove(path string) error
	// pages is pageFiles: the store's page files, sorted.
	pages(mem string) ([]string, error)
	loadRegistry(path string) (map[string]any, error)
}

// diskStore is the store on disk, written for real.
type diskStore struct{}

func (diskStore) read(p string) (string, bool, error) { return triStateRead(p) }
func (diskStore) write(p, content string) error       { return writeStringAtomic(p, content) }
func (diskStore) remove(p string) error               { return os.Remove(p) }
func (diskStore) pages(mem string) ([]string, error)  { return pageFiles(mem) }
func (diskStore) loadRegistry(p string) (map[string]any, error) {
	return LoadRegistry(p)
}

// overlayStore reads the disk through every write it has taken, and writes
// nothing: the plan's view of the store mid-sequence.
type overlayStore struct {
	// files maps a written path to its content; a removed path maps to nil.
	files map[string]*string
	// order is each path in the order it was first written or removed.
	order []string
}

func newOverlayStore() *overlayStore { return &overlayStore{files: map[string]*string{}} }

func (o *overlayStore) note(p string) {
	if _, seen := o.files[p]; !seen {
		o.order = append(o.order, p)
	}
}

func (o *overlayStore) read(p string) (string, bool, error) {
	if c, ok := o.files[p]; ok {
		if c == nil {
			return "", false, nil
		}
		return *c, true, nil
	}
	return triStateRead(p)
}

func (o *overlayStore) write(p, content string) error {
	o.note(p)
	o.files[p] = &content
	return nil
}

func (o *overlayStore) remove(p string) error {
	o.note(p)
	o.files[p] = nil
	return nil
}

func (o *overlayStore) pages(mem string) ([]string, error) {
	onDisk, err := pageFiles(mem)
	if err != nil {
		return nil, err
	}
	set := map[string]bool{}
	for _, name := range onDisk {
		set[name] = true
	}
	for p, c := range o.files {
		if filepath.Dir(p) != filepath.Clean(mem) || !IsMemoryPageName(filepath.Base(p)) {
			continue
		}
		set[filepath.Base(p)] = c != nil
	}
	var out []string
	for name, live := range set {
		if live {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (o *overlayStore) loadRegistry(p string) (map[string]any, error) {
	if c, ok := o.files[p]; ok {
		if c == nil {
			return map[string]any{}, nil
		}
		return decodeRegistry([]byte(*c), nil, p)
	}
	return LoadRegistry(p)
}

// removeEffect is the effect a planned removal is reported as.
const removeEffect = "remove"

// plan reports the overlay against the disk: each path whose final content
// differs from what is on disk is a write (a create where nothing is there),
// and each path removed that is on disk is a removal. Paths are
// repository-relative.
func (o *overlayStore) plan(repoRoot string) dryrun.Plan {
	var out dryrun.Plan
	for _, p := range o.order {
		rel, err := filepath.Rel(repoRoot, p)
		if err != nil {
			rel = p
		}
		rel = filepath.ToSlash(rel)
		cur, present, _ := triStateRead(p)
		c := o.files[p]
		switch {
		case c == nil && present:
			out.Effects = append(out.Effects, dryrun.Effect{Effect: removeEffect, Target: rel})
		case c == nil:
		case present && cur == *c:
		case present:
			out.Writes = append(out.Writes, dryrun.Write{Path: rel, Action: dryrun.Replace, Bytes: len(*c), Content: []byte(*c)})
		default:
			out.Writes = append(out.Writes, dryrun.Write{Path: rel, Action: dryrun.Create, Bytes: len(*c), Content: []byte(*c)})
		}
	}
	return out
}

// storePlanner is a plan in the making: the overlay every write of one verb
// lands in. A nil planner is the real write.
type storePlanner struct {
	repoRoot string
	overlay  *overlayStore
	// lockLaid is set when the store's lock file is absent, so the real write
	// would lay it.
	lockLaid bool
}

func newStorePlanner(repoRoot string) *storePlanner {
	return &storePlanner{repoRoot: repoRoot, overlay: newOverlayStore()}
}

// writePages is WritePages for the planner: the same pre-lock judgement, the
// store resolved without being created and its lock path held to the lock's
// own refusal, then the locked sequence run against the overlay.
func (pl *storePlanner) writePages(writes []PageWrite, merge RegistryMerge, now time.Time) (WriteReport, error) {
	rendered, redactor, err := prepareWrites(pl.repoRoot, writes, merge)
	if err != nil {
		return WriteReport{}, err
	}
	mem, _, err := safeMemoryDir(pl.repoRoot)
	if err != nil {
		return WriteReport{}, err
	}
	lock := filepath.Join(mem, ".lock")
	if fi, err := os.Lstat(lock); err == nil {
		if fi.Mode()&os.ModeSymlink != 0 || !fi.Mode().IsRegular() {
			return WriteReport{}, &UnsafeStorePathError{Msg: "memory store lock path is a symlink or non-regular file: " + lock}
		}
	} else if os.IsNotExist(err) {
		pl.lockLaid = true
	} else {
		return WriteReport{}, err
	}
	return writePagesLocked(pl.overlay, pl.repoRoot, rendered, merge, redactor, now)
}

// keep plans the kept original at sources/<name>, through the same redaction
// the real copy applies.
func (pl *storePlanner) keep(material sourceMaterial, contentHash string, redactor *storeRedactor) (string, error) {
	payload, err := redactor.redactOriginalBytes(material)
	if err != nil {
		return "", err
	}
	name := contentHash + material.ext
	abs := filepath.Join(Dir(pl.repoRoot), "sources", name)
	if err := pl.overlay.write(abs, string(payload)); err != nil {
		return "", err
	}
	return filepath.Join(".abcd", "memory", "sources", name), nil
}

// report is the plan: every write the overlay took, then the store's lock
// file when the real write would lay it.
func (pl *storePlanner) report() dryrun.Plan {
	p := pl.overlay.plan(pl.repoRoot)
	if pl.lockLaid {
		p.Writes = append(p.Writes, dryrun.Write{Path: path.Join(RelDir, ".lock"), Action: dryrun.Create,
			Pending: "the store's lock file, laid empty when the write takes the lock"})
	}
	return p
}

// writeThrough writes through the planner when there is one, and to the
// store when there is none.
func writeThrough(pl *storePlanner, repoRoot string, writes []PageWrite, merge RegistryMerge, now time.Time) (WriteReport, error) {
	if pl != nil {
		return pl.writePages(writes, merge, now)
	}
	return WritePages(repoRoot, writes, merge, now)
}

// IngestPlan is what Ingest would do: the result it would report and the
// writes it would make.
type IngestPlan struct {
	Result IngestResult
	Plan   dryrun.Plan
}

// PlanIngest does everything Ingest does, the source read as the real run
// reads it, the distilled pages judged, and the writer's locked sequence run,
// against an overlay instead of the store. It writes nothing; a refusal is the
// real run's.
func PlanIngest(req IngestRequest) (IngestPlan, error) {
	pl := newStorePlanner(req.RepoRoot)
	res, err := ingest(req, pl)
	if err != nil {
		return IngestPlan{}, err
	}
	return IngestPlan{Result: res, Plan: pl.report()}, nil
}

// AskPlan is what Ask would do: the result it would report and the writes a
// file-back would make.
type AskPlan struct {
	Result AskResult
	Plan   dryrun.Plan
}

// PlanAsk is Ask with its file-back planned rather than written: retrieval and
// synthesis run as they do for the real ask, the answer page is judged, and
// the writer's locked sequence runs against an overlay. It writes nothing.
func PlanAsk(req AskRequest) (AskPlan, error) {
	pl := newStorePlanner(req.RepoRoot)
	res, err := ask(req, pl)
	if err != nil {
		return AskPlan{}, err
	}
	return AskPlan{Result: res, Plan: pl.report()}, nil
}
