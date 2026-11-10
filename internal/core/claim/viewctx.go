package claim

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/intentdriven/abcd/internal/core/peers"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/localtier"
)

// viewGit is the git the view's readers run through (iss-2611100548341325).
// Its zero value runs the calls they made before it existed, with no deadline,
// for View and every reader that keeps state; ViewContext alone builds one
// holding a context, under which each git it starts is bound to that context
// and none starts once it has ended. The readers keep one body either way, so
// the view and the stop check read every tip identically.
type viewGit struct{ ctx context.Context }

// bound reports whether g runs its git under a context.
func (g viewGit) bound() bool { return g.ctx != nil }

// local is localGit, bound to g's context when it holds one.
func (g viewGit) local(root string, maxBytes int, args ...string) (string, error) {
	if !g.bound() {
		return localGit(root, maxBytes, args...)
	}
	trace(args...)
	return gitutil.RunCappedContext(g.ctx, root, maxBytes, args...)
}

// stored is storedGit, bound to g's context when it holds one.
func (g viewGit) stored(root string, maxBytes int, args ...string) (string, error) {
	if !g.bound() {
		return storedGit(root, maxBytes, args...)
	}
	return storedGitCtx(g.ctx, root, maxBytes, args...)
}

// emptyTreeName is emptyTreeName, bound to g's context when it holds one.
func (g viewGit) emptyTreeName(root string) (string, error) {
	if !g.bound() {
		return emptyTreeName(root)
	}
	trace("hash-object", "-t", "tree", "--stdin")
	return gitutil.EmptyTreeNameContext(g.ctx, root)
}

// originDefault is originDefault, bound to g's context when it holds one.
func (g viewGit) originDefault(root string) string {
	if !g.bound() {
		return originDefault(root)
	}
	trace("symbolic-ref", "--quiet", "refs/remotes/"+Remote+"/HEAD")
	return gitutil.TrackedDefaultBranchContext(g.ctx, root, Remote)
}

// recordsAt is peers.RecordsAt, bound to g's context when it holds one.
// Both arms trace the listing, as every git this package's readers start is
// traced, though peers runs it.
func (g viewGit) recordsAt(root, ref string) (map[string][]peers.RecordAt, error) {
	trace("ls-tree", "-r", "-z", "--full-tree", ref, "--")
	if !g.bound() {
		return peers.RecordsAt(root, ref)
	}
	return peers.RecordsAtContext(g.ctx, root, ref)
}

// resolveRef is resolveRef, bound to g's context when it holds one.
func (g viewGit) resolveRef(root, ref string) (name, typ string, ok bool, err error) {
	if !g.bound() {
		return resolveRef(root, ref)
	}
	return resolveRefCtx(g.ctx, root, ref)
}

// ownCopyKey is ownCopyKey, its tracked check bound to g's context when it
// holds one.
func (g viewGit) ownCopyKey(root string) string {
	if !g.bound() {
		return ownCopyKey(root)
	}
	r, err := os.OpenRoot(root)
	if err != nil {
		return ""
	}
	defer r.Close()
	return copyKeyOf(localtier.ReadInRootContext(g.ctx, r, copyKeyRel, 256))
}

// ViewContext is View bound to ctx, for a hook on a deadline (the stop
// check): every git the view starts runs under ctx, and none starts once ctx
// has ended. The view folds several failures into an empty answer (a default
// branch it could not name, a copy key or a kept first sighting it could not
// read, the generated files, and a tip whose commit read failed past its cap,
// which reads as an object too big), so a view ctx ended is not a view:
// whenever ctx has ended by the time it returns, ViewContext answers ctx's
// error and an empty snapshot, never a snapshot read in part. A commit read
// ctx ended is not folded into an object too big at all (readCommitIn).
func ViewContext(ctx context.Context, root string, now time.Time) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, fmt.Errorf("reading the reservations: %w", err)
	}
	snap, err := viewOf(viewGit{ctx: ctx}, root, now, true, nil)
	if cerr := ctx.Err(); cerr != nil {
		return Snapshot{}, fmt.Errorf("reading the reservations: %w", cerr)
	}
	return snap, err
}

// OwnCopyKeyContext is OwnCopyKey with its tracked check bound to ctx. ""
// answers both "none kept" and "ctx ended first": a caller that must tell
// them apart checks ctx.Err() after it.
func OwnCopyKeyContext(ctx context.Context, root string) string {
	return viewGit{ctx: ctx}.ownCopyKey(root)
}
