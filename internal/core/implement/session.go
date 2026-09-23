package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
)

// Role is a session's standing in the run. The first session is the run: it may
// cut a release and take any lane, and it never waits on the second. The second
// is optional and bounded (the intent's decision 3): at most one lane, never the
// release, never a lane that touches the reading corpus, and it backs off on
// contention.
//
// The role lives in the session's record in the run state, written when the
// session joins. It is deliberately not read from the environment: a variable a
// shell, a task runner or a repository's configuration can set is not a
// statement the session made, and the bounds key on it. What the record gives
// is consistency, not authentication — two sessions of one account can each
// write anything under that account's home — so the bounds are a discipline two
// cooperating sessions keep, checked at every verb, not a wall against a hostile
// one.
type Role string

// The two roles.
const (
	RoleFirst  Role = "first"
	RoleSecond Role = "second"
)

// Roles returns the closed role vocabulary.
func Roles() []Role { return []Role{RoleFirst, RoleSecond} }

// ParseRole accepts exactly one of Roles.
func ParseRole(s string) (Role, error) {
	for _, r := range Roles() {
		if string(r) == s {
			return r, nil
		}
	}
	return "", refusal("unknown role %q (first or second)", s)
}

// Mode is a window's division of work between the sessions. `single` is a
// window with one session; the three the intent measures are `claim` (a session
// claims a record before opening its lane), `batch` (the run file assigns whole
// batches per session) and `split-roles` (the first builds; the second reviews,
// audits and lands).
type Mode string

// The four modes.
const (
	ModeSingle     Mode = "single"
	ModeClaim      Mode = "claim"
	ModeBatch      Mode = "batch"
	ModeSplitRoles Mode = "split-roles"
)

// Modes returns the closed mode vocabulary, in the order the run tries them.
func Modes() []Mode { return []Mode{ModeSingle, ModeClaim, ModeBatch, ModeSplitRoles} }

// ParseMode accepts exactly one of Modes.
func ParseMode(s string) (Mode, error) {
	for _, m := range Modes() {
		if string(m) == s {
			return m, nil
		}
	}
	return "", refusal("unknown mode %q (one of: single, claim, batch, split-roles)", s)
}

// Session is a joined session's record.
type Session struct {
	Session  string    `json:"session"`
	Role     Role      `json:"role"`
	JoinedAt time.Time `json:"joined_at"`
	Model    string    `json:"model,omitempty"`
	// Ceiling is the session's own agent ceiling as it stated it on joining,
	// zero when it stated none.
	Ceiling int `json:"ceiling,omitempty"`
}

// MaxCeiling bounds a stated agent ceiling.
//
// The ceiling is the second session's own limit on the agents it runs at once,
// on top of the first session's (itd-2609221656373558, criterion 5). abcd runs
// no agent and counts none — agent_start and agent_end are lines the session
// writes by hand — so the ceiling is recorded and reported, never enforced
// here: the session states it on joining, the record and the session_open line
// carry it, and every check reports it back to the session about to act.
const MaxCeiling = 64

// maxRecordBytes caps a session or claim record on read.
const maxRecordBytes = 64 << 10

// sessionRel is a session record's path inside the run directory.
func sessionRel(id string) string { return sessionsDirName + "/" + id + ".json" }

// JoinResult is what Join reports.
type JoinResult struct {
	Session Session `json:"session"`
	// Rejoined is true when the session's record already existed with the same
	// role: a resumed session opens again rather than being refused.
	Rejoined bool `json:"rejoined"`
}

// Join records a session in the run and logs its session_open. Nothing signals
// any other session: joining is a file the joiner writes and a log line, and
// the first session learns of a second only if it reads the run state. The
// record is taken by an exclusive create; a session that joins again with the
// role it holds is a resume, and one that asks for a different role is refused.
// ceiling is the session's own agent ceiling (see MaxCeiling), zero for none; a
// resume keeps the ceiling it joined with and refuses a different one.
func (r *Run) Join(id string, role Role, model, reason string, ceiling int) (JoinResult, error) {
	if err := validName("session", id); err != nil {
		return JoinResult{}, err
	}
	if _, err := ParseRole(string(role)); err != nil {
		return JoinResult{}, err
	}
	if ceiling < 0 || ceiling > MaxCeiling {
		return JoinResult{}, refusal("ceiling %d is outside 0 (none stated) to %d", ceiling, MaxCeiling)
	}
	var out JoinResult
	err := r.withLock(func() error {
		root, err := r.root()
		if err != nil {
			return err
		}
		defer root.Close()
		s := Session{Session: id, Role: role, JoinedAt: r.now(), Model: model, Ceiling: ceiling}
		data, err := json.MarshalIndent(s, "", "  ")
		if err != nil {
			return err
		}
		err = fsutil.CreateExclusiveIn(root, sessionRel(id), append(data, '\n'), fileMode)
		switch {
		case err == nil:
			out.Session = s
		case errors.Is(err, os.ErrExist):
			prev, rerr := readSession(root, id)
			if rerr != nil {
				return rerr
			}
			if prev.Role != role {
				return refusal("session %s joined as %s; a session keeps its role (leave first to rejoin as %s)", id, prev.Role, role)
			}
			if ceiling != 0 && ceiling != prev.Ceiling {
				return refusal("session %s joined with ceiling %d; a session keeps its ceiling (leave first to rejoin with %d)", id, prev.Ceiling, ceiling)
			}
			out.Session, out.Rejoined = prev, true
		default:
			return fmt.Errorf("cannot record the session: %w", err)
		}
		fields := map[string]any{"role": string(role)}
		if model != "" {
			fields["model"] = model
		}
		if out.Session.Ceiling > 0 {
			fields["ceiling"] = out.Session.Ceiling
		}
		if reason != "" {
			fields["reason"] = reason
		}
		if out.Rejoined {
			fields["rejoin"] = true
		}
		if _, err := r.append(id, EventSessionOpen, fields); err != nil {
			if !out.Rejoined {
				_ = root.Remove(sessionRel(id))
			}
			return err
		}
		return nil
	})
	return out, err
}

// LeaveResult is what Leave reports.
type LeaveResult struct {
	Session  Session `json:"session"`
	Released []Claim `json:"released"`
}

// Leave closes a session: it releases every claim the session holds, logs
// session_close with the reason, and removes the session's record. A session
// that stops without leaving is covered by its claims' leases instead.
func (r *Run) Leave(id, reason string) (LeaveResult, error) {
	if err := validName("session", id); err != nil {
		return LeaveResult{}, err
	}
	out := LeaveResult{Released: []Claim{}}
	err := r.withLock(func() error {
		s, err := r.requireSession(id)
		if err != nil {
			return err
		}
		out.Session = s
		root, err := r.root()
		if err != nil {
			return err
		}
		defer root.Close()
		claims, err := r.listClaims(root)
		if err != nil {
			return err
		}
		for _, c := range claims {
			if c.Claim.Session != id {
				continue
			}
			if err := r.releaseLocked(root, c.Claim, "session closed"); err != nil {
				return err
			}
			out.Released = append(out.Released, c.Claim)
		}
		fields := map[string]any{"role": string(s.Role)}
		if reason != "" {
			fields["reason"] = reason
		}
		if _, err := r.append(id, EventSessionClose, fields); err != nil {
			return err
		}
		return root.Remove(sessionRel(id))
	})
	return out, err
}

// readSession reads one session record.
func readSession(root *os.Root, id string) (Session, error) {
	data, err := fsutil.ReadGuardedInRoot(root, sessionRel(id), maxRecordBytes)
	if err != nil {
		return Session{}, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil || s.Session != id {
		return Session{}, fmt.Errorf("the session record for %s is unreadable; remove %s by hand to rejoin", id, sessionRel(id))
	}
	if _, err := ParseRole(string(s.Role)); err != nil {
		return Session{}, fmt.Errorf("the session record for %s carries an unknown role", id)
	}
	return s, nil
}

// requireSession returns a joined session's record, or refuses: every verb that
// writes acts for a session, and a session the run does not know has no role to
// bound it by.
func (r *Run) requireSession(id string) (Session, error) {
	if err := validName("session", id); err != nil {
		return Session{}, err
	}
	if !r.exists {
		return Session{}, refusal("session %s has not joined this run (run `abcd implement join` first)", id)
	}
	root, err := r.root()
	if err != nil {
		return Session{}, err
	}
	defer root.Close()
	s, err := readSession(root, id)
	if errors.Is(err, os.ErrNotExist) {
		return Session{}, refusal("session %s has not joined this run (run `abcd implement join` first)", id)
	}
	return s, err
}

// Sessions lists the joined sessions, by join time.
func (r *Run) Sessions() ([]Session, error) {
	out := []Session{}
	if !r.exists {
		return out, nil
	}
	entries, err := os.ReadDir(r.Dir + "/" + sessionsDirName)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return out, nil
		}
		return nil, err
	}
	root, err := r.root()
	if err != nil {
		return nil, err
	}
	defer root.Close()
	for _, e := range entries {
		id, ok := strings.CutSuffix(e.Name(), ".json")
		if !ok || e.IsDir() || !nameRe.MatchString(id) {
			continue
		}
		s, err := readSession(root, id)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].JoinedAt.Before(out[j].JoinedAt) })
	return out, nil
}

// WindowState is the division mode in force, as the log last recorded it.
type WindowState struct {
	Mode   Mode      `json:"mode"`
	Window int       `json:"window,omitempty"`
	Since  time.Time `json:"since"`
	SetBy  string    `json:"set_by"`
}

// SetMode opens a window: it logs window_mode with the mode and, when given, the
// window's number. Only the first session sets it — the second tells the first
// nothing, and the window's mode is how the first divides its own run.
func (r *Run) SetMode(session string, m Mode, window int) (WindowState, error) {
	if _, err := ParseMode(string(m)); err != nil {
		return WindowState{}, err
	}
	if window < 0 {
		return WindowState{}, refusal("window %d is negative", window)
	}
	var out WindowState
	err := r.withLock(func() error {
		s, err := r.requireSession(session)
		if err != nil {
			return err
		}
		if s.Role != RoleFirst {
			return r.refuseLogged(session, "second_session_sets_no_mode", map[string]any{"mode": string(m)},
				"the window's mode is the first session's to set; the second session works within it")
		}
		fields := map[string]any{"mode": string(m)}
		if window > 0 {
			fields["window"] = window
		}
		ts, err := r.append(session, EventWindowMode, fields)
		out = WindowState{Mode: m, Window: window, Since: ts, SetBy: session}
		return err
	})
	return out, err
}

// CurrentMode returns the mode the log last recorded, whoever wrote the line (a
// hand-written window_mode counts the same as one SetMode wrote). ok is false
// when no window has been opened; an unrecognised mode word is returned as it
// was written, and the bounds treat it as no split.
func (r *Run) CurrentMode() (WindowState, bool, error) {
	events, _, err := r.ReadLog()
	if err != nil {
		return WindowState{}, false, err
	}
	var last *Event
	for i := range events {
		if events[i].Event != EventWindowMode {
			continue
		}
		if last == nil || !events[i].TS.Before(last.TS) {
			last = &events[i]
		}
	}
	if last == nil {
		return WindowState{}, false, nil
	}
	w := WindowState{Mode: Mode(last.String("mode")), Since: last.TS, SetBy: last.Session}
	if n, ok := last.Number("window"); ok {
		w.Window = int(n)
	}
	return w, true, nil
}

// refuseLogged logs a refusal event for a bound the session met and returns the
// ErrRefused-classed error naming it. The act is refused whether or not the log
// line lands; a log failure is folded into the message rather than masking the
// refusal.
func (r *Run) refuseLogged(session, condition string, fields map[string]any, msg string) error {
	f := map[string]any{"condition": condition}
	for k, v := range fields {
		f[k] = v
	}
	if _, err := r.append(session, EventRefusal, f); err != nil {
		return refusal("%s (and the refusal could not be logged: %v)", msg, err)
	}
	return refusal("%s", msg)
}
