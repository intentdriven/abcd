package scribe

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/core/sessionkind"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// AssembleRequest is one scribe assembly: the run, the researcher's supplied
// dispositions, and where the two artefacts go.
type AssembleRequest struct {
	RepoRoot string
	// Run is the ingested reading run the session transcribes for (rdg-N).
	Run string
	// DispositionsPath is the researcher's dispositions text, in whatever form
	// they wrote it. It is the one supplied path, and it is read, never walked.
	DispositionsPath string
	// OutDir is where the context and the manifest go; empty means the local-tier
	// default for the run. OutDirLabel is the operator's spelling of it, for
	// refusals.
	OutDir      string
	OutDirLabel string
	// DryRun writes nothing unless OutDir names somewhere to write.
	DryRun bool
}

// AssembleResult is what one assembly produced. The two artefacts are carried
// for an in-memory caller and named by basename.
type AssembleResult struct {
	Run           string   `json:"run"`
	ContextStamp  string   `json:"context_stamp"`
	ContextSHA256 string   `json:"context_sha256"`
	ItemCount     int      `json:"item_count"`
	AllowList     []string `json:"allow_list"`
	OutDir        string   `json:"out_dir,omitempty"`
	Artefacts     []string `json:"artefacts"`
	Written       bool     `json:"written"`

	Context  Context  `json:"-"`
	Manifest Manifest `json:"-"`
}

// collectHook is the collector, behind a seam so a test can inject an item by a
// route the allow list does not own and watch assertAllowList refuse it.
var collectHook = collectLedger

// Assemble builds the scribe's context for one ingested run.
func Assemble(req AssembleRequest) (AssembleResult, error) {
	if strings.TrimSpace(req.RepoRoot) == "" {
		return AssembleResult{}, errors.New("scribe: no repository root given")
	}
	if err := requireCommittedRun(req.RepoRoot, req.Run); err != nil {
		return AssembleResult{}, err
	}
	if strings.TrimSpace(req.DispositionsPath) == "" {
		return AssembleResult{}, errors.New("scribe: no dispositions supplied; the scribe transcribes the " +
			"researcher's dispositions and authors none, so an assembly without them has nothing to hand it")
	}
	label := req.OutDirLabel
	if label == "" {
		label = req.OutDir
	}
	if err := reading.RefuseReachableOutDir(req.RepoRoot, req.OutDir, label,
		ContextFileName, ManifestFileName); err != nil {
		return AssembleResult{}, fmt.Errorf("scribe: %w", err)
	}
	raw, err := fsutil.ReadGuarded(req.DispositionsPath, reading.MaxFileBytes)
	if err != nil {
		return AssembleResult{}, fmt.Errorf("scribe: reading the supplied dispositions: %w", err)
	}
	supplied := scrub(req.RepoRoot, string(raw))

	entries, err := collectHook(req.RepoRoot)
	if err != nil {
		return AssembleResult{}, err
	}
	if err := assertAllowList(entries); err != nil {
		return AssembleResult{}, err
	}
	if !holdsRun(entries, req.Run) {
		return AssembleResult{}, fmt.Errorf("scribe: the ledger holds no reading record of %s, so there is "+
			"nothing for the scribe to transcribe dispositions against; a run with an empty item set has no "+
			"item to answer", req.Run)
	}

	ledgerRaw, err := encode(entries)
	if err != nil {
		return AssembleResult{}, err
	}
	stamp, err := sessionkind.Stamp(sessionkind.Scribe, req.Run, sha256Hex(ledgerRaw))
	if err != nil {
		return AssembleResult{}, fmt.Errorf("scribe: stamping the context: %w", err)
	}
	ctx := Context{
		Type: ContextType, SchemaVersion: SchemaVersion, ContextStamp: stamp, Run: req.Run,
		Ledger: entries, Supplied: Supplied{Dispositions: supplied},
	}
	contextRaw, err := encode(ctx)
	if err != nil {
		return AssembleResult{}, err
	}
	defHash, err := definitionHash(req.RepoRoot)
	if err != nil {
		return AssembleResult{}, err
	}
	m := Manifest{
		Type: ManifestType, SchemaVersion: SchemaVersion, ContextStamp: stamp, Run: req.Run,
		DefinitionSHA256: defHash, ContextSHA256: sha256Hex(contextRaw),
		Supplied:  SuppliedHashes{DispositionsSHA256: sha256Hex([]byte(supplied))},
		Items:     make([]ManifestItem, 0, len(entries)),
		AllowList: AllowList(), Exclusions: Exclusions(),
	}
	for _, e := range entries {
		m.Items = append(m.Items, ManifestItem{Path: e.Path, Bytes: len(e.Text), SHA256: sha256Hex([]byte(e.Text))})
	}
	manifestRaw, err := encode(m)
	if err != nil {
		return AssembleResult{}, err
	}

	res := AssembleResult{
		Run: req.Run, ContextStamp: stamp, ContextSHA256: m.ContextSHA256, ItemCount: len(entries),
		AllowList: m.AllowList, Artefacts: []string{}, Context: ctx, Manifest: m,
	}
	outDir := req.OutDir
	switch {
	case outDir != "":
	case req.DryRun:
		return res, nil
	default:
		outDir = DefaultRunDir + "/" + req.Run
	}
	res.OutDir = outDir
	dir := outDir
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(req.RepoRoot, filepath.FromSlash(outDir))
	}
	if label == "" {
		label = outDir
	}
	if err := writePair(dir, label, contextRaw, manifestRaw); err != nil {
		return AssembleResult{}, err
	}
	res.Written = true
	res.Artefacts = []string{ContextFileName, ManifestFileName}
	return res, nil
}

// requireCommittedRun refuses a run id that is not one, and a run with no
// commit marker: the scribe transcribes dispositions against a reading the
// ledger already holds, so its items come from the store and never from a raw
// output handed over a second time (adr-2609021016275803).
func requireCommittedRun(repoRoot, run string) error {
	if !recordid.ValidReadingRunID(run) {
		return fmt.Errorf("scribe: run %q is not a reading run id (rdg-N)", echo(run))
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return fmt.Errorf("scribe: opening the repository root: %w", err)
	}
	defer root.Close()
	marker := issueschema.ReadingsRecordDir + "/" + run + "/" + issueschema.RunRecordFileName
	fi, err := root.Lstat(marker)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("scribe: %s has no commit marker at %s, so it was never ingested; the scribe "+
			"transcribes a reading the ledger already holds — ingest the run first", run, marker)
	case err != nil:
		return fmt.Errorf("scribe: probing %s: %w", marker, err)
	case !fi.Mode().IsRegular():
		return fmt.Errorf("scribe: %s is not a regular file, so it marks nothing", marker)
	}
	return nil
}

// collectLedger walks the allow list and nothing else, returning every ledger
// record sorted by path. A symlinked directory or leaf is refused, a hidden
// entry (a lock, a placeholder) is skipped, and each record is read behind the
// guarded reader at the ledger's record limit.
func collectLedger(repoRoot string) ([]LedgerEntry, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("scribe: opening the repository root: %w", err)
	}
	defer root.Close()

	// The walk below runs inside the repository root, which follows a symlink
	// that stays inside it, and the Lstat on each allow-list directory sees only
	// the leaf. So the ANCESTORS are judged first, by the rule capture's own
	// readers apply: the ledger moved into the shipped tree behind a committed
	// link would otherwise reach the context under ledger paths.
	if err := refuseRedirectedLedger(repoRoot); err != nil {
		return nil, err
	}

	var out []LedgerEntry
	for _, dir := range AllowList() {
		fi, err := root.Lstat(dir)
		if os.IsNotExist(err) {
			continue // a family with no records yet contributes nothing
		}
		if err != nil {
			return nil, fmt.Errorf("scribe: probing %s: %w", dir, err)
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("%w: %s is a symlink; the context is drawn from the ledger's own "+
				"directories and a link is a route out of them", ErrSymlink, dir)
		}
		if !fi.IsDir() {
			return nil, fmt.Errorf("scribe: %s is not a directory", dir)
		}
		err = fs.WalkDir(root.FS(), dir, func(p string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if p != dir && strings.HasPrefix(d.Name(), ".") {
				if d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				return fmt.Errorf("%w: %s is a symlink; the context is drawn from the ledger's own files "+
					"and a link is a route out of them", ErrSymlink, p)
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".md") {
				return nil
			}
			raw, err := fsutil.ReadGuardedInRoot(root, p, issueschema.RecordReadLimit)
			if err != nil {
				if errors.Is(err, fsutil.ErrNotRegular) {
					return fmt.Errorf("%w: %s is not a regular file", ErrSymlink, p)
				}
				return fmt.Errorf("scribe: reading %s: %w", p, err)
			}
			out = append(out, LedgerEntry{Path: p, Text: scrub(repoRoot, string(raw))})
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// refuseRedirectedLedger is capture.RefuseRedirectedLedger under this package's
// symlink sentinel, so a caller tests one refusal whichever level held the link.
func refuseRedirectedLedger(repoRoot string) error {
	if err := capture.RefuseRedirectedLedger(repoRoot); err != nil {
		return fmt.Errorf("%w: the ledger is reached through a directory that is not a real one (%w); the "+
			"context is drawn from the ledger's own directories and a link above them is a route out", ErrSymlink, err)
	}
	return nil
}

// assertAllowList is the fail-closed half: every item must sit strictly inside
// one allow-list directory, whatever route it arrived by, or the assembly is
// refused naming it (adr-56). The collector's walk is the positive half; this is
// what makes an item that reached the context by any future route a refusal
// rather than a disclosure.
func assertAllowList(entries []LedgerEntry) error {
	allowed := AllowList()
	for _, e := range entries {
		clean := path.Clean(e.Path)
		ok := clean == e.Path && !strings.Contains(e.Path, "..")
		if ok {
			ok = false
			for _, dir := range allowed {
				if strings.HasPrefix(clean, dir+"/") {
					ok = true
					break
				}
			}
		}
		if !ok {
			return fmt.Errorf("scribe: %s is outside the scribe's allow list (%s); the scribe receives "+
				"ledger content and nothing else, so the assembly is refused rather than disclosing it",
				echo(e.Path), strings.Join(allowed, ", "))
		}
	}
	return nil
}

// holdsRun reports whether the ledger entries include a reading record of run.
func holdsRun(entries []LedgerEntry, run string) bool {
	prefix := runRecordsDir(run) + "/"
	for _, e := range entries {
		if strings.HasPrefix(e.Path, prefix) {
			return true
		}
	}
	return false
}

// runRecordsDir is where the ledger files one run's reading records.
func runRecordsDir(run string) string {
	return capture.LedgerRelPath + "/" + issueschema.ReadingsDir + "/" + run
}

// scrub takes the caller's home and the repository root out of a text, so the
// context names no absolute local path. The ledger is redacted at capture; the
// supplied text is the researcher's own and has been through nothing.
func scrub(repoRoot, s string) string {
	if abs, err := filepath.Abs(repoRoot); err == nil {
		s = fsutil.RedactRoot(s, abs, ".")
		if real, err := filepath.EvalSymlinks(abs); err == nil && real != abs {
			s = fsutil.RedactRoot(s, real, ".")
		}
	}
	return fsutil.RedactHome(s)
}

// definitionHash is the sha256 of the scribe definition, read through the
// repository root so a symlinked ancestor cannot have it hash a file outside
// the repository.
func definitionHash(repoRoot string) (string, error) {
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return "", fmt.Errorf("scribe: opening the repository root: %w", err)
	}
	defer root.Close()
	raw, err := fsutil.ReadGuardedInRoot(root, DefinitionPath, reading.MaxFileBytes)
	if err != nil {
		return "", fmt.Errorf("scribe: the definition at %s: %w", DefinitionPath, err)
	}
	return sha256Hex(raw), nil
}

// writePair writes the context and the manifest into an empty or absent
// directory, both or neither, on the reading assembler's rules.
func writePair(dir, label string, contextRaw, manifestRaw []byte) error {
	entries, err := os.ReadDir(dir)
	switch {
	case os.IsNotExist(err):
	case err != nil:
		return fmt.Errorf("scribe: reading the output directory: %w", err)
	case len(entries) > 0:
		return fmt.Errorf("scribe: the output directory %s is not empty (%d entr(y|ies)); one scribe "+
			"session's artefacts are one session's evidence, so ingest or clear the session parked there, "+
			"or name an empty directory", label, len(entries))
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("scribe: creating the output directory: %w", err)
	}
	ctxPath := filepath.Join(dir, ContextFileName)
	if err := fsutil.WriteFileAtomic(ctxPath, contextRaw, 0o644); err != nil {
		return fmt.Errorf("scribe: writing the context: %w", err)
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, ManifestFileName), manifestRaw, 0o644); err != nil {
		os.Remove(ctxPath)
		return fmt.Errorf("scribe: writing the manifest: %w", err)
	}
	return nil
}
