// Package source is the personal sources corpus and its provenance ledger (itd-76,
// spc-31): a local-only store of documents the agent may consult, a CSL-JSON
// bibliography describing them, and one append-only influence ledger per consuming
// repository. It never prints and never exits — front doors under
// internal/surface/* format its results.
//
// The trust boundary it enforces is adr-41 / brief invariant 9, cited rather than
// restated: documents and ledgers never leave the user tier, and a public citation
// needs both the source's permission_status and a human-flipped ledger line.
//
// Layout of a corpus directory (the user-level home's `sources/`, by default
// ~/.abcd/sources):
//
//	sources.json            CSL-JSON array; each entry's `custom` block carries
//	                        confidential, permission_status, keywords, aliases,
//	                        ban_authors and file
//	confidential/<key>/     original.<ext>, text.md, and derived artefacts
//	public/<key>/           the same shape for a freely citable source
//	ledger/<repo>.jsonl     append-only influence records for one repository
//
// FOLDER LOCATION IS THE CLASSIFICATION. `custom.confidential` mirrors it, and a
// corpus where the two disagree is refused wholesale by every step that derives a
// ban from it (the safe direction: the block already written keeps banning). The
// corpus is itself a git repository with no remote; its history is the
// tamper-evidence layer, and every write here is committed.
//
// Matching is not this package's. The projection of a confidential entry into
// patterns and every scan run through banlist.PhrasePattern and banlist.ScanText,
// the private name layer's one matcher, so the pre-commit guard and cite-check
// cannot disagree about what a confidential source is called.
package source

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// Classes. The folder a source sits in is one of these, and nothing else decides it.
const (
	ClassConfidential = "confidential"
	ClassPublic       = "public"
)

// Permission statuses. PermissionCitable is the ONLY value that grants the right to
// cite (adr-41 gate 1); every other value, including one this vocabulary does not
// know, withholds it.
const (
	PermissionCitable          = "citable"
	PermissionNoPublicCitation = "no-public-citation"
	PermissionInternal         = "internal-never-cite"
	PermissionAIGenerated      = "ai-generated-never-cite"
	PermissionAskAuthor        = "ask-author"
)

// Permissions is the closed vocabulary `add` and `declassify` accept.
var Permissions = []string{PermissionCitable, PermissionNoPublicCitation, PermissionInternal, PermissionAIGenerated, PermissionAskAuthor}

// Influences is the closed vocabulary of a ledger line's influence.
var Influences = []string{"supports", "contradicts", "method", "background"}

// File and directory names inside a corpus.
const (
	SourcesFile = "sources.json"
	LedgerDir   = "ledger"
	TextFile    = "text.md"
	readmeFile  = "README.md"
	lockFile    = "abcd-source.lock"
)

// maxSourcesBytes caps the bibliography read (trust boundary).
const maxSourcesBytes = 32 << 20

// lockTimeout bounds the wait for the corpus lock.
const lockTimeout = 5 * time.Second

// Sentinel errors. No message carries a confidential title, alias or author: a
// refusal is output like any other, and keys are the only handle output names.
var (
	// ErrNoCorpus reports that the corpus directory does not exist. Every step that
	// needs a corpus returns it, and a front door turns it into one loud line.
	ErrNoCorpus = errors.New("sources corpus is absent")
	// ErrCorpusInvalid reports a corpus directory that exists but cannot be used.
	ErrCorpusInvalid = errors.New("sources corpus is not usable")
	// ErrInvalidEntry rejects a source entry.
	ErrInvalidEntry = errors.New("invalid source entry")
	// ErrIdentifyingKey rejects a confidential entry whose key names it: the key is
	// the one handle every refusal and every scan prints.
	ErrIdentifyingKey = errors.New("a confidential source's key would name it")
	// ErrDuplicateSource rejects a key the corpus already carries.
	ErrDuplicateSource = errors.New("source key already exists")
	// ErrUnknownSource reports a key the corpus does not carry.
	ErrUnknownSource = errors.New("unknown source key")
	// ErrClassMismatch reports a corpus whose folders and entries disagree.
	ErrClassMismatch = errors.New("the corpus's classes disagree")
	// ErrInvalidLedger rejects a ledger record, or reports a ledger that does not read.
	ErrInvalidLedger = errors.New("invalid ledger record")
	// ErrCitationRefused is the two-gate refusal of a cited_publicly flip.
	ErrCitationRefused = errors.New("public citation refused")
)

// keyRe is a source key: lowercase ASCII, path-safe, and inside the banlist key
// charset so it can head a generated entry's key.
var keyRe = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{0,63}$`)

// repoRe is a ledger's repository handle.
var repoRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$`)

// Name is a CSL name: family and given, or one literal.
type Name struct {
	Family  string `json:"family,omitempty"`
	Given   string `json:"given,omitempty"`
	Literal string `json:"literal,omitempty"`
}

// Custom is the entry's abcd block.
type Custom struct {
	Confidential     bool     `json:"confidential"`
	PermissionStatus string   `json:"permission_status"`
	Keywords         []string `json:"keywords,omitempty"`
	Aliases          []string `json:"aliases,omitempty"`
	BanAuthors       bool     `json:"ban_authors,omitempty"`
	File             string   `json:"file,omitempty"`
}

// Date is a CSL date.
type Date struct {
	DateParts [][]int `json:"date-parts"`
}

// Entry is one CSL-JSON item as this package reads it. Fields it does not name are
// kept byte-for-byte when the bibliography is rewritten.
type Entry struct {
	ID             string `json:"id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Author         []Name `json:"author,omitempty"`
	Issued         *Date  `json:"issued,omitempty"`
	ContainerTitle string `json:"container-title,omitempty"`
	URL            string `json:"URL,omitempty"`
	Custom         Custom `json:"custom"`
}

// Corpus is a loaded corpus: its entries, the raw bytes of each (so a rewrite
// preserves what this package does not model), and the folder each key sits in.
type Corpus struct {
	Dir     string
	Entries []Entry
	raw     []json.RawMessage
	// folders maps a key to the classes it has a folder under.
	folders map[string][]string
}

// Problem is one inconsistency, named by key only.
type Problem struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}

// DefaultDir is the corpus's default location: the user-level home's `sources/`.
// Relocating the home is itd-77's concern; a caller may pass any directory.
func DefaultDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return "", fmt.Errorf("cannot resolve the home directory for the default corpus: %v", err)
	}
	return filepath.Join(home, ".abcd", "sources"), nil
}

// present reports whether dir exists, and refuses one that is not a real directory.
func present(dir string) error {
	fi, err := os.Lstat(dir)
	switch {
	case os.IsNotExist(err):
		return fmt.Errorf("%w: there is no corpus at the configured location", ErrNoCorpus)
	case err != nil:
		return fmt.Errorf("%w: the corpus location cannot be read: %v", ErrCorpusInvalid, err)
	case fi.Mode()&os.ModeSymlink != 0:
		return fmt.Errorf("%w: the corpus location is a symlink; point at the real directory", ErrCorpusInvalid)
	case !fi.IsDir():
		return fmt.Errorf("%w: the corpus location is not a directory", ErrCorpusInvalid)
	}
	return nil
}

// loadCorpus reads a corpus. An absent directory is ErrNoCorpus; a directory without a
// bibliography or a git repository is ErrCorpusInvalid.
func loadCorpus(dir string) (*Corpus, error) {
	if err := present(dir); err != nil {
		return nil, err
	}
	if fi, err := os.Lstat(filepath.Join(dir, ".git")); err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("%w: the corpus directory is not a git repository (run `abcd source init` on an empty location)", ErrCorpusInvalid)
	}
	data, err := fsutil.ReadGuarded(filepath.Join(dir, SourcesFile), maxSourcesBytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %s cannot be read (absent, oversize, or not a regular file)", ErrCorpusInvalid, SourcesFile)
	}
	c := &Corpus{Dir: dir, folders: map[string][]string{}}
	if err := json.Unmarshal(data, &c.raw); err != nil {
		return nil, fmt.Errorf("%w: %s is not a CSL-JSON array", ErrCorpusInvalid, SourcesFile)
	}
	for i, r := range c.raw {
		var e Entry
		if err := json.Unmarshal(r, &e); err != nil {
			return nil, fmt.Errorf("%w: entry %d of %s does not read as CSL-JSON", ErrCorpusInvalid, i+1, SourcesFile)
		}
		c.Entries = append(c.Entries, e)
	}
	for _, class := range []string{ClassConfidential, ClassPublic} {
		des, err := os.ReadDir(filepath.Join(dir, class))
		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s/ cannot be listed", ErrCorpusInvalid, class)
		}
		for _, de := range des {
			if de.IsDir() {
				c.folders[de.Name()] = append(c.folders[de.Name()], class)
			}
		}
	}
	return c, nil
}

// Lookup returns the entry for key.
func (c *Corpus) Lookup(key string) (Entry, bool) {
	for _, e := range c.Entries {
		if e.ID == key {
			return e, true
		}
	}
	return Entry{}, false
}

// Class is the class a key's folder declares, or "" when it has none or two.
func (c *Corpus) Class(key string) string {
	if f := c.folders[key]; len(f) == 1 {
		return f[0]
	}
	return ""
}

// Problems lists every inconsistency between entries and folders, by key only.
func (c *Corpus) Problems() []Problem {
	var out []Problem
	seen := map[string]bool{}
	for _, e := range c.Entries {
		switch {
		case !keyRe.MatchString(e.ID):
			out = append(out, Problem{Key: "(entry with an invalid id)", Reason: "the id is not a valid source key"})
			continue
		case seen[e.ID]:
			out = append(out, Problem{Key: e.ID, Reason: "the key appears twice in " + SourcesFile})
			continue
		}
		seen[e.ID] = true
		f := c.folders[e.ID]
		switch len(f) {
		case 0:
			out = append(out, Problem{Key: e.ID, Reason: "no folder under confidential/ or public/"})
			continue
		case 2:
			out = append(out, Problem{Key: e.ID, Reason: "a folder under both confidential/ and public/"})
			continue
		}
		folderConf := f[0] == ClassConfidential
		if folderConf != e.Custom.Confidential {
			out = append(out, Problem{Key: e.ID, Reason: fmt.Sprintf("the folder is under %s/ but the entry says confidential: %v", f[0], e.Custom.Confidential)})
			continue
		}
		if folderConf && e.Custom.PermissionStatus == PermissionCitable {
			out = append(out, Problem{Key: e.ID, Reason: "a confidential source cannot be citable; declassify it instead"})
		}
	}
	var orphans []string
	for k := range c.folders {
		if !seen[k] && keyRe.MatchString(k) {
			orphans = append(orphans, k)
		}
	}
	sort.Strings(orphans)
	for _, k := range orphans {
		out = append(out, Problem{Key: k, Reason: "a folder with no entry in " + SourcesFile})
	}
	return out
}

// requireConsistent refuses a corpus with any problem, naming every key.
func (c *Corpus) requireConsistent() error {
	probs := c.Problems()
	if len(probs) == 0 {
		return nil
	}
	parts := make([]string, len(probs))
	for i, p := range probs {
		parts[i] = p.Key + " (" + p.Reason + ")"
	}
	return fmt.Errorf("%w: repair these entries first — %s; nothing was written, and the generated banlist block (if any) is left as it was",
		ErrClassMismatch, strings.Join(parts, "; "))
}

// StatusReport is the corpus as the bare verb shows it. It holds counts and keys'
// numbers only — no title, alias or author.
type StatusReport struct {
	Present      bool          `json:"present"`
	Dir          string        `json:"dir"`
	Confidential int           `json:"confidential"`
	Public       int           `json:"public"`
	Problems     []Problem     `json:"problems"`
	Ledgers      []LedgerCount `json:"ledgers"`
	Remotes      int           `json:"remotes"`
}

// LedgerCount is one ledger file's size.
type LedgerCount struct {
	Repo  string `json:"repo"`
	Lines int    `json:"lines"`
}

// Status reports the corpus at dir. An absent corpus is Present false and no error.
func Status(dir string) (StatusReport, error) {
	st := StatusReport{Dir: dir, Problems: []Problem{}, Ledgers: []LedgerCount{}}
	c, err := loadCorpus(dir)
	switch {
	case errors.Is(err, ErrNoCorpus):
		return st, nil
	case err != nil:
		return st, err
	}
	st.Present = true
	for _, e := range c.Entries {
		switch c.Class(e.ID) {
		case ClassConfidential:
			st.Confidential++
		case ClassPublic:
			st.Public++
		}
	}
	st.Problems = append(st.Problems, c.Problems()...)
	des, _ := os.ReadDir(filepath.Join(dir, LedgerDir))
	for _, de := range des {
		repo, ok := strings.CutSuffix(de.Name(), ".jsonl")
		if !ok || de.IsDir() || !repoRe.MatchString(repo) {
			continue
		}
		lines, _ := readLedger(dir, repo)
		st.Ledgers = append(st.Ledgers, LedgerCount{Repo: repo, Lines: len(lines)})
	}
	if out, err := corpusGit(dir, "remote"); err == nil && strings.TrimSpace(out) != "" {
		st.Remotes = len(strings.Fields(out))
	}
	return st, nil
}

// readme is the manual Init lays in a new corpus.
const readme = `# Sources corpus

Local-only. This directory is a git repository with no remote, and nothing in it
is ever committed to a project repository (adr-41).

- sources.json          CSL-JSON bibliography; each entry's "custom" block carries
                        confidential, permission_status, keywords, aliases,
                        ban_authors and file
- confidential/<key>/   original.<ext>, text.md, and anything derived from them
- public/<key>/         the same shape for a freely citable source
- ledger/<repo>.jsonl   append-only influence records, one file per repository

The folder a source sits in IS its classification. Maintain the corpus with
` + "`abcd source`" + ` (add, declassify, ledger, sync-banlist, cite-check); a hand
edit that leaves an entry and its folder disagreeing is refused by every step that
derives a ban from the corpus until it is repaired.

Durability is yours: back this directory up, and keep an offline
` + "`git bundle`" + ` snapshot; abcd cannot do either for you.
`

// Init creates a corpus at dir: the directory (0700), a no-remote git repository,
// an empty bibliography, and the manual, in one commit. An existing corpus is
// refused, as is a location inside another repository's working tree, where one
// `git add -A` would carry documents into it (adr-41).
func Init(dir string) (StatusReport, error) {
	if !filepath.IsAbs(dir) {
		return StatusReport{}, fmt.Errorf("%w: the corpus location must be an absolute path", ErrCorpusInvalid)
	}
	switch err := present(dir); {
	case err == nil:
		if _, lerr := loadCorpus(dir); lerr == nil {
			return StatusReport{}, fmt.Errorf("%w: a corpus already exists at the configured location", ErrCorpusInvalid)
		}
		if has, _ := fsutil.DirHasEntries(dir); has {
			return StatusReport{}, fmt.Errorf("%w: the location exists and is not empty; init only creates a corpus", ErrCorpusInvalid)
		}
	case !errors.Is(err, ErrNoCorpus):
		return StatusReport{}, err
	}
	if outer := gitutil.RepoShapedRoot(filepath.Dir(dir)); outer != "" {
		return StatusReport{}, fmt.Errorf("%w: the location is inside another git working tree, where a corpus is one `git add -A` from being committed; choose a location outside every repository", ErrCorpusInvalid)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return StatusReport{}, fmt.Errorf("%w: cannot create the corpus directory: %v", ErrCorpusInvalid, err)
	}
	if _, err := corpusGit(dir, "init", "-q"); err != nil {
		return StatusReport{}, fmt.Errorf("%w: git init failed: %v", ErrCorpusInvalid, err)
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, SourcesFile), []byte("[]\n"), 0o600); err != nil {
		return StatusReport{}, err
	}
	if err := fsutil.WriteFileAtomic(filepath.Join(dir, readmeFile), []byte(readme), 0o600); err != nil {
		return StatusReport{}, err
	}
	if err := commit(dir, "source: init corpus", SourcesFile, readmeFile); err != nil {
		return StatusReport{}, err
	}
	return Status(dir)
}

// withLock serialises writers of one corpus. The lock lives inside the corpus's
// git directory, which nothing tracks.
func withLock(dir string, fn func() error) error {
	err := fsutil.WithFileLock(filepath.Join(dir, ".git", lockFile), lockTimeout, fn)
	if errors.Is(err, fsutil.ErrLockContention) {
		return fmt.Errorf("%w: another process holds the corpus lock", ErrCorpusInvalid)
	}
	return err
}

// writeSources rewrites the bibliography from raw entries, atomically.
func writeSources(dir string, raw []json.RawMessage) error {
	if raw == nil {
		raw = []json.RawMessage{}
	}
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(filepath.Join(dir, SourcesFile), append(out, '\n'), 0o600)
}

// validPermission reports membership of the closed vocabulary.
func validPermission(p string) bool { return slices.Contains(Permissions, p) }
