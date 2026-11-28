package cli

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/claim"
	"github.com/intentdriven/abcd/internal/core/implement/loop"
)

// What a drive's start says about how it ends (itd-2611251809580662,
// spc-2611251810508418 step 3): the end with its source, the full plan's
// ends, restart and holds lines, and the one command that carries a drive on
// to the same end.

// driveStart is a fixed start instant with a fraction of a second, which the
// end drops.
var driveStart = time.Date(2026, 11, 27, 9, 30, 15, 750_000_000, time.FixedZone("CET", 3600))

// TestTheResumeCommandNamesTheInstantAndTheProfile: the command is the
// drive's start with the end as an RFC 3339 UTC instant in whole seconds,
// which --until reads back as the very end the drive holds and the start
// shows, and --profile only when one was given.
func TestTheResumeCommandNamesTheInstantAndTheProfile(t *testing.T) {
	if got, want := loop.DriveResumeCommand(driveStart, ""), "abcd drain --drive --until 2026-11-27T08:30:15Z"; got != want {
		t.Fatalf("an instant in another zone is named in UTC: %q, want %q", got, want)
	}
	b := driveBoundFromDays(driveStart, loop.PaceValue{Value: 7, Layer: "bundled", Origin: "bundled"})
	want := "abcd drain --drive --until 2026-12-04T08:30:15Z"
	if got := loop.DriveResumeCommand(b.end, ""); got != want {
		t.Fatalf("with no profile: %q, want %q", got, want)
	}
	if got := loop.DriveResumeCommand(b.end, "work-2"); got != want+" --profile work-2" {
		t.Fatalf("with a profile: %q", got)
	}
	if got := b.restartLine("work-2"); !strings.HasSuffix(got, want+" --profile work-2") ||
		!strings.Contains(got, "a reboot, a logout or a crash ends the drive") || !strings.Contains(got, "nothing starts it again") {
		t.Fatalf("the restart line says what ends a drive and ends on the command: %q", got)
	}
	until := driveStart.Add(48*time.Hour + 400*time.Millisecond)
	for name, bd := range map[string]driveBound{"default": b, "--until": driveBoundUntil(until)} {
		given := strings.TrimPrefix(loop.DriveResumeCommand(bd.end, ""), loop.TerminalResumeCommand+" --until ")
		if given != bd.instant() {
			t.Errorf("%s: the command names %q, the start shows %q", name, given, bd.instant())
		}
		at, err := parseDriveUntil(given, driveStart)
		if err != nil || !at.Equal(bd.end) || at.Location() != bd.end.Location() {
			t.Errorf("%s: --until reads the printed instant back as %v (%v), the drive holds %v", name, at, err, bd.end)
		}
	}
}

// TestTheResumeCommandQuotesNothingARepositoryChose: an end set by the
// repository's days, whose origin names a file, a branch and shell words,
// gives a command of fixed words, an instant and a plain profile name alone;
// a profile that is not a plain name is left out, never quoted. Every line
// the start composes reads the same whatever origin the figure came from.
func TestTheResumeCommandQuotesNothingARepositoryChose(t *testing.T) {
	planted := ".abcd/config.json at origin/$(touch pwned);`id`|x\n--profile evil"
	b := driveBoundFromDays(driveStart, loop.PaceValue{Value: 14, Layer: "repo", Origin: planted})
	shape := regexp.MustCompile(`^abcd drain --drive --until \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ( --profile [\p{L}\p{N}][\p{L}\p{N}._-]*)?$`)
	for _, profile := range []string{"", "home", "x; rm -rf ~", "-x", "a b", "a\nb", "$(id)"} {
		got := loop.DriveResumeCommand(b.end, profile)
		if !shape.MatchString(got) {
			t.Errorf("profile %q: the command is not fixed words, an instant and a plain profile: %q", profile, got)
		}
		if profile != "home" && strings.Contains(got, "--profile") {
			t.Errorf("profile %q: a name that is not plain is left out: %q", profile, got)
		}
	}
	// Every line is a function of the days and the layer alone: an end set by
	// the same days from another origin reads the same, word for word.
	other := driveBoundFromDays(driveStart, loop.PaceValue{Value: 14, Layer: "repo", Origin: ".abcd/config.json at origin/main"})
	lines := func(b driveBound) []string {
		return []string{b.endsLine(), b.endsPlanLine(), b.restartLine("home"), loop.DriveResumeCommand(b.end, "home")}
	}
	for i, line := range lines(b) {
		if line != lines(other)[i] {
			t.Errorf("a line changes with the origin the repository's figure came from: %q, and from another origin %q", line, lines(other)[i])
		}
		if strings.Contains(line, "pwned") || strings.Contains(line, "evil") || strings.Contains(line, "`") {
			t.Errorf("a line repeats what the repository's origin named: %q", line)
		}
	}
	if !strings.Contains(b.endsPlanLine(), "the repository's default of 14 days") {
		t.Fatalf("the full plan names the repository's days by its figure alone: %q", b.endsPlanLine())
	}
}

// TestTheEndsLineNamesItsSourceInOneWord: the start's ends line is the
// instant with its source as one word, (default), (repository) or (--until),
// and the full plan's names it in full.
func TestTheEndsLineNamesItsSourceInOneWord(t *testing.T) {
	if loop.BundledDriveDays != 7 {
		t.Fatalf("the default's words say seven days; the bundled figure is %d", loop.BundledDriveDays)
	}
	until := time.Date(2026, 12, 1, 18, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		b          driveBound
		ends, full string
	}{
		{driveBoundFromDays(driveStart, loop.PaceValue{Value: 7, Layer: "bundled", Origin: "bundled"}),
			"at 2026-12-04T08:30:15Z (default)", "at 2026-12-04T08:30:15Z, seven days on, abcd's default;"},
		{driveBoundFromDays(driveStart, loop.PaceValue{Value: 14, Layer: "repo", Origin: ".abcd/config.json at origin/main"}),
			"at 2026-12-11T08:30:15Z (repository)", "at 2026-12-11T08:30:15Z, the repository's default of 14 days;"},
		{driveBoundFromDays(driveStart, loop.PaceValue{Value: 1, Layer: "repo", Origin: ".abcd/config.json at origin/main"}),
			"at 2026-11-28T08:30:15Z (repository)", "the repository's default of 1 day;"},
		{driveBoundUntil(until), "at 2026-12-01T18:00:00Z (--until)", "at 2026-12-01T18:00:00Z, --until;"},
	} {
		if got := c.b.endsLine(); got != c.ends {
			t.Errorf("ends line %q, want %q", got, c.ends)
		}
		if got := c.b.endsPlanLine(); !strings.Contains(got, c.full) {
			t.Errorf("full plan's ends line %q, want it to hold %q", got, c.full)
		}
	}
}

// TestTheHoldsLineNamesBothLimits: the full plan's holds line names the one
// lane a drive holds in work and the person's ceiling with its layer, read
// at what this checkout last fetched of origin and said to be so: the act
// that adds a reservation asks origin itself.
func TestTheHoldsLineNamesBothLimits(t *testing.T) {
	for _, c := range []struct {
		n           int
		layer, want string
	}{
		{10, claim.LimitRepository, "1 lane in work; 10 reservations per person (repository), as this checkout last fetched them"},
		{10, claim.LimitBundled, "1 lane in work; 10 reservations per person (default), as this checkout last fetched them"},
		{4, claim.LimitMachine, "1 lane in work; 4 reservations per person (this machine), as this checkout last fetched them"},
		{1, claim.LimitRepository, "1 lane in work; 1 reservation per person (repository), as this checkout last fetched them"},
		{3, "elsewhere", "1 lane in work; 3 reservations per person, as this checkout last fetched them"},
		{0, "", "1 lane in work; the reservations per person are as the first move reads them at origin"},
	} {
		if got := driveHoldsLine(c.n, c.layer); got != c.want {
			t.Errorf("holds line %q, want %q", got, c.want)
		}
	}
}
