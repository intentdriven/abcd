package loop

// drivestate.go is a drive's state as one value (spc-2611141617199772 step 4;
// itd-2611141610197753 decisions 3, 6, 7 and 9). The loop already knows which
// wait it is in, since when, until when and the act its refusal names, so it
// composes the state itself, and every surface shows the one line DriveLine
// composes from it: no surface reads the state back out of a sentence, which
// is what the lab's state-line prototype did and read wrong twice (F-11).
//
// The value says working, waiting, stalled or stopped; since when; until when,
// only where the wait names an end; on what; and the one act open to a person,
// or nothing. It is written whole, by rename, to DriveStateRel in the drive's
// checkout each time a field changes. It is not run state: it carries a
// version of its own, the next drive overwrites it, and nobody clears it. It
// carries the drive's process, when it was written and when the drive will
// have written again, so a reader can tell a drive that ended without a last
// line from one that is working.
//
// Stalled is said two ways. By time: a wait with no named end that has lasted
// the stuck bound's minutes (pace.stuck_minutes) since anything last moved,
// whatever reasons the waits gave in between (stallWord). By hand-backs: the
// stuck bound's count (pace.stuck_refusals) of lane hand-backs in a row with
// one key (handBackStreak); the drive keeps opening lanes either way
// (decision 7), so a false stalled costs one look.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DriveStateVersion is the version of the drive's state file. A reader shows
// no line for a file stating a newer one.
const DriveStateVersion = 1

// DriveStateRel is where a drive keeps its state, beside its log, relative to
// the checkout it runs from.
const DriveStateRel = driveStateDir + "/drive-state.json"

// driveStateDir is the drive's log folder, which the drive's log is also kept
// in, created 0700 as the log's is.
const driveStateDir = ".abcd/.work.local/logs"

// The four words a drive's state says.
const (
	DriveWorking = "working"
	DriveWaiting = "waiting"
	DriveStalled = "stalled"
	DriveStopped = "stopped"
)

// DriveLanes counts the drive's lanes: landed, in flight, set aside and handed
// back.
type DriveLanes struct {
	Done       int `json:"done"`
	InFlight   int `json:"in_flight"`
	SetAside   int `json:"set_aside"`
	HandedBack int `json:"handed_back"`
}

// DriveProcess is the drive's own process as a run's driver record names one
// (LiveDriver): its id, and its start in Unix seconds with how early that may
// be, so a reader tells it from a later process given the same id.
type DriveProcess struct {
	PID       int   `json:"pid"`
	StartUnix int64 `json:"start_unix"`
	SlackMS   int64 `json:"slack_ms,omitempty"`
}

// DriveState is a drive's state as the loop knows it.
type DriveState struct {
	Version int    `json:"version"`
	State   string `json:"state"`
	// Since is when the state began: for waiting and stalled, when anything
	// last moved.
	Since time.Time `json:"since"`
	// Until is the wait's named end, nil where it names none.
	Until *time.Time `json:"until,omitempty"`
	// On is the issue and its stage, or the thing outside the drive.
	On string `json:"on,omitempty"`
	// Do is the one act open to a person at a terminal, "" for none.
	Do      string       `json:"do,omitempty"`
	Lanes   DriveLanes   `json:"lanes"`
	Process DriveProcess `json:"process"`
	// Written is when the value was written, and Next the time by which the
	// drive will have written it again.
	Written time.Time `json:"written"`
	Next    time.Time `json:"next"`
}

// driveAct is a remedy with whom it is written for: Person is true only where
// the words are a person's act at a terminal. Most refusals' remedies are
// written for whoever calls the verb, which under a drive is the drive itself
// or a host session, so the caller says which, and nothing is inferred from
// the words.
type driveAct struct {
	Words  string
	Person bool
}

// hostOnlyVerbs are the commands that hand a host session's work to a run: a
// person who typed one beside a drive would step or hand a receipt to a run
// the drive is driving (F-21). A remedy naming one is never shown, whoever it
// is said to be for.
var hostOnlyVerbs = []string{"implement receipt", "implement step"}

// shown is the act as the line shows it: the words where they are a person's
// act at a terminal, "" otherwise.
func (a driveAct) shown() string {
	if !a.Person {
		return ""
	}
	for _, v := range hostOnlyVerbs {
		if strings.Contains(a.Words, v) {
			return ""
		}
	}
	return strings.TrimSpace(a.Words)
}

// driveWait is one wait of the drive: Start is when anything last moved, which
// the time rule measures from, and Until the wait's named end, zero for none.
//
// Three ends are named, and every other time the loop holds is the time of its
// next look (decisions 6 and 9): the pace window's next eligible time
// (moveWait), a rate limit's reset where the provider names one (limited),
// and, while every lane left is set aside, the next hourly retry of those
// lanes. An outage's next try, a look at origin, at a cap or at work others
// hold, a probe of a limit that names no reset, and idle's look-again in every
// other case are not ends, and carry no Until.
type driveWait struct {
	Start, Until time.Time
	On           string
	Act          driveAct
}

// driveStallBound is the bound the time rule reads: the stuck bound's minutes,
// the bundled sixty where none is given.
func driveStallBound(bound time.Duration) time.Duration {
	if bound <= 0 {
		return BundledStuckMinutes * time.Minute
	}
	return bound
}

// stallWord is the stall rule by time. A wait whose named end is still ahead
// is waiting, however long the drive has gone without moving. A wait with no
// end, or whose end has passed, is waiting until bound has passed since start
// and stalled from then on. The wait's reason plays no part, so waits whose
// reasons alternate keep the one start. A zero start is a wait not yet
// measured, which is waiting.
func stallWord(start, until time.Time, bound time.Duration, now time.Time) string {
	if !until.IsZero() && now.Before(until) {
		return DriveWaiting
	}
	if start.IsZero() || now.Sub(start) < driveStallBound(bound) {
		return DriveWaiting
	}
	return DriveStalled
}

// waitState is the state of the wait w at now: waiting or stalled by
// stallWord, since w's start (now where it has none), until its named end
// where that is still ahead, on what w names, and its act where that is a
// person's.
func waitState(w driveWait, bound time.Duration, now time.Time) DriveState {
	start := w.Start
	if start.IsZero() {
		start = now
	}
	s := DriveState{State: stallWord(start, w.Until, bound, now), Since: start, On: w.On, Do: w.Act.shown()}
	if !w.Until.IsZero() && now.Before(w.Until) {
		u := w.Until
		s.Until = &u
	}
	return s
}

// DriveLine is the one form every surface shows a drive's state in:
// `<state> <elapsed> · on <what> · you: <act>`, the on part left out where the
// state names nothing. Where there is no act it reads "nothing to do" while
// working or waiting and "needs a look" while stalled or stopped. What the
// value carries is masked for a terminal, so a stored line that was tampered
// with prints no control code.
func DriveLine(s DriveState, now time.Time) string {
	elapsed := time.Duration(0)
	if !s.Since.IsZero() {
		elapsed = now.Sub(s.Since)
	}
	line := driveText(s.State) + " " + driveElapsed(elapsed)
	if on := driveText(s.On); on != "" {
		line += " · on " + on
	}
	act := driveText(s.Do)
	switch {
	case act != "":
	case s.State == DriveStalled || s.State == DriveStopped:
		act = "needs a look"
	default:
		act = "nothing to do"
	}
	return line + " · you: " + act
}

// driveElapsed is a time the line shows: whole minutes under an hour ("40m"),
// hours and minutes from there ("2h44m").
func driveElapsed(d time.Duration) string {
	m := max(int(d/time.Minute), 0)
	if m < 60 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dh%02dm", m/60, m%60)
}

// driveText is a value of the line as it may be stored and shown: the home
// redacted and every control rune masked, on one line.
func driveText(s string) string {
	return strings.TrimSpace(termsafe.SanitizeLine(fsutil.RedactHome(s)))
}

// writeDriveState writes s whole, by rename, to DriveStateRel under repoRoot,
// mode 0600, stamped with this version, written at now and due again by
// now+every, and returns the value as stored, which is what a surface is
// handed, so the terminal shows what the file says. The logs folder and each
// folder above it within the checkout must be real directories, as the
// drive's log requires (fsutil.EnsureRealDirAll); the write itself goes through
// the checkout's root, so nothing swapped in after that check moves it outside.
// The file is rebuilt at the drive's next write, so it is not flushed to the
// disk at each one: a power cut loses the last value, never half of one.
func writeDriveState(repoRoot string, s DriveState, now time.Time, every time.Duration) (DriveState, error) {
	s = storedDriveState(s, now, every)
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return DriveState{}, err
	}
	if err := fsutil.EnsureRealDirAll(repoRoot, driveStateDir, 0o700); err != nil {
		return DriveState{}, err
	}
	root, err := os.OpenRoot(repoRoot)
	if err != nil {
		return DriveState{}, err
	}
	defer root.Close()
	if err := fsutil.WriteRebuildableInRoot(root, DriveStateRel, append(data, '\n'), 0o600); err != nil {
		return DriveState{}, err
	}
	return s, nil
}

// storedDriveState is s as writeDriveState stores it: stamped with this version,
// written at now and due again by now+every, its words masked and its home
// redacted (driveText), and its times in UTC. The terminal is handed this value
// whether or not the write succeeded, so it never shows what a file could not
// hold.
func storedDriveState(s DriveState, now time.Time, every time.Duration) DriveState {
	s.Version = DriveStateVersion
	s.State = driveText(s.State)
	s.On = driveText(s.On)
	s.Do = driveText(s.Do)
	s.Since = s.Since.UTC()
	if s.Until != nil {
		u := s.Until.UTC()
		s.Until = &u
	}
	s.Written = now.UTC()
	s.Next = now.Add(every).UTC()
	return s
}

// handBackKey is what makes two lane hand-backs the same: the hand-back's
// kind, the stage the lane was at and the check of its last logged refusal.
// The refusal's reason is not in it, since it carries run ids, counts and
// minutes that differ between two lanes failing one way.
type handBackKey struct {
	Kind, Stage, Check string
}

// handBackStreak is the stall rule by hand-backs (T7 of the lane plan): lane
// hand-backs in a row that share a key, and when the first of them came. A
// lane that reaches its pull request clears it, and a lane hand-back with
// another key starts it again at one. Hand-backs at the judgement, by the
// rule or at a reservation neither count nor clear it: most of a drain's
// hand-backs are judgements, which is the drain working.
type handBackStreak struct {
	key   handBackKey
	count int
	since time.Time
}

// note counts the hand-back r, routed at at, with refused the lane's last
// logged refusal (nil for none). Only a lane's hand-back is counted.
func (s *handBackStreak) note(r DrainRoute, refused *LaneRefused, at time.Time) {
	if r.From != "lane" {
		return
	}
	k := handBackKey{Kind: r.Kind}
	if refused != nil {
		k.Stage, k.Check = refused.Stage, refused.Check
	}
	if s.count > 0 && k == s.key {
		s.count++
		return
	}
	*s = handBackStreak{key: k, count: 1, since: at}
}

// landed clears the streak: a lane reached its pull request.
func (s *handBackStreak) landed() { *s = handBackStreak{} }

// stalled reports whether the streak has reached bound, the stuck bound's
// count of identical refusals (pace.stuck_refusals), the bundled three where
// none is given: three identical refusals make a lane stuck, and three
// identical hand-backs make the drive stalled.
func (s handBackStreak) stalled(bound int) bool {
	if bound < 1 {
		bound = BundledStuckRefusals
	}
	return s.count >= bound
}

// on names the streak: its count, the kind and where the lanes were.
func (s handBackStreak) on() string {
	on := fmt.Sprintf("%d lanes handed back in a row, %s", s.count, s.key.Kind)
	if s.key.Stage != "" {
		on += " at " + refusedAt(LaneRefused{Stage: s.key.Stage, Check: s.key.Check})
	}
	return on
}

// over is st with the streak applied: where the streak has reached bound, a
// working or waiting drive reads stalled since the streak's first hand-back,
// on the streak, with no act, which the line shows as needing a look. A drive
// already stalled, or stopped, keeps its own state, and so does a wait whose
// act is a person's: that act is the one thing the drive waits on, and the
// streak, which never holds the drive up (decision 7), shows again once the
// wait ends, so the line never hides an act behind a look.
func (s handBackStreak) over(st DriveState, bound int) DriveState {
	if !s.stalled(bound) || st.Do != "" || (st.State != DriveWorking && st.State != DriveWaiting) {
		return st
	}
	st.State, st.Since, st.Until, st.On, st.Do = DriveStalled, s.since, nil, s.on(), ""
	return st
}
