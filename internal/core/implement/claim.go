package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/core/recordid"
	"github.com/intentdriven/abcd/internal/fsutil"
)

// A claim is how a session takes a record before opening its lane: one file per
// record, `claims/<record>.json`, taken by an exclusive create that fails if the
// file exists. That create IS the exclusion — two sessions reaching for one
// record at the same instant are decided by the kernel, and exactly one of them
// holds it. The file carries who holds it, for which lane, since when, and until
// when: a claim is a lease, so a session that dies holding one strands nothing,
// and the record is claimable again once the lease has passed. The lapse is
// logged, as is every refused claim with the session holding it.

// DefaultLease is how long a claim holds when the caller names no lease: long
// enough for a lane's build-and-gate cycle, short enough that a session that
// died mid-lane frees its record within the same working window.
const DefaultLease = 2 * time.Hour

// Lease bounds. A lease shorter than a minute lapses under its own gate run; a
// lease longer than a day outlives any window the run keeps.
const (
	MinLease = time.Minute
	MaxLease = 24 * time.Hour
)

// Claim is one claim file's content.
type Claim struct {
	Record    string    `json:"record"`
	Session   string    `json:"session"`
	Lane      string    `json:"lane"`
	ClaimedAt time.Time `json:"claimed_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ClaimState is a claim as read at a moment: whether its lease still holds.
type ClaimState struct {
	Claim
	Live bool `json:"live"`
	// Unreadable marks a claim file nobody can parse: it carries only its
	// record, and its lease is the grace after the file was written.
	Unreadable bool `json:"unreadable,omitempty"`
}

// UnreadableClaimGrace is how long a claim file nobody can parse holds its
// record, counted from the file's modification time. Every claim is written
// under the run's lock in one create, so an unparseable file is a writer killed
// between the create and the write; the grace is only long enough that a
// reader racing that writer backs off rather than taking the record.
const UnreadableClaimGrace = time.Minute

// ClaimRequest is what a session asks for.
type ClaimRequest struct {
	Session string
	Record  string
	Lane    string
	// Lease is the claim's lifetime; zero means DefaultLease.
	Lease time.Duration
	// Paths are the repository-relative files the lane expects to touch, when the
	// caller knows them. For the second session a path in the reading corpus —
	// or any path, when the corpus cannot be derived — refuses the claim (see
	// ReadingCorpus).
	Paths []string
}

// ClaimResult is what a granted claim reports.
type ClaimResult struct {
	Claim Claim `json:"claim"`
	// Renewed is true when the session already held the record and the lease was
	// extended instead of a second claim being taken.
	Renewed bool `json:"renewed"`
	// Lapsed is the expired claim this one replaced, when there was one.
	Lapsed *Claim `json:"lapsed,omitempty"`
}

// HeldError is the contention a refused claim returns: the record and the
// session that holds it.
type HeldError struct {
	Holder Claim
}

func (e *HeldError) Error() string {
	return fmt.Sprintf("%s is claimed by session %s for lane %s until %s; back off and take other work",
		e.Holder.Record, e.Holder.Session, e.Holder.Lane, e.Holder.ExpiresAt.Format(time.RFC3339))
}

// Is makes a HeldError an ErrContention.
func (e *HeldError) Is(target error) bool { return target == ErrContention }

// claimRel is a record's claim file inside the run directory.
func claimRel(record string) string { return claimsDirName + "/" + record + ".json" }

// validRecord refuses anything that is not a record id.
func validRecord(record string) error {
	if !recordid.CitedIDRe.MatchString(record) {
		return refusal("%q is not a record id (itd-N, iss-N, spc-N or adr-N)", record)
	}
	return nil
}

// Claim takes req.Record for req.Session. In order, under the run's lock:
//
//  1. the session must have joined; its role decides the bounds;
//  2. a second session is refused when it already holds a live claim on another
//     record (one lane at a time), when the window's mode is split-roles (the
//     second builds nothing), or when req.Paths reach the reading corpus;
//  3. the claim file is created exclusively. If it exists and this session
//     holds it, the lease is renewed. If another session's lease has passed, the
//     lapse is logged, the stale file removed, and the create retried. If
//     another session's lease holds, the denial is logged naming the holder —
//     and for the second session a backoff is logged beside it — and a
//     *HeldError (ErrContention) is returned.
//
// A granted claim is logged as `claim`. When that line cannot be written the
// claim is removed again, so the claims directory never holds a claim the log
// does not.
func (r *Run) Claim(req ClaimRequest) (ClaimResult, error) {
	if err := validRecord(req.Record); err != nil {
		return ClaimResult{}, err
	}
	if err := validName("lane", req.Lane); err != nil {
		return ClaimResult{}, err
	}
	lease := req.Lease
	if lease == 0 {
		lease = DefaultLease
	}
	if lease < MinLease || lease > MaxLease {
		return ClaimResult{}, refusal("lease %s is outside %s to %s", lease, MinLease, MaxLease)
	}
	for _, p := range req.Paths {
		if !fsutil.ValidRelPath(p) {
			return ClaimResult{}, refusal("path %q is not a repository-relative path", p)
		}
	}
	var out ClaimResult
	err := r.withLock(func() error {
		s, err := r.requireSession(req.Session)
		if err != nil {
			return err
		}
		root, err := r.root()
		if err != nil {
			return err
		}
		defer root.Close()
		if s.Role == RoleSecond {
			if err := r.secondClaimBounds(root, req); err != nil {
				return err
			}
		}
		now := r.now()
		c := Claim{Record: req.Record, Session: req.Session, Lane: req.Lane, ClaimedAt: now, ExpiresAt: now.Add(lease)}
		data, err := encodeClaim(c)
		if err != nil {
			return err
		}
		for attempt := 0; ; attempt++ {
			err = fsutil.CreateExclusiveIn(root, claimRel(req.Record), data, fileMode)
			if err == nil {
				break
			}
			if !errors.Is(err, os.ErrExist) || attempt > 0 {
				return fmt.Errorf("cannot take the claim: %w", err)
			}
			held, rerr := r.readClaim(root, req.Record)
			var bad *UnreadableClaimError
			if errors.As(rerr, &bad) {
				// Nobody can say who holds it. Within the grace it is contention;
				// after it the file has lapsed, and the lapse is logged as one.
				if now.Before(bad.LapsesAt) {
					return fmt.Errorf("%w: %v; it lapses at %s, so back off and retry after then",
						ErrContention, bad, bad.LapsesAt.Format(time.RFC3339))
				}
				if _, err := r.append(req.Session, EventClaimLapsed, map[string]any{
					"record": req.Record, "reason": "unparseable", "path": claimRel(req.Record),
					"expired_at": bad.LapsesAt.Format(time.RFC3339),
				}); err != nil {
					return err
				}
				if err := root.Remove(claimRel(req.Record)); err != nil {
					return fmt.Errorf("cannot remove the unreadable claim %s: %w", bad.Path, err)
				}
				continue
			}
			if rerr != nil {
				return rerr
			}
			switch {
			case held.Session == req.Session:
				// Renewal: the holder extends its own lease, lane included.
				if err := fsutil.WriteFileAtomicInRoot(root, claimRel(req.Record), data, fileMode); err != nil {
					return fmt.Errorf("cannot renew the claim: %w", err)
				}
				out.Renewed = true
				out.Claim = c
				_, err := r.append(req.Session, EventClaim, claimFields(c, map[string]any{"renewed": true}))
				return err
			case !now.Before(held.ExpiresAt):
				if _, err := r.append(req.Session, EventClaimLapsed, map[string]any{
					"record": held.Record, "lane": held.Lane, "holder": held.Session, "reason": "expired",
					"expired_at": held.ExpiresAt.Format(time.RFC3339),
				}); err != nil {
					return err
				}
				if err := root.Remove(claimRel(req.Record)); err != nil {
					return fmt.Errorf("cannot remove the lapsed claim: %w", err)
				}
				lapsed := held
				out.Lapsed = &lapsed
				continue
			default:
				if _, err := r.append(req.Session, EventClaimDenied, map[string]any{
					"record": req.Record, "lane": req.Lane, "holder": held.Session, "holder_lane": held.Lane,
				}); err != nil {
					return err
				}
				if s.Role == RoleSecond {
					if _, err := r.append(req.Session, EventBackoff, map[string]any{
						"on": "claim", "record": req.Record,
						"reason": "record claimed by session " + held.Session, "minutes": 0,
					}); err != nil {
						return err
					}
				}
				return &HeldError{Holder: held}
			}
		}
		if _, err := r.append(req.Session, EventClaim, claimFields(c, nil)); err != nil {
			_ = root.Remove(claimRel(req.Record))
			return err
		}
		out.Claim = c
		return nil
	})
	return out, err
}

// secondClaimBounds applies the second session's bounds to a claim.
func (r *Run) secondClaimBounds(root *os.Root, req ClaimRequest) error {
	claims, err := r.listClaims(root)
	if err != nil {
		return err
	}
	now := r.now()
	for _, c := range claims {
		if c.Claim.Session == req.Session && c.Claim.Record != req.Record && now.Before(c.Claim.ExpiresAt) {
			return r.refuseLogged(req.Session, "second_session_lane_cap",
				map[string]any{"record": req.Record, "held": c.Claim.Record},
				fmt.Sprintf("the second session holds at most one lane, and it holds %s (release it first)", c.Claim.Record))
		}
	}
	w, ok, err := r.CurrentMode()
	if err != nil {
		return err
	}
	if ok && w.Mode == ModeSplitRoles {
		return r.refuseLogged(req.Session, "split_roles_second_builds_nothing", map[string]any{"record": req.Record},
			"in a split-roles window the second session reviews, audits and lands; it opens no lane")
	}
	return r.corpusBound(req.Session, map[string]any{"record": req.Record}, req.Paths)
}

// Release gives up a session's claim on a record and logs claim_released. Only
// the holder releases: a session cannot free another's claim, which lapses on
// its own lease instead.
func (r *Run) Release(session, record string) (Claim, error) {
	if err := validRecord(record); err != nil {
		return Claim{}, err
	}
	var out Claim
	err := r.withLock(func() error {
		if _, err := r.requireSession(session); err != nil {
			return err
		}
		root, err := r.root()
		if err != nil {
			return err
		}
		defer root.Close()
		held, err := r.readClaim(root, record)
		if errors.Is(err, os.ErrNotExist) {
			return refusal("%s is not claimed", record)
		}
		var bad *UnreadableClaimError
		if errors.As(err, &bad) {
			return refusal("%v; no session holds it to release, and it lapses at %s, when a claim replaces it",
				bad, bad.LapsesAt.Format(time.RFC3339))
		}
		if err != nil {
			return err
		}
		if held.Session != session {
			return refusal("%s is claimed by session %s, not %s; only the holder releases a claim", record, held.Session, session)
		}
		out = held
		return r.releaseLocked(root, held, "released")
	})
	return out, err
}

// releaseLocked removes a claim and logs it. The caller holds the lock.
func (r *Run) releaseLocked(root *os.Root, c Claim, reason string) error {
	if err := root.Remove(claimRel(c.Record)); err != nil {
		return fmt.Errorf("cannot release the claim on %s: %w", c.Record, err)
	}
	_, err := r.append(c.Session, EventClaimReleased, claimFields(c, map[string]any{"reason": reason}))
	return err
}

// Claims lists every claim file, live or lapsed, by record.
func (r *Run) Claims() ([]ClaimState, error) {
	if !r.exists {
		return []ClaimState{}, nil
	}
	root, err := r.root()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	return r.listClaims(root)
}

// listClaims reads the claims directory.
func (r *Run) listClaims(root *os.Root) ([]ClaimState, error) {
	out := []ClaimState{}
	dir, err := root.Open(claimsDirName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return nil, err
	}
	sort.Strings(names)
	now := r.now()
	for _, n := range names {
		record, ok := strings.CutSuffix(n, ".json")
		if !ok || !recordid.CitedIDRe.MatchString(record) {
			continue
		}
		c, err := r.readClaim(root, record)
		var bad *UnreadableClaimError
		if errors.As(err, &bad) {
			// Listed, never fatal: one torn file must not stop the status, leave
			// or another record's claim from reading every other claim.
			out = append(out, ClaimState{Claim: Claim{Record: record, ExpiresAt: bad.LapsesAt},
				Live: now.Before(bad.LapsesAt), Unreadable: true})
			continue
		}
		if err != nil {
			return nil, err
		}
		out = append(out, ClaimState{Claim: c, Live: now.Before(c.ExpiresAt)})
	}
	return out, nil
}

// UnreadableClaimError is a claim file nobody can parse — a session killed
// between the exclusive create and the write leaves an empty one. It names the
// file's full path and when it lapses, never a guess at who holds it.
type UnreadableClaimError struct {
	Record   string
	Path     string
	LapsesAt time.Time
}

func (e *UnreadableClaimError) Error() string {
	return fmt.Sprintf("the claim file %s is unreadable; nothing can say who holds %s", e.Path, e.Record)
}

// readClaim reads one claim file. An unparseable one is an
// *UnreadableClaimError lapsing UnreadableClaimGrace after the file was last
// written.
func (r *Run) readClaim(root *os.Root, record string) (Claim, error) {
	rel := claimRel(record)
	data, err := fsutil.ReadGuardedInRoot(root, rel, maxRecordBytes)
	if err != nil {
		return Claim{}, err
	}
	var c Claim
	if err := json.Unmarshal(data, &c); err != nil || c.Record != record || c.Session == "" || c.ExpiresAt.IsZero() {
		bad := &UnreadableClaimError{Record: record, Path: filepath.Join(r.Dir, filepath.FromSlash(rel))}
		fi, serr := root.Stat(rel)
		if serr != nil {
			return Claim{}, fmt.Errorf("%v, and its age cannot be read: %w", bad, serr)
		}
		bad.LapsesAt = fi.ModTime().UTC().Add(UnreadableClaimGrace)
		return Claim{}, bad
	}
	return c, nil
}

// encodeClaim renders a claim file.
func encodeClaim(c Claim) ([]byte, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// claimFields is a claim's log fields plus extras.
func claimFields(c Claim, extra map[string]any) map[string]any {
	f := map[string]any{
		"record": c.Record, "lane": c.Lane,
		"expires_at":    c.ExpiresAt.Format(time.RFC3339),
		"lease_minutes": int(c.ExpiresAt.Sub(c.ClaimedAt) / time.Minute),
	}
	for k, v := range extra {
		f[k] = v
	}
	return f
}
