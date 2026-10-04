package loop

// pace.go is the run's pace (itd-2609201925079472, spc-2609202134341288 piece
// 1): the working window, the pause after it and the ceiling on lanes alive
// at once, and the fix rounds a lane may take before it is handed back
// (ruling DR1, 2026-09-29), resolved once when a run starts through the one
// layered configuration reader — the --pace, --sub-agents and --fix-rounds
// flags, then the
// repository's .abcd/config.json, then the machine's ~/.abcd.noindex/config.json,
// then the bundled default — and written into the run's state with the layer
// each value came from, so the run record names it and a later invocation
// honours the pace the run started on, whatever the files say by then.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/layered"
)

// The bundled pace: 120 minutes of work, 300 of pause, two lanes (decision 5,
// the product thinker's numbers for this repository's runs on 2026-09-20). A
// repository that measured otherwise writes its own under `pace` in its
// .abcd/config.json.
const (
	BundledWorkMinutes  = 120
	BundledPauseMinutes = 300
	BundledSubAgents    = 2
)

// BundledFixRounds is the fix rounds a lane may take before it is handed back
// when nothing sets it (ruling DR1 of 2026-09-29: "tied to the pace settings, a
// per-run value set alongside --pace, default 3"; itd-50 decision 2).
const BundledFixRounds = 3

// The accepted ranges. A week bounds the minutes, so the window arithmetic
// stays far inside time.Duration and a typed extra digit is refused rather than
// run; 64 bounds the ceiling for the same reason. A pause of 0 minutes is a run
// that does not pause.
const (
	maxPaceMinutes = 7 * 24 * 60
	maxSubAgents   = 64
	maxFixRounds   = 64
)

// The configuration keys the pace claims, under the `pace` namespace of
// .abcd/config.json and ~/.abcd.noindex/config.json.
const (
	paceNamespace   = "pace"
	keyWorkMinutes  = "work_minutes"
	keyPauseMinutes = "pause_minutes"
	keySubAgents    = "sub_agents"
	keyFixRounds    = "fix_rounds"
)

// StagePace is the refusal stage of a pace or ceiling the loop cannot run on.
const StagePace = "pace"

// paceForm is the accepted form every pace refusal names.
var paceForm = "--pace takes <work-minutes>/<pause-minutes> in whole minutes (work 1 to 10080, pause 0 to 10080, e.g. 120/300) " +
	"--sub-agents a whole number of lanes from 1 to 64, and --fix-rounds the fix rounds a lane may take before it is handed back, " +
	"a whole number from 0 to 64; in .abcd/config.json or " + abcdhome.Display("config.json") + " the same numbers are " +
	"pace.work_minutes, pace.pause_minutes, pace.sub_agents and pace.fix_rounds"

// PaceValue is one of the pace's numbers with the layer that supplied it:
// "flag", "repo", "machine" or "bundled", and the origin — the flag as typed,
// the file, or "bundled".
type PaceValue struct {
	Value  int    `json:"value"`
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
}

// Pace is a run's pace: the working window, the pause after it, the ceiling
// on this run's lanes and validators alive at once, and the fix rounds a lane
// may take before it is handed back. FixRounds is zero in a run a state file
// before version 6 holds: it started before the pace carried the cap, and
// FixRoundCap reads the bundled value for it.
type Pace struct {
	WorkMinutes  PaceValue `json:"work_minutes"`
	PauseMinutes PaceValue `json:"pause_minutes"`
	SubAgents    PaceValue `json:"sub_agents"`
	FixRounds    PaceValue `json:"fix_rounds,omitzero"`
}

// FixRoundCap is the fix rounds a lane of the run may take before it is
// handed back: the run's own, or the bundled value for a run that started
// before the pace carried one (an unpaced run, or a state file before version
// 6).
func (s State) FixRoundCap() int {
	if s.Pace == nil || s.Pace.FixRounds.Layer == "" {
		return BundledFixRounds
	}
	return s.Pace.FixRounds.Value
}

// String renders the pace and where each number came from, as the run record
// and the text surfaces name it.
func (p Pace) String() string {
	head := fmt.Sprintf("%d/%d minutes, %d sub-agents", p.WorkMinutes.Value, p.PauseMinutes.Value, p.SubAgents.Value)
	w, pa, s := p.WorkMinutes.Origin, p.PauseMinutes.Origin, p.SubAgents.Origin
	if w == pa && pa == s {
		head += " (" + describeOrigin(p.WorkMinutes) + ")"
	} else {
		head = fmt.Sprintf("%s (work from %s, pause from %s, sub-agents from %s)", head,
			describeOrigin(p.WorkMinutes), describeOrigin(p.PauseMinutes), describeOrigin(p.SubAgents))
	}
	if p.FixRounds.Layer == "" {
		return head
	}
	return fmt.Sprintf("%s; %s before a lane is handed back (%s)", head, fixRoundsPhrase(p.FixRounds.Value), describeOrigin(p.FixRounds))
}

// fixRoundsPhrase is a number of fix rounds in words a reader counts.
func fixRoundsPhrase(n int) string {
	if n == 1 {
		return "1 fix round"
	}
	return fmt.Sprintf("%d fix rounds", n)
}

func describeOrigin(v PaceValue) string {
	if v.Layer == v.Origin {
		return v.Layer
	}
	return v.Origin + ", the " + v.Layer + " layer"
}

// same reports whether two paces run on the same numbers, wherever they came
// from.
func (p Pace) same(q Pace) bool {
	return p.WorkMinutes.Value == q.WorkMinutes.Value && p.PauseMinutes.Value == q.PauseMinutes.Value &&
		p.SubAgents.Value == q.SubAgents.Value && p.FixRounds.Value == q.FixRounds.Value
}

// paceFlags are the --pace, --sub-agents and --fix-rounds flags as typed,
// parsed.
type paceFlags struct {
	set                               bool
	work, pause, subs, fix            *int
	paceOrigin, subsOrigin, fixOrigin string
}

// parsePaceFlags checks the flags' form: --pace is two runs of digits around
// one slash, --sub-agents and --fix-rounds one run of digits each. The ranges
// are the resolver's, checked there for every layer alike.
func parsePaceFlags(pace, subs, fix *string) (paceFlags, error) {
	var f paceFlags
	if pace != nil {
		f.set = true
		f.paceOrigin = "--pace " + layered.BoundKey(*pace)
		w, p, ok := strings.Cut(*pace, "/")
		wn, werr := wholeNumber(w)
		pn, perr := wholeNumber(p)
		if !ok || werr != nil || perr != nil {
			return f, paceRefusal(fmt.Sprintf("--pace %q is not a pace", layered.BoundKey(*pace)))
		}
		f.work, f.pause = &wn, &pn
	}
	if subs != nil {
		f.set = true
		f.subsOrigin = "--sub-agents " + layered.BoundKey(*subs)
		n, err := wholeNumber(*subs)
		if err != nil {
			return f, paceRefusal(fmt.Sprintf("--sub-agents %q is not a number of lanes", layered.BoundKey(*subs)))
		}
		f.subs = &n
	}
	if fix != nil {
		f.set = true
		f.fixOrigin = "--fix-rounds " + layered.BoundKey(*fix)
		n, err := wholeNumber(*fix)
		if err != nil {
			return f, paceRefusal(fmt.Sprintf("--fix-rounds %q is not a number of fix rounds", layered.BoundKey(*fix)))
		}
		f.fix = &n
	}
	return f, nil
}

// wholeNumber reads a run of ASCII digits, and nothing else, as an int.
func wholeNumber(s string) (int, error) {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return 0, fmt.Errorf("%q is not a whole number", s)
	}
	return strconv.Atoi(s)
}

func paceRefusal(reason string) error {
	return refuse(StagePace, "", "", reason, paceForm)
}

// resolvePace reads the pace through the layered configuration: the flags,
// then the repository's file, then the machine's, then the bundled default,
// each key on its own. It claims the `pace` namespace, so a misspelt key is
// refused rather than letting a default apply unannounced, and a value outside
// its range is refused naming its origin, never replaced by a lower layer.
func resolvePace(roots layered.Roots, f paceFlags) (Pace, error) {
	s, err := layered.Load(layered.Config, roots)
	if err != nil {
		return Pace{}, paceRefusal(err.Error())
	}
	if f.work != nil {
		if err := s.SetFlag(paceNamespace+"."+keyWorkMinutes, *f.work, f.paceOrigin); err != nil {
			return Pace{}, paceRefusal(err.Error())
		}
		if err := s.SetFlag(paceNamespace+"."+keyPauseMinutes, *f.pause, f.paceOrigin); err != nil {
			return Pace{}, paceRefusal(err.Error())
		}
	}
	if f.subs != nil {
		if err := s.SetFlag(paceNamespace+"."+keySubAgents, *f.subs, f.subsOrigin); err != nil {
			return Pace{}, paceRefusal(err.Error())
		}
	}
	if f.fix != nil {
		if err := s.SetFlag(paceNamespace+"."+keyFixRounds, *f.fix, f.fixOrigin); err != nil {
			return Pace{}, paceRefusal(err.Error())
		}
	}
	if err := s.Claim(paceNamespace, keyWorkMinutes, keyPauseMinutes, keySubAgents, keyFixRounds); err != nil {
		return Pace{}, paceRefusal(err.Error())
	}
	get := func(key string, bundled, lo, hi int, unit string) (PaceValue, error) {
		v, err := layered.Get(s, paceNamespace+"."+key, bundled, func(n int) error {
			if n < lo || n > hi {
				return fmt.Errorf("want %s from %d to %d", unit, lo, hi)
			}
			return nil
		})
		if err != nil {
			return PaceValue{}, paceRefusal(err.Error())
		}
		return PaceValue{Value: v.V, Layer: v.Layer.String(), Origin: v.Origin}, nil
	}
	var p Pace
	if p.WorkMinutes, err = get(keyWorkMinutes, BundledWorkMinutes, 1, maxPaceMinutes, "whole minutes of work"); err != nil {
		return Pace{}, err
	}
	if p.PauseMinutes, err = get(keyPauseMinutes, BundledPauseMinutes, 0, maxPaceMinutes, "whole minutes of pause"); err != nil {
		return Pace{}, err
	}
	if p.SubAgents, err = get(keySubAgents, BundledSubAgents, 1, maxSubAgents, "a whole number of lanes"); err != nil {
		return Pace{}, err
	}
	if p.FixRounds, err = get(keyFixRounds, BundledFixRounds, 0, maxFixRounds, "a whole number of fix rounds"); err != nil {
		return Pace{}, err
	}
	return p, nil
}
