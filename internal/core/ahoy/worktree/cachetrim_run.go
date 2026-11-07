package worktree

// cachetrim_run.go paces the trim that runs at a lane's start. Prune is too rare
// on a machine that goes days without one and the load gate only refuses, so the
// build loop asks for a trim whenever it makes a lane's worktree; a marker in the
// person's abcd home holds that to once an hour across every process of the
// account, whose cache it is. The walk takes minutes on a large cache, so the
// trim runs detached, which leaves nobody to tell of a failure: the run saves its
// answer here and the next claim hands it back for the lane to say.

import (
	"encoding/json"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// CacheTrimInterval is the least time between two trims a lane's start asks for.
const CacheTrimInterval = time.Hour

const (
	cacheTrimRunFile  = "go-cache-trim.json"
	cacheTrimLockFile = "go-cache-trim.lock"
	// cacheTrimClaimWait bounds how long a lane waits on another's claim, which
	// holds the lock only to read and write one small file.
	cacheTrimClaimWait = 2 * time.Second
	cacheTrimRunMax    = 64 << 10
)

// CacheTrimRun is the marker: when the last trim was claimed or started, and what
// it answered. Trim is nil for a run that has not finished, or never did.
type CacheTrimRun struct {
	StartedAt time.Time  `json:"started_at"`
	Trim      *CacheTrim `json:"trim,omitempty"`
}

// ClaimCacheTrim reports whether a trim is due as of now and, when it is, records
// the claim so no other process of the account starts one for an hour, handing
// back the run before it. The marker only paces the trim, so one that cannot be
// used, or a time ahead of the clock, is due. An error is a home it cannot trust
// or a lock it could not take: nothing is due, and the caller says so.
func ClaimCacheTrim(home string, now time.Time) (last CacheTrimRun, due bool, err error) {
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return CacheTrimRun{}, false, err
	}
	defer root.Close()
	err = fsutil.WithFileLock(abcdhome.Path(home, cacheTrimLockFile), cacheTrimClaimWait, func() error {
		if raw, err := fsutil.ReadGuardedInRoot(root, cacheTrimRunFile, cacheTrimRunMax); err == nil {
			if json.Unmarshal(raw, &last) != nil {
				last = CacheTrimRun{}
			}
		}
		if !last.StartedAt.IsZero() && !now.Before(last.StartedAt) && now.Sub(last.StartedAt) < CacheTrimInterval {
			return nil
		}
		due = true
		return writeCacheTrimRun(home, CacheTrimRun{StartedAt: now})
	})
	if err != nil {
		return CacheTrimRun{}, false, err
	}
	return last, due, nil
}

// SaveCacheTrim records what a real trim, started at started, answered, under the
// claim's lock. A run that outlasted the hour may find a later claim in the
// marker; it leaves that claim alone, so the marker never moves back in time and
// the next lane does not start another run early.
func SaveCacheTrim(home string, started time.Time, t CacheTrim) error {
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer root.Close()
	return fsutil.WithFileLock(abcdhome.Path(home, cacheTrimLockFile), cacheTrimClaimWait, func() error {
		var now CacheTrimRun
		if raw, err := fsutil.ReadGuardedInRoot(root, cacheTrimRunFile, cacheTrimRunMax); err == nil && json.Unmarshal(raw, &now) == nil && now.StartedAt.After(started) {
			return nil
		}
		return writeCacheTrimRun(home, CacheTrimRun{StartedAt: started, Trim: &t})
	})
}

func writeCacheTrimRun(home string, r CacheTrimRun) error {
	raw, err := json.Marshal(r)
	if err != nil {
		return err
	}
	root, err := fsutil.EnsureHomeScope(home, abcdhome.Rel(), abcdhome.DirMode)
	if err != nil {
		return err
	}
	defer root.Close()
	return fsutil.WriteFileAtomicInRoot(root, cacheTrimRunFile, raw, 0o600)
}
