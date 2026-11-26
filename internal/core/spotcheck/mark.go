package spotcheck

// mark.go is the store's high-water mark (adr-2611251806496719): this
// machine's own note, in the person's home store under
// ~/.abcd.noindex/spotcheck/<root-sha>/<origin>/mark.json (<origin> the first
// sixteen hex digits of the SHA-256 of origin's URL, so a fork's checkout and
// upstream's keep marks of their own), of the last copy of the
// store's branch it read and verified, the window's opening and the newest
// verified cycle in it. The branch on origin is kept for ever and only ever
// moves forward, so a copy that no longer holds the mark's commit, opening or
// cycle was deleted or rewritten: the store refuses it in one line, draws
// nothing, carries nothing and opens no first window over it. Nothing in the
// store's branch can move the mark, and a mark that cannot be read refuses as
// loudly as a rewritten branch. Its folder is named by origin's URL as the
// checkout's configuration spells it, so a remote renamed to another spelling
// starts a mark of its own, trusting what it first reads: the mark guards
// against the branch being rewritten, the forge's ruleset against the
// rewriting itself.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

// markLeaf is the mark's file in its folder.
const markLeaf = "mark.json"

// maxMarkBytes bounds the mark on read.
const maxMarkBytes = 4 << 10

// mark is the high-water mark: the store's commit last read and verified,
// the window's opening, and the newest verified cycle with its window.
type mark struct {
	SchemaVersion int       `json:"schema_version"`
	Commit        string    `json:"commit"`
	Opened        string    `json:"opened"`
	OpenedAt      time.Time `json:"opened_at"`
	Cycle         string    `json:"cycle,omitempty"`
	CycleFrom     string    `json:"cycle_from,omitempty"`
	CycleTo       string    `json:"cycle_to,omitempty"`
}

// errRewritten is the store's refusal of a branch that no longer holds what
// this machine last verified on it.
var errRewritten = errors.New("the spot-check branch on origin was rewritten")

// rewritten is errRewritten naming what the branch no longer holds.
func rewritten(what string) error {
	return fmt.Errorf("%w: it no longer holds %s this machine last verified there", errRewritten, what)
}

// markFolder is the mark's folder for the repository at root and its
// origin, "" when the repository has no commit to key it on.
func markFolder(ctx context.Context, root string) (string, error) {
	sha, err := gitutil.RootCommitContext(ctx, root)
	if err != nil {
		return "", fmt.Errorf("the spot-check's mark cannot be found: %s", redact(err))
	}
	if !gitutil.IsFullSHA(sha) {
		return "", nil
	}
	url, err := gitutil.RunCappedContext(ctx, root, 64<<10, "remote", "get-url", claim.Remote)
	if err != nil || strings.TrimSpace(url) == "" {
		return "", errors.New("the spot-check's mark cannot be found: origin's URL does not read")
	}
	sum := sha256.Sum256([]byte(strings.TrimSpace(url)))
	return abcdhome.Rel("spotcheck", sha, hex.EncodeToString(sum[:8])), nil
}

// readMark is the mark for the repository at root; ok is false when there is
// none.
func readMark(ctx context.Context, root string) (m mark, ok bool, err error) {
	folder, err := markFolder(ctx, root)
	if err != nil || folder == "" {
		return mark{}, false, err
	}
	home, err := abcdhome.MachineHome()
	if err != nil {
		return mark{}, false, fmt.Errorf("the spot-check's mark cannot be found: %s", redact(err))
	}
	data, refusal, err := fsutil.ReadHomeDeclaration(home, folder+"/"+markLeaf, maxMarkBytes)
	if refusal == fsutil.DeclarationAbsent {
		return mark{}, false, nil
	}
	if err != nil {
		return mark{}, false, fmt.Errorf("the spot-check's mark %s cannot be read: %s", abcdhome.Display("spotcheck", "<root-sha>", "<origin>", markLeaf), redact(err))
	}
	if err := jsonstrict.Decode(data, &m); err != nil || m.SchemaVersion != SchemaVersion || !gitutil.IsFullSHA(m.Commit) {
		return mark{}, false, fmt.Errorf("the spot-check's mark %s does not read", abcdhome.Display("spotcheck", "<root-sha>", "<origin>", markLeaf))
	}
	return m, true, nil
}

// writeMark keeps m as the mark for the repository at root.
func writeMark(ctx context.Context, root string, m mark) error {
	folder, err := markFolder(ctx, root)
	if err != nil || folder == "" {
		return err
	}
	home, err := abcdhome.MachineHome()
	if err != nil {
		return err
	}
	m.SchemaVersion = SchemaVersion
	data, err := encode(m)
	if err != nil {
		return err
	}
	dir, err := fsutil.EnsureHomeDeclarationScope(home, folder, abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer dir.Close()
	return fsutil.WriteFileAtomicInRoot(dir, markLeaf, data, abcdhome.FileMode)
}

// errBehind is origin's copy as this checkout last fetched it standing
// behind the mark: another checkout of the repository on this machine, which
// shares the mark, read a later copy, or a push's own move of the tracking
// ref did not take. It is answered by a fetch, never as a rewritten branch.
var errBehind = errors.New("this checkout's copy of the spot-check's branch is behind the one this machine last verified, so it is fetched first")

// holdsMark is nil when tracking, origin's copy ("" for none), holds the
// mark's commit, and the refusal otherwise; nil too when this machine keeps
// no mark. fetched says tracking is what origin answered just now: a copy
// not fetched that holds no such commit, or that the mark's commit descends
// from, is errBehind, since a fetch may bring it; once fetched, any copy
// that does not hold it was deleted or rewritten on origin.
func holdsMark(ctx context.Context, root, tracking string, fetched bool) error {
	m, ok, err := readMark(ctx, root)
	if err != nil || !ok {
		return err
	}
	if tracking == m.Commit {
		return nil
	}
	if tracking != "" {
		if held, err := isAncestor(ctx, root, m.Commit, tracking); err == nil && held {
			return nil
		}
	}
	if !fetched && behind(ctx, root, m.Commit, tracking) {
		return errBehind
	}
	return rewritten("the copy (" + m.Commit[:12] + ")")
}

// behind reports whether tracking, a copy not fetched just now, may only be
// behind the mark's commit: there is no copy, the commit is not in this
// checkout's object store, or the commit descends from the copy.
func behind(ctx context.Context, root, commit, tracking string) bool {
	if tracking == "" {
		return true
	}
	if _, err := gitutil.RunCappedBytesAsStoredContext(ctx, root, "", 256, "cat-file", "-e", commit+"^{commit}"); err != nil {
		return ctx.Err() == nil
	}
	on, err := isAncestor(ctx, root, tracking, commit)
	return err == nil && on
}
