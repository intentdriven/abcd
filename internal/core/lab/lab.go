// Package lab mechanises the lab conventions three hand-run experiments proved
// (itd-2609212137128014): a lab is a throwaway world pinned at one commit of a
// repository, run to answer one question, whose evidence lives at the operator
// level and whose knowledge enters the record only through capture.
//
// The store is machine-scoped and keyed on the repository's root commit, the way
// the transcript and worktree stores are:
//
//	~/.abcd/lab/<root-sha>/index.jsonl        one registry line per lab
//	~/.abcd/lab/<root-sha>/<lab-id>/          one lab home
//	    INTENTION.md                          the question, before any mutation
//	    snapshot/                             a standalone clone at the pin
//	    home/                                 the lab's own HOME
//	    bin/                                  the work binary (and a test binary)
//	    state/preflight.md                    the preflight artefact
//	    state/probes/<probe>/                 one probe record per probe
//	    state/sweep.md                        the retraction sweep's artefact
//	    findings.md, corrections.md           running logs the harvest mines
//	    amendments.md                         procedure deltas the lab contributed
//	    harvest/harvest.md                    the assembled harvest
//
// Every verb writes only under one lab home (the registry line is the store's
// own), and nothing is ever written into the repository a lab studies. Hand-run
// labs that predate the keyed layout sit beside the root-commit directories and
// are neither read nor written.
//
// The package never writes to stdout: it returns results, and the front doors
// under internal/surface render them.
package lab

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// storeRelPath is the lab store relative to the caller's home.
const storeRelPath = ".abcd/lab"

// StoreDisplay names the store in a message, in tilde form, so no output
// carries the account's home path.
const StoreDisplay = "~/" + storeRelPath

// Names inside a lab home and the store.
const (
	indexName       = "index.jsonl"
	intentionName   = "INTENTION.md"
	findingsName    = "findings.md"
	correctionsName = "corrections.md"
	amendmentsName  = "amendments.md"
	snapshotDir     = "snapshot"
	labHomeDir      = "home"
	binDir          = "bin"
	stateDir        = "state"
	probesDir       = "state/probes"
	harvestDir      = "harvest"
	harvestName     = "harvest/harvest.md"
	preflightMD     = "state/preflight.md"
	preflightJSON   = "state/preflight.json"
	workBinPinName  = "state/work-binary.json"
	sweepMD         = "state/sweep.md"
	workBinary      = "bin/abcd"
	testBinary      = "bin/abcd-test"
)

// storeDirPerm and fileMode keep the store the account's own business: a lab
// touches private-tier material, so nothing in it is group- or world-readable.
const (
	storeDirPerm = 0o700
	fileMode     = 0o600
)

// maxDocBytes caps every lab document the verbs read back. A lab's prose is a
// few kilobytes; the cap bounds a planted device or a runaway log.
const maxDocBytes = 4 << 20

// maxQuestionBytes caps the question a lab is minted for: one line.
const maxQuestionBytes = 300

// ErrRefused marks a refusal the caller can act on (a bad argument, an unknown
// lab, a store path occupied by the wrong kind of thing). A front door maps it
// to a usage exit.
var ErrRefused = errors.New("refused")

// ErrHalted marks a gate that failed: the verb wrote its artefact and a finding,
// and the lab is halted until that gate passes again.
var ErrHalted = errors.New("lab halted")

// now is the clock the id and the stamps read. Tests pin it.
var now = time.Now

// idRe is a lab id: the UTC mint time and the pin's first seven hex digits.
var idRe = regexp.MustCompile(`^lab-[0-9]{12}-[0-9a-f]{7}$`)

// ValidID reports whether s is a lab id as the store names one.
func ValidID(s string) bool { return idRe.MatchString(s) }

// Entry is one registry line: what a lab was minted for, and at which pin.
type Entry struct {
	ID       string `json:"id"`
	RootSHA  string `json:"root_sha"`
	Pin      string `json:"pin"`
	Created  string `json:"created"`
	Question string `json:"question"`
}

// store is one repository's lane of the lab store.
type store struct {
	home    string // the caller's home
	rootSHA string
	dir     string // <home>/.abcd/lab/<root-sha>
}

// rel is the lane's path relative to the caller's home.
func (s store) rel() string { return storeRelPath + "/" + s.rootSHA }

// Display is a path inside the store in tilde form.
func (s store) display(parts ...string) string {
	return filepath.ToSlash(filepath.Join(append([]string{StoreDisplay, s.rootSHA}, parts...)...))
}

// resolveStore names this repository's lane of the store without touching disk.
func resolveStore(repoRoot string) (store, error) {
	sha := gitutil.RootCommit(repoRoot)
	if !gitutil.IsFullSHA(sha) {
		return store{}, fmt.Errorf("%w: the repository has no root commit to key the lab store on (a repository with no commits has no labs)", ErrRefused)
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return store{}, fmt.Errorf("cannot resolve the caller's home directory: %v", err)
	}
	return store{home: home, rootSHA: sha, dir: filepath.Join(home, filepath.FromSlash(storeRelPath), sha)}, nil
}

// ensure creates the lane one real directory at a time, never through a
// symlink, and proves every level.
func (s store) ensure() error {
	if err := fsutil.EnsureRealDirAll(s.home, s.rel(), storeDirPerm); err != nil {
		return fmt.Errorf("%w: cannot create the lab store %s: %v", ErrRefused, s.display(), redact(err, s.home))
	}
	return nil
}

// exists reports whether the lane exists, refusing a lane path occupied by
// anything but a real directory.
func (s store) exists() (bool, error) {
	for _, p := range []string{filepath.Join(s.home, ".abcd"), filepath.Join(s.home, ".abcd", "lab"), s.dir} {
		if fsutil.IsRealDir(p) {
			continue
		}
		if ok, _ := fsutil.ExistsNoFollow(p); ok {
			return false, fmt.Errorf("%w: %s is not a real directory (a symlink or a file occupies it)", ErrRefused, tilde(p, s.home))
		}
		return false, nil
	}
	return true, nil
}

// entries reads the lane's registry. A line that does not parse is skipped and
// counted, never guessed at.
func (s store) entries() ([]Entry, int, error) {
	ok, err := s.exists()
	if err != nil || !ok {
		return nil, 0, err
	}
	r, err := os.OpenRoot(s.dir)
	if err != nil {
		return nil, 0, fmt.Errorf("cannot open the lab store: %v", redact(err, s.home))
	}
	defer r.Close()
	data, err := fsutil.ReadGuardedInRoot(r, indexName, maxDocBytes)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0, nil
	}
	if err != nil {
		return nil, 0, fmt.Errorf("cannot read %s: %v", s.display(indexName), redact(err, s.home))
	}
	var out []Entry
	bad := 0
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 0, 64<<10), maxDocBytes)
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var e Entry
		if json.Unmarshal(line, &e) != nil || !ValidID(e.ID) || !gitutil.IsFullSHA(e.Pin) {
			bad++
			continue
		}
		out = append(out, e)
	}
	return out, bad, nil
}

// lab is one opened lab home.
type lab struct {
	store store
	entry Entry
	dir   string   // absolute lab home
	root  *os.Root // the lab home as a containment root
}

func (l *lab) close() { _ = l.root.Close() }

// display names a path inside the lab home in tilde form.
func (l *lab) display(rel string) string { return l.store.display(l.entry.ID, rel) }

// open resolves and opens an existing lab of this repository. The id is held to
// its shape before it becomes a path segment, and the lab must be in the
// registry: a directory the verb did not mint is not a lab.
func open(repoRoot, id string) (*lab, error) {
	if !ValidID(id) {
		return nil, fmt.Errorf("%w: %q is not a lab id (lab-<yymmddHHMMSS>-<pin7>)", ErrRefused, clip(id))
	}
	s, err := resolveStore(repoRoot)
	if err != nil {
		return nil, err
	}
	es, _, err := s.entries()
	if err != nil {
		return nil, err
	}
	var entry *Entry
	for i := range es {
		if es[i].ID == id {
			entry = &es[i]
		}
	}
	if entry == nil {
		return nil, fmt.Errorf("%w: no lab %s in this repository's lane of %s", ErrRefused, id, StoreDisplay)
	}
	dir := filepath.Join(s.dir, id)
	if !fsutil.IsRealDir(dir) {
		return nil, fmt.Errorf("%w: the lab home %s is missing or not a real directory", ErrRefused, s.display(id))
	}
	r, err := os.OpenRoot(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot open the lab home %s: %v", s.display(id), redact(err, s.home))
	}
	return &lab{store: s, entry: *entry, dir: dir, root: r}, nil
}

// readDoc reads one lab document through the containment root. An absent file
// reads as empty.
func (l *lab) readDoc(rel string) (string, error) {
	data, err := fsutil.ReadGuardedInRoot(l.root, rel, maxDocBytes)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("cannot read %s: %v", l.display(rel), redact(err, l.store.home))
	}
	return string(data), nil
}

// writeDoc replaces one lab document atomically, inside the containment root.
func (l *lab) writeDoc(rel, content string) error {
	if err := fsutil.WriteFileAtomicInRoot(l.root, rel, []byte(content), fileMode); err != nil {
		return fmt.Errorf("cannot write %s: %v", l.display(rel), redact(err, l.store.home))
	}
	return nil
}

// Summary is one lab as the bare listing shows it.
type Summary struct {
	Entry
	Home    string   `json:"home"`
	Probes  int      `json:"probes"`
	Halted  []string `json:"halted"`
	Missing bool     `json:"missing,omitempty"`
}

// Listing is the bare verb's read-only answer.
type Listing struct {
	Store     string    `json:"store"`
	Labs      []Summary `json:"labs"`
	Unparsed  int       `json:"unparsed,omitempty"`
	RootSHA   string    `json:"root_sha"`
	StoreSeen bool      `json:"store_seen"`
}

// List reads this repository's labs. It writes nothing and creates nothing.
func List(repoRoot string) (Listing, error) {
	s, err := resolveStore(repoRoot)
	if err != nil {
		return Listing{}, err
	}
	out := Listing{Store: s.display(), RootSHA: s.rootSHA, Labs: []Summary{}}
	seen, err := s.exists()
	if err != nil {
		return Listing{}, err
	}
	out.StoreSeen = seen
	es, bad, err := s.entries()
	if err != nil {
		return Listing{}, err
	}
	out.Unparsed = bad
	for _, e := range es {
		sum := Summary{Entry: e, Home: s.display(e.ID), Halted: []string{}}
		l, err := open(repoRoot, e.ID)
		if err != nil {
			sum.Missing = true
			out.Labs = append(out.Labs, sum)
			continue
		}
		sum.Probes = len(l.probeNames())
		sum.Halted = l.haltedGates()
		l.close()
		out.Labs = append(out.Labs, sum)
	}
	return out, nil
}

// tilde writes a path under home in tilde form.
func tilde(p, home string) string {
	if rel, err := filepath.Rel(home, p); err == nil && !strings.HasPrefix(rel, "..") {
		return "~/" + filepath.ToSlash(rel)
	}
	return fsutil.RedactHome(p)
}

// redact scrubs the home path out of an error's text.
func redact(err error, home string) string {
	if err == nil {
		return ""
	}
	return fsutil.RedactRoot(err.Error(), home, "~")
}

// clip bounds an untrusted token quoted back in a refusal.
func clip(s string) string {
	if len(s) <= 80 {
		return s
	}
	cut := 80
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut] + "…"
}

// stamp is the current time as the lab's documents record it.
func stamp() string { return now().UTC().Format(time.RFC3339) }
