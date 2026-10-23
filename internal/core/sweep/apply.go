package sweep

// apply.go is `--apply` for worktrees (spc-2610230730269295, step 2): the dry
// run's judgement, then each row acted on. A remove row is reclaimed through
// the store's PruneOne, the one reclaim primitive, whose last check before
// `git worktree remove` is a fresh liveness probe of that one worktree. Every
// other row is kept. Nothing is forced, no process is stopped, and the
// embargo store and a worktree outside the store are never acted on: the dry
// run keeps both, and PruneOne refuses a name its lane does not hold.

import (
	"context"
	"errors"
	"path/filepath"

	store "github.com/intentdriven/abcd/internal/core/ahoy/worktree"
)

// Apply judges the repository at repoRoot as DryRun does, then acts on each
// row, which carries its Outcome:
//
//   - a remove row is reclaimed through the store's PruneOne, whose PreRemove
//     makes a fresh probe of that one worktree immediately before removal: a
//     new process listing, the reservations the caller read, the build
//     loop's runs and the owner check. Any hold keeps it, naming the hold;
//   - a save-patch row is kept: patch salvage is not built yet, so its
//     changes are not saved;
//   - every other row is kept, with the dry run's reason.
//
// A dry run that refuses is returned as DryRun returns it. A prune that
// refuses or faults keeps that worktree, naming why, and the run carries the
// first such fault as its Refusal, which is also returned as the error, so
// ExitCode reads ExitRefused; the other rows are still acted on, each prune
// being its own judgement.
func Apply(ctx context.Context, repoRoot string, o Options) (Result, error) {
	res, srcs, j, err := judgeRepo(ctx, repoRoot, o)
	if err != nil {
		return res, err
	}
	res.Applied = true
	prune := o.pruneOne
	if prune == nil {
		prune = store.PruneOne
	}
	var fault error
	for i := range res.Rows {
		row := &res.Rows[i]
		row.Outcome = OutcomeKept
		switch row.Action {
		case ActionRemove:
			if err := j.remove(ctx, repoRoot, srcs[i], o, prune, row); err != nil && fault == nil {
				fault = err
			}
		case ActionSavePatch:
			row.Reason = "dirty, and nothing holds it: patch salvage is not built yet, so its changes were not saved; the worktree is kept"
		}
	}
	if fault != nil {
		_, err := refused(fault)
		res.Refusal = err.Error()
		return res, err
	}
	return res, nil
}

// remove reclaims the store worktree src through prune, marking row with the
// outcome. A prune refusal or fault is returned, and row kept naming it.
func (j judge) remove(ctx context.Context, repoRoot string, src store.Row, o Options,
	prune func(string, string, store.PruneOptions) (store.Pruned, error), row *Row) error {
	var last Liveness
	opts := store.PruneOptions{
		Forge: o.Forge,
		Held:  o.Held,
		Cwd:   o.Cwd,
		// The probe the dry run made read the process table once, before any
		// removal; a process that started since is seen only by a new one, so
		// each removal makes its own, last, immediately before git removes.
		PreRemove: func(abs string) (bool, string) {
			fresh := j
			fresh.probe = NewProbe(ctx, o.Probe)
			last = fresh.liveness(abs)
			return last.Held, last.Reason
		},
	}
	got, err := prune(repoRoot, filepath.Base(filepath.Clean(src.Abs)), opts)
	if err != nil {
		row.Reason = clean("the store's prune refused: " + err.Error())
		return err
	}
	if len(got.Rows) != 1 {
		row.Reason = "the store's prune answered for no single worktree"
		return errors.New("sweep: the store's prune answered for no single worktree")
	}
	pr := got.Rows[0]
	if pr.Outcome == store.OutcomeReclaimed {
		row.Outcome = OutcomeRemoved
		if pr.Archive != "" {
			row.Reason += "; its notes were archived to " + pr.Archive
		}
		return nil
	}
	// Kept by the store, or by the fresh probe: the prune row's reason is
	// already masked and home-redacted.
	if last.Held {
		row.Reason = "held at the last check before removal: " + pr.Reason
		for _, e := range last.Evidence {
			e.Detail = clean(e.Detail)
			row.Evidence = append(row.Evidence, e)
		}
		return nil
	}
	row.Reason = "kept by the store's prune: " + pr.Reason
	return nil
}
