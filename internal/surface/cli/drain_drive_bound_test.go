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
// which --until reads back as the same end, and --profile only when one was
// given.
func TestTheResumeCommandNamesTheInstantAndTheProfile(t *testing.T) {
	b := driveBoundFromDays(driveStart, loop.PaceValue{Value: 7, Layer: "bundled", Origin: "bundled"})
	want := "abcd drain --drive --until 2026-12-04T08:30:15Z"
	if got := driveResumeCommand(b.end, ""); got != want {
		t.Fatalf("with no profile: %q, want %q", got, want)
	}
	if got := driveResumeCommand(b.end, "work-2"); got != want+" --profile work-2" {
		t.Fatalf("with a profile: %q", got)
	}
	if got := b.restartLine("work-2"); !strings.HasSuffix(got, want+" --profile work-2") ||
		!strings.Contains(got, "a reboot, a logout or a crash ends the drive") || !strings.Contains(got, "nothing starts it again") {
		t.Fatalf("the restart line says what ends a drive and ends on the command: %q", got)
	}
	given := strings.TrimPrefix(driveResumeCommand(b.end, ""), loop.TerminalResumeCommand+" --until ")
	at, err := parseDriveUntil(given, driveStart)
	if err != nil || !at.Equal(b.end) {
		t.Fatalf("--until reads the printed instant back as the same end: %v %v, want %v", at, err, b.end)
	}
	u := driveBoundUntil(at.In(time.Local).Add(400 * time.Millisecond))
	if got := driveResumeCommand(u.end, ""); got != want {
		t.Fatalf("an --until end names the same instant, in UTC and whole seconds: %q", got)
	}
}

// TestTheResumeCommandQuotesNothingARepositoryChose: an end set by the
// repository's days, whose origin names a file, a branch and shell words,
// gives a command of fixed words, an instant and a plain profile name alone;
// a profile that is not a plain name is left out, never quoted. No line the
// start composes repeats the origin.
func TestTheResumeCommandQuotesNothingARepositoryChose(t *testing.T) {
	planted := ".abcd/config.json at origin/$(touch pwned);`id`|x\n--profile evil"
	b := driveBoundFromDays(driveStart, loop.PaceValue{Value: 14, Layer: "repo", Origin: planted})
	shape := regexp.MustCompile(`^abcd drain --drive --until \d{4}-\d\d-\d\dT\d\d:\d\d:\d\dZ( --profile [\p{L}\p{N}][\p{L}\p{N}._-]*)?$`)
	for _, profile := range []string{"", "home", "x; rm -rf ~", "-x", "a b", "a\nb", "$(id)"} {
		got := driveResumeCommand(b.end, profile)
		if !shape.MatchString(got) {
			t.Errorf("profile %q: the command is not fixed words, an instant and a plain profile: %q", profile, got)
		}
		if profile != "home" && strings.Contains(got, "--profile") {
			t.Errorf("profile %q: a name that is not plain is left out: %q", profile, got)
		}
	}
	for _, line := range []string{b.endsLine(), b.endsPlanLine(), b.restartLine("home"), driveResumeCommand(b.end, "home")} {
		if strings.Contains(line, "origin") || strings.Contains(line, "config.json") || strings.Contains(line, "pwned") || strings.Contains(line, "evil") {
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
// lane a drive holds in work and the person's ceiling with its layer.
func TestTheHoldsLineNamesBothLimits(t *testing.T) {
	for _, c := range []struct {
		n           int
		layer, want string
	}{
		{10, claim.LimitRepository, "1 lane in work; 10 reservations per person (repository)"},
		{10, claim.LimitBundled, "1 lane in work; 10 reservations per person (default)"},
		{4, claim.LimitMachine, "1 lane in work; 4 reservations per person (this machine)"},
		{1, claim.LimitRepository, "1 lane in work; 1 reservation per person (repository)"},
		{3, "elsewhere", "1 lane in work; 3 reservations per person"},
	} {
		if got := driveHoldsLine(c.n, c.layer); got != c.want {
			t.Errorf("holds line %q, want %q", got, c.want)
		}
	}
}
