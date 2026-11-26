package landing

// A person's own facilitator mode for one project (itd-2611252049031692,
// dcn-2611252132144530). Where the project's default branch commits no
// facilitator.mode, it leaves the mode to each contributor, and the person's
// own entry for this project decides before the computer's file does.
//
// The entries are one file, ~/.abcd.noindex/facilitator.json, under the home
// the account database names for this account (AccountHome), never $HOME
// where the two differ: under another HOME, a hosted session's among them, a
// person has no setting, and the computer's file is not read for the mode
// either. The file is read with a home declaration's guards (a regular file
// this account owns that no one else can write, within its cap, with no
// symlinked folder above it), and is decoded strictly; a file that is there
// and refused reads human where the project leaves it to each, and is said
// beside a project value that stands.
//
// An entry is selected by the identity record alone (identity.ReadRecord):
// its email names the person, and git's would-be author and committer here
// must be the record's, or the mode is human and the reading names the
// difference. The project is known by its first commits, every parentless
// commit reachable from the default branch's tip, and by its whole address.
// An entry of human matches where its root is any of the first commits; an
// entry of auto only where the project has exactly one first commit, the
// entry's, its whole address is the entry's, and nothing in the environment
// sets git's identity. So a fork, which carries the same first commit at
// another address, shares a person's human and not their auto. Where the
// file holds entries and the first commits cannot be established, the mode is
// human, and the computer's file is not read in its place.
//
// The merge gate reads a person's human; a person's own auto arms only a pull
// request their own account opened (the intent's decision 7), which the gate
// does not yet read, so there it reads human and says so.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/identity"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// AccountHome is the home the account database names for this account
// (gitutil.AccountHome), where a person's own setting and the computer's file
// are read for the mode. A test names its own HOME.
var AccountHome = gitutil.AccountHome

// personLayer is the layer name a person's own entry decides from.
const personLayer = "person"

// personLeaf is the person's file, and its caps and schema.
const (
	personLeaf     = "facilitator.json"
	personSchema   = 1
	maxPersonBytes = 64 << 10
)

// personEntry is one entry: whose (the identity record's email), for which
// project (the default branch's root commit and the repository's name), the
// mode, and when it was set.
type personEntry struct {
	Email string    `json:"email"`
	Root  string    `json:"root"`
	Forge string    `json:"forge"`
	Mode  string    `json:"mode"`
	SetAt time.Time `json:"set_at"`
}

type personFile struct {
	Schema  int           `json:"schema_version"`
	Entries []personEntry `json:"entries"`
}

// What git's identity in the checkout was against the identity record when a
// reading was taken, as the kept reading keeps it.
const (
	gitIsRecord = "record"
	gitIsOther  = "other"
)

// foreignHomeLayer is the layer a reading under a HOME that is not this
// account's own decides from where the project leaves the mode to each: no
// layer but the project's is read there, so the mode is human.
const foreignHomeLayer = "home"

// projectFacts is what selects an entry for one project, as a reading found
// it and the kept reading keeps it: every first commit (parentless commit)
// reachable from the default branch's tip, or why they could not be
// established (unread); the repository's whole address (addressOf, "" for
// none); and whether the environment sets git's identity.
type projectFacts struct {
	roots    []string
	unread   string
	address  string
	envIdent bool
}

// projectKey is how one reading comes by the facts: facts answers them,
// asked only once a file with entries needs them (nil where the reading is
// not at a commit); ident asks git's identity against a record (nil where git
// is not asked, as on the row, which takes gitIs from the kept reading); gate
// is the merge gate's reading, which holds a person's own auto back.
type projectKey struct {
	facts func() (projectFacts, error)
	ident func(rec identity.Record) (diff string, err error)
	gitIs string
	gate  bool
}

// personReading is what a person's own setting says for one project: the
// mode ("" for no setting), the origin it is said with, whether it was
// withheld (refused, or not this person's to use here), git's identity
// against the record as found, and the project's facts where an entry needed
// them, for the kept reading.
type personReading struct {
	mode, origin string
	withheld     bool
	// note is a refusal of the person's file, said beside a project value.
	note  string
	gitIs string
	facts *projectFacts
}

// homeIsAccounts says whether home is the account database's home for this
// account, after every link is resolved on both sides. A home that cannot be
// told is not.
func homeIsAccounts(home string) bool {
	if home == "" {
		return false
	}
	acct, err := AccountHome()
	if err != nil || acct == "" {
		return false
	}
	a, errA := filepath.EvalSymlinks(acct)
	h, errH := filepath.EvalSymlinks(home)
	return errA == nil && errH == nil && a == h
}

// foreignHomeOrigin says why neither a person's own setting nor the
// computer's file is read for the mode.
func foreignHomeOrigin() string {
	return "HOME is not the home the account database names for this account, so neither your own setting nor this computer's file is read for " + KeyFacilitatorMode + " here, on purpose; run abcd with your own HOME to read them"
}

// personDisplay is the person's file in its tilde form.
func personDisplay() string { return abcdhome.Display(personLeaf) }

// readPersonFile reads the person's file under home: ok false with no error
// when it is not there; an error, naming the file, when it is there and
// refused.
func readPersonFile(home string) (personFile, bool, error) {
	raw, refusal, err := fsutil.ReadHomeDeclaration(home, abcdhome.Rel(personLeaf), maxPersonBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return personFile{}, false, nil
	default:
		why := "it cannot be trusted or read"
		if err != nil {
			why = fsutil.RedactHome(err.Error())
		}
		return personFile{}, false, fmt.Errorf("%s is refused (%s); it must be a regular file you own that no other account can write, within %d KiB", personDisplay(), why, maxPersonBytes>>10)
	}
	var f personFile
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&f); err != nil {
		return personFile{}, false, fmt.Errorf("%s is refused: it is not the file's JSON shape (%v)", personDisplay(), err)
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return personFile{}, false, fmt.Errorf("%s is refused: it holds more than one JSON value", personDisplay())
	}
	if f.Schema != personSchema {
		return personFile{}, false, fmt.Errorf("%s is refused: schema_version %d is not %d", personDisplay(), f.Schema, personSchema)
	}
	for i, e := range f.Entries {
		if err := checkMode(e.Mode); err != nil {
			return personFile{}, false, fmt.Errorf("%s is refused: entry %d: %v; it takes human or auto", personDisplay(), i+1, err)
		}
		if strings.TrimSpace(e.Email) == "" || !gitutil.IsFullSHA(e.Root) {
			return personFile{}, false, fmt.Errorf("%s is refused: entry %d names no email or no full root commit", personDisplay(), i+1)
		}
	}
	return f, true, nil
}

// readPerson is a person's own setting for the project key names, from the
// file under home (the account's own, which the caller has judged). The
// project's facts are asked only once the file holds entries; an error is a
// deadline or cancellation met while asking them, which no reading stands on.
func readPerson(home string, key projectKey) (personReading, error) {
	f, ok, err := readPersonFile(home)
	if err != nil {
		return personReading{mode: ModeHuman, origin: err.Error(), withheld: true, note: err.Error()}, nil
	}
	if !ok || len(f.Entries) == 0 {
		return personReading{}, nil
	}
	facts := projectFacts{unread: "the project's default branch is not read at a commit here"}
	if key.facts != nil {
		if facts, err = key.facts(); err != nil {
			return personReading{}, err
		}
	}
	p := personReading{facts: &facts}
	withhold := func(why string) (personReading, error) {
		p.mode, p.withheld, p.origin = ModeHuman, true, why
		return p, nil
	}
	if facts.unread == "" && len(facts.roots) == 0 {
		facts.unread = "git named no first commit of the project"
	}
	if facts.unread != "" {
		return withhold(facts.unread + ", so no entry of " + personDisplay() + " is used here and the mode is human")
	}
	rec, ref, err := recordFor(home, facts.roots)
	switch {
	case errors.Is(err, errNoRecord):
		return withhold(personDisplay() + " holds entries, and no identity record says who you are here, so none is selected; record one with abcd ahoy --identity")
	case err != nil:
		return withhold("your identity record cannot be read (" + fsutil.RedactHome(err.Error()) + "), so no entry of " + personDisplay() + " is selected here; fix it with abcd ahoy --identity")
	}
	e, ok := selectEntry(f.Entries, rec.Email, facts)
	if !ok {
		return p, nil
	}
	gitIs, diff := key.gitIs, ""
	switch {
	case gitIs != "":
	case key.ident == nil:
		gitIs, diff = gitIsOther, "git's identity here was not read with the last reading, which the next prompt takes"
	default:
		d, err := key.ident(rec)
		switch {
		case errors.Is(err, context.DeadlineExceeded), errors.Is(err, context.Canceled):
			return personReading{}, err
		case err != nil:
			gitIs, diff = gitIsOther, "git's identity here could not be read ("+fsutil.RedactHome(err.Error())+")"
		case d != "":
			gitIs, diff = gitIsOther, d
		default:
			gitIs = gitIsRecord
		}
	}
	p.gitIs = gitIs
	if gitIs != gitIsRecord {
		if diff == "" {
			diff = "git's identity here was not your identity record's when it was last read"
		}
		return withhold(diff + ", so your own setting for this project is not used (your identity record is " + ref.Display + ")")
	}
	if e.Mode == ModeAuto && facts.envIdent {
		return withhold("variables in this environment set git's identity (GIT_AUTHOR_* or GIT_COMMITTER_*), so your own auto for this project is not used here")
	}
	if key.gate && e.Mode == ModeAuto {
		return withhold("your own auto in " + personDisplay() + " arms only a pull request your own account opened, which this gate does not read yet, so it waits as for human")
	}
	p.mode, p.origin = e.Mode, "set in "+personDisplay()+" on "+e.SetAt.UTC().Format(time.DateOnly)
	return p, nil
}

// errNoRecord says no identity record applies to the project.
var errNoRecord = errors.New("no identity record")

// recordFor is the identity record that applies to a project whose first
// commits are roots: the one record every root reads (identity.ReadRecord),
// errNoRecord where none does, and an error where they name different people
// or one is refused.
func recordFor(home string, roots []string) (identity.Record, identity.RecordRef, error) {
	var rec identity.Record
	var ref identity.RecordRef
	have := false
	for _, root := range roots {
		r, rf, found, err := identity.ReadRecord(home, root)
		if err != nil {
			return identity.Record{}, identity.RecordRef{}, err
		}
		if !found {
			continue
		}
		if have && (r.Name != rec.Name || !strings.EqualFold(r.Email, rec.Email)) {
			return identity.Record{}, identity.RecordRef{}, errors.New("the identity records for this project's first commits name different people")
		}
		rec, ref, have = r, rf, true
	}
	if !have {
		return identity.Record{}, identity.RecordRef{}, errNoRecord
	}
	return rec, ref, nil
}

// selectEntry is the entry for email in the project facts describe: an entry
// of human where its root is any of the project's first commits; an entry of
// auto only where the project has exactly one first commit, the entry's, and
// the repository's whole address is the entry's. With both, human is taken.
func selectEntry(es []personEntry, email string, facts projectFacts) (personEntry, bool) {
	var auto *personEntry
	for i, e := range es {
		if !strings.EqualFold(strings.TrimSpace(e.Email), strings.TrimSpace(email)) {
			continue
		}
		switch {
		case e.Mode == ModeHuman && slices.Contains(facts.roots, e.Root):
			return e, true
		case e.Mode == ModeAuto && auto == nil && len(facts.roots) == 1 && facts.roots[0] == e.Root &&
			facts.address != "" && e.Forge == facts.address:
			auto = &es[i]
		}
	}
	if auto != nil {
		return *auto, true
	}
	return personEntry{}, false
}

// keyFor is the project key of the reading r names: the facts of its tip
// (r.RepoAt), asked of git once and only when an entry needs them, and git's
// identity in r.Repo. A repository layer that is not read at a commit has no
// facts, so where the file holds entries the mode is human.
func keyFor(ctx context.Context, r layered.Roots, gate bool) projectKey {
	k := projectKey{gate: gate}
	if r.Repo == "" || !gitutil.IsFullSHA(r.RepoAt) {
		return k
	}
	repo, tip := r.Repo, r.RepoAt
	var once sync.Once
	var facts projectFacts
	var err error
	k.facts = func() (projectFacts, error) {
		once.Do(func() { facts, err = factsOf(ctx, repo, tip) })
		return facts, err
	}
	k.ident = func(rec identity.Record) (string, error) { return gitIdentDiff(ctx, repo, rec) }
	return k
}

// maxRootBytes caps the list of the project's first commits.
var maxRootBytes = 1 << 20

// listRoots lists every parentless commit reachable from tip, one per line,
// as stored (replace refs off). A test substitutes it.
var listRoots = func(ctx context.Context, repo, tip string) (string, error) {
	return gitutil.RunCappedContext(ctx, repo, maxRootBytes, "-c", "core.useReplaceRefs=false", "rev-list", "--max-parents=0", tip, "--")
}

// ctxErr is the error of a context that ended, from ctx or as err carries
// it, nil where none did.
func ctxErr(ctx context.Context, err error) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

// factsOf is the facts of the project whose default branch's tip is tip in
// repo. Where its first commits cannot be established (an environment or a
// repository that rewrites where history begins, a shallow clone, git failing
// or answering past its cap or with nothing), unread says why; a context that
// ends is an error.
func factsOf(ctx context.Context, repo, tip string) (projectFacts, error) {
	f := projectFacts{envIdent: envSetsIdent()}
	for _, name := range []string{"GIT_GRAFT_FILE", "GIT_SHALLOW_FILE"} {
		if os.Getenv(name) != "" {
			f.unread = "this environment sets " + name + ", which changes where git's history begins"
			return f, nil
		}
	}
	common, ok := commonGitDir(repo)
	if !ok {
		f.unread = "git's directory for this checkout could not be read"
		return f, nil
	}
	if _, err := os.Lstat(filepath.Join(common, "info", "grafts")); !errors.Is(err, fs.ErrNotExist) {
		f.unread = "this repository declares grafts, which change where its history begins"
		return f, nil
	}
	out, err := gitutil.RunCappedContext(ctx, repo, 64, "rev-parse", "--is-shallow-repository")
	if e := ctxErr(ctx, err); e != nil {
		return f, e
	}
	switch {
	case err != nil:
		f.unread = "git could not say whether this clone is shallow (" + fsutil.RedactHome(err.Error()) + ")"
		return f, nil
	case strings.TrimSpace(out) == "true":
		f.unread = "this clone is shallow, so the project's first commits are not all here"
		return f, nil
	case strings.TrimSpace(out) != "false":
		f.unread = "git did not say whether this clone is shallow"
		return f, nil
	}
	out, err = listRoots(ctx, repo, tip)
	if e := ctxErr(ctx, err); e != nil {
		return f, e
	}
	if err != nil {
		f.unread = "git could not list the project's first commits (" + fsutil.RedactHome(err.Error()) + ")"
		return f, nil
	}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}
		if !gitutil.IsFullSHA(line) {
			f.unread = "git's list of the project's first commits holds something that is not a commit"
			return f, nil
		}
		f.roots = append(f.roots, line)
	}
	if len(f.roots) == 0 {
		f.unread = "git named no first commit of the project"
		return f, nil
	}
	out, err = gitutil.RunCappedContext(ctx, repo, 4096, "config", "--get", "remote."+Remote+".url")
	if e := ctxErr(ctx, err); e != nil {
		return f, e
	}
	if err == nil {
		f.address = addressOf(out)
	}
	return f, nil
}

// envSetsIdent says whether any GIT_AUTHOR_* or GIT_COMMITTER_* variable is
// set in this process's environment.
func envSetsIdent() bool {
	for _, kv := range os.Environ() {
		k, _, _ := strings.Cut(kv, "=")
		if strings.HasPrefix(k, "GIT_AUTHOR_") || strings.HasPrefix(k, "GIT_COMMITTER_") {
			return true
		}
	}
	return false
}

// addressOf is a repository's whole address from any of git's spellings of
// it (https://host/path.git, ssh://user@host/path, user@host:path): the host
// lower-cased, a slash, and the full path, with .git and any leading or
// trailing slash dropped; "" where it names no host or no path.
func addressOf(raw string) string {
	raw = strings.TrimSpace(raw)
	var host, p string
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil || u.Host == "" {
			return ""
		}
		host, p = u.Host, u.Path
	} else {
		before, rest, ok := strings.Cut(raw, ":")
		if !ok || before == "" || strings.Contains(before, "/") {
			return ""
		}
		if at := strings.LastIndex(before, "@"); at >= 0 {
			before = before[at+1:]
		}
		host, p = before, rest
	}
	host = strings.ToLower(host)
	p = strings.TrimRight(strings.TrimSuffix(strings.Trim(p, "/"), ".git"), "/")
	if host == "" || p == "" {
		return ""
	}
	return host + "/" + p
}

// gitIdentDiff says how git's would-be author and committer in repo differ
// from rec, "" where both are rec's. git is asked as the person's own git
// commits (gitutil.WorkingTreeCommandContext): with their global and system
// settings, where most people keep their identity, which the isolated
// environment reads as absent; `git var` reads settings and starts nothing.
// A context that ends is its error.
func gitIdentDiff(ctx context.Context, repo string, rec identity.Record) (string, error) {
	for _, role := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		out, err := personGitVar(ctx, repo, role)
		if e := ctxErr(ctx, err); e != nil {
			return "", e
		}
		if err != nil {
			return "", err
		}
		name, email := parseIdent(out)
		if name != rec.Name || !strings.EqualFold(email, rec.Email) {
			who := "author"
			if role == "GIT_COMMITTER_IDENT" {
				who = "commit"
			}
			return fmt.Sprintf("git would %s here as %s <%s>, not as your identity record's %s <%s>", who, name, email, rec.Name, rec.Email), nil
		}
	}
	return "", nil
}

// maxIdentBytes caps one of git's identity lines.
const maxIdentBytes = 4096

// personGitVar is `git var role` in repo as the person's own git answers it,
// its output past maxIdentBytes an error.
func personGitVar(ctx context.Context, repo, role string) (string, error) {
	cmd := gitutil.WorkingTreeCommandContext(ctx, repo, "var", role)
	var out bytes.Buffer
	cmd.Stdout = &capped{w: &out, left: maxIdentBytes + 1}
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git var %s: %w", role, err)
	}
	if out.Len() > maxIdentBytes {
		return "", fmt.Errorf("git var %s: the answer ran past %d bytes", role, maxIdentBytes)
	}
	return out.String(), nil
}

// capped keeps the first left bytes written to it and drops the rest.
type capped struct {
	w    io.Writer
	left int
}

func (c *capped) Write(p []byte) (int, error) {
	n := len(p)
	if c.left > 0 {
		k := min(len(p), c.left)
		c.left -= k
		if _, err := c.w.Write(p[:k]); err != nil {
			return 0, err
		}
	}
	return n, nil
}

// parseIdent is the name and email of one of git's identity lines.
func parseIdent(line string) (name, email string) {
	line = strings.TrimSpace(line)
	open, close := strings.LastIndex(line, "<"), strings.LastIndex(line, ">")
	if open < 0 || close < open {
		return line, ""
	}
	return strings.TrimSpace(line[:open]), line[open+1 : close]
}

// modeInput is what the resolver takes beside the layers' values: a person's
// own reading, and why the computer's file is not read where it is not (a
// refused home, or, foreign, a HOME that is not the account's).
type modeInput struct {
	person  personReading
	homeOff string
	foreign bool
}

// inputFor reads what the resolver takes beside vals for the reading r names.
// The machine layer's values are dropped from vals where the home is not the
// account's. An error is a context that ended while the project's facts were
// asked, which no reading stands on.
func inputFor(r layered.Roots, vals []layerValue[string], key projectKey) ([]layerValue[string], modeInput, error) {
	var in modeInput
	switch {
	case r.HomeRefused:
		in.homeOff = homeRefusedOrigin(ModeHuman)
	case !homeIsAccounts(r.Home):
		in.homeOff, in.foreign = foreignHomeOrigin(), true
		kept := vals[:0:0]
		for _, v := range vals {
			if v.layer != layered.Machine.String() {
				kept = append(kept, v)
			}
		}
		vals = kept
	case repoLayerMode(vals) != "":
		// The project's value stands: only a refusal of the file is said.
		if _, _, err := readPersonFile(r.Home); err != nil {
			in.person.note = err.Error()
		}
	default:
		p, err := readPerson(r.Home, key)
		if err != nil {
			return vals, in, err
		}
		in.person = p
	}
	return vals, in, nil
}
