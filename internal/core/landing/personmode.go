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
// difference. An entry of auto is matched on the default branch's root commit
// and the repository's name together, an entry of human on the root alone, so
// a fork, which carries the same root under another name, shares a person's
// human and not their auto.
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
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
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

// projectKey is what selects an entry for one reading: the default branch's
// root commit, the repository's name ("" where it is not read), and git's
// identity against the record: ident names it, and gitIs (when not "")
// stands in for asking git, as the row's kept reading does.
type projectKey struct {
	root, name string
	ident      func(rec identity.Record) (diff string, err error)
	gitIs      string
	// gate is the merge gate's reading, which holds a person's own auto back.
	gate bool
}

// personReading is what a person's own setting says for one project: the
// mode ("" for no setting), the origin it is said with, whether it was
// withheld (refused, or not this person's to use here), and git's identity
// against the record as found.
type personReading struct {
	mode, origin string
	withheld     bool
	// note is a refusal of the person's file, said beside a project value.
	note  string
	gitIs string
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
	return "HOME is not the home the account database names for this account, so neither your own setting nor this computer's file is read for " + KeyFacilitatorMode + "; run abcd with your own HOME"
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
// file under home (the account's own, which the caller has judged).
func readPerson(home string, key projectKey) personReading {
	f, ok, err := readPersonFile(home)
	if err != nil {
		return personReading{mode: ModeHuman, origin: err.Error(), withheld: true, note: err.Error()}
	}
	if !ok || len(f.Entries) == 0 || key.root == "" {
		// No file, no entries, or no project to select one for.
		return personReading{}
	}
	rec, ref, found, err := identity.ReadRecord(home, key.root)
	if err != nil {
		return personReading{mode: ModeHuman, withheld: true,
			origin: "your identity record cannot be read (" + fsutil.RedactHome(err.Error()) + "), so no entry of " + personDisplay() + " is selected here; fix it with abcd ahoy --identity"}
	}
	if !found {
		return personReading{mode: ModeHuman, withheld: true,
			origin: personDisplay() + " holds entries, and no identity record says who you are here, so none is selected; record one with abcd ahoy --identity"}
	}
	e, ok := selectEntry(f.Entries, rec.Email, key)
	if !ok {
		return personReading{}
	}
	gitIs, diff := key.gitIs, ""
	if gitIs == "" && key.ident == nil {
		gitIs, diff = gitIsOther, "git's identity here was not read with the last reading, which the next prompt takes"
	}
	if gitIs == "" {
		gitIs = gitIsOther
		switch d, err := identOf(key, rec); {
		case err != nil:
			diff = "git's identity here could not be read (" + fsutil.RedactHome(err.Error()) + ")"
		case d != "":
			diff = d
		default:
			gitIs = gitIsRecord
		}
	}
	if gitIs != gitIsRecord {
		if diff == "" {
			diff = "git's identity here was not your identity record's when it was last read"
		}
		return personReading{mode: ModeHuman, withheld: true, gitIs: gitIs,
			origin: diff + ", so your own setting for this project is not used (your identity record is " + ref.Display + ")"}
	}
	if key.gate && e.Mode == ModeAuto {
		return personReading{mode: ModeHuman, withheld: true, gitIs: gitIs,
			origin: "your own auto in " + personDisplay() + " arms only a pull request your own account opened, which this gate does not read yet, so it waits as for human"}
	}
	return personReading{mode: e.Mode, origin: "set in " + personDisplay() + " on " + e.SetAt.UTC().Format(time.DateOnly), gitIs: gitIs}
}

// identOf is git's identity here against rec, through the key's reader.
func identOf(key projectKey, rec identity.Record) (string, error) {
	if key.ident == nil {
		return "", errors.New("no checkout to ask")
	}
	return key.ident(rec)
}

// selectEntry is the entry for email in the project key names: an entry of
// human on the root alone, an entry of auto on the root and the name. With
// both, human is taken.
func selectEntry(es []personEntry, email string, key projectKey) (personEntry, bool) {
	var auto *personEntry
	for i, e := range es {
		if !strings.EqualFold(strings.TrimSpace(e.Email), strings.TrimSpace(email)) || key.root == "" || e.Root != key.root {
			continue
		}
		if e.Mode == ModeHuman {
			return e, true
		}
		if key.name != "" && strings.EqualFold(e.Forge, key.name) && auto == nil {
			auto = &es[i]
		}
	}
	if auto != nil {
		return *auto, true
	}
	return personEntry{}, false
}

// keyFor is the project key of the reading r names: the oldest root commit of
// its tip (r.RepoAt), the name origin's address spells, and git's identity in
// r.Repo. A repository layer that is not read at
// a commit has no root, and so no entry.
func keyFor(ctx context.Context, r layered.Roots, gate bool) projectKey {
	k := projectKey{gate: gate}
	if r.Repo == "" || !gitutil.IsFullSHA(r.RepoAt) {
		return k
	}
	k.root = oldestRoot(ctx, r.Repo, r.RepoAt)
	k.name = originName(ctx, r.Repo)
	repo := r.Repo
	k.ident = func(rec identity.Record) (string, error) { return gitIdentDiff(ctx, repo, rec) }
	return k
}

// oldestRoot is the oldest root commit of tip as stored, "" where git does not
// say.
func oldestRoot(ctx context.Context, repo, tip string) string {
	out, err := gitutil.RunCappedContext(ctx, repo, 1<<20, "-c", "core.useReplaceRefs=false", "rev-list", "--max-parents=0", "--timestamp", tip, "--")
	if err != nil {
		return ""
	}
	best, bestAt := "", int64(0)
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 2 || !gitutil.IsFullSHA(f[1]) {
			continue
		}
		at, err := strconv.ParseInt(f[0], 10, 64)
		if err != nil {
			continue
		}
		if best == "" || at < bestAt || (at == bestAt && f[1] < best) {
			best, bestAt = f[1], at
		}
	}
	return best
}

// originName is the repository's name as origin's address spells it,
// owner/name in lower case, "" where it spells none.
func originName(ctx context.Context, repo string) string {
	out, err := gitutil.RunCappedContext(ctx, repo, 4096, "config", "--get", "remote."+Remote+".url")
	if err != nil {
		return ""
	}
	return nameFromURL(strings.TrimSpace(out))
}

// nameFromURL is owner/name from a forge address in any of git's spellings
// (https://host/owner/name.git, ssh://git@host/owner/name, git@host:owner/name).
func nameFromURL(raw string) string {
	p := ""
	if u, err := url.Parse(raw); err == nil && u.Scheme != "" && u.Host != "" {
		p = u.Path
	} else if _, rest, ok := strings.Cut(raw, ":"); ok && !strings.Contains(raw, "://") {
		p = rest
	} else {
		return ""
	}
	p = strings.TrimSuffix(strings.Trim(p, "/"), ".git")
	parts := strings.Split(p, "/")
	if len(parts) < 2 || parts[len(parts)-2] == "" || parts[len(parts)-1] == "" {
		return ""
	}
	return strings.ToLower(parts[len(parts)-2] + "/" + parts[len(parts)-1])
}

// gitIdentDiff says how git's would-be author and committer in repo differ
// from rec, "" where both are rec's. git is asked as the person's own git
// commits (gitutil.WorkingTreeCommandContext): with their global and system
// settings, where most people keep their identity, which the isolated
// environment reads as absent; `git var` reads settings and starts nothing.
func gitIdentDiff(ctx context.Context, repo string, rec identity.Record) (string, error) {
	for _, role := range []string{"GIT_AUTHOR_IDENT", "GIT_COMMITTER_IDENT"} {
		out, err := personGitVar(ctx, repo, role)
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
// refused home, or a HOME that is not the account's).
type modeInput struct {
	person  personReading
	homeOff string
}

// inputFor reads what the resolver takes beside vals for the reading r names.
// The machine layer's values are dropped from vals where the home is not the
// account's.
func inputFor(r layered.Roots, vals []layerValue[string], key projectKey) ([]layerValue[string], modeInput) {
	var in modeInput
	switch {
	case r.HomeRefused:
		in.homeOff = homeRefusedOrigin(ModeHuman)
	case !homeIsAccounts(r.Home):
		in.homeOff = foreignHomeOrigin()
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
		in.person = readPerson(r.Home, key)
	}
	return vals, in
}
