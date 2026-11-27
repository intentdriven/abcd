package cli

// drain_drive_bound.go composes what a drive's start says about how it ends
// (itd-2611251809580662 decisions 1, 6 and 9, spc-2611251810508418 step 3):
// the end and its source, the full plan's `ends`, `restart` and `holds`
// lines, and the one command that carries a drive on to the same end. The
// end is fixed once, before the consent, and every word here is built from
// that instant, a source from a closed set and figures, so nothing a
// repository's file or origin chose (a path, a branch name, an origin
// string) reaches a line a person reads as a command.

import (
	"fmt"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/runner"
)

// driveEndSource is what set a drive's end.
type driveEndSource int

const (
	// driveEndDefault is abcd's own seven days.
	driveEndDefault driveEndSource = iota
	// driveEndRepository is the repository's drive.until_days.
	driveEndRepository
	// driveEndUntil is the person's --until.
	driveEndUntil
)

// driveBound is a drive's end as its start fixed it: the instant, in UTC and
// whole seconds so the printed command names exactly it; what set it; and the
// default's days (0 for --until).
type driveBound struct {
	end    time.Time
	source driveEndSource
	days   int
}

// driveBoundFromDays is the end of a drive starting at now that runs the
// days loop's drive_bound.go resolved: the repository's where its layer is the
// repository's, else abcd's default.
func driveBoundFromDays(now time.Time, d loop.PaceValue) driveBound {
	source := driveEndDefault
	if d.Layer == layered.Repo.String() {
		source = driveEndRepository
	}
	return driveBound{
		end:    now.Add(time.Duration(d.Value) * 24 * time.Hour).UTC().Truncate(time.Second),
		source: source,
		days:   d.Value,
	}
}

// driveBoundUntil is the end the person gave with --until.
func driveBoundUntil(at time.Time) driveBound {
	return driveBound{end: at.UTC().Truncate(time.Second), source: driveEndUntil}
}

// instant is the end as the start shows it and the command names it.
func (b driveBound) instant() string {
	return b.end.Format(time.RFC3339)
}

// sourceWord is the end's source in one word, for the start's `ends` line.
func (b driveBound) sourceWord() string {
	switch b.source {
	case driveEndRepository:
		return "repository"
	case driveEndUntil:
		return "--until"
	}
	return "default"
}

// sourceInFull is the end's source in full, for the full plan's `ends` line.
func (b driveBound) sourceInFull() string {
	switch b.source {
	case driveEndRepository:
		return fmt.Sprintf("the repository's default of %s", countOf(b.days, "day"))
	case driveEndUntil:
		return "--until"
	}
	return "seven days on, abcd's default"
}

// endsLine is the start's `ends` line after its label: the end with its
// source in one word.
func (b driveBound) endsLine() string {
	return fmt.Sprintf("at %s (%s)", b.instant(), b.sourceWord())
}

// endsPlanLine is the full plan's `ends` line after its label: the end with
// its source in full.
func (b driveBound) endsPlanLine() string {
	return fmt.Sprintf("at %s, %s; waits through limits and outages until then", b.instant(), b.sourceInFull())
}

// restartLine is the full plan's `restart` line after its label: what ends a
// drive that nothing starts again, and the command that carries it on.
func (b driveBound) restartLine(profile string) string {
	return "a reboot, a logout or a crash ends the drive, and nothing starts it again; to carry on to the same end, type " +
		driveResumeCommand(b.end, profile)
}

// driveResumeCommand is the one command that carries a drive on to end: the
// drive's start with the end as an explicit --until, which a re-execution and
// a person's restart both pass, and the --profile the person typed. A profile
// that is not a plain name is left out rather than quoted, so the command is
// always words a shell reads as they stand.
func driveResumeCommand(end time.Time, profile string) string {
	cmd := loop.TerminalResumeCommand + " --until " + end.UTC().Format(time.RFC3339)
	if profile != "" && runner.PlainProfileName(profile) {
		cmd += " --profile " + profile
	}
	return cmd
}

// driveHoldsLine is the full plan's `holds` line after its label: the one
// lane a drive holds in work, whatever the repository's drain.in_work_max,
// and one person's ceiling on live reservations with the layer that set it
// (claim.MaxLiveFor).
func driveHoldsLine(ceiling int, layer string) string {
	line := fmt.Sprintf("1 lane in work; %s per person", countOf(ceiling, "reservation"))
	if w := ceilingLayerWord(layer); w != "" {
		line += " (" + w + ")"
	}
	return line
}

// ceilingLayerWord is the ceiling's layer in a word, "" for one claim does
// not name.
func ceilingLayerWord(layer string) string {
	switch layer {
	case claim.LimitRepository:
		return "repository"
	case claim.LimitMachine:
		return "this machine"
	case claim.LimitBundled:
		return "default"
	}
	return ""
}
