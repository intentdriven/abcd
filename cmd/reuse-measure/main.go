// Command reuse-measure reads the reports make preflight printed for the last
// landings on the default branch and sets their reuse and check time against the
// baseline, the measure criterion 6 of itd-2611011325421304 asks for (step 3 of
// spc-2611012047503720).
//
// Usage, from the repository root:
//
//	go run ./cmd/reuse-measure [-ref <ref>] [-n 10] [-baseline <log>] <log or directory>...
//
// A landing is a merge on the first-parent line of -ref (the default branch
// when it is not given), newest first, and its head is the merge's last parent,
// the commit the preflight checked. A landing's report is a run in a preflight
// log, a regular file named outright or a .log file directly inside a directory
// named, that minted a push receipt for that head: only a passing run mints one.
// One log can hold several runs, so each is judged alone: its figures are the
// lines its own mint printed straight after its receipt, the reuse counts of both
// stages and the times, and a run whose mint was refused reports nothing, though
// it prints its times too. A log over 64 MiB, or a line over 64 KiB, is not read;
// a full run whose counts do not read is named and counted nowhere.
//
// The baseline is the count iss-2611011343052330 recorded on 2026-11-01, 0 of
// 133 packages reused, which has no time. -baseline names the log of a timed run
// that reused nothing (make preflight PREFLIGHT_FRESH=1), and only against it is
// a time saving stated: the run's time less the median of the landings that ran
// test stages, for the whole check and for the race stage. A records-only
// landing counts as a landing and stays out of the reuse shares and both
// medians, since it ran no test stage either way.
//
// It prints the table and the shares in Markdown, then the saving or every
// reason none is stated: a landing with no report here, a report that prints no
// time, and no timed baseline. It exits 0 when it states the saving, 1 when the
// measure is short of something, and 2 for a usage mistake, a baseline that
// reused a package or prints no time, or a path or a git history it cannot read.
//
// It is developer tooling, never part of a release.
package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/intentdriven/abcd/internal/fsutil"
	"github.com/intentdriven/abcd/internal/gitutil"
)

const (
	exitIncomplete = 1
	exitUsage      = 2
)

// The 2026-11-01 baseline, from the body of iss-2611011343052330.
const (
	baselineDate     = "2026-11-01"
	baselineIssue    = "iss-2611011343052330"
	baselinePackages = 133
)

// report is one passing preflight's figures, as its log prints them.
type report struct {
	path, head, scope, mode                      string
	unitReused, unitTotal, raceReused, raceTotal int
	race, checked, waited                        time.Duration
	// counted is whether a full run's log prints its reuse counts; timed is
	// whether it prints the check's time; race is set only for a full run that
	// prints the race stage's time too.
	counted, timed bool
}

// landing is one merge on the default branch and the head it landed.
type landing struct{ pr, head string }

// The lines a run's end prints, as scripts/preflight-receipt.sh's mint and
// clock_report word them (TestTheScriptsOwnLinesAreTheOnesRead runs the script
// and reads what it prints). A run's report opens at its mint's first line, the
// receipt or a refusal, and holds only the lines the script prints straight after
// it: the counts, each stage's time and the checked-in line, which closes it.
var (
	receiptLine = regexp.MustCompile(`^preflight: push receipt minted for ([0-9a-f]{12})(, records-only)?`)
	refusedLine = regexp.MustCompile(`^preflight: no push receipt minted`)
	countsLine  = regexp.MustCompile(`^\s+the unit stage reused (\d+) of (\d+) packages and the race stage (\d+) of (\d+) \(([a-z]+)\)`)
	stageLine   = regexp.MustCompile(`^preflight: stage [a-z]+: `)
	raceLine    = regexp.MustCompile(`^preflight: stage race: (\S+)$`)
	checkedLine = regexp.MustCompile(`^preflight: checked in (\S+) \(waited ([0-9hms]+)`)
	prNumber    = regexp.MustCompile(`#[0-9]+`)
)

// The caps on what is read: a log over maxLog bytes is not read, and a line
// over maxLine is read as no line of a report. The largest preflight log kept
// on 2026-11-21 was 1.2 MiB and its longest line under 2 KiB. Variables so a
// test can lower them.
var maxLog, maxLine int64 = 64 << 20, 64 << 10

// errUnreadable marks a log that is no regular file or is over maxLog: refused
// when named outright, and named as not read when found in a directory.
var errUnreadable = errors.New("not read")

func main() {
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "reuse-measure:", err)
		os.Exit(exitUsage)
	}
	os.Exit(run(os.Args[1:], wd, os.Stdout, os.Stderr))
}

// readRuns reads every run in the log at path that minted a push receipt, in
// the order the log holds them, each judged alone: its figures are the lines
// its own mint printed, never a later run's. A run whose mint was refused, or
// that failed before its mint, reports nothing. The log is opened once, refused
// unless it is a regular file within maxLog (a symlink or a FIFO is refused,
// never followed or waited on), and a line over maxLine closes any open report
// and is read as nothing else.
func readRuns(path string) ([]report, error) {
	f, fi, err := fsutil.OpenRegular(path)
	switch {
	case errors.Is(err, fsutil.ErrNotRegular), errors.Is(err, syscall.ELOOP):
		return nil, fmt.Errorf("%s: %w: it is not a regular file", path, errUnreadable)
	case err != nil:
		return nil, err
	}
	defer f.Close()
	if fi.Size() > maxLog {
		return nil, fmt.Errorf("%s: %w: it holds %d bytes, over the %d a log may hold", path, errUnreadable, fi.Size(), maxLog)
	}
	var (
		runs   []report
		open   *report
		minted bool
	)
	end := func() {
		if open != nil && minted {
			runs = append(runs, *open)
		}
		open = nil
	}
	br := bufio.NewReader(io.LimitReader(f, maxLog))
	for {
		line, long, readErr := readLine(br, maxLine)
		switch {
		case long:
			end()
		case receiptLine.MatchString(line):
			end()
			m := receiptLine.FindStringSubmatch(line)
			open, minted = &report{path: path, head: m[1], scope: "full"}, true
			if m[2] != "" {
				open.scope = "records-only"
			}
		case refusedLine.MatchString(line):
			end()
			open, minted = &report{path: path}, false
		case open != nil:
			closes, err := open.take(line)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			if closes {
				end()
			}
		}
		if readErr == io.EOF {
			end()
			return runs, nil
		}
		if readErr != nil {
			return nil, readErr
		}
	}
}

// readLine reads one line without its line ending; long is true, and line
// empty, when it is over limit bytes, which are read past and never held.
func readLine(br *bufio.Reader, limit int64) (line string, long bool, err error) {
	var buf []byte
	for {
		frag, err := br.ReadSlice('\n')
		if !long {
			if int64(len(buf)+len(frag)) > limit+2 {
				long, buf = true, nil
			} else {
				buf = append(buf, frag...)
			}
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		return strings.TrimRight(string(buf), "\r\n"), long, err
	}
}

// take fills r from one line printed after its mint's first line, and reports
// whether the line closes the run's report: the checked-in line, the line saying
// no time is reported, or any line the mint and its clock do not print.
func (r *report) take(line string) (closes bool, err error) {
	if m := countsLine.FindStringSubmatch(line); m != nil {
		n := make([]int, 4)
		for i := range n {
			v, err := strconv.Atoi(m[i+1])
			if err != nil {
				return true, err
			}
			n[i] = v
		}
		r.unitReused, r.unitTotal, r.raceReused, r.raceTotal, r.mode = n[0], n[1], n[2], n[3], m[5]
		r.counted = true
		return false, nil
	}
	if m := raceLine.FindStringSubmatch(line); m != nil {
		d, err := time.ParseDuration(m[1])
		if err != nil {
			return true, fmt.Errorf("race stage time %q: %w", m[1], err)
		}
		r.race = d
		return false, nil
	}
	if m := checkedLine.FindStringSubmatch(line); m != nil {
		c, err := time.ParseDuration(m[1])
		if err != nil {
			return true, fmt.Errorf("check time %q: %w", m[1], err)
		}
		w, err := time.ParseDuration(m[2])
		if err != nil {
			return true, fmt.Errorf("wait %q: %w", m[2], err)
		}
		r.checked, r.waited, r.timed = c, w, true
		return true, nil
	}
	// The mint's continuation lines are indented; each stage's time keeps the
	// report open for the checked-in line after it.
	if stageLine.MatchString(line) || strings.HasPrefix(line, " ") {
		return false, nil
	}
	return true, nil
}

// readLandings lists the last n merges on ref's first-parent line, newest first.
func readLandings(root, ref string, n int) ([]landing, error) {
	out, err := gitutil.Run(root, "log", "--first-parent", "--merges", "-n", strconv.Itoa(n), "--format=%P%x09%s", "--end-of-options", ref, "--")
	if err != nil {
		return nil, err
	}
	var ls []landing
	for line := range strings.SplitSeq(out, "\n") {
		parents, subject, found := strings.Cut(line, "\t")
		if !found {
			continue
		}
		fields := strings.Fields(parents)
		head := fields[len(fields)-1]
		pr := prNumber.FindString(subject)
		if pr == "" {
			pr = head[:12]
		}
		ls = append(ls, landing{pr: pr, head: head})
	}
	return ls, nil
}

// readReports reads every report in the paths named, keyed by head. A path
// named outright is a directory, whose .log files directly inside it are read,
// or a regular file; anything else, or a named file that cannot be read, is
// refused. A log in a directory that cannot be read is named in unread and the
// rest are read. Every run that minted a receipt reports, and a head with more
// than one report takes the one that prints its time, then the later: the later
// name, or the later run in one log.
func readReports(paths []string) (reports map[string]report, unread []string, err error) {
	type logFile struct {
		path  string
		named bool
	}
	var files []logFile
	for _, p := range paths {
		info, err := os.Lstat(p)
		if err != nil {
			return nil, nil, err
		}
		switch {
		case info.Mode().IsRegular():
			files = append(files, logFile{p, true})
			continue
		case !info.IsDir():
			return nil, nil, fmt.Errorf("%s is neither a directory nor a regular file", p)
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil, nil, err
		}
		for _, e := range entries {
			if e.Type().IsRegular() && strings.HasSuffix(e.Name(), ".log") {
				files = append(files, logFile{filepath.Join(p, e.Name()), false})
			}
		}
	}
	slices.SortFunc(files, func(a, b logFile) int { return strings.Compare(a.path, b.path) })
	reports = map[string]report{}
	for _, f := range files {
		runs, err := readRuns(f.path)
		if err != nil {
			if f.named {
				return nil, nil, err
			}
			unread = append(unread, fmt.Sprintf("%s (%s)", filepath.Base(f.path), strings.TrimPrefix(err.Error(), f.path+": ")))
			continue
		}
		for _, r := range runs {
			if have, seen := reports[r.head]; seen && have.timed && !r.timed {
				continue
			}
			reports[r.head] = r
		}
	}
	return reports, unread, nil
}

// readBaseline reads the timed run that reused nothing, refusing any other. A
// log holding more than one passing run gives its last.
func readBaseline(path string) (report, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return report{}, err
	}
	if !info.Mode().IsRegular() {
		return report{}, fmt.Errorf("%s is not a regular file", path)
	}
	runs, err := readRuns(path)
	if err != nil {
		return report{}, err
	}
	if len(runs) == 0 {
		return report{}, fmt.Errorf("%s holds no push receipt, so it is no passing run", path)
	}
	r := runs[len(runs)-1]
	switch {
	case r.scope != "full":
		return report{}, fmt.Errorf("%s ran no test stage, so it is no baseline", path)
	case !r.counted:
		return report{}, fmt.Errorf("%s prints no reuse counts, so it cannot show it reused nothing", path)
	case r.unitTotal+r.raceTotal == 0:
		return report{}, fmt.Errorf("%s ran no test stage, so it is no baseline", path)
	case r.unitReused+r.raceReused != 0:
		return report{}, fmt.Errorf("%s reused %d packages; the baseline is a run that reused nothing (make preflight PREFLIGHT_FRESH=1)", path, r.unitReused+r.raceReused)
	case !r.timed || r.race == 0:
		return report{}, fmt.Errorf("%s prints no check time or no race stage time, so no time can be set against it", path)
	}
	return r, nil
}

func run(args []string, wd string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reuse-measure", flag.ContinueOnError)
	fs.SetOutput(stderr)
	ref := fs.String("ref", "", "the default branch to read landings from (its remote-tracking ref when not given)")
	n := fs.Int("n", 10, "how many landings to measure")
	baselinePath := fs.String("baseline", "", "the log of a timed run that reused nothing (make preflight PREFLIGHT_FRESH=1)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if fs.NArg() == 0 || *n < 1 {
		fmt.Fprintln(stderr, "usage: reuse-measure [-ref <ref>] [-n 10] [-baseline <log>] <log or directory>...")
		return exitUsage
	}
	fail := func(err error) int {
		fmt.Fprintln(stderr, "reuse-measure:", err)
		return exitUsage
	}
	var base *report
	if *baselinePath != "" {
		b, err := readBaseline(*baselinePath)
		if err != nil {
			return fail(err)
		}
		base = &b
	}
	if *ref == "" {
		if *ref = gitutil.DefaultRef(wd); *ref == "" {
			return fail(errors.New("no default branch resolves here; name one with -ref"))
		}
	}
	ls, err := readLandings(wd, *ref, *n)
	if err != nil {
		return fail(fmt.Errorf("reading the landings on %s: %w", *ref, err))
	}
	reports, unread, err := readReports(fs.Args())
	if err != nil {
		return fail(err)
	}
	if write(stdout, *ref, *n, ls, reports, unread, base) {
		return 0
	}
	return exitIncomplete
}

// write prints the measure and reports whether it states a saving.
func write(w io.Writer, ref string, n int, ls []landing, reports map[string]report, unread []string, base *report) bool {
	fmt.Fprintf(w, "# Reuse over the last %d landings on %s\n\n", n, ref)
	fmt.Fprintln(w, "| PR | Head | Report | Unit reused | Race reused | Race stage | Checked in | Waited |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|")
	var gaps []string
	if len(ls) < n {
		gaps = append(gaps, fmt.Sprintf("%s holds %d landings, not %d", ref, len(ls), n))
	}
	var unitR, unitT, raceR, raceT int
	var checks, races []time.Duration
	for _, l := range ls {
		short := l.head[:12]
		r, ok := reports[short]
		if !ok {
			fmt.Fprintf(w, "| %s | %s | no report here | | | | | |\n", l.pr, short)
			gaps = append(gaps, l.pr+" has no report here")
			continue
		}
		name := filepath.Base(r.path)
		checked, waited := "not printed", "not printed"
		if r.timed {
			checked, waited = clock(r.checked), clock(r.waited)
		} else {
			gaps = append(gaps, l.pr+"'s report prints no time")
		}
		if r.scope == "records-only" {
			fmt.Fprintf(w, "| %s | %s | %s | records-only | records-only | records-only | %s | %s |\n", l.pr, short, name, checked, waited)
			continue
		}
		race := "not printed"
		if r.race > 0 {
			race = clock(r.race)
		}
		if !r.counted {
			// Counts that do not read are no reuse at all: the report stays out of
			// the shares and the medians, and is named.
			fmt.Fprintf(w, "| %s | %s | %s | not printed | not printed | %s | %s | %s |\n", l.pr, short, name, race, checked, waited)
			gaps = append(gaps, l.pr+"'s report prints no reuse counts")
			continue
		}
		fmt.Fprintf(w, "| %s | %s | %s | %d of %d | %d of %d | %s | %s | %s |\n", l.pr, short, name, r.unitReused, r.unitTotal, r.raceReused, r.raceTotal, race, checked, waited)
		unitR, unitT, raceR, raceT = unitR+r.unitReused, unitT+r.unitTotal, raceR+r.raceReused, raceT+r.raceTotal
		if r.timed && r.race > 0 {
			checks, races = append(checks, r.checked), append(races, r.race)
		} else if r.timed {
			gaps = append(gaps, l.pr+"'s report prints no race stage time")
		}
	}
	fmt.Fprintln(w)
	if len(unread) > 0 {
		fmt.Fprintf(w, "Not read: %s.\n", strings.Join(unread, "; "))
	}
	if unitT+raceT > 0 {
		fmt.Fprintf(w, "Reused, pooled over the landings that ran test stages: unit %s, race %s, both %s.\n",
			share(unitR, unitT), share(raceR, raceT), share(unitR+raceR, unitT+raceT))
	}
	if len(checks) > 0 {
		fmt.Fprintf(w, "Check time of the %d timed landings that ran test stages: median %s, race stage median %s.\n",
			len(checks), clock(median(checks)), clock(median(races)))
	}
	fmt.Fprintf(w, "Baseline (%s, %s): 0 of %d packages reused, and no time recorded.\n", baselineDate, baselineIssue, baselinePackages)
	if base == nil {
		gaps = append(gaps, "no timed run that reused nothing was given (-baseline, the log of make preflight PREFLIGHT_FRESH=1)")
	} else {
		fmt.Fprintf(w, "Timed run that reused nothing: %s, head %s, reused 0 of %d packages: race stage %s, checked in %s.\n",
			filepath.Base(base.path), base.head, base.unitTotal+base.raceTotal, clock(base.race), clock(base.checked))
	}
	if len(checks) == 0 {
		gaps = append(gaps, "no landing here ran test stages with a printed time")
	}
	fmt.Fprintln(w)
	if len(gaps) > 0 {
		fmt.Fprintf(w, "No saving is stated: %s.\n", strings.Join(gaps, "; "))
		return false
	}
	c, r := median(checks), median(races)
	fmt.Fprintf(w, "Saving: the median check of the %d landings that ran test stages took %s against the cold run's %s, %s; its race stage %s against %s, %s.\n",
		len(checks), clock(c), clock(base.checked), less(base.checked, c), clock(r), clock(base.race), less(base.race, r))
	return true
}

// less says how much shorter got is than want, or longer.
func less(want, got time.Duration) string {
	if got > want {
		return clock(got-want) + " more"
	}
	return clock(want-got) + " less"
}

// share is reused of total with its percentage to one place.
func share(reused, total int) string {
	if total == 0 {
		return "none"
	}
	return fmt.Sprintf("%d of %d (%.1f%%)", reused, total, 100*float64(reused)/float64(total))
}

// median is the middle duration, or the mean of the middle two to the second.
func median(d []time.Duration) time.Duration {
	if len(d) == 0 {
		return 0
	}
	s := slices.Clone(d)
	slices.Sort(s)
	mid := len(s) / 2
	if len(s)%2 == 1 {
		return s[mid]
	}
	return ((s[mid-1] + s[mid]) / 2).Round(time.Second)
}

// clock prints whole seconds as the preflight does: 42s, 3m12s, 41m07s, 1h02m03s.
func clock(d time.Duration) string {
	s := int(d.Round(time.Second) / time.Second)
	h, m := s/3600, s%3600/60
	s %= 60
	switch {
	case h > 0:
		return fmt.Sprintf("%dh%02dm%02ds", h, m, s)
	case m > 0:
		return fmt.Sprintf("%dm%02ds", m, s)
	}
	return fmt.Sprintf("%ds", s)
}
