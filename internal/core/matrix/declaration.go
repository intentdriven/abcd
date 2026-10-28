package matrix

// declaration.go is the person's matrix (decision 16: everything about a run is
// configurable): the tasks and how each is judged, the builder and reviewer
// cells, the repeats, the load cap, the concurrency and, for a run with a
// metered cell, the spending cap. It is read at a trust boundary, so it is
// decoded strictly through jsonstrict.Decode: a repeated key, a key no field
// names, at any depth, and anything after the document are refused, never
// read last-wins or ignored.

import (
	"errors"
	"fmt"
	"math"
	"path"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/intentdriven/abcd/internal/core/jsonstrict"
	"github.com/intentdriven/abcd/internal/core/oracle"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// DeclarationSchema is the declaration's schema version.
const DeclarationSchema = 1

// MaxDeclarationBytes bounds a declaration, the bound the rules overlay holds
// its file to; a larger one is refused before it is parsed.
const MaxDeclarationBytes = 256 << 10

// The pilot's bounds (decision 8): three to five runs a cell estimate the
// spread the repeat count is then sized from.
const (
	MinPilot = 3
	MaxPilot = 5
)

// MaxRepeats bounds the repeats a cell may be sized up to. The SOTA note's
// separations needed about 20–113 runs an arm; the bound sits well past that
// and refuses a typo that would plan a run no machine finishes.
const MaxRepeats = 1000

// maxPaths bounds the paths one task names, and maxNameBytes its prose fields.
//
// MaxConcurrency bounds the copies a run makes at once; the run's plan holds it
// to the machine (spec step 2).
const MaxConcurrency = 64

// maxTasks and maxCells bound what one declaration names, so the run record
// built from it fits the bound its lab reads a record back within.
const (
	maxCells     = 64
	maxTasks     = 64
	maxPaths     = 64
	maxNameBytes = 200
	maxCriteria  = 4 << 10
)

// TaskKind is how a task is done and judged (decision 18).
type TaskKind string

const (
	// KindCode is a code feature: built through the runners, judged by
	// hidden tests and reviewer models.
	KindCode TaskKind = "code"
	// KindWriting is a written doc: judged by people, blind.
	KindWriting TaskKind = "writing"
	// KindCheck is a check such as doc fidelity: its copies' findings
	// compared side by side.
	KindCheck TaskKind = "check"
)

// Role is which side of the matrix a cell sits on.
type Role string

const (
	RoleBuilder  Role = "builder"
	RoleReviewer Role = "reviewer"
)

// Declaration is the person's matrix.
type Declaration struct {
	SchemaVersion int    `json:"schema_version"`
	Name          string `json:"name"`
	Tasks         []Task `json:"tasks"`
	Builders      []Cell `json:"builders"`
	Reviewers     []Cell `json:"reviewers,omitempty"`
	// Repeats is the pilot and the most each cell may be sized up to; the
	// person consents to the most (spec step 2).
	Repeats Repeats `json:"repeats"`
	// SpendCap is the run's own spending cap, nil when the declaration names
	// none; a run with a metered cell needs one from here or the person's own
	// configuration (decision 17), never a default abcd supplies.
	SpendCap *Money `json:"spend_cap,omitempty"`
	// LoadCap is the cores the run's local cells may load (decision 12). A
	// declaration may leave it out; the run record asks for it when a cell
	// runs locally.
	LoadCap     *int    `json:"load_cap,omitempty"`
	Concurrency int     `json:"concurrency"`
	Judging     Judging `json:"judging"`
}

// Task is one task the matrix does. Which fields apply follows its kind: a code
// task names its intent; a writing task its source material and the judges'
// criteria; a check the material it checks (scope condition 2).
type Task struct {
	ID       string   `json:"id"`
	Kind     TaskKind `json:"kind"`
	Intent   string   `json:"intent,omitempty"`
	Source   []string `json:"source,omitempty"`
	Criteria string   `json:"criteria,omitempty"`
	Material []string `json:"material,omitempty"`
}

// Cell is one model route the matrix compares. Route is the per-invocation
// route override's text for the cell's role (decision 14):
// `<role>=runner:<name>[@<provider>/<model>]`, or `host`. Its grammar is
// parsed where the run starts (spec step 2), against the runner rung
// iss-2610161801358637 adds; here it is bounded text. Settings are the
// vendor's recommended sampling settings, pinned for the cell (decision 5).
//
// CostPerCopy is the person's own expected cost of one copy on a metered cell
// (decision 36): the plan reckons the expected cost from it, and never fetches
// a price before consent (decision 12). MinutesPerCopy is the person's own
// estimate of one copy's time (decision 35): with one on every cell the plan
// shows a bound before the pilot; otherwise the time is known after it.
type Cell struct {
	ID             string          `json:"id"`
	Route          string          `json:"route"`
	Settings       oracle.Settings `json:"settings,omitempty"`
	CostPerCopy    *Money          `json:"cost_per_copy,omitempty"`
	MinutesPerCopy *float64        `json:"minutes_per_copy,omitempty"`
}

// Repeats is the pilot each cell runs first and the most it may be sized up to.
type Repeats struct {
	Pilot int `json:"pilot"`
	Max   int `json:"max"`
}

// Money is an amount in a unit. The only unit is usd: a cap is the person's
// own figure, and a cost reported in another unit (OpenRouter's credits) is
// set against it only once their exchange is established (spec step 2).
type Money struct {
	Amount float64 `json:"amount"`
	Unit   string  `json:"unit"`
}

// Judging is the run's judging-session settings (decision 24): whether guests
// admitted by code may judge, their verdicts reported apart.
type Judging struct {
	GuestsMayJudge bool `json:"guests_may_judge"`
}

// idRe is a task's or a cell's id: it becomes a path segment of a trial's
// record (trials/<task>/<cell>/<repeat>.json) and a Harbor trial name's part.
var idRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const maxIDBytes = 64

// intentRe is an intent id.
var intentRe = regexp.MustCompile(`^itd-[0-9]{1,20}$`)

// validID reports whether s is a task or cell id.
func validID(s string) bool { return len(s) <= maxIDBytes && idRe.MatchString(s) }

// parseDeclaration reads a declaration: bounded, strictly decoded, validated.
func parseDeclaration(data []byte) (Declaration, error) {
	if len(data) > MaxDeclarationBytes {
		return Declaration{}, fmt.Errorf("the declaration is %d bytes, past its bound of %d bytes", len(data), MaxDeclarationBytes)
	}
	var d Declaration
	if err := jsonstrict.Decode(data, &d); err != nil {
		return Declaration{}, fmt.Errorf("the declaration does not parse: %w", err)
	}
	if err := d.validate(); err != nil {
		return Declaration{}, fmt.Errorf("the declaration is refused: %w", err)
	}
	return d, nil
}

func (d Declaration) validate() error {
	if d.SchemaVersion != DeclarationSchema {
		return fmt.Errorf("schema_version is %d; this abcd reads %d", d.SchemaVersion, DeclarationSchema)
	}
	if err := prose("name", d.Name, maxNameBytes); err != nil {
		return err
	}
	if len(d.Tasks) == 0 {
		return errors.New("tasks names no task")
	}
	if len(d.Tasks) > maxTasks {
		return fmt.Errorf("tasks names %d tasks; a run does at most %d", len(d.Tasks), maxTasks)
	}
	seen := map[string]bool{}
	for i, t := range d.Tasks {
		if !validID(t.ID) {
			return fmt.Errorf("tasks[%d]: id %q is not lower-case words joined by '-', at most %d bytes", i, clip(t.ID), maxIDBytes)
		}
		if seen[t.ID] {
			return fmt.Errorf("tasks[%d]: the task id %s is named twice", i, t.ID)
		}
		seen[t.ID] = true
		if err := t.validate(); err != nil {
			return fmt.Errorf("task %s: %w", t.ID, err)
		}
	}
	if len(d.Builders) == 0 {
		return errors.New("builders names no builder cell; a matrix compares at least one")
	}
	if n := len(d.Builders) + len(d.Reviewers); n > maxCells {
		return fmt.Errorf("builders and reviewers name %d cells; a run compares at most %d", n, maxCells)
	}
	cells := map[string]bool{}
	for _, c := range d.cells() {
		if !validID(c.ID) {
			return fmt.Errorf("%s cell: id %q is not lower-case words joined by '-', at most %d bytes", c.role, clip(c.ID), maxIDBytes)
		}
		if cells[c.ID] {
			return fmt.Errorf("the cell id %s is named twice; a cell is one route, in one role", c.ID)
		}
		cells[c.ID] = true
		if err := c.validate(); err != nil {
			return fmt.Errorf("cell %s: %w", c.ID, err)
		}
	}
	r := d.Repeats
	if r.Pilot < MinPilot || r.Pilot > MaxPilot {
		return fmt.Errorf("repeats.pilot is %d; a pilot runs %d to %d copies a cell", r.Pilot, MinPilot, MaxPilot)
	}
	if r.Max < r.Pilot || r.Max > MaxRepeats {
		return fmt.Errorf("repeats.max is %d; it is at least the pilot (%d) and at most %d", r.Max, r.Pilot, MaxRepeats)
	}
	if d.SpendCap != nil {
		if err := d.SpendCap.validate(); err != nil {
			return fmt.Errorf("spend_cap: %w", err)
		}
	}
	if d.LoadCap != nil && *d.LoadCap < 1 {
		return fmt.Errorf("load_cap is %d; a load cap is at least one core, below the machine's count", *d.LoadCap)
	}
	if d.Concurrency < 1 || d.Concurrency > MaxConcurrency {
		return fmt.Errorf("concurrency is %d; from 1 to %d copies run at a time", d.Concurrency, MaxConcurrency)
	}
	return nil
}

func (t Task) validate() error {
	switch t.Kind {
	case KindCode:
		if !intentRe.MatchString(t.Intent) {
			return fmt.Errorf("a code task names its intent (itd-<N>); intent is %q", clip(t.Intent))
		}
		return only(t, "intent")
	case KindWriting:
		if len(t.Source) == 0 {
			return errors.New("a writing task names its source material in source")
		}
		if err := paths("source", t.Source); err != nil {
			return err
		}
		if err := prose("criteria", t.Criteria, maxCriteria); err != nil {
			return fmt.Errorf("a writing task names the judges' criteria: %w", err)
		}
		return only(t, "source", "criteria")
	case KindCheck:
		if len(t.Material) == 0 {
			return errors.New("a check names the material it checks in material")
		}
		if err := paths("material", t.Material); err != nil {
			return err
		}
		return only(t, "material")
	}
	return fmt.Errorf("kind %q is not code, writing or check", clip(string(t.Kind)))
}

// only refuses a field the task's kind does not read, so a declaration never
// carries a value that silently does nothing.
func only(t Task, keep ...string) error {
	set := map[string]bool{"intent": t.Intent != "", "source": len(t.Source) > 0,
		"criteria": t.Criteria != "", "material": len(t.Material) > 0}
	for _, k := range keep {
		delete(set, k)
	}
	for _, k := range []string{"intent", "source", "criteria", "material"} {
		if set[k] {
			return fmt.Errorf("%s does not apply to a %s task", k, t.Kind)
		}
	}
	return nil
}

func (c roleCell) validate() error {
	if c.Route == "" {
		return errors.New("route names no route; a cell is `host` or a runner route")
	}
	if len(c.Route) > oracle.MaxRouteBytes || strings.IndexFunc(c.Route, notPrintable) >= 0 || termsafe.Sanitize(c.Route) != c.Route {
		return fmt.Errorf("route is longer than %d bytes or carries a space or a control, bidirectional or zero-width character", oracle.MaxRouteBytes)
	}
	// The oracle's own check, the one a routing row and a --route pass: at
	// most oracle.MaxSettings, each a parameter-shaped name and a bounded,
	// plain JSON scalar.
	if _, err := oracle.CheckSettings(c.Settings); err != nil {
		return fmt.Errorf("settings: %w", err)
	}
	if c.CostPerCopy != nil {
		if err := c.CostPerCopy.validate(); err != nil {
			return fmt.Errorf("cost_per_copy: %w", err)
		}
	}
	if m := c.MinutesPerCopy; m != nil && (!(*m > 0) || *m > maxMinutesPerCopy || math.IsInf(*m, 0)) {
		return fmt.Errorf("minutes_per_copy is %v; it is a positive figure of at most %d", *m, maxMinutesPerCopy)
	}
	return nil
}

// maxMinutesPerCopy bounds one copy's declared time: a week.
const maxMinutesPerCopy = 7 * 24 * 60

// maxAmount bounds one figure of money a run names, so a plan's sums stay
// finite and the plan always has a digest.
const maxAmount = 1_000_000

func (m Money) validate() error {
	if m.Unit != "usd" {
		return fmt.Errorf("unit is %q; money here is in usd", clip(m.Unit))
	}
	if !(m.Amount > 0) || m.Amount > maxAmount {
		return fmt.Errorf("amount is %v; it is a positive figure of at most %d", m.Amount, maxAmount)
	}
	return nil
}

// roleCell is a cell with the role it was declared in.
type roleCell struct {
	Cell
	role Role
}

// cells is every cell, builders first, in declaration order.
func (d Declaration) cells() []roleCell {
	out := make([]roleCell, 0, len(d.Builders)+len(d.Reviewers))
	for _, c := range d.Builders {
		out = append(out, roleCell{c, RoleBuilder})
	}
	for _, c := range d.Reviewers {
		out = append(out, roleCell{c, RoleReviewer})
	}
	return out
}

// paths holds each path a task names to a repository-relative, slash-separated
// path that stays inside the repository.
func paths(field string, ps []string) error {
	if len(ps) > maxPaths {
		return fmt.Errorf("%s names %d paths; at most %d", field, len(ps), maxPaths)
	}
	for _, p := range ps {
		if p == "" || strings.HasPrefix(p, "/") || strings.Contains(p, `\`) || path.Clean(p) != p ||
			p == ".." || strings.HasPrefix(p, "../") || strings.IndexFunc(p, unicode.IsControl) >= 0 {
			return fmt.Errorf("%s: %q is not a repository-relative path inside the repository", field, clip(p))
		}
	}
	return nil
}

// prose holds a prose field to one non-empty, bounded value of valid UTF-8,
// its lines plain text: no control character but the line break, and no
// bidirectional or zero-width one.
func prose(field, s string, limit int) error {
	if strings.TrimSpace(s) == "" || len(s) > limit || !utf8.ValidString(s) ||
		strings.ContainsRune(s, '\r') || termsafe.SanitizeBlock(s) != s {
		return fmt.Errorf("%s is empty, longer than %d bytes or carries a control, bidirectional or zero-width character", field, limit)
	}
	return nil
}

func notPrintable(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }

// clip bounds an untrusted value quoted back in a refusal.
func clip(s string) string {
	const n = 80
	if len(s) <= n {
		return s
	}
	cut := n
	for cut > 0 && s[cut]&0xC0 == 0x80 {
		cut--
	}
	return s[:cut] + "…"
}
