package loop

// drain_inwork.go is the drain's limit on its lanes in work
// (itd-2611251809580662 decision 2, spc-2611251810508418 step 2):
// `drain.in_work_max`, resolved once when a drain begins and kept on the
// drain's state with its layer and origin, as the bundle cap is. A lane is in
// work while the drain reads it in progress for any reason but the withdrawal
// its hand-back owes (owedArming); a lane whose landing armed its merge reads
// as a pull request and leaves the count, keeping its reservation until the
// merge. The move opens its next lane only while the count is below the
// limit, so with the bundled one a drain works one lane at a time, as it
// always has.
//
// The figure is the repository's and the run's, never the machine's: the
// --in-work flag, then the repository's .abcd/config.json at the commit origin
// itself names as its default branch's tip (claim.AskOriginDefault, the read
// reservation.max_live's act makes), as the object store holds it and never
// from a working tree, then the bundled one. A drain.in_work_max in the
// machine's file is refused naming the file: a machine that wants fewer lanes
// starts fewer drains. Every misreading leans to the lower number or a
// refusal: a repository value outside 1 to 10 refuses the drain's start
// naming its origin, and an origin that cannot be asked, or names no default
// branch, leaves the bundled one, the lowest the setting can be, said in the
// move's next step.

import (
	"fmt"
	"slices"
	"strings"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/termsafe"
)

// BundledInWorkMax is the lanes one drain holds in work when nothing sets
// drain.in_work_max: one at a time.
const BundledInWorkMax = 1

// MaxInWorkMax bounds drain.in_work_max and --in-work.
const MaxInWorkMax = 10

// keyInWorkMax is the limit's key under the `drain` namespace.
const keyInWorkMax = "in_work_max"

// drainKeys are every key the `drain` namespace holds: both of its
// resolvers claim the namespace with all of them, so a file that sets one is
// never refused by the other's claim as a misspelt key.
var drainKeys = []string{keyBundleMax, keyInWorkMax}

// inWorkMaxKey is the limit's key as a person writes it.
var inWorkMaxKey = drainNamespace + "." + keyInWorkMax

// inWorkMaxLayers are the layers a stored limit may name: the ones
// resolveInWorkMax writes. The machine is not among them.
var inWorkMaxLayers = []string{layered.Flag.String(), layered.Repo.String(), layered.Bundled.String()}

// inWorkMaxForm is the accepted form every refusal of the limit names.
var inWorkMaxForm = "drain.in_work_max in the repository's .abcd/config.json, read at origin's default branch, or --in-work <n> for one drain, is the most lanes one drain holds in work, a whole number from 1 to 10 (1, the default, works one lane at a time); " +
	abcdhome.Display("config.json") + " does not hold it"

// drainAskOrigin is claim.AskOriginDefault, a seam a test replaces to make
// origin unanswerable.
var drainAskOrigin = claim.AskOriginDefault

// bundledInWorkMax is the limit a drain carries when nothing set it, and the
// one a drain state of version 4 or lower is read with.
func bundledInWorkMax() PaceValue {
	return PaceValue{Value: BundledInWorkMax, Layer: layered.Bundled.String(), Origin: layered.Bundled.String()}
}

// resolveInWorkMax resolves the limit a drain beginning now holds its lanes
// in work to: flag (--in-work, 0 when not given), else the repository's
// figure at origin's default branch, else the bundled one, with a note saying
// why no repository figure was read where one could not be. roots name the
// machine's file, which is read only to refuse the key there.
func resolveInWorkMax(repoRoot string, roots layered.Roots, flag int) (PaceValue, string, error) {
	if err := refuseMachineInWork(roots); err != nil {
		return PaceValue{}, "", err
	}
	if flag != 0 {
		if err := checkInWorkMax(flag); err != nil {
			return PaceValue{}, "", inWorkMaxRefusal(fmt.Sprintf("--in-work %d: %v", flag, err))
		}
		return PaceValue{Value: flag, Layer: layered.Flag.String(), Origin: fmt.Sprintf("--in-work %d", flag)}, "", nil
	}
	v := bundledInWorkMax()
	has, err := claim.HasOrigin(repoRoot)
	if err != nil || !has {
		// No origin, no repository figure: a checkout's own file never
		// speaks for the repository.
		return v, "", nil
	}
	at, err := drainAskOrigin(repoRoot, inWorkMaxKey)
	if err != nil {
		return v, fmt.Sprintf("%s could not be asked which branch is its default (%s), so no repository figure for %s was read and this drain holds %d lane in work, abcd's default",
			claim.Remote, termsafe.Sanitize(fsutil.RedactHome(err.Error())), inWorkMaxKey, v.Value), nil
	}
	if at.None != "" {
		return v, fmt.Sprintf("%s, so it has no repository figure for %s and this drain holds %d lane in work, abcd's default",
			termsafe.Sanitize(at.None), inWorkMaxKey, v.Value), nil
	}
	s, err := layered.Load(layered.Config, layered.Roots{Repo: repoRoot, RepoAt: at.Tip, RepoAtName: at.Name, RepoAtAsStored: true, HomeRefused: true})
	if err != nil {
		return PaceValue{}, "", inWorkMaxRefusal(err.Error())
	}
	if err := s.Claim(drainNamespace, drainKeys...); err != nil {
		return PaceValue{}, "", inWorkMaxRefusal(err.Error())
	}
	found, err := s.Lookup(inWorkMaxKey)
	if err != nil {
		return PaceValue{}, "", inWorkMaxRefusal(err.Error())
	}
	for _, f := range found {
		if f.Layer != layered.Repo {
			continue
		}
		n, err := layered.Decode[int](f.Raw)
		if err == nil {
			err = checkInWorkMax(n)
		}
		if err != nil {
			return PaceValue{}, "", inWorkMaxRefusal(fmt.Sprintf("%s (commit %s): %s is %s: %v", f.Origin, at.Tip[:12], inWorkMaxKey, compactRaw(f.Raw), err))
		}
		v = PaceValue{Value: n, Layer: layered.Repo.String(), Origin: f.Origin}
	}
	return v, "", nil
}

// refuseMachineInWork refuses a drain.in_work_max in the machine's file,
// naming the file, as a key that layer does not hold. The machine layer is
// loaded alone, so no working tree is read here; a machine file that cannot
// be read is the bundle cap's resolver's to refuse, which reads it too.
func refuseMachineInWork(roots layered.Roots) error {
	if roots.HomeRefused || roots.Home == "" {
		return nil
	}
	s, err := layered.Load(layered.Config, layered.Roots{Home: roots.Home})
	if err != nil {
		return inWorkMaxRefusal(err.Error())
	}
	found, err := s.Lookup(inWorkMaxKey)
	if err != nil {
		return inWorkMaxRefusal(err.Error())
	}
	if slices.ContainsFunc(found, func(f layered.Found) bool { return f.Layer == layered.Machine }) {
		return inWorkMaxRefusal(fmt.Sprintf("%s sets %s, which that file does not hold: the lanes one drain holds in work are the repository's and the run's, not one machine's; remove it, and give --in-work to the drain instead",
			layered.Config.MachineOrigin(), inWorkMaxKey))
	}
	return nil
}

// checkInWorkMax holds a figure to 1 to MaxInWorkMax.
func checkInWorkMax(n int) error {
	if n < 1 || n > MaxInWorkMax {
		return fmt.Errorf("want a whole number of lanes from 1 to %d", MaxInWorkMax)
	}
	return nil
}

func inWorkMaxRefusal(reason string) error {
	return refuse(StageDrain, "", "", termsafe.Sanitize(fsutil.RedactHome(reason)), inWorkMaxForm)
}

// inWorkMaxWritten reports whether a stored limit is one resolveInWorkMax
// writes: a value in range, a layer it writes and an origin with no control,
// bidi or zero-width rune.
func inWorkMaxWritten(v *PaceValue) bool {
	return v == nil || v.Value >= 1 && v.Value <= MaxInWorkMax &&
		slices.Contains(inWorkMaxLayers, v.Layer) && termsafe.Sanitize(v.Origin) == v.Origin
}

// inWorkCap is the limit the drain started with: the bundled one for a state
// that records none.
func (s DrainState) inWorkCap() PaceValue {
	if s.InWorkMax == nil {
		return bundledInWorkMax()
	}
	return *s.InWorkMax
}

// LanesInWork is a drain's lanes in work against its limit: the count, and
// the limit with the layer and origin it came from.
type LanesInWork struct {
	Count  int    `json:"count"`
	Limit  int    `json:"limit"`
	Layer  string `json:"layer"`
	Origin string `json:"origin"`
}

// HeldCount is the person's live reservations against their ceiling, as the
// move's view told them: the count, the ceiling and the layer that set it
// (claim.LimitBundled, LimitRepository or LimitMachine).
type HeldCount struct {
	Live  int    `json:"live"`
	Limit int    `json:"limit"`
	Layer string `json:"layer"`
}

// inWorkLane is a lane in work as the move read it, with its run.
type inWorkLane struct {
	lane DrainLane
	run  State
}

// lanesInWork counts st's lanes in work: in progress, not passed over by the
// drive, and not in progress only for the withdrawal its hand-back owes. A
// lane whose run cannot be read is counted, so an unreadable state never
// reads the count low.
func (st DrainState) lanesInWork(repoRoot string, d DrainOptions) LanesInWork {
	n := 0
	for _, l := range st.Lanes {
		if l.Outcome != DrainLaneInProgress || d.passedLane(l.RunID) {
			continue
		}
		if run, err := ReadState(repoRoot, l.RunID); err == nil && owedArming(run) > 0 {
			continue
		}
		n++
	}
	c := st.inWorkCap()
	return LanesInWork{Count: n, Limit: c.Value, Layer: c.Layer, Origin: c.Origin}
}

// DrainInWork is the drain in progress's lanes in work against its limit, as
// `abcd peers` shows them; nil when no drain is in progress.
func DrainInWork(repoRoot string) (*LanesInWork, error) {
	st, live, err := readDrain(repoRoot)
	if err != nil || !live {
		return nil, err
	}
	iw := st.lanesInWork(repoRoot, DrainOptions{})
	return &iw, nil
}

// inWorkPhrase names the lanes in work against the limit: "1 of 1 lane in
// work".
func inWorkPhrase(n, limit int) string {
	return fmt.Sprintf("%d of %d %s in work", n, limit, plural(limit, "lane", "lanes"))
}

// heldPhrase names the person's holds against the ceiling, "" when the move
// could not tell them: "you hold 7 of the 10 reservations the repository
// allows".
func heldPhrase(h *HeldCount) string {
	if h == nil {
		return ""
	}
	who := "abcd's default"
	switch h.Layer {
	case claim.LimitRepository:
		who = "the repository"
	case claim.LimitMachine:
		who = "this machine"
	}
	return fmt.Sprintf("you hold %d of the %d reservations %s allows", h.Live, h.Limit, who)
}

// countsPhrase is both counts, as the drain's wait sentences name them.
func countsPhrase(n, limit int, h *HeldCount) string {
	return joinNext(inWorkPhrase(n, limit), heldPhrase(h))
}

// inHandNext is the drain's next step while it holds lanes in work: drive
// each, by its run, with both counts.
func inHandNext(working []inWorkLane, limit int, h *HeldCount, b stuckBounds) string {
	counts := countsPhrase(len(working), limit, h)
	if len(working) == 1 {
		l, run := working[0].lane, working[0].run
		next := fmt.Sprintf("drive %s's lane, run %s: `abcd implement step --run %s` (%s); run `abcd drain` again once it is handed back or its pull request is open",
			l.Issue, l.RunID, l.RunID, counts)
		if note := refusingNote(run, b); note != "" {
			next += "; " + note
		}
		return next
	}
	parts := make([]string, 0, len(working))
	for _, w := range working {
		p := fmt.Sprintf("%s's lane, run %s (`abcd implement step --run %s`)", w.lane.Issue, w.lane.RunID, w.lane.RunID)
		if note := refusingNote(w.run, b); note != "" {
			p += ", " + note
		}
		parts = append(parts, p)
	}
	return fmt.Sprintf("drive each lane in work: %s (%s); run `abcd drain` again once one is handed back or its pull request is open",
		strings.Join(parts, "; "), counts)
}

// besideNext names the lanes already in work beside the one a move goes on to
// ask or open, with both counts.
func besideNext(working []inWorkLane, limit int, h *HeldCount) string {
	parts := make([]string, 0, len(working))
	for _, w := range working {
		parts = append(parts, fmt.Sprintf("%s's lane, run %s (`abcd implement step --run %s`)", w.lane.Issue, w.lane.RunID, w.lane.RunID))
	}
	return fmt.Sprintf("in work beside it: %s (%s before this move)", strings.Join(parts, "; "), countsPhrase(len(working), limit, h))
}
