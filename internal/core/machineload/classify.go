package machineload

import (
	"sort"
	"time"
)

// Self is the check's own invocation: its pid, whose parent chain is exempt,
// and its effective uid, which decides whose strays are the caller's own.
type Self struct {
	PID int
	UID uint32
}

// NearFullShare is the part of its fair share (FairShare) at which a process is
// using nearly all the CPU it could get: on a machine loaded no higher than its
// cores, 0.9 of a core.
const NearFullShare = 0.9

// FairShare is the CPU, in cores, one runnable process can get on the machine as
// the snapshot finds it loaded: the online cores divided by the one-minute load,
// the runnable demand, and never more than one core. With no load reading, no
// core count, or a load no higher than the cores, it is one core. Forty busy
// loops on 16 cores each get 0.4 of a core, and each uses all of it.
func FairShare(snap Snapshot) float64 {
	if !snap.HasLoad || snap.Cores <= 0 || snap.Load1 <= float64(snap.Cores) {
		return 1
	}
	return float64(snap.Cores) / snap.Load1
}

// MaxOwnNamed caps how many of the caller's own strays a warning names; the rest
// are counted.
const MaxOwnNamed = 20

// The two triggers.
const (
	TriggerStray   = "stray"
	TriggerExtreme = "extreme"
)

// The two remedy forms, and why a stray gets the process form.
const (
	// RemedyGroup: `pgrep -g <pgid>`, then `kill -- -<pgid>`.
	RemedyGroup = "group"
	// RemedyProcess: `ps -o pid=,comm= -p <pid>`, then `kill <pid>`.
	RemedyProcess = "process"

	// WhyMixedGroup: the stray's group holds a program the warning does not name.
	WhyMixedGroup = "mixed-group"
	// WhyCallersGroup: the stray's group is the check's own, or an ancestor's.
	WhyCallersGroup = "callers-group"
)

// OwnStray is one of the caller's own strays, named in full.
type OwnStray struct {
	Name  string
	PID   int
	PGID  int
	Age   time.Duration
	Share float64
}

// OtherStrays is every other account's strays: how many, and how many cores they
// use between them. The type has no field that could hold a name, a command
// line, a uid or an account, so no renderer can leak one
// (TestOtherStraysTypeCarriesNoIdentity).
type OtherStrays struct {
	Count int
	Cores float64
}

// Remedy is one way to stop some of the caller's own strays. It is a plan for
// the caller to carry out; nothing here signals anything.
type Remedy struct {
	// Form is RemedyGroup or RemedyProcess.
	Form string
	// PGID and PIDs are the group form's group and the strays it holds.
	PGID int
	PIDs []int
	// PID and Why are the process form's process and why it is not the group form.
	PID int
	Why string
}

// Verdict is what one snapshot says.
type Verdict struct {
	// Triggers holds TriggerStray and TriggerExtreme, in that order, when they
	// fire; empty is nothing to warn about.
	Triggers []string
	// Own is the caller's own strays, at most MaxOwnNamed, by pid; OwnMore counts
	// the rest.
	Own     []OwnStray
	OwnMore int
	// Others is every other account's strays, anonymously.
	Others OtherStrays
	// Remedy covers every stray in Own.
	Remedy []Remedy
}

// Classify decides what a snapshot says under the limits, for the invocation
// self. It is a pure function.
//
// A process is a stray when it is older than the stray limit, its lifetime share
// is at least NearFullShare of its FairShare, and it is not the invocation or
// one of its ancestors: busy for its share, it uses nearly all the CPU it could
// get on the machine as loaded, so forty loops each at a fortieth of the machine
// are all strays, as one loop at a full core of an idle machine is (the product
// thinker's ruling H1 of 2026-09-25). The one snapshot decides; there is no
// second sample. The share test applies before the uid split, so the caller's
// own processes and every other account's are judged alike. Nothing is exempt by name: abcd's own lanes are exempt by time,
// because everything they start (go, compile, link, vet, the package test
// binaries) lives for seconds to minutes, so a hung test binary reads as a stray,
// as the intent's third scope condition says it must. A stray whose effective uid
// is the caller's is the caller's own; any other uid, root included, is another
// account's.
//
// The extreme trigger fires when the one-minute load average is strictly above
// the extreme limit. The five- and fifteen-minute averages remember work that
// finished long ago, so they are reported and never decide.
func Classify(snap Snapshot, lim Limits, self Self) Verdict {
	var v Verdict
	byPID := make(map[int]Proc, len(snap.Procs))
	for _, p := range snap.Procs {
		byPID[p.PID] = p
	}

	// The invocation and its ancestors, and their groups.
	ancestry := map[int]bool{}
	callerGroups := map[int]bool{}
	for pid, hops := self.PID, 0; pid > 0 && !ancestry[pid] && hops < 4096; hops++ {
		ancestry[pid] = true
		p, ok := byPID[pid]
		if !ok {
			break
		}
		callerGroups[p.PGID] = true
		pid = p.PPID
	}

	var own []OwnStray
	if snap.HasProcs {
		limit := time.Duration(lim.StrayMinutes) * time.Minute
		threshold := NearFullShare * FairShare(snap)
		for _, p := range snap.Procs {
			if ancestry[p.PID] || p.Age <= limit {
				continue
			}
			share := p.Share()
			if share < threshold {
				continue
			}
			if p.UID == self.UID {
				own = append(own, OwnStray{Name: p.Name, PID: p.PID, PGID: p.PGID, Age: p.Age, Share: share})
				continue
			}
			v.Others.Count++
			v.Others.Cores += share
		}
	}
	sort.Slice(own, func(i, j int) bool { return own[i].PID < own[j].PID })
	if len(own) > MaxOwnNamed {
		v.OwnMore = len(own) - MaxOwnNamed
		own = own[:MaxOwnNamed]
	}
	v.Own = own

	if len(own) > 0 || v.Others.Count > 0 {
		v.Triggers = append(v.Triggers, TriggerStray)
	}
	if snap.HasLoad && snap.Load1 > lim.ExtremeLoad {
		v.Triggers = append(v.Triggers, TriggerExtreme)
	}
	v.Remedy = remedies(snap.Procs, own, callerGroups)
	return v
}

// remedies plans how to stop the named strays. The group form is offered only
// when every live member of the group is a stray the warning names and the group
// is neither the invocation's nor an ancestor's, so a group kill can never reach
// the caller's shell, its agent host or a program nobody named. Every other
// stray gets the process form.
func remedies(procs []Proc, own []OwnStray, callerGroups map[int]bool) []Remedy {
	named := map[int]bool{}
	for _, s := range own {
		named[s.PID] = true
	}
	mixed := map[int]bool{}
	for _, p := range procs {
		if !named[p.PID] {
			mixed[p.PGID] = true
		}
	}
	var out []Remedy
	grouped := map[int]int{} // pgid -> index in out
	for _, s := range own {
		switch {
		case callerGroups[s.PGID] || s.PGID <= 1:
			out = append(out, Remedy{Form: RemedyProcess, PID: s.PID, Why: WhyCallersGroup})
		case mixed[s.PGID]:
			out = append(out, Remedy{Form: RemedyProcess, PID: s.PID, Why: WhyMixedGroup})
		default:
			if i, ok := grouped[s.PGID]; ok {
				out[i].PIDs = append(out[i].PIDs, s.PID)
				continue
			}
			grouped[s.PGID] = len(out)
			out = append(out, Remedy{Form: RemedyGroup, PGID: s.PGID, PIDs: []int{s.PID}})
		}
	}
	return out
}
