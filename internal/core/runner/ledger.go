package runner

// ledger.go is the launch ledger (itd-2611031342011261): one entry per agent
// abcd starts, written under abcd's home before the harness starts and
// completed when it ends, so a session or a run left on its own can account
// for every agent it started. RecordLaunch is the one way an entry is written:
// the dispatcher's launches, its hand-offs to the host and any launcher outside
// the dispatcher all pass through it.
//
// The store is one file per launch, each holding one line of JSON, so two
// launches never write one file and the folder read end to end is one line per
// launch. Only the process that launched completes its own entry, by replacing
// the file; a reader takes what it knows and never writes back (ledger_read.go).
// The run state names nothing of the ledger, and the ledger keeps its own
// version: its readers are more often behind the writer than a run's driver is
// (iss-2611020719494343).
//
// A cost is what the harness reported, an estimate at its own price table. A
// key is named, never carried: nothing here reads a credential.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/adapter/scanner"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/core/sessionname"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// LedgerSchema is the version of the entries this build writes. A reader takes
// an entry above it as far as it can and names it as newer.
const LedgerSchema = 1

// ledgerFolder is the store, under abcd's home.
const ledgerFolder = "agents"

// The billing routes an entry names. Only RouteKey's cost counts as spend.
const (
	RouteKey          = "key"
	RouteSubscription = "subscription"
	RouteHost         = "host"
	RouteUnknown      = "unknown"
)

// The outcomes an entry names. OutcomeLost is never written: a reader judges
// it, of a running entry whose process is gone.
const (
	OutcomeRunning  = "running"
	OutcomeFinished = "finished"
	OutcomeFailed   = "failed"
	OutcomeStopped  = "stopped"
	OutcomeHanded   = "handed-to-host"
	OutcomeLost     = "lost"
)

// The kinds of orchestrator: a host session, or a run with no session over it.
const (
	OrchestratorSession = "session"
	OrchestratorRun     = "run"
)

// Orchestrator is who asked for a launch.
type Orchestrator struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
	// Name is a session's persona name, "" when it has none.
	Name string `json:"name,omitempty"`
}

// LaunchWork is the piece of work a launch is for: the run, lane and await, or
// an interview and its turn. Each is free of meaning to the ledger but for the
// listing, and two launches naming one await are named as a pair.
type LaunchWork struct {
	Run   string
	Lane  string
	Await string
}

// LaunchStart is what is known of a launch before it starts.
type LaunchStart struct {
	Role string
	// Runner is the runner that will run it, or Host for a role handed over.
	Runner string
	// ModelAsked is the model the runner asks for, "" for the harness's own.
	ModelAsked string
	// Repo is the checkout the launch works for.
	Repo string
	Work LaunchWork
	// Fallback names the orchestrator when no session and no run does: the
	// launch's own transcript session.
	Fallback string
	// Key is the configured name of the key abcd handed this launch, never
	// its value; "" when it handed none.
	Key string
}

// LaunchEnd is what a launch returned.
type LaunchEnd struct {
	Answer Answer
	// Transcript is the harness's event stream, what a failed or killed launch
	// has to show for itself.
	Transcript []byte
	// Err is the launch's own failure, which RecordLaunch records and returns
	// to the caller untouched.
	Err error
}

// LedgerEntry is one launch: one line of one file.
type LedgerEntry struct {
	Schema       int          `json:"schema_version"`
	ID           string       `json:"id"`
	Orchestrator Orchestrator `json:"orchestrator"`
	Repo         string       `json:"repo,omitempty"`
	Run          string       `json:"run,omitempty"`
	Lane         string       `json:"lane,omitempty"`
	Await        string       `json:"await,omitempty"`
	Role         string       `json:"role"`
	Runner       string       `json:"runner"`
	ModelAsked   string       `json:"model_asked,omitempty"`
	// Route is the billing route, Key the configured name of a key route's
	// key, and RouteWhy why a route reads unknown.
	Route    string `json:"route"`
	Key      string `json:"key,omitempty"`
	RouteWhy string `json:"route_why,omitempty"`
	// Model is the model the harness says answered.
	Model string        `json:"model,omitempty"`
	Usage *oracle.Usage `json:"usage,omitempty"`
	Cost  *oracle.Cost  `json:"cost,omitempty"`
	// APIMillis is the time the harness says it spent waiting on the model,
	// absent when it reports none.
	APIMillis  *int64     `json:"api_ms,omitempty"`
	WallMillis int64      `json:"wall_ms,omitempty"`
	Outcome    string     `json:"outcome"`
	Reason     string     `json:"reason,omitempty"`
	StartedAt  time.Time  `json:"started_at"`
	EndedAt    *time.Time `json:"ended_at,omitempty"`
	// PID is the process a reader judges a running entry by: the launcher's
	// until the harness has started, then the harness's. PIDStartUnix is the
	// harness's start, which tells it from a later process given the same id;
	// zero while PID is the launcher's.
	PID          int   `json:"pid,omitempty"`
	PIDStartUnix int64 `json:"pid_start_unix,omitempty"`
}

// Retention is how long the store keeps entries: the next writer prunes the
// oldest whole files past either limit, and never one inside the floor.
type Retention struct {
	Days  int
	Count int
}

// The shipped retention (decision 14).
const (
	DefaultRetentionDays  = 30
	DefaultRetentionCount = 20000
)

// The most a figure may say: past them a figure is no retention at all, and the
// ledger never grows without bound.
const (
	maxRetentionDays  = 36500
	maxRetentionCount = 1000000
)

// RetentionFloorDays is the shortest the store ever keeps an entry: the longest
// cap period a key's budget counts, a calendar month of 31 days
// (itd-2611011230226453), plus one day, so a month's totals are always summed
// from entries still held.
const RetentionFloorDays = 32

// defaultRetention is the shipped retention as the floor leaves it.
func defaultRetention() Retention {
	return Retention{Days: max(DefaultRetentionDays, RetentionFloorDays), Count: DefaultRetentionCount}
}

// Ledger is the store one machine's launches are written to.
type Ledger struct {
	home      string
	now       func() time.Time
	retention Retention
}

// newLedger is the ledger under home's abcd folder, at the shipped retention.
// "" names no home, and a launch recorded there is refused.
func newLedger(home string) *Ledger {
	return &Ledger{home: home, now: func() time.Time { return time.Now().UTC() }, retention: defaultRetention()}
}

// keyValuePrefix opens a provider's key value, never a name someone chose.
const keyValuePrefix = "sk-"

// keyValueRunRe is an unbroken run of letters and digits long enough to be a
// token: what a value holds and a name someone chose does not.
var keyValueRunRe = regexp.MustCompile(`[a-z0-9]{20,}`)

// keyNameOK reports whether s can be a key's configured name: the credential
// store's own name, and nothing shaped like a value. A name is all an entry
// keeps of a key, so a caller's mistake stops here rather than in the store.
func keyNameOK(s string) bool {
	if !credential.ValidName(s) || strings.HasPrefix(s, keyValuePrefix) {
		return false
	}
	return !slices.ContainsFunc(keyValueRunRe.FindAllString(s, -1), func(run string) bool {
		return strings.ContainsAny(run, "0123456789")
	}) && len(secretFindings(s)) == 0
}

// maxLedgerReason bounds a reason as an entry carries it.
const maxLedgerReason = 300

// reasonValidator is all an entry says of a validator's refusal. The refusal's
// own words quote the answer, which the agent wrote and which can repeat its
// brief, so they stay on the run's record and off this one.
const reasonValidator = "the contract's validator refused its answer"

// staleLedgerTemp is how old the atomic writer's temporary file must be before
// the pruner takes it for one a crash left: a racing launch's lasts a moment.
const staleLedgerTemp = 24 * time.Hour

type startedKey struct{}

// launchStarted tells the launch ctx belongs to the process it started, so its
// entry is judged by that process. at is read just before the start, since a
// process's start time is when it forked and exec can take seconds after that.
// The runners' launcher calls it; a launcher that starts its own process inside
// RecordLaunch calls it too.
func launchStarted(ctx context.Context, pid int, at time.Time) {
	if f, ok := ctx.Value(startedKey{}).(func(int, time.Time)); ok {
		f(pid, at)
	}
}

// launchExited tells the launch ctx belongs to that its process has gone and the
// launcher still holds it: validating an answer takes time, and an entry whose
// launch is still being finished is not lost.
func launchExited(ctx context.Context) { launchStarted(ctx, 0, time.Time{}) }

// RecordLaunch is the one way an entry is written. It writes the entry marked
// running, runs run with a context launchStarted answers on, and completes the
// entry from what run returned; a nil run is a role handed to the host, which
// gets a placeholder with no figures (decision 21).
//
// An error means nothing ran: the entry could not be written, and a launch that
// could not be recorded is not started, since the claim is that none escapes
// the record. The launch's own failure comes back in LaunchEnd.Err. A completion
// that cannot be written leaves the answer standing and the entry as it was,
// which a reader shows as lost once the process is gone.
func RecordLaunch(ctx context.Context, l *Ledger, s LaunchStart, run func(context.Context) LaunchEnd) (LaunchEnd, error) {
	if l == nil {
		return LaunchEnd{}, errors.New("runner: no launch ledger was opened, so nothing is started")
	}
	if s.Key != "" && !keyNameOK(s.Key) {
		return LaunchEnd{}, errors.New("runner: the key handed to this launch is not named by its configured name, " +
			"so nothing is started and nothing is written to the ledger")
	}
	h, err := l.begin(s)
	if err != nil {
		return LaunchEnd{}, fmt.Errorf("runner: the launch ledger at %s could not record this launch, so nothing is started: %w",
			abcdhome.Display(ledgerFolder), err)
	}
	if run == nil {
		h.handed()
		return LaunchEnd{}, nil
	}
	end := run(context.WithValue(ctx, startedKey{}, h.started))
	h.end(end)
	return end, nil
}

// openStore is the store, made if it is missing, held so a link swapped in
// cannot move a write.
func (l *Ledger) openStore() (*os.Root, error) {
	if l.home == "" {
		return nil, errors.New("this process has no home for it")
	}
	return fsutil.EnsureHomeScope(l.home, abcdhome.Rel(ledgerFolder), abcdhome.DirMode)
}

// launchHandle is one running entry and the store it is written to.
type launchHandle struct {
	mu    sync.Mutex
	l     *Ledger
	entry LedgerEntry
}

func (l *Ledger) begin(s LaunchStart) (*launchHandle, error) {
	id, err := newLaunchID(l.now())
	if err != nil {
		return nil, err
	}
	route, key := RouteUnknown, ""
	if s.Key != "" {
		route, key = RouteKey, s.Key
	}
	h := &launchHandle{l: l, entry: LedgerEntry{
		Schema: LedgerSchema, ID: id, Orchestrator: l.orchestrator(s), Repo: s.Repo,
		Run: cleanHarness(s.Work.Run), Lane: cleanHarness(s.Work.Lane), Await: cleanHarness(s.Work.Await),
		Role: s.Role, Runner: s.Runner, ModelAsked: s.ModelAsked, Route: route, Key: key,
		Outcome: OutcomeRunning, StartedAt: l.now(), PID: os.Getpid(),
	}}
	if err := h.write(); err != nil {
		return nil, err
	}
	l.prune()
	return h, nil
}

// prune removes the oldest whole files past the retention, the next writer's
// chore. Names sort oldest first and say when each launch started, so no entry
// is opened. Nothing inside the floor goes, whatever the figures say, and a
// name that says no time is passed over. A failure is the next writer's to
// retry: it costs nothing a launch needs.
func (l *Ledger) prune() {
	root, err := l.openStore()
	if err != nil {
		return
	}
	defer root.Close()
	dir, err := root.Open(".")
	if err != nil {
		return
	}
	names, err := dir.Readdirnames(-1)
	dir.Close()
	if err != nil {
		return
	}
	now := l.now()
	for _, n := range names {
		if !strings.HasPrefix(n, tempFilePrefix) {
			continue
		}
		if fi, err := root.Lstat(n); err == nil && fi.Mode().IsRegular() && now.Sub(fi.ModTime()) > staleLedgerTemp {
			_ = root.Remove(n)
		}
	}
	names = slices.DeleteFunc(names, func(n string) bool { return !launchFileRe.MatchString(n) })
	slices.Sort(names)
	floor := now.AddDate(0, 0, -RetentionFloorDays)
	oldest := now.AddDate(0, 0, -l.retention.Days)
	for i, name := range names {
		at, err := launchFileStarted(name)
		if err != nil {
			continue
		}
		if !at.Before(floor) {
			return
		}
		if at.Before(oldest) || len(names)-i > l.retention.Count {
			_ = root.Remove(name)
		}
	}
}

// launchFileStarted is when the launch a file name belongs to started, to the
// second.
func launchFileStarted(name string) (time.Time, error) {
	return time.Parse(launchIDTime, strings.TrimPrefix(name, "lch-")[:len(launchIDTime)])
}

// orchestrator names who asked for s: the session this process runs under,
// else the run, else the launch's own session, so no entry is unattributed.
func (l *Ledger) orchestrator(s LaunchStart) Orchestrator {
	if o, ok := CurrentSession(l.home); ok {
		return o
	}
	id := s.Work.Run
	if id == "" {
		id = s.Fallback
	}
	return Orchestrator{Kind: OrchestratorRun, ID: cleanHarness(id)}
}

// HostSessionEnv is where the host exports the id of the session running a
// command (decision 10).
const HostSessionEnv = "CLAUDE_CODE_SESSION_ID"

// CurrentSession is the session this process runs under, with the persona name
// home's register holds for it: the one the host exports, else one whose host
// process it descends from. It reports false under neither, as a run driven
// from a plain terminal is.
func CurrentSession(home string) (Orchestrator, bool) {
	m := sessionname.Live()
	m.Home, m.From = home, os.Getpid()
	if id := os.Getenv(HostSessionEnv); sessionRe.MatchString(id) {
		return Orchestrator{Kind: OrchestratorSession, ID: id, Name: sessionname.NameOf(m, id)}, true
	}
	if id, name := sessionname.SessionAbove(m); id != "" {
		return Orchestrator{Kind: OrchestratorSession, ID: id, Name: name}, true
	}
	return Orchestrator{}, false
}

// write replaces the entry's one file, whole.
func (h *launchHandle) write() error {
	line, err := json.Marshal(h.entry)
	if err != nil {
		return err
	}
	// An entry the reader would refuse records nothing.
	if len(line) >= maxLedgerEntry {
		return fmt.Errorf("the entry is %d bytes, past the %d a reader takes", len(line), maxLedgerEntry)
	}
	root, err := h.l.openStore()
	if err != nil {
		return err
	}
	defer root.Close()
	return fsutil.WriteFileAtomicInRoot(root, h.entry.ID+ledgerExt, append(line, '\n'), abcdhome.FileMode)
}

// started moves the entry onto the process that runs the launch, or back onto
// the launcher when pid is 0. A write that fails leaves it judged by the
// launcher, which waits on that process.
func (h *launchHandle) started(pid int, at time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if pid == 0 {
		h.entry.PID, h.entry.PIDStartUnix = os.Getpid(), 0
	} else {
		h.entry.PID, h.entry.PIDStartUnix = pid, at.Unix()
	}
	_ = h.write()
}

// handed completes a placeholder. The entry stands whatever this write does.
func (h *launchHandle) handed() {
	h.mu.Lock()
	defer h.mu.Unlock()
	at := h.l.now()
	h.entry.Route, h.entry.Key, h.entry.Outcome, h.entry.EndedAt = RouteHost, "", OutcomeHanded, &at
	h.entry.PID, h.entry.PIDStartUnix = 0, 0
	_ = h.write()
}

// end completes the entry from what the launch returned.
func (h *launchHandle) end(end LaunchEnd) {
	h.mu.Lock()
	defer h.mu.Unlock()
	at := h.l.now()
	e := &h.entry
	e.EndedAt, e.WallMillis = &at, at.Sub(e.StartedAt).Milliseconds()
	e.PID, e.PIDStartUnix = 0, 0
	e.Model, e.Usage, e.Cost = end.Answer.Model, end.Answer.Usage, end.Answer.Cost
	var facts claudeFacts
	if e.Runner == Claude {
		// A failed launch returns no answer; its stream shows what it spent,
		// and the model the harness said it started on.
		facts = claudeLaunchFacts(end.Transcript)
		e.APIMillis = facts.apiMillis
		if e.Model == "" {
			e.Model = facts.model
		}
		if end.Err != nil && e.Usage == nil && e.Cost == nil {
			spent := claudeSpent(end.Transcript)
			e.Usage, e.Cost = spent.Usage, spent.Cost
		}
	}
	if e.Route != RouteKey {
		// Decision 11: a stream carrying the subscription's rate-limit windows
		// is the subscription's, and only claude's carries them; a key-billed
		// launch gets none.
		if facts.window {
			e.Route = RouteSubscription
		} else {
			e.RouteWhy = "abcd handed this launch no key and its stream shows no rate-limit window"
		}
	}
	e.Outcome = OutcomeFinished
	if end.Err != nil {
		e.Outcome, e.Reason = OutcomeFailed, ledgerReason(end.Err)
		var fl *Failure
		if errors.As(end.Err, &fl) {
			e.Reason = ledgerReason(errors.New(fl.Detail))
			if fl.Reason == ReasonInvalid {
				e.Reason = reasonValidator
			}
			if errors.Is(end.Err, ErrKilled) {
				e.Outcome = OutcomeStopped
			}
			if fl.Spent != nil && e.Usage == nil && e.Cost == nil {
				e.Usage, e.Cost = fl.Spent.Usage, fl.Spent.Cost
			}
		}
	}
	_ = h.write()
}

// ledgerReason is an error as an entry carries it: every secret span sealed,
// plain, bounded and with the person's home written as ~. Reasons are abcd's
// own words, but a word can quote what it failed on. It fails closed: text the
// seal leaves a secret in is described rather than kept.
func ledgerReason(err error) string {
	s := err.Error()
	sealed, _ := scanner.Redact(s, secretFindings(s))
	if len(secretFindings(sealed)) > 0 {
		sealed = termsafe.DescribeRefused(s)
	}
	s = termsafe.Sanitize(fsutil.RedactHome(sealed))
	if len(s) > maxLedgerReason {
		s = strings.ToValidUTF8(s[:maxLedgerReason], "") + "..."
	}
	return s
}

// secretFindings is every token the bundled patterns find in s. It reads no
// repository's configuration: a store shared by every repository is not one
// repository's to widen a hole in.
func secretFindings(s string) []scanner.Finding {
	found := scanner.ScanText(s, scanner.Identity{}, scanner.DefaultPatterns(), nil, ledgerFolder)
	return slices.DeleteFunc(found, func(f scanner.Finding) bool { return scanner.IsIdentityKind(f.Kind) })
}

// ledgerExt is an entry file's extension. The atomic writer's temporary files
// carry none, so no reader or pruner takes one for an entry.
const ledgerExt = ".json"

// launchIDTime is the leading stamp of a launch's id, so a file's name says
// when its launch started and names sort oldest first.
const launchIDTime = "060102150405"

// launchFileRe is an entry's file name.
var launchFileRe = regexp.MustCompile(`^lch-[0-9]{18}-[0-9a-f]{8}\.json$`)

// newLaunchID is a fresh id: the stamp to the microsecond, then eight random
// hex digits, so two launches in one instant, on any lane, never share a file.
func newLaunchID(at time.Time) (string, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	at = at.UTC()
	return fmt.Sprintf("lch-%s%06d-%s", at.Format(launchIDTime), at.Nanosecond()/1000, hex.EncodeToString(b[:])), nil
}
