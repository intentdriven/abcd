package matrix

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/oracle"
)

// A matrix run starts only once the person has seen its plan and said yes to
// it (acceptance criterion 1, decisions 12 and 17). The plan names, before
// anything runs or spends: how many copies each cell makes, the expected cost
// by source against the cap, the load cap, the time, and the judges the run
// needs. Building it reads the declaration and this machine's configuration
// and nothing else: no network call, no price fetched (decision 12), nothing
// written.

// PlannedCell is one cell as the plan resolved it: its role, the source its
// route reaches (decision 2), and the runner and model it asks for.
type PlannedCell struct {
	Cell   string `json:"cell"`
	Role   Role   `json:"role"`
	Route  string `json:"route"`
	Source Source `json:"source"`
	Runner string `json:"runner"`
	Model  string `json:"model"`
	// Provider is the oracle provider block the model is asked of, "" for a
	// host cell.
	Provider string `json:"provider,omitempty"`
	// ProviderPin is an OpenRouter cell's pin and LocalPin a local cell's,
	// read from this machine's configuration, so the plan the person sees
	// names exactly what each copy is held to (decision 6).
	ProviderPin *ProviderPin `json:"provider_pin,omitempty"`
	LocalPin    *LocalPin    `json:"local_pin,omitempty"`
}

// Copies is how many copies the run makes: every cell does every task, the
// pilot's repeats first, and the run is sized up to at most Max repeats
// (decision 8). The person consents to the most.
type Copies struct {
	PilotTotal int            `json:"pilot_total"`
	MaxTotal   int            `json:"max_total"`
	MaxByCell  map[string]int `json:"max_by_cell"`
}

// SourceCost is the expected cost of the cells from one source at the most
// copies the run may make. Expected is nil when the source is not metered, or
// when a metered cell declared no cost per copy; Note then says so.
type SourceCost struct {
	Source   Source `json:"source"`
	Metered  bool   `json:"metered"`
	Expected *Money `json:"expected,omitempty"`
	Note     string `json:"note,omitempty"`
}

// Estimate is the run's time: a bound when every cell declares its minutes per
// copy, otherwise known after the pilot (decision 35).
type Estimate struct {
	BoundMinutes *float64 `json:"bound_minutes,omitempty"`
	Note         string   `json:"note"`
}

// JudgeNeed is who judges one task and how many of them it needs (decisions
// 21, 22, 26): a code task is judged by its hidden tests and the reviewer
// models; a writing task by people, each ranking at most eight copies so that
// every pair of models meets; a check by the person and one outside judge.
type JudgeNeed struct {
	Task  string   `json:"task"`
	Kind  TaskKind `json:"kind"`
	Who   string   `json:"who"`
	Count int      `json:"count"`
	Note  string   `json:"note,omitempty"`
}

// Plan is what the person consents to.
type Plan struct {
	Lab            string        `json:"lab"`
	FrozenCommit   string        `json:"frozen_commit"`
	Declaration    ContentRef    `json:"declaration"`
	Cells          []PlannedCell `json:"cells"`
	Copies         Copies        `json:"copies"`
	Cost           []SourceCost  `json:"cost"`
	SpendCap       *Money        `json:"spend_cap"`
	SpendCapOrigin string        `json:"spend_cap_origin"`
	// LoadCap is the cores the local cells may load, nil with no local cell;
	// Cores is this machine's count it is held below (decision 12).
	LoadCap     *int        `json:"load_cap"`
	Cores       int         `json:"cores"`
	Concurrency int         `json:"concurrency"`
	Time        Estimate    `json:"time"`
	Judges      []JudgeNeed `json:"judges"`
	// Digest is the sha256 of the plan with Digest left blank: consent names
	// it, so a plan that changed after it was shown is never started.
	Digest string `json:"digest"`
}

// PlanInputs is what a plan is built from. CheckRoute is the runner
// configuration's own check of one cell's route (an enabled runner, a model its
// provider lists, no claude for a pinned model), the front door's to supply,
// so building the plan changes no configuration.
type PlanInputs struct {
	Lab          string
	FrozenCommit string
	Declaration  []byte
	API          *oracle.APIConfig
	CheckRoute   func(oracle.FlagRoute) error
	Cap          ResolvedCap
	Cores        int
}

// maxJudgeSubset is the most copies one judge ranks (decision 21).
const maxJudgeSubset = 8

// judgeFloor is the fewest judges whose rankings are pooled; below it each
// judge's ranking is reported on its own (decision 26).
const judgeFloor = 3

// buildPlan builds and checks a matrix run's plan. Everything it refuses is
// refused before anything is written: a declaration that does not parse, a
// cell route the grammar or the runner configuration refuses, a cell on a
// runner that does not name its model, a provider the allowlist denies, a
// metered cell with no cap, a local cell with no load cap or one at or above
// the cores.
func buildPlan(in PlanInputs) (Plan, error) {
	d, err := parseDeclaration(in.Declaration)
	if err != nil {
		return Plan{}, err
	}
	if in.API == nil || in.CheckRoute == nil {
		return Plan{}, fmt.Errorf("the plan needs this machine's provider configuration and route check")
	}
	if in.Cores < 1 {
		return Plan{}, fmt.Errorf("the core count %d is not a machine's", in.Cores)
	}
	p := Plan{
		Lab: in.Lab, FrozenCommit: in.FrozenCommit,
		Declaration: contentOf("declaration.json", in.Declaration),
		Concurrency: d.Concurrency, Cores: in.Cores,
		SpendCap: in.Cap.Cap, SpendCapOrigin: in.Cap.Origin,
	}
	metered, local := false, false
	for _, c := range d.cells() {
		pc, err := resolveCell(c, in.API, in.CheckRoute)
		if err != nil {
			return Plan{}, fmt.Errorf("cell %s: %w", c.ID, err)
		}
		p.Cells = append(p.Cells, pc)
		metered = metered || pc.Source.Metered()
		local = local || pc.Source == SourceLocal
	}
	if metered && p.SpendCap == nil {
		return Plan{}, fmt.Errorf("a cell spends money, and no spending cap is named: give the run one in spend_cap, or set %s in %s; abcd never supplies one", CapKey, layered.Config.MachineOrigin())
	}
	if local {
		if d.LoadCap == nil {
			return Plan{}, fmt.Errorf("a cell runs a local model, and the declaration names no load_cap; name the cores it may load, below this machine's %d", in.Cores)
		}
		if *d.LoadCap >= in.Cores {
			return Plan{}, fmt.Errorf("load_cap %d is not below this machine's %d cores; leave cores for the system", *d.LoadCap, in.Cores)
		}
		lc := *d.LoadCap
		p.LoadCap = &lc
	}
	p.Copies = copiesOf(d)
	p.Cost = costOf(d, p.Cells, p.Copies)
	p.Time = planTime(d, p.Copies)
	p.Judges = judgesOf(d)
	p.Digest = p.digest()
	return p, nil
}

// resolveCell resolves one cell's route to its source (decision 2). host is the
// host leg; a runner route parses by the runner rung's grammar for the cell's
// own role, and names its model unless the runner is claude on the host's
// own model: a runner's machine model can change between the plan and the
// run, and the plan names each cell's source before consent.
func resolveCell(c roleCell, api *oracle.APIConfig, check func(oracle.FlagRoute) error) (PlannedCell, error) {
	pc := PlannedCell{Cell: c.ID, Role: c.role, Route: c.Route}
	if c.Route == "host" {
		pc.Source, pc.Model = SourceHost, "host"
		return pc, nil
	}
	routes, err := oracle.ParseRunnerRoutes([]string{c.Route}, []string{string(c.role)})
	if err != nil {
		return PlannedCell{}, err
	}
	fr := routes[0]
	if err := check(fr); err != nil {
		return PlannedCell{}, err
	}
	pc.Runner, pc.Model = fr.Runner, fr.Model
	if fr.Model == "" {
		if fr.Runner == "claude" {
			pc.Source, pc.Model = SourceHost, "host"
			return pc, nil
		}
		return PlannedCell{}, fmt.Errorf("the route names runner %s and no model; a matrix cell names its model (@<provider>/<model>), since the runner's own can change before the run", fr.Runner)
	}
	provider, model, ok := strings.Cut(fr.Model, "/")
	if !ok || provider == "" || model == "" {
		return PlannedCell{}, fmt.Errorf("the model %q is not <provider>/<model>", clip(fr.Model))
	}
	if err := api.Admit(provider, model); err != nil {
		return PlannedCell{}, err
	}
	pc.Provider = provider
	switch {
	case api.IsLocal(provider):
		pc.Source = SourceLocal
		lp, ok := api.LocalPinFor(provider, model)
		if !ok {
			return PlannedCell{}, fmt.Errorf("the local model %s on %s has no pin in %s; a local cell is pinned down to its weights", clip(model), clip(provider), layered.Config.MachineOrigin())
		}
		pin := LocalPin(lp)
		pc.LocalPin = &pin
	case api.IsOpenRouter(provider):
		pc.Source = SourceOpenRouter
		op, ok := api.PinFor(provider, model)
		if !ok {
			return PlannedCell{}, fmt.Errorf("the OpenRouter model %s has no provider pin in %s; without one OpenRouter falls back across providers, which changes the cell", clip(model), layered.Config.MachineOrigin())
		}
		if op.AllowFallbacks {
			return PlannedCell{}, fmt.Errorf("the OpenRouter pin for %s allows fallbacks; a matrix cell keeps them off", clip(model))
		}
		pin := ProviderPin(op)
		pc.ProviderPin = &pin
	default:
		pc.Source = SourceProvider
	}
	return pc, nil
}

// copiesOf counts the copies: every cell does every task.
func copiesOf(d Declaration) Copies {
	per := len(d.Tasks)
	c := Copies{MaxByCell: map[string]int{}}
	for _, rc := range d.cells() {
		c.PilotTotal += per * d.Repeats.Pilot
		c.MaxTotal += per * d.Repeats.Max
		c.MaxByCell[rc.ID] = per * d.Repeats.Max
	}
	return c
}

// costOf is the expected cost by source at the most copies (decision 36): a
// metered cell's declared cost per copy times its copies, summed per source;
// a metered source with any cell that declared none is not estimated.
func costOf(d Declaration, cells []PlannedCell, copies Copies) []SourceCost {
	byID := map[string]Cell{}
	for _, rc := range d.cells() {
		byID[rc.ID] = rc.Cell
	}
	sums := map[Source]*SourceCost{}
	var order []Source
	for _, pc := range cells {
		sc, ok := sums[pc.Source]
		if !ok {
			sc = &SourceCost{Source: pc.Source, Metered: pc.Source.Metered()}
			sums[pc.Source] = sc
			order = append(order, pc.Source)
			if sc.Metered {
				sc.Expected = &Money{Unit: "usd"}
			} else {
				sc.Note = "spends nothing a cap bounds"
			}
		}
		if !sc.Metered || sc.Expected == nil {
			continue
		}
		cpc := byID[pc.Cell].CostPerCopy
		if cpc == nil {
			sc.Expected, sc.Note = nil, "not estimated; bounded by your cap"
			continue
		}
		sc.Expected.Amount += cpc.Amount * float64(copies.MaxByCell[pc.Cell])
	}
	out := make([]SourceCost, 0, len(order))
	for _, s := range order {
		sc := *sums[s]
		if sc.Expected != nil {
			sc.Expected.Amount = math.Round(sc.Expected.Amount*100) / 100
		}
		out = append(out, sc)
	}
	return out
}

// planTime is the run's time (decision 35): a bound at the most copies when every
// cell declares its minutes per copy, spread over the run's concurrency;
// otherwise known after the pilot.
func planTime(d Declaration, copies Copies) Estimate {
	total := 0.0
	for _, rc := range d.cells() {
		if rc.MinutesPerCopy == nil {
			return Estimate{Note: "known after the pilot"}
		}
		total += *rc.MinutesPerCopy * float64(copies.MaxByCell[rc.ID])
	}
	b := math.Ceil(total / float64(max(1, d.Concurrency)))
	return Estimate{BoundMinutes: &b, Note: "at most, at the most copies; the pilot sizes the repeats"}
}

// judgesOf is who judges each task and how many.
func judgesOf(d Declaration) []JudgeNeed {
	var out []JudgeNeed
	for _, t := range d.Tasks {
		switch t.Kind {
		case KindCode:
			out = append(out, JudgeNeed{Task: t.ID, Kind: t.Kind, Who: "hidden tests and the reviewer models", Count: len(d.Reviewers)})
		case KindWriting:
			// Each judge ranks at most eight copies, and every pair of
			// builder models must meet in some judge's subset: enough judges
			// that the models' copies cover each pair, never fewer than the
			// floor below which rankings are not pooled.
			models := len(d.Builders)
			n := judgeFloor
			if models > maxJudgeSubset {
				half := maxJudgeSubset / 2
				groups := (models + half - 1) / half
				n = max(n, groups*(groups-1)/2)
			}
			out = append(out, JudgeNeed{Task: t.ID, Kind: t.Kind, Who: "people who have not read the copies, blind to their authors", Count: n,
				Note: fmt.Sprintf("each ranks at most %d copies; with fewer than %d judges each ranking is reported on its own", maxJudgeSubset, judgeFloor)})
		case KindCheck:
			out = append(out, JudgeNeed{Task: t.ID, Kind: t.Kind, Who: "you and one outside judge", Count: 2})
		}
	}
	return out
}

// digest is the sha256 of the plan's canonical JSON with Digest blank.
func (p Plan) digest() string {
	p.Digest = ""
	cells := append([]PlannedCell(nil), p.Cells...)
	sort.SliceStable(cells, func(i, j int) bool { return cells[i].Cell < cells[j].Cell })
	p.Cells = cells
	data, _ := json.Marshal(p)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
