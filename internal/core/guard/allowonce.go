package guard

// allowonce.go is the person's way past a delete the guard could not prove
// safe (iss-2610241239492159): `abcd guard allow --once --command '<line>'`,
// which only a person runs, writes a record in their home folder that lets
// that one exact line through the hook once, within AllowWindow. The hook
// takes the record as it lets the line run, so it is used once, and both acts
// go into the store's log: the exact line, when it was allowed and when it
// was used, and nothing that names who.
//
// The record is read only when it is a regular file this account owns that no
// one else can write, in a directory chain no one else can change: the rules
// loader's rule for ~/.abcd.noindex/rules.json, through the same reader
// (fsutil.ReadHomeDeclaration). The trust model is the guard's: a careless
// agent, not a hostile one. An agent running the verb, or writing into the
// store from the shell, is refused (AllowVerbSignal, allowStoreSignal); a
// file-writing tool the shell guard does not see is outside what it reads.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// AllowWindow is how long a one-shot allow waits to be used.
const AllowWindow = 10 * time.Minute

// allowStoreLeaf is the store's directory below abcd's home folder; each
// repository's allows sit in a directory of their own, keyed on its root
// commit as the worktree store is.
const allowStoreLeaf = "guard-allow"

// maxAllowBytes bounds one allow record.
const maxAllowBytes = 64 * 1024

// Allowed is one one-shot allow.
type Allowed struct {
	Command   string    `json:"command"`
	AllowedAt time.Time `json:"allowed_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// allowLogLine is one line of the store's log.
type allowLogLine struct {
	Event     string    `json:"event"`
	Command   string    `json:"command"`
	AllowedAt time.Time `json:"allowed_at"`
	UsedAt    time.Time `json:"used_at,omitzero"`
}

// AllowStoreDisplay names the store for a message, in the tilde form.
func AllowStoreDisplay() string { return abcdhome.Display(allowStoreLeaf) + "/" }

// allowRel is the store's directory for one repository, relative to home.
func allowRel(rootSHA string) string { return abcdhome.Rel(allowStoreLeaf, rootSHA) }

// allowName is the record's file name for one exact line.
func allowName(command string) string {
	sum := sha256.Sum256([]byte(command))
	return "allow-" + hex.EncodeToString(sum[:16]) + ".json"
}

// GrantAllow records a one-shot allow of command for the repository whose root
// commit is rootSHA, in home, at now. A second grant of the same line before
// it is used replaces the first, restarting its window.
func GrantAllow(home, rootSHA, command string, now time.Time) (Allowed, error) {
	if command == "" {
		return Allowed{}, errors.New("an allow needs the exact command line it lets through")
	}
	if !validRootSHA(rootSHA) {
		return Allowed{}, fmt.Errorf("%q is not a repository's root commit", rootSHA)
	}
	root, err := fsutil.EnsureHomeScope(home, allowRel(rootSHA), abcdhome.DirMode)
	if err != nil {
		return Allowed{}, err
	}
	defer root.Close()
	a := Allowed{Command: command, AllowedAt: now.UTC(), ExpiresAt: now.Add(AllowWindow).UTC()}
	data, err := json.Marshal(a)
	if err != nil {
		return Allowed{}, err
	}
	if err := fsutil.WriteFileAtomicInRoot(root, allowName(command), data, 0o600); err != nil {
		return Allowed{}, err
	}
	logAllow(root, allowLogLine{Event: "allowed", Command: command, AllowedAt: a.AllowedAt})
	return a, nil
}

// TakeAllow uses the one-shot allow of command, when one is recorded, still
// in its window, and readable as the person's own: it is removed as it is
// used, so a second run of the line finds none. A record past its window is
// removed and not used. ok is false whenever the line may not run on it.
func TakeAllow(home, rootSHA, command string, now time.Time) (a Allowed, ok bool) {
	if command == "" || !validRootSHA(rootSHA) {
		return Allowed{}, false
	}
	rel := allowRel(rootSHA) + "/" + allowName(command)
	data, refusal, err := fsutil.ReadHomeDeclaration(home, rel, maxAllowBytes)
	if err != nil || refusal != fsutil.DeclarationOK {
		return Allowed{}, false
	}
	if json.Unmarshal(data, &a) != nil || a.Command != command {
		return Allowed{}, false
	}
	root, err := fsutil.OpenHomeDeclarationScope(home, allowRel(rootSHA))
	if err != nil {
		return Allowed{}, false
	}
	defer root.Close()
	// Taking it is its removal: of two hooks racing for one record, only the
	// one whose remove succeeds may run the line.
	if err := root.Remove(allowName(command)); err != nil {
		return Allowed{}, false
	}
	if now.After(a.ExpiresAt) {
		return Allowed{}, false
	}
	logAllow(root, allowLogLine{Event: "used", Command: command, AllowedAt: a.AllowedAt, UsedAt: now.UTC()})
	return a, true
}

// logAllow appends one line to the store's log. A log that cannot be written
// changes no outcome: the record is the control, and the log its account.
func logAllow(root *os.Root, l allowLogLine) {
	if line, err := json.Marshal(l); err == nil {
		_ = fsutil.AppendLineIn(root, "log.jsonl", line, 0o600)
	}
}

// validRootSHA reports whether s is a full hex object name: forty digits
// under SHA-1, sixty-four under SHA-256.
func validRootSHA(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// allowStoreEntryID is the reserved id a shell write into the allow store is
// refused under.
const allowStoreEntryID = "guard-allow-store-write"

// allowStoreSignal refuses a line that writes into the allow store: an
// allow written there by hand is one no person granted. Only a line that
// names the store is read, so every other line costs nothing; on one that
// does, a write whose target the line does not fix is refused too, since it
// may land there.
func (r Registry) allowStoreSignal(command string, rc *readCtx, v verdicts) (payloadSignal, bool) {
	if rc == nil || r.files == nil || r.files.home == "" || !strings.Contains(command, allowStoreLeaf) {
		return payloadSignal{}, false
	}
	store := abcdhome.Path(r.files.home, allowStoreLeaf)
	stores := []string{store, fsutil.RealExistingPath(store)}
	for _, p := range placedLine(rc.line, v.segs) {
		for _, st := range p.states {
			for _, u := range pathUses(rc, p.seg, st) {
				if u.remove {
					// Removing an allow only takes one away.
					continue
				}
				if !u.ok || inAny(u.path, stores) || inAny(fsutil.RealExistingPath(u.path), stores) {
					return allowStoreRefusal(), true
				}
			}
		}
	}
	return payloadSignal{}, false
}

// inAny reports whether p is one of dirs or below one.
func inAny(p string, dirs []string) bool {
	p = filepath.Clean(p)
	for _, d := range dirs {
		if p == d || strings.HasPrefix(p, d+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func allowStoreRefusal() payloadSignal {
	return payloadSignal{
		id:      allowStoreEntryID,
		verdict: VerdictBlock,
		family:  "guard allow",
		reason: "This writes into " + AllowStoreDisplay() + ", where a person's one-shot allows of a delete live: " +
			"an allow an agent writes there is one no person granted.",
		successor: "Leave the allow to the person: tell them the exact command and why it is safe, and let them run " +
			"`abcd guard allow --once --command '<the command>'` themselves.",
	}
}
