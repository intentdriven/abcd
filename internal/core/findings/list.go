// Package findings is the list of findings abcd holds a session and a landing
// to (itd-2611251831138387, spc-2611252115253843): one JSON file for each
// finding under ~/.abcd.noindex/findings/<root-sha>/, in open/, private/ or
// settled/. The readings that put an entry here (an agent's block, the words
// of a closing text, a gate failing at a head) are the callers'; the hold and
// the landing read this list and not the readings, so a better reader changes
// neither.
//
// Every level is made and opened through fsutil's declaration scope, which
// judges each directory's owner and mode, and every file is read through
// ReadHomeDeclarationDenyingIn denying 0o077. A file is written whole by
// rename. A put and a settle each judge and write under the root's one lock
// file, so two writers cannot pass a bound together or lose an entry between
// them; a read takes no lock. Nothing is removed by age, by a new session or
// by the worktree going.
//
// Where a settle or a move to private/ was cut short, the entry stands both
// where it went and where it was; where it went wins, for every reader, and
// the stale twin is removed when a settle meets it.
//
// A private entry's text is read only through PrivateText, and only for the
// person: every other return of this package carries it blank.
package findings

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/capture"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// schemaVersion is the version every entry this build writes carries. A file
// of a later version is a newer abcd's: counted and shown by its handle, never
// settled here.
const schemaVersion = 1

// The bounds the store holds. The two a reading of a closing text holds (its
// bytes and its entries) are the reader's. None is exported: a caller meets
// each as the refusal that names it (ErrFull, ErrTooManyFiles, ErrTooLarge).
const (
	// maxFileBytes is one entry file: a larger one is never written, and one
	// found is unreadable, named.
	maxFileBytes = 4 << 10
	// maxTextBytes is an entry's text and a settlement's reason, cut on a rune.
	maxTextBytes = 600
	// maxOpen is the open entries of one root; past it nothing more is put.
	maxOpen = 1000
	// maxPrivate is the private entries of one root; past it nothing more is
	// put there.
	maxPrivate = 100
	// maxIDBytes and maxPathBytes cut every other string an entry carries.
	maxIDBytes   = 128
	maxPathBytes = 512
	// maxFilesRead is the files a hook's read takes: every open and private
	// entry the bounds allow, and as many again for the twins a cut-short move
	// leaves and the temporary files of a write cut short. Past it the list is
	// unreadable to that hook; a command names its own (ReadWithin).
	maxFilesRead = 2 * (maxOpen + maxPrivate)
	// lockWait is how long a put or a settle waits for the root's lock.
	lockWait = 10 * time.Second
	lockName = ".lock"
)

// Kind is what made an entry, and decides what clears it (clears).
type Kind string

// The closed set of kinds this build knows.
const (
	KindUntested Kind = "untested"
	KindHandedUp Kind = "handed-up"
	KindDoubt    Kind = "doubt"
	KindKnown    Kind = "known"
	KindLine     Kind = "line"
	KindCheck    Kind = "check"
	KindPrivate  Kind = "private"
)

// Act is what settles an entry. All but ActReceipt are the command's clearing
// flags; ActReceipt is a landing's preflight receipt naming the offered head,
// taken only through SettleByReceipt.
type Act string

// The acts.
const (
	ActCaptured    Act = "captured"
	ActSettled     Act = "settled"
	ActRefuted     Act = "refuted"
	ActOwn         Act = "own"
	ActNotAFinding Act = "not-a-finding"
	ActPrivate     Act = "private"
	ActReceipt     Act = "receipt"
)

// clears is the spec's kind table: the acts that clear each kind. A private
// entry's acts are the person's alone (SettleRequest.Person).
var clears = map[Kind][]Act{
	KindUntested: {ActCaptured, ActSettled, ActRefuted, ActPrivate},
	KindHandedUp: {ActCaptured, ActSettled, ActRefuted, ActPrivate},
	KindDoubt:    {ActCaptured, ActSettled},
	KindKnown:    {ActCaptured, ActSettled, ActRefuted, ActPrivate},
	KindLine:     {ActCaptured, ActSettled, ActRefuted, ActNotAFinding, ActPrivate},
	KindCheck:    {ActCaptured, ActSettled, ActOwn, ActPrivate, ActReceipt},
	KindPrivate:  {ActCaptured, ActSettled, ActRefuted},
}

// clearedBy reports whether act clears an entry of kind, as the table has it.
func clearedBy(kind Kind, act Act) bool {
	for _, a := range clears[kind] {
		if a == act {
			return true
		}
	}
	return false
}

// The refusals. Each is wrapped with what it names.
var (
	ErrFull         = errors.New("findings: the list is full")
	ErrTooManyFiles = errors.New("findings: more files than the read's budget; the list is unreadable")
	ErrTooLarge     = fmt.Errorf("findings: the entry would pass %d bytes with its text withheld; nothing written", maxFileBytes)
	ErrNewer        = errors.New("findings: the entry is a newer abcd's")
	ErrUnknownKind  = errors.New("findings: a kind this build does not know")
	ErrNotCleared   = errors.New("findings: that act does not clear the entry")
	ErrAmbiguous    = errors.New("findings: the handle names more than one entry")
	ErrNoEntry      = errors.New("findings: no entry has that handle")
	ErrUnreadable   = errors.New("findings: the entry cannot be read")
)

// textRedactor is what the store asks to redact a text. An error is a scan
// that did not run whole, before or during this text.
type textRedactor interface {
	Redact(string) (string, error)
}

// newRedactor builds capture's ledger redactor for one checkout; the seam a
// test replaces. An error, a scanner that cannot be made, runs degraded or
// lacks its configured augmenter, withholds the text.
var newRedactor = func(ctx context.Context, root string) (textRedactor, error) {
	return capture.NewTextRedactor(ctx, root)
}

// redactors builds one redactor for each checkout a call meets, once.
type redactors struct {
	ctx  context.Context
	made map[string]textRedactor
}

func newRedactors(ctx context.Context) *redactors {
	return &redactors{ctx: ctx, made: map[string]textRedactor{}}
}

// clean is text as stored, uncut, or withheld true when no full scan can
// clear it: capture's redactor for the checkout as given, termsafe, the home
// redacted. A redactor that cannot be built, or whose scan degrades, is
// dropped for the rest of the call, and so is every text once the caller's
// deadline has passed (the augmenter's own run is not bound by it).
func (r *redactors) clean(checkout, text string) (string, bool) {
	if r.ctx.Err() != nil {
		return "", true
	}
	red, ok := r.made[checkout]
	if !ok {
		made, err := newRedactor(r.ctx, checkout)
		if err != nil {
			// A failed build may hand back a typed nil; keep a true nil.
			made = nil
		}
		red, r.made[checkout] = made, made
	}
	if red == nil {
		return "", true
	}
	out, err := red.Redact(text)
	if err != nil || r.ctx.Err() != nil {
		r.made[checkout] = nil
		return "", true
	}
	return fsutil.RedactHome(termsafe.SanitizeBlock(out)), false
}

// cut bounds s at n bytes on a rune.
func cut(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return strings.ToValidUTF8(s[:n], "")
}

// id is an identifier as stored: one line, hidden runes masked, cut.
func id(s string) string { return cut(termsafe.Sanitize(s), maxIDBytes) }

// Entry is one finding. Name is the file's, never stored in it.
type Entry struct {
	SchemaVersion int    `json:"schema_version"`
	Kind          Kind   `json:"kind"`
	Session       string `json:"session,omitempty"`
	Agent         string `json:"agent,omitempty"`
	AgentType     string `json:"agent_type,omitempty"`
	// Checkout is the checkout the payload's directory is in; Place is that
	// checkout when a run's state names it as a lane's worktree, else "".
	Checkout string `json:"checkout,omitempty"`
	Place    string `json:"place,omitempty"`
	Run      string `json:"run,omitempty"`
	Lane     string `json:"lane,omitempty"`
	// Head names a gate's entry: set, the entry is named by kind, place and
	// head, so a failure at a new head is a new entry.
	Head string `json:"head,omitempty"`
	// Position is where in the transcript a reading took the text from; a
	// withheld reading's entry is named by it.
	Position string `json:"position,omitempty"`
	Text     string `json:"text,omitempty"`
	// Withheld is why no text was kept, or "".
	Withheld Withholding `json:"text_withheld,omitempty"`
	At       time.Time   `json:"at"`
	Settled  *Settlement `json:"settled,omitempty"`
	Name     string      `json:"-"`
}

// Withholding is why an entry's text was not kept, or "".
type Withholding string

// The two reasons a text is withheld.
const (
	// WithheldScan is a text no full scan could clear.
	WithheldScan Withholding = "scan"
	// WithheldSize is a cleared text that would take its file past the bound.
	WithheldSize Withholding = "size"
)

// WithheldLine is the sentence a stop or a list says in place of e's
// withheld text, or "" for a text kept. A withheld entry holds like any
// other.
func (e Entry) WithheldLine() string {
	switch e.Withheld {
	case WithheldScan:
		return "its text was withheld because the full scan could not run; fix the repository's scanner configuration (.abcd/config/pii.json, and .abcd/config/gitleaks.json where it configures the augmenter)"
	case WithheldSize:
		return "its text was withheld because the entry would pass its file's bound"
	}
	return ""
}

// Handle is the entry's name as it is shown: f- and eight characters.
func (e Entry) Handle() string { return handleOf(e.Name) }

func handleOf(name string) string {
	if len(name) < 8 {
		return "f-" + name
	}
	return "f-" + name[:8]
}

// shown is e as every return but PrivateText carries it.
func shown(e Entry) Entry {
	if e.Kind == KindPrivate {
		e.Text = ""
	}
	return e
}

// Settlement is what settled an entry: the act, the record it names, or the
// reason, test or run given.
type Settlement struct {
	Act    Act    `json:"act"`
	Record string `json:"record,omitempty"`
	Reason string `json:"reason,omitempty"`
	// ReasonWithheld is true where no full scan could clear the reason, or
	// it would take the file past its bound.
	ReasonWithheld bool      `json:"reason_withheld,omitempty"`
	At             time.Time `json:"at"`
}

// Store is one root's list under one home.
type Store struct{ home, root string }

var rootSHARe = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// NewStore is the list of the repository whose root commit is rootSHA, under
// home (abcdhome.MachineHome in production).
func NewStore(home, rootSHA string) (Store, error) {
	if !rootSHARe.MatchString(rootSHA) {
		return Store{}, fmt.Errorf("findings: %q is not a root commit's object name", rootSHA)
	}
	return Store{home: home, root: rootSHA}, nil
}

const (
	dirOpen    = "open"
	dirPrivate = "private"
	dirSettled = "settled"
)

func (s Store) rel(sub string) string { return abcdhome.Rel("findings", s.root, sub) }

func (s Store) ensure(sub string) (*os.Root, error) {
	return fsutil.EnsureHomeDeclarationScope(s.home, s.rel(sub), abcdhome.DirMode)
}

// open opens sub for reading; an absent directory is (nil, nil).
func (s Store) open(sub string) (*os.Root, error) {
	r, err := fsutil.OpenHomeDeclarationScope(s.home, s.rel(sub))
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return r, err
}

// locked runs fn holding the root's lock, with the three directories made.
func (s Store) locked(fn func(d dirs) error) error {
	top, err := fsutil.EnsureHomeDeclarationScope(s.home, abcdhome.Rel("findings", s.root), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer top.Close()
	return fsutil.WithFileLockIn(top, lockName, lockWait, func() error {
		d, err := s.dirs(true)
		if err != nil {
			return err
		}
		defer d.close()
		return fn(d)
	})
}

// dirs is the three directories, each opened (or made, with ensure) once.
type dirs map[string]*os.Root

func (s Store) dirs(ensure bool) (dirs, error) {
	d := dirs{}
	for _, sub := range []string{dirSettled, dirPrivate, dirOpen} {
		var r *os.Root
		var err error
		if ensure {
			r, err = s.ensure(sub)
		} else {
			r, err = s.open(sub)
		}
		if err != nil {
			d.close()
			return nil, err
		}
		d[sub] = r
	}
	return d, nil
}

func (d dirs) close() {
	for _, r := range d {
		if r != nil {
			r.Close()
		}
	}
}

func (d dirs) holds(sub, name string) bool {
	if d[sub] == nil {
		return false
	}
	_, err := d[sub].Lstat(name + ".json")
	return err == nil
}

// went is where an entry of sub went if a cut-short move left it in sub too:
// settled/ for an open or private one, private/ for an open one, or "".
func (d dirs) went(sub, name string) string {
	switch {
	case sub != dirSettled && d.holds(dirSettled, name):
		return dirSettled
	case sub == dirOpen && d.holds(dirPrivate, name):
		return dirPrivate
	}
	return ""
}

// live is the names in sub no other directory answers for.
func (d dirs) live(sub string) (map[string]bool, error) {
	out := map[string]bool{}
	if d[sub] == nil {
		return out, nil
	}
	names, _, err := listNames(d[sub], -1)
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		if d.went(sub, n) == "" {
			out[n] = true
		}
	}
	return out, nil
}

// nameOf is the entry's name. A gate's entry is named by its kind, place and
// head, withheld or not. A reading's is named by its kind, its redacted text
// before the cut, its place and its session; withheld, by where it came from
// (kind, session, agent and its position in the transcript), never its text,
// so a re-read under the same degraded scan is the same entry. A withheld
// entry with no position is named at random, and a repeat is its own entry.
func nameOf(e Entry, text string, withheld bool) (string, error) {
	var parts []string
	switch {
	case e.Head != "":
		parts = []string{"gate", string(e.Kind), e.Place, e.Head}
	case withheld && e.Position != "":
		parts = []string{"withheld", string(e.Kind), e.Session, e.Agent, e.Position}
	case withheld:
		return randomName()
	default:
		parts = []string{"reading", string(e.Kind), normalise(text), e.Place, e.Session}
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:]), nil
}

// randomName names a withheld entry with no position to name it by.
func randomName() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("findings: no name for a withheld entry: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}

func normalise(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

func subOf(k Kind) string {
	if k == KindPrivate {
		return dirPrivate
	}
	return dirOpen
}

// prepared is one entry made ready to write: its bytes encoded.
type prepared struct {
	e    Entry
	body []byte
}

// prepare bounds, redacts, names and encodes e. A cleared text that takes the
// file past its bound is withheld and the entry kept; one past it even so is
// ErrTooLarge.
func prepare(rs *redactors, e Entry) (prepared, error) {
	if _, ok := clears[e.Kind]; !ok {
		return prepared{}, fmt.Errorf("%w: %q", ErrUnknownKind, e.Kind)
	}
	e.SchemaVersion = schemaVersion
	e.Settled = nil
	e.Withheld = ""
	for _, f := range []*string{&e.Session, &e.Agent, &e.AgentType, &e.Run, &e.Lane, &e.Head, &e.Position} {
		*f = id(*f)
	}
	// The redactor is built for the checkout as given; only what is stored
	// is cut.
	text, withheld := rs.clean(e.Checkout, e.Text)
	e.Checkout, e.Place = cut(e.Checkout, maxPathBytes), cut(e.Place, maxPathBytes)
	name, err := nameOf(e, text, withheld)
	if err != nil {
		return prepared{}, err
	}
	e.Name = name
	e.Text = cut(text, maxTextBytes)
	if withheld {
		e.Withheld = WithheldScan
	}
	body, err := encode(e)
	if errors.Is(err, ErrTooLarge) && e.Text != "" {
		e.Text, e.Withheld = "", WithheldSize
		body, err = encode(e)
	}
	if err != nil {
		return prepared{}, err
	}
	return prepared{e: e, body: body}, nil
}

// encode is e as its file holds it, or ErrTooLarge past maxFileBytes. JSON's
// HTML escaping is off: it turns one byte of text into six.
func encode(e Entry) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(e); err != nil {
		return nil, err
	}
	if buf.Len() > maxFileBytes {
		return nil, fmt.Errorf("%w: %s", ErrTooLarge, e.Handle())
	}
	return buf.Bytes(), nil
}

func writeBody(dir *os.Root, name string, body []byte) error {
	return fsutil.WriteFileAtomicInRoot(dir, name+".json", body, abcdhome.FileMode)
}

// Put puts every entry on the list or none, and returns them as stored,
// building one redactor for each checkout the call meets. Every entry is
// made and encoded before any is written, and the bounds are judged and the
// files written under the root's lock. An entry already settled stays settled
// and is returned so, unless a receipt settled it; one the private list holds
// is returned from there; one already listed is written again whole.
func (s Store) Put(ctx context.Context, es ...Entry) ([]Entry, error) {
	rs := newRedactors(ctx)
	ps := make([]prepared, 0, len(es))
	for _, e := range es {
		p, err := prepare(rs, e)
		if err != nil {
			return nil, err
		}
		ps = append(ps, p)
	}
	out := make([]Entry, len(ps))
	err := s.locked(func(d dirs) error {
		live := map[string]map[string]bool{}
		for _, sub := range []string{dirOpen, dirPrivate} {
			l, err := d.live(sub)
			if err != nil {
				return err
			}
			live[sub] = l
		}
		var writes []prepared
		var reopen []string
		for i, p := range ps {
			prior, err := readOne(s, d[dirSettled], dirSettled, p.e.Name)
			switch {
			case err == nil && prior.Settled != nil && prior.Settled.Act != ActReceipt:
				out[i] = shown(prior)
				continue
			case err != nil && !errors.Is(err, fs.ErrNotExist):
				return err
			}
			receiptSettled := err == nil
			if p.e.Kind != KindPrivate && !receiptSettled {
				if held, err := readOne(s, d[dirPrivate], dirPrivate, p.e.Name); err == nil {
					out[i] = shown(held)
					continue
				}
			}
			sub := subOf(p.e.Kind)
			live[sub][p.e.Name] = true
			writes = append(writes, p)
			if receiptSettled {
				reopen = append(reopen, p.e.Name)
			}
			out[i] = shown(p.e)
		}
		if n := len(live[dirOpen]); n > maxOpen {
			return fmt.Errorf("%w: the open list would hold %d entries, past its bound of %d", ErrFull, n, maxOpen)
		}
		if n := len(live[dirPrivate]); n > maxPrivate {
			return fmt.Errorf("%w: the private list would hold %d entries, past its bound of %d", ErrFull, n, maxPrivate)
		}
		for _, p := range writes {
			if err := writeBody(d[subOf(p.e.Kind)], p.e.Name, p.body); err != nil {
				return err
			}
		}
		// Written open first, so a crash here leaves the settled one winning
		// and the put to be made again, never neither.
		for _, name := range reopen {
			if err := d[dirSettled].Remove(name + ".json"); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

var entryFileRe = regexp.MustCompile(`^([0-9a-f]{64})\.json$`)

// listNames is the entry names in dir and how many directory entries were
// read for them, at most limit (-1 for every one). A stray file counts
// against the limit as an entry does.
func listNames(dir *os.Root, limit int) ([]string, int, error) {
	f, err := dir.Open(".")
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()
	ents, err := f.ReadDir(limit)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, 0, err
	}
	var out []string
	for _, d := range ents {
		if m := entryFileRe.FindStringSubmatch(d.Name()); m != nil {
			out = append(out, m[1])
		}
	}
	return out, len(ents), nil
}

// readOne reads one entry of sub through dir (nil: absent). An absent file
// is fs.ErrNotExist; a refused one is ErrUnreadable naming the guard, the
// home in tilde form; a later schema or an unknown kind ErrNewer.
func readOne(s Store, dir *os.Root, sub, name string) (Entry, error) {
	if dir == nil {
		return Entry{}, fs.ErrNotExist
	}
	raw, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, s.home, s.rel(sub)+"/"+name+".json", maxFileBytes, 0o077)
	if refusal == fsutil.DeclarationAbsent {
		return Entry{}, fs.ErrNotExist
	}
	if err != nil {
		return Entry{}, fmt.Errorf("%w: %s", ErrUnreadable, fsutil.RedactHome(err.Error()))
	}
	var head struct {
		SchemaVersion int  `json:"schema_version"`
		Kind          Kind `json:"kind"`
	}
	if err := json.Unmarshal(raw, &head); err != nil || head.SchemaVersion < 1 {
		return Entry{}, fmt.Errorf("%w: not an entry of any schema", ErrUnreadable)
	}
	if _, known := clears[head.Kind]; head.SchemaVersion > schemaVersion || !known {
		return Entry{}, ErrNewer
	}
	var e Entry
	if err := json.Unmarshal(raw, &e); err != nil {
		return Entry{}, fmt.Errorf("%w: %v", ErrUnreadable, err)
	}
	e.Name = name
	return e, nil
}

// Unreadable is a file a read refused, by handle and why.
type Unreadable struct {
	Handle string `json:"handle"`
	Reason string `json:"reason"`
}

// View is the list as one read found it. Private entries carry no text.
type View struct {
	Open       []Entry      `json:"open"`
	Private    []Entry      `json:"private"`
	Newer      []string     `json:"newer"`
	Unreadable []Unreadable `json:"unreadable"`
	// Full is true once the open entries reach their bound.
	Full bool `json:"full"`
}

// Ordered is the open entries with checkout's first, each group by time.
func (v View) Ordered(checkout string) []Entry {
	out := append([]Entry(nil), v.Open...)
	sort.SliceStable(out, func(i, j int) bool {
		hi, hj := out[i].Checkout == checkout, out[j].Checkout == checkout
		if hi != hj {
			return hi
		}
		if !out[i].At.Equal(out[j].At) {
			return out[i].At.Before(out[j].At)
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// Read is the list as a hook reads it, within maxFilesRead files.
func (s Store) Read() (View, error) { return s.ReadWithin(maxFilesRead) }

// ReadWithin is the open and private entries, reading at most files directory
// entries across the two, strays included (past it, ErrTooManyFiles). An
// absent store is an empty view, and nothing is made. A directory refused is
// the error; a file refused is named in Unreadable.
func (s Store) ReadWithin(files int) (View, error) {
	var v View
	d, err := s.dirs(false)
	if err != nil {
		return View{}, err
	}
	defer d.close()
	budget := files
	for _, sub := range []string{dirOpen, dirPrivate} {
		if d[sub] == nil {
			continue
		}
		names, seen, err := listNames(d[sub], budget+1)
		if err == nil && seen > budget {
			err = fmt.Errorf("%w (%d)", ErrTooManyFiles, files)
		}
		if err != nil {
			return View{}, err
		}
		budget -= seen
		live := 0
		for _, name := range names {
			if d.went(sub, name) != "" {
				continue
			}
			live++
			e, err := readOne(s, d[sub], sub, name)
			switch {
			case errors.Is(err, ErrNewer):
				v.Newer = append(v.Newer, handleOf(name))
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				v.Unreadable = append(v.Unreadable, Unreadable{Handle: handleOf(name), Reason: err.Error()})
			case sub == dirPrivate:
				v.Private = append(v.Private, shown(e))
			default:
				v.Open = append(v.Open, shown(e))
			}
		}
		if sub == dirOpen && live >= maxOpen {
			v.Full = true
		}
	}
	return v, nil
}

// SettledView is the last n settled entries, newest first by the time each
// file was written, and every file read on the way that could not be.
func (s Store) SettledView(n int) ([]Entry, []Unreadable, error) {
	dir, err := s.open(dirSettled)
	if err != nil || dir == nil {
		return nil, nil, err
	}
	defer dir.Close()
	names, _, err := listNames(dir, -1)
	if err != nil {
		return nil, nil, err
	}
	type dated struct {
		name string
		at   time.Time
	}
	var order []dated
	for _, name := range names {
		if fi, err := dir.Lstat(name + ".json"); err == nil {
			order = append(order, dated{name, fi.ModTime()})
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].at.After(order[j].at) })
	var out []Entry
	var bad []Unreadable
	for _, o := range order {
		if len(out) == n {
			break
		}
		e, err := readOne(s, dir, dirSettled, o.name)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case errors.Is(err, ErrNewer):
			bad = append(bad, Unreadable{Handle: handleOf(o.name), Reason: "a newer abcd's"})
		case err != nil:
			bad = append(bad, Unreadable{Handle: handleOf(o.name), Reason: err.Error()})
		case e.Settled == nil:
			bad = append(bad, Unreadable{Handle: handleOf(o.name), Reason: "settled/ holds an entry with no settlement"})
		default:
			out = append(out, shown(e))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Settled.At.After(out[j].Settled.At) })
	return out, bad, nil
}

// Answer is what a record says to a claim on it: whether it holds, the state
// found, and where it was found.
type Answer struct {
	Holds bool   `json:"holds"`
	State string `json:"state"`
	Where string `json:"where,omitempty"`
}

// Answerer answers a captured or settled claim on a record id; *Records is
// the one production uses.
type Answerer interface {
	Answer(act Act, id string) (Answer, error)
}

// SettleRequest settles entries by handle with one act.
type SettleRequest struct {
	Handles []string
	Act     Act
	// Record is the record a captured or settled act names, judged by Records.
	Record  string
	Records Answerer
	// Reason is the reason, test or run a not-a-finding, refuted or own act
	// gives, redacted against each entry's own checkout.
	Reason string
	// Person is true where the caller is a person at a terminal, the one
	// caller that may settle a private entry.
	Person bool
	At     time.Time
}

var handleRe = regexp.MustCompile(`^f-([0-9a-f]{8,64})$`)

type located struct {
	e   Entry
	sub string
}

// Settle settles every handle or none: each handle, the act and its record are
// judged first, and every file is encoded before any is written, all under the
// root's lock; a refusal writes nothing. A handle given twice is one entry.
// Each settled entry is written to settled/ and then removed from where it
// was. ActReceipt is refused here: SettleByReceipt is its one entry point.
func (s Store) Settle(ctx context.Context, req SettleRequest) ([]Entry, error) {
	if req.Act == ActReceipt {
		return nil, fmt.Errorf("%w: a receipt settles through SettleByReceipt alone", ErrNotCleared)
	}
	return s.settle(ctx, req)
}

// SettleByReceipt settles check entries by a preflight receipt naming the
// offered head: the landing's act, never a command's.
func (s Store) SettleByReceipt(ctx context.Context, handles []string, receipt string, at time.Time) ([]Entry, error) {
	return s.settle(ctx, SettleRequest{Handles: handles, Act: ActReceipt, Reason: receipt, At: at})
}

func (s Store) settle(ctx context.Context, req SettleRequest) ([]Entry, error) {
	if len(req.Handles) == 0 {
		return nil, fmt.Errorf("%w: no handle given", ErrNoEntry)
	}
	switch req.Act {
	case ActCaptured, ActSettled:
		if req.Record == "" || req.Records == nil {
			return nil, fmt.Errorf("%w: --%s names a record", ErrNotCleared, req.Act)
		}
	case ActRefuted, ActOwn, ActNotAFinding, ActReceipt:
		if strings.TrimSpace(req.Reason) == "" {
			return nil, fmt.Errorf("%w: --%s takes its reason, test or run", ErrNotCleared, req.Act)
		}
	case ActPrivate:
	default:
		return nil, fmt.Errorf("%w: %q is not an act", ErrNotCleared, req.Act)
	}
	var out []Entry
	err := s.locked(func(d dirs) error {
		var found []located
		seen := map[string]bool{}
		for _, h := range req.Handles {
			l, err := s.locate(d, h, true)
			if err != nil {
				return err
			}
			if !clearedBy(l.e.Kind, req.Act) || l.e.Kind == KindPrivate && !req.Person {
				return fmt.Errorf("%w: %s is %s, which --%s does not clear", ErrNotCleared, h, l.e.Kind, req.Act)
			}
			if !seen[l.e.Name] {
				seen[l.e.Name] = true
				found = append(found, l)
			}
		}
		if req.Act == ActCaptured || req.Act == ActSettled {
			ans, err := req.Records.Answer(req.Act, req.Record)
			if err != nil {
				return err
			}
			if !ans.Holds {
				return fmt.Errorf("%w: %q is %s on %s, which does not answer --%s", ErrNotCleared, termsafe.Sanitize(req.Record), ans.State, orNowhere(ans.Where), req.Act)
			}
		}
		if req.Act == ActPrivate {
			live, err := d.live(dirPrivate)
			if err != nil {
				return err
			}
			for _, l := range found {
				live[l.e.Name] = true
			}
			if len(live) > maxPrivate {
				return fmt.Errorf("%w: the private list would hold %d entries, past its bound of %d", ErrFull, len(live), maxPrivate)
			}
		}
		rs := newRedactors(ctx)
		type move struct {
			e        Entry
			from, to string
			body     []byte
		}
		var moves []move
		for _, l := range found {
			e := l.e
			to := dirSettled
			if req.Act == ActPrivate {
				e.Kind, to = KindPrivate, dirPrivate
			} else {
				reason, withheld := rs.clean(e.Checkout, req.Reason)
				e.Settled = &Settlement{Act: req.Act, Record: id(req.Record), Reason: cut(reason, maxTextBytes), ReasonWithheld: withheld, At: req.At}
			}
			body, err := encode(e)
			if errors.Is(err, ErrTooLarge) && e.Settled != nil && e.Settled.Reason != "" {
				e.Settled.Reason, e.Settled.ReasonWithheld = "", true
				body, err = encode(e)
			}
			if err != nil {
				return err
			}
			moves = append(moves, move{e: e, from: l.sub, to: to, body: body})
		}
		for _, m := range moves {
			if err := writeBody(d[m.to], m.e.Name, m.body); err != nil {
				return err
			}
			if err := d[m.from].Remove(m.e.Name + ".json"); err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			out = append(out, shown(m.e))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func orNowhere(where string) string {
	if where == "" {
		return "no branch or checkout read"
	}
	return where
}

// PrivateText is the text of the private entry handle names, and whether it
// was withheld: the one place this package hands a private text back, and to
// the person alone.
func (s Store) PrivateText(handle string, person bool) (text string, withheld bool, err error) {
	if !person {
		return "", false, fmt.Errorf("%w: a private entry's text is the person's alone", ErrNotCleared)
	}
	d, err := s.dirs(false)
	if err != nil {
		return "", false, err
	}
	defer d.close()
	l, err := s.locate(d, handle, false)
	if err != nil {
		return "", false, err
	}
	if l.e.Kind != KindPrivate {
		return "", false, fmt.Errorf("%w: %s is not on the private list", ErrNoEntry, handle)
	}
	return l.e.Text, l.e.Withheld != "", nil
}

// locate finds the one open or private entry handle names. A name a cut-short
// move left behind is passed by, as Read passes it, and with prune (under the
// lock) its stale twin is removed.
func (s Store) locate(d dirs, handle string, prune bool) (located, error) {
	m := handleRe.FindStringSubmatch(handle)
	if m == nil {
		return located{}, fmt.Errorf("%w: %q is not a handle (f- and eight characters)", ErrNoEntry, termsafe.Sanitize(handle))
	}
	var hits []located
	var newer bool
	for _, sub := range []string{dirOpen, dirPrivate} {
		if d[sub] == nil {
			continue
		}
		names, _, err := listNames(d[sub], -1)
		if err != nil {
			return located{}, err
		}
		for _, name := range names {
			if !strings.HasPrefix(name, m[1]) {
				continue
			}
			if d.went(sub, name) != "" {
				if prune {
					_ = d[sub].Remove(name + ".json")
				}
				continue
			}
			e, err := readOne(s, d[sub], sub, name)
			switch {
			case errors.Is(err, ErrNewer):
				newer = true
				hits = append(hits, located{})
			case errors.Is(err, fs.ErrNotExist):
			case err != nil:
				return located{}, fmt.Errorf("%s: %w", handle, err)
			default:
				hits = append(hits, located{e: e, sub: sub})
			}
		}
	}
	switch {
	case len(hits) > 1:
		return located{}, fmt.Errorf("%w: %s", ErrAmbiguous, handle)
	case newer:
		return located{}, fmt.Errorf("%w: %s; no command of this build settles it", ErrNewer, handle)
	case len(hits) == 0:
		return located{}, fmt.Errorf("%w: %s", ErrNoEntry, handle)
	}
	return hits[0], nil
}
