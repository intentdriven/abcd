package implement

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/intentdriven/abcd/internal/core/banlist"
	"github.com/intentdriven/abcd/internal/core/machineload"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// The load check (itd-2609231434459890, spc-2609231542463113) runs once at the
// start of `make preflight` and once at the start of the eval harness. It reads
// the machine's load and process table, warns when a program outside the running
// lanes has held a near-full core for longer than the stray limit or when the
// one-minute load is above the extreme limit, and, inside an autonomous run,
// writes the same warning to the run log as a `load` event. It never refuses,
// never waits and never signals anything: every status is a report, and the
// front door exits 0 on each of them. It is a sibling of `implement check`
// because it shares that verb's report-never-enforce verdict and this package's
// single run-log writer; it is not a Run.Check step, because a step can refuse
// and this check never can.

// The sites the check runs at.
const (
	SitePreflight   = "preflight"
	SiteEvalHarness = "eval-harness"
)

// LoadSites returns the closed site vocabulary.
func LoadSites() []string { return []string{SitePreflight, SiteEvalHarness} }

// The four statuses.
const (
	// LoadOK: nothing to warn about.
	LoadOK = "ok"
	// LoadWarning: at least one trigger fired.
	LoadWarning = "warning"
	// LoadSkipped: a CI runner, where the check does not run.
	LoadSkipped = "skipped"
	// LoadUnchecked: the machine could not be read (an unsupported platform, or
	// a read that failed) and nothing fired on what could be read.
	LoadUnchecked = "unchecked"
)

// Where the limits came from.
const (
	LimitsDefault               = "default"
	LimitsFile                  = "file"
	LimitsDefaultAfterMalformed = "default-after-malformed"
)

// The core-count sources.
const (
	CoresOnline  = "online"
	CoresRuntime = "runtime"
)

// PrivateNameMask replaces an own stray's name the private banned-names layer
// matches; WithheldNameMask replaces every name when that layer cannot be read.
const (
	PrivateNameMask  = "[private name]"
	WithheldNameMask = "[name withheld: private-names layer unreadable]"
)

// LimitsFileDisplay is the settings file as every report names it.
const LimitsFileDisplay = "~/.abcd/" + machineload.LimitsFileName

// loadCheckedEnv is the marker `make preflight` exports to its prerequisites and
// recipe, so a check started inside a preflight knows it.
const loadCheckedEnv = "ABCD_LOAD_CHECKED"

// maxLimitsBytes caps the settings file.
const maxLimitsBytes = 4 << 10

// LoadRequest is one check. Every field but Site has a working zero value; the
// others exist so a test never depends on the real machine, environment or home.
type LoadRequest struct {
	// Site is where the check runs: SitePreflight or SiteEvalHarness.
	Site string
	// Getenv reads the environment; nil is os.Getenv.
	Getenv func(string) string
	// Read reads the machine; nil is machineload.Read.
	Read func() (machineload.Snapshot, error)
	// Home is the caller's home, where ~/.abcd/load-limits lives; "" resolves it.
	Home string
	// RepoRoot is the checkout the check runs in, whose private banned-names
	// layer scrubs the own strays' names; "" is outside any checkout.
	RepoRoot string
	// RootSHA keys the run state; "" derives it from RepoRoot.
	RootSHA string
	// Self is the invocation; the zero value is this process.
	Self machineload.Self
	// Now is the clock for the run-log line; nil is time.Now.
	Now func() time.Time
	// GOOS names the platform in the cannot-check reason; "" is runtime.GOOS.
	GOOS string
}

// LoadAverages are the three load averages.
type LoadAverages struct {
	One     float64 `json:"one"`
	Five    float64 `json:"five"`
	Fifteen float64 `json:"fifteen"`
}

// LoadLimits are the limits in force and where they came from.
type LoadLimits struct {
	StrayMinutes    int     `json:"stray_minutes"`
	ExtremeLoad     float64 `json:"extreme_load"`
	Source          string  `json:"source"`
	StrayFromFile   bool    `json:"stray_from_file"`
	ExtremeFromFile bool    `json:"extreme_from_file"`
	// Malformed says why the settings file is unusable, when it is: a line
	// number and a fault class, never a value.
	Malformed string `json:"malformed,omitempty"`
}

// LoadOwnStray is one of the caller's own strays as it is printed and logged.
type LoadOwnStray struct {
	Name   string `json:"name"`
	PID    int    `json:"pid"`
	PGID   int    `json:"pgid"`
	AgeS   int64  `json:"age_s"`
	CPUPct int    `json:"cpu_pct"`
}

// LoadOtherStrays is every other account's strays: a count and a CPU total in
// cores, rounded to one decimal, and nothing else.
type LoadOtherStrays struct {
	Count int     `json:"count"`
	Cores float64 `json:"cores"`
}

// LoadRemedy is one printed way to stop some of the caller's own strays.
type LoadRemedy struct {
	// Form is "group" (pgrep -g, then kill -- -PGID) or "process" (ps -p, then
	// kill PID).
	Form string `json:"form"`
	PGID int    `json:"pgid,omitempty"`
	PIDs []int  `json:"pids,omitempty"`
	PID  int    `json:"pid,omitempty"`
	// Why says why a process form is not the group form.
	Why string `json:"why,omitempty"`
}

// LoadRunLog is what happened to the run-log event.
type LoadRunLog struct {
	Logged  bool   `json:"logged"`
	Session string `json:"session,omitempty"`
	Error   string `json:"error,omitempty"`
}

// LoadResult is the check's whole report.
type LoadResult struct {
	Status          string          `json:"status"`
	Site            string          `json:"site"`
	Load            *LoadAverages   `json:"load"`
	Cores           int             `json:"cores"`
	CoresSource     string          `json:"cores_source,omitempty"`
	Limits          *LoadLimits     `json:"limits"`
	Triggers        []string        `json:"triggers"`
	OwnStrays       []LoadOwnStray  `json:"own_strays"`
	OwnStraysMore   int             `json:"own_strays_more"`
	OtherStrays     LoadOtherStrays `json:"other_strays"`
	Remedy          []LoadRemedy    `json:"remedy"`
	NamesWithheld   bool            `json:"names_withheld,omitempty"`
	WithinPreflight bool            `json:"within_preflight"`
	RunLog          LoadRunLog      `json:"run_log"`
	// Reason says why the check was skipped or could not read everything.
	Reason string `json:"reason,omitempty"`
}

// CheckLoad runs the load check once. It never returns an error: a machine it
// cannot read is a status with a reason, and a run log it cannot write is
// reported on the result beside a warning that stands.
func CheckLoad(req LoadRequest) LoadResult {
	getenv := req.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	res := LoadResult{
		Site: req.Site, Triggers: []string{}, OwnStrays: []LoadOwnStray{}, Remedy: []LoadRemedy{},
		WithinPreflight: getenv(loadCheckedEnv) == SitePreflight,
	}
	if reason, ci := ciRunner(getenv); ci {
		res.Status = LoadSkipped
		res.Reason = "skipped on a CI runner (" + reason + "): a fresh runner carries no programs left from earlier work, so there is nothing to warn about"
		return res
	}

	read := req.Read
	if read == nil {
		read = machineload.Read
	}
	snap, readErr := read()
	if errors.Is(readErr, machineload.ErrUnsupported) {
		goos := req.GOOS
		if goos == "" {
			goos = runtime.GOOS
		}
		res.Status = LoadUnchecked
		res.Reason = "abcd reads load and processes on macOS and Linux only, and this is " + termsafe.Sanitize(goos)
		return res
	}
	if readErr != nil {
		res.Reason = termsafe.Sanitize(readErr.Error())
	}
	if !snap.HasLoad {
		res.Status = LoadUnchecked
		if res.Reason == "" {
			res.Reason = "could not read the load average"
		}
		return res
	}
	res.Load = &LoadAverages{One: snap.Load1, Five: snap.Load5, Fifteen: snap.Load15}
	if snap.Cores < 1 {
		snap.Cores, snap.CoresFallback = runtime.NumCPU(), true
	}
	res.Cores, res.CoresSource = snap.Cores, CoresOnline
	if snap.CoresFallback {
		res.CoresSource = CoresRuntime
	}

	lim, limits := readLimits(req.Home, snap.Cores)
	res.Limits = &limits

	self := req.Self
	if self == (machineload.Self{}) {
		self = machineload.Self{PID: os.Getpid(), UID: uint32(os.Geteuid())}
	}
	v := machineload.Classify(snap, lim, self)
	res.Triggers = append(res.Triggers, v.Triggers...)
	for _, s := range v.Own {
		res.OwnStrays = append(res.OwnStrays, LoadOwnStray{
			Name: termsafe.Sanitize(s.Name), PID: s.PID, PGID: s.PGID,
			AgeS: int64(s.Age / time.Second), CPUPct: int(math.Round(s.Share * 100)),
		})
	}
	res.OwnStraysMore = v.OwnMore
	res.OtherStrays = LoadOtherStrays{Count: v.Others.Count, Cores: math.Round(v.Others.Cores*10) / 10}
	for _, r := range v.Remedy {
		res.Remedy = append(res.Remedy, LoadRemedy{Form: r.Form, PGID: r.PGID, PIDs: r.PIDs, PID: r.PID, Why: r.Why})
	}
	res.NamesWithheld = scrubOwnNames(req.RepoRoot, res.OwnStrays)

	switch {
	case len(res.Triggers) > 0:
		res.Status = LoadWarning
	case readErr != nil:
		res.Status = LoadUnchecked
	default:
		res.Status = LoadOK
	}
	if res.Status == LoadWarning {
		res.RunLog = logLoadWarning(req, res)
	}
	return res
}

// ciRunner reports whether the environment is a CI runner, and names the
// variable that says so: GITHUB_ACTIONS=true, or a CI value other than empty,
// "false" or "0".
func ciRunner(getenv func(string) string) (string, bool) {
	if getenv("GITHUB_ACTIONS") == "true" {
		return "GITHUB_ACTIONS=true", true
	}
	switch v := getenv("CI"); v {
	case "", "false", "0":
		return "", false
	default:
		shown := termsafe.Sanitize(v)
		if len(shown) > 32 {
			shown = shown[:32]
		}
		return "CI=" + shown, true
	}
}

// readLimits reads ~/.abcd/load-limits through the guarded declaration read the
// other home-scope files use: a regular file, not a symlink, owned by the
// caller, writable by nobody else, at most 4 KiB. The check never creates the
// file or ~/.abcd/. An absent file is silent; any refusal or parse fault makes
// the whole file unusable, and both limits take their defaults.
func readLimits(home string, cores int) (machineload.Limits, LoadLimits) {
	def := machineload.DefaultLimits(cores)
	out := func(l machineload.Limits, source, malformed string) (machineload.Limits, LoadLimits) {
		return l, LoadLimits{StrayMinutes: l.StrayMinutes, ExtremeLoad: l.ExtremeLoad, Source: source,
			StrayFromFile: l.StrayFromFile, ExtremeFromFile: l.ExtremeFromFile, Malformed: malformed}
	}
	if home == "" {
		h, err := os.UserHomeDir()
		if err != nil || h == "" {
			return out(def, LimitsDefault, "")
		}
		home = h
	}
	path := filepath.Join(home, ".abcd", machineload.LimitsFileName)
	raw, refusal, err := fsutil.ReadDeclaration(path, maxLimitsBytes)
	switch refusal {
	case fsutil.DeclarationOK:
	case fsutil.DeclarationAbsent:
		return out(def, LimitsDefault, "")
	case fsutil.DeclarationNotRegular:
		return out(def, LimitsDefaultAfterMalformed, "it is not a regular file (a symlink, a directory or a device)")
	case fsutil.DeclarationWritableByOthers:
		return out(def, LimitsDefaultAfterMalformed, "it is writable by group or other, so its contents are not necessarily yours")
	case fsutil.DeclarationForeignOwner:
		return out(def, LimitsDefaultAfterMalformed, "it is not owned by you")
	default:
		why := "it could not be read"
		if errors.Is(err, fsutil.ErrTooBig) {
			why = fmt.Sprintf("it could not be read (larger than %d KiB)", maxLimitsBytes>>10)
		}
		return out(def, LimitsDefaultAfterMalformed, why)
	}
	l, fault := machineload.ParseLimits(raw, cores)
	if fault != nil {
		return out(def, LimitsDefaultAfterMalformed, fault.Error())
	}
	if l.StrayFromFile || l.ExtremeFromFile {
		return out(l, LimitsFile, "")
	}
	return out(l, LimitsDefault, "")
}

// scrubOwnNames masks every own stray's name the private banned-names layer
// matches, asking the engine the pre-commit guard enforces with. When the layer
// cannot be read, every name is withheld and it reports true. The scrub runs
// before anything is rendered or logged, so the printed block and the run-log
// event carry the same masked names.
func scrubOwnNames(repoRoot string, own []LoadOwnStray) bool {
	if repoRoot == "" || len(own) == 0 {
		return false
	}
	names := make([]string, len(own))
	for i, s := range own {
		names[i] = s.Name
	}
	match, err := banlist.MatchPrivate(repoRoot, names)
	if err != nil {
		for i := range own {
			own[i].Name = WithheldNameMask
		}
		return true
	}
	for i, m := range match {
		if m {
			own[i].Name = PrivateNameMask
		}
	}
	return false
}

// logLoadWarning writes the warning to the run log when a run is live: the run
// keyed on the repository's root commit exists and holds at least one joined
// session. It creates nothing otherwise. The line is attributed to the run's
// first-role session, else its earliest joined: the check runs inside a lane on
// no session's word, and the role bounds never read a session from the
// environment, so the line records the machine's state for the run and `site`
// records where it was seen.
func logLoadWarning(req LoadRequest, res LoadResult) LoadRunLog {
	sha := req.RootSHA
	if sha == "" && req.RepoRoot != "" {
		sha = gitutil.RootCommit(req.RepoRoot)
	}
	if !gitutil.IsFullSHA(sha) {
		return LoadRunLog{}
	}
	run, err := Peek(sha)
	if err != nil {
		return LoadRunLog{Error: err.Error()}
	}
	if !run.exists {
		return LoadRunLog{}
	}
	run.Now = req.Now
	sessions, err := run.Sessions()
	if err != nil {
		return LoadRunLog{Error: err.Error()}
	}
	if len(sessions) == 0 {
		return LoadRunLog{}
	}
	session := sessions[0].Session
	for _, s := range sessions {
		if s.Role == RoleFirst {
			session = s.Session
			break
		}
	}
	fields := loadEventFields(res)
	err = run.withLock(func() error {
		_, err := run.append(session, EventLoad, fields)
		return err
	})
	if err != nil {
		return LoadRunLog{Session: session, Error: err.Error()}
	}
	return LoadRunLog{Logged: true, Session: session}
}

// loadEvent is the `load` event's own fields: the verdict's facts, flat, so the
// warning renders from a logged line exactly as it was printed. A reader tells
// it from the run's hand-written load samples by `triggers`, which they do not
// carry.
type loadEvent struct {
	Site              string          `json:"site"`
	Triggers          []string        `json:"triggers"`
	Load1             float64         `json:"load1"`
	Load5             float64         `json:"load5"`
	Load15            float64         `json:"load15"`
	Cores             int             `json:"cores"`
	CoresSource       string          `json:"cores_source"`
	StrayLimitMin     int             `json:"stray_limit_min"`
	ExtremeLimit      float64         `json:"extreme_limit"`
	LimitsSource      string          `json:"limits_source"`
	StrayFromFile     bool            `json:"stray_from_file"`
	ExtremeFromFile   bool            `json:"extreme_from_file"`
	OwnStrays         []LoadOwnStray  `json:"own_strays"`
	OwnStraysMore     int             `json:"own_strays_more"`
	OtherStrays       LoadOtherStrays `json:"other_strays"`
	Remedy            []LoadRemedy    `json:"remedy"`
	NamesWithheld     bool            `json:"names_withheld"`
	WithinPreflight   bool            `json:"within_preflight"`
	ProcessTableError string          `json:"process_table_error,omitempty"`
}

// loadEventFields flattens a warning into the event's fields.
func loadEventFields(res LoadResult) map[string]any {
	e := loadEvent{
		Site: res.Site, Triggers: res.Triggers, Cores: res.Cores, CoresSource: res.CoresSource,
		OwnStrays: res.OwnStrays, OwnStraysMore: res.OwnStraysMore, OtherStrays: res.OtherStrays,
		Remedy: res.Remedy, NamesWithheld: res.NamesWithheld, WithinPreflight: res.WithinPreflight,
		ProcessTableError: res.Reason,
	}
	if res.Load != nil {
		e.Load1, e.Load5, e.Load15 = res.Load.One, res.Load.Five, res.Load.Fifteen
	}
	if res.Limits != nil {
		e.StrayLimitMin, e.ExtremeLimit, e.LimitsSource = res.Limits.StrayMinutes, res.Limits.ExtremeLoad, res.Limits.Source
		e.StrayFromFile, e.ExtremeFromFile = res.Limits.StrayFromFile, res.Limits.ExtremeFromFile
	}
	raw, _ := json.Marshal(e)
	fields := map[string]any{}
	_ = json.Unmarshal(raw, &fields)
	return fields
}

// LoadResultFromEvent reads a logged `load` event back into the warning it
// records, so a reader renders exactly what was printed. The run-log outcome is
// not part of the event.
//
// No production code reads it yet; its readers are tests in two packages, this
// one and the CLI surface's (TestLoggedEventRendersToThePrintedWarning, which
// needs the surface's unexported renderer). It stays here, exported, because Go
// has no test-only export across packages and the decoding needs the unexported
// loadEvent, so moving it into a test file would duplicate the event's shape.
func LoadResultFromEvent(ev Event) (LoadResult, error) {
	if ev.Event != EventLoad {
		return LoadResult{}, fmt.Errorf("a %s event is not a load warning", ev.Event)
	}
	if _, ok := ev.Fields["triggers"]; !ok {
		return LoadResult{}, errors.New("a load line without triggers is a hand-written sample, not a warning")
	}
	raw, err := json.Marshal(ev.Fields)
	if err != nil {
		return LoadResult{}, err
	}
	var e loadEvent
	if err := json.Unmarshal(raw, &e); err != nil {
		return LoadResult{}, fmt.Errorf("the load event does not read back: %w", err)
	}
	res := LoadResult{
		Status: LoadWarning, Site: e.Site, Triggers: e.Triggers,
		Load:  &LoadAverages{One: e.Load1, Five: e.Load5, Fifteen: e.Load15},
		Cores: e.Cores, CoresSource: e.CoresSource,
		Limits: &LoadLimits{StrayMinutes: e.StrayLimitMin, ExtremeLoad: e.ExtremeLimit, Source: e.LimitsSource,
			StrayFromFile: e.StrayFromFile, ExtremeFromFile: e.ExtremeFromFile},
		OwnStrays: e.OwnStrays, OwnStraysMore: e.OwnStraysMore, OtherStrays: e.OtherStrays,
		Remedy: e.Remedy, NamesWithheld: e.NamesWithheld, WithinPreflight: e.WithinPreflight,
		Reason: e.ProcessTableError,
	}
	if res.Triggers == nil {
		res.Triggers = []string{}
	}
	if res.OwnStrays == nil {
		res.OwnStrays = []LoadOwnStray{}
	}
	if res.Remedy == nil {
		res.Remedy = []LoadRemedy{}
	}
	return res, nil
}
