package rules

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// What abcd keeps about one sub-agent of one session (spc-2611251757228271):
// that it was told where the conventions are, and what the guard and the stop
// hook then found. Each fact is one small file beside the per-session ledgers,
// written once and whole and never rewritten from a read, so two hooks
// answering for one sub-agent at the same moment cannot lose one another's
// write. They are not fields of SessionState: the prompt router rewrites that
// file whole on every prompt and the reset hook removes it, either of which
// would drop a fact while its sub-agent still runs. PruneState sweeps them by
// the ledgers' own age.

// SubagentFact names one fact; its value is the fact file's suffix.
type SubagentFact string

const (
	// SubagentTold is that the sub-agent was told: by the start hook, or by
	// the guard's refusal where no start hook told it.
	SubagentTold SubagentFact = "told"
	// SubagentPassed is that the guard found the conventions read, by the
	// file or by a lane's brief that carries them.
	SubagentPassed SubagentFact = "passed"
	// SubagentHeld is that the guard refused a write.
	SubagentHeld SubagentFact = "held"
	// SubagentUnjudged is that the guard could not judge the sub-agent, and
	// why. It is final: that sub-agent is not judged again and never held.
	SubagentUnjudged SubagentFact = "unjudged"
	// SubagentJudged is that the stop kept the sub-agent's one line.
	SubagentJudged SubagentFact = "judged"
)

// Who wrote a told fact.
const (
	SubagentToldByStart = "start"
	SubagentToldByGuard = "guard"
)

// SubagentConventionsFiles are the conventions files a sub-agent is told of,
// in the order they are looked for. A told fact names one of them or none.
var SubagentConventionsFiles = []string{"AGENTS.md", "CLAUDE.md"}

// maxSubagentFactBytes caps a fact on read: a file name, a directory and a
// word.
const maxSubagentFactBytes = 16 * 1024

// SubagentRecord is a fact's content. File, Root and By are the told fact's:
// the conventions file's name, empty for none, the checkout root looked in,
// and who wrote it. Reason is the passed and the unjudged facts'.
type SubagentRecord struct {
	File   string    `json:"file,omitempty"`
	Root   string    `json:"root,omitempty"`
	By     string    `json:"by,omitempty"`
	Reason string    `json:"reason,omitempty"`
	At     time.Time `json:"at"`
}

// ErrSubagentStore is that the folder the facts are kept in could not be
// opened, so nothing is known of any sub-agent, as against one fact that
// cannot be read.
var ErrSubagentStore = errors.New("rules: the folder the sub-agents' facts are kept in cannot be opened")

// subagentLeaf is the fact's file name: the hash of the session and the agent,
// each part ended, so neither id can name a path and no two pairs meet.
func subagentLeaf(session, agent string, fact SubagentFact) string {
	sum := sha256.Sum256([]byte("subagent\x00" + session + "\x00" + agent))
	return hex.EncodeToString(sum[:]) + "." + string(fact)
}

// WriteSubagentFact keeps one fact of one sub-agent at mode 0600. A fact that
// is already there stands as it is, so a fact's age runs from its first
// writing. With no home it keeps nothing and returns ErrNoStateHome.
func WriteSubagentFact(session, agent string, fact SubagentFact, rec SubagentRecord) error {
	dir, _, _, err := stateFolder(true)
	if err != nil {
		return err
	}
	defer dir.Close()
	leaf := subagentLeaf(session, agent, fact)
	if _, err := dir.Lstat(leaf); err == nil {
		return nil
	}
	rec.At = time.Now().UTC()
	data, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomicInRoot(dir, leaf, data, 0o600)
}

// ReadSubagentFact reads one fact of one sub-agent, through the guarded read
// LoadState uses. An absent fact, and an absent folder, are found false and no
// error. A folder that cannot be opened is ErrSubagentStore. A fact this
// account does not own, one others can write, a link, one that does not parse,
// and a told fact naming anything but the two conventions files or none, or a
// writer that is neither of the two, is an error: it cannot be read, and the
// caller says so rather than taking it for absent.
func ReadSubagentFact(session, agent string, fact SubagentFact) (SubagentRecord, bool, error) {
	dir, home, rel, err := stateFolder(false)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return SubagentRecord{}, false, nil
	case err != nil:
		return SubagentRecord{}, false, fmt.Errorf("%w: %v", ErrSubagentStore, pathCause(err))
	}
	defer dir.Close()
	data, refusal, err := fsutil.ReadHomeDeclarationDenyingIn(dir, home, path.Join(rel, subagentLeaf(session, agent, fact)), maxSubagentFactBytes, 0o022)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return SubagentRecord{}, false, nil
	case err != nil:
		return SubagentRecord{}, false, fmt.Errorf("rules: a sub-agent's %s fact cannot be read: %v", fact, pathCause(err))
	case refusal != fsutil.DeclarationOK:
		return SubagentRecord{}, false, fmt.Errorf("rules: a sub-agent's %s fact is not this account's own file", fact)
	}
	var rec SubagentRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return SubagentRecord{}, false, fmt.Errorf("rules: a sub-agent's %s fact does not parse", fact)
	}
	if fact == SubagentTold && !validTold(rec) {
		return SubagentRecord{}, false, errors.New("rules: a sub-agent's told fact names a file or a writer abcd does not write")
	}
	return rec, true, nil
}

// validTold reports whether a told fact holds what abcd writes in one.
func validTold(rec SubagentRecord) bool {
	if rec.By != SubagentToldByStart && rec.By != SubagentToldByGuard {
		return false
	}
	if rec.File == "" {
		return true
	}
	for _, f := range SubagentConventionsFiles {
		if rec.File == f {
			return true
		}
	}
	return false
}
