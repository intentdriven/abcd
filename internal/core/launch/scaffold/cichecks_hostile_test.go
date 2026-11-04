package scaffold

import (
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

// hostileBytes is how large the attacks that grow with the file are written: all the
// read accepts, less a little for what surrounds each.
const hostileBytes = maxWorkflowBytes - 4<<10

// hostileAllocPerByte bounds what one read may allocate for each byte of the file:
// it is linear, and building the parsed tree is its largest part, about 260 bytes a
// byte on the worst shape measured (a flow mapping of one-letter keys).
const hostileAllocPerByte = 400

// hostileReadSeconds is a hard stop on one read: a linear read takes well under a
// second even on a loaded machine, and one that is not would otherwise hold the test
// until its timeout.
const hostileReadSeconds = 60

// hostileWorkflows are files written to cost far more to read than their size, each
// within the size the read accepts, with whether the read refuses it.
func hostileWorkflows() map[string]struct {
	src     string
	refused bool
} {
	const good = "on: pull_request\njobs:\n  build:\n    runs-on: x\n"
	lines := func(n int, format string) string {
		var b strings.Builder
		for i := range n {
			fmt.Fprintf(&b, format, i)
		}
		return b.String()
	}
	repeat := func(item string, room int) string { return strings.Repeat(item, room/len(item)) }
	type file = struct {
		src     string
		refused bool
	}
	// Many jobs that alias one thing, the way that costs most for each alias.
	aliased := func(body string, job string) string {
		return body + lines(hostileBytes/2/len(fmt.Sprintf(job, 0)), job)
	}
	files := map[string]file{
		"a job body behind many aliases": {refused: true, src: aliased(
			"on: pull_request\njobs:\n  a: &x\n"+lines(hostileBytes/2/13, "    k%06d:\n"), "  j%06d: *x\n")},
		"one key repeated at the top": {refused: true, src: "{" + repeat("a,", hostileBytes) + "a}"},
		"one key repeated in a job": {refused: true, src: "on: pull_request\njobs:\n  a: {" +
			repeat("k: 1,", hostileBytes) + "k: 1}\n"},
		"one key repeated where the read does not look": {refused: true, src: good +
			"    env: {" + repeat("k: 1,", hostileBytes) + "k: 1}\n"},
		"a matrix list behind many aliases": {refused: true, src: aliased(
			"on: pull_request\njobs:\n  a: &b\n    strategy:\n      matrix:\n        k: ["+
				repeat("a,", hostileBytes/2)+"a]\n", "  j%06d: *b\n")},
		"a strategy behind many aliases": {refused: true, src: aliased(
			"x-s: &s\n  matrix:\n    k: ["+repeat("a,", hostileBytes/2)+"a]\non: pull_request\njobs:\n",
			"  j%06d: {strategy: *s}\n")},
		"a needs list behind many aliases": {refused: true, src: aliased(
			"x-n: &n ["+repeat("a,", hostileBytes/2)+"a]\non: pull_request\njobs:\n  a: {}\n",
			"  j%06d: {needs: *n}\n")},
		"more nodes than the read takes": {refused: true, src: "x: [" +
			strings.Repeat("a,", maxWorkflowNodes) + "a]\n" + good},
		"more depth than the read takes": {refused: true, src: "x: " +
			strings.Repeat("[", maxWorkflowDepth+1) + strings.Repeat("]", maxWorkflowDepth+1) + "\n" + good},
		"merge keys in a job": {refused: true, src: "x-m: &m {a: 1}\non: pull_request\njobs:\n  a:\n    <<: [" +
			repeat("*m,", hostileBytes) + "*m]\n"},
		"an alias inside what it names": {src: "x-s: &s [*s]\non: pull_request\njobs:\n  build:\n    runs-on: x\n" +
			"  loop:\n    strategy: &t {matrix: *t}\n"},
		"aliases of aliases where the read does not look": {src: func() string {
			var b strings.Builder
			b.WriteString("x-0: &a0 [l,l,l,l,l,l,l,l,l,l]\n")
			for i := 1; i <= 30; i++ {
				fmt.Fprintf(&b, "x-%d: &a%d [%s*a%d]\n", i, i, strings.Repeat(fmt.Sprintf("*a%d,", i-1), 9), i-1)
			}
			return b.String() + good
		}()},
	}
	// Each anchor followed no more than the limit allows, so that only the count of
	// everything the read visits stops it.
	var many strings.Builder
	const anchors, items = 40, 2000
	for a := range anchors {
		fmt.Fprintf(&many, "x-%d: &a%d [%sa]\n", a, a, strings.Repeat("a,", items))
	}
	many.WriteString("on: pull_request\njobs:\n  a: {}\n")
	for a := range anchors {
		for j := range maxAliasFollows {
			fmt.Fprintf(&many, "  j%d-%d: {needs: *a%d}\n", a, j, a)
		}
	}
	files["many anchors, each within its limit"] = file{refused: true, src: many.String()}
	return files
}

// TestAHostileWorkflowIsReadInBoundedWork: whatever a file holds, reading it costs
// work in proportion to its size, and a file past the read's limits is refused, loudly
// and with nothing proposed, as an unreadable one is. Both readers are held. The cost
// is measured in bytes allocated, with a hard stop on time.
func TestAHostileWorkflowIsReadInBoundedWork(t *testing.T) {
	for name, c := range hostileWorkflows() {
		t.Run(name, func(t *testing.T) {
			if len(c.src) > maxWorkflowBytes {
				t.Fatalf("the fixture is %d bytes, past the %d the read accepts, so it would test only that", len(c.src), maxWorkflowBytes)
			}
			repo := writeWorkflows(t, map[string]string{"ci.yml": "on: pull_request\njobs:\n  fine:\n    runs-on: x\n", "hostile.yml": c.src})
			limit := uint64(len(c.src))*hostileAllocPerByte + 4<<20

			var checks []string
			var err error
			bounded(t, limit, func() { checks, _, err = RequiredChecks(repo, "main", false) })
			if c.refused {
				if err == nil {
					t.Fatalf("the read went ahead, with checks %q", checks)
				}
				if len(checks) != 0 {
					t.Errorf("checks = %q; a refused read proposes none", checks)
				}
				if !strings.Contains(err.Error(), "hostile.yml") {
					t.Errorf("the refusal does not name the file: %v", err)
				}
				if strings.Contains(err.Error(), "cannot be read safely") {
					t.Errorf("the file was refused for its size, not what it holds: %v", err)
				}
			} else if err != nil {
				t.Errorf("the read was refused: %v", err)
			} else if !slices.Contains(checks, "fine") {
				t.Errorf("checks = %q, which leaves out the other file's job", checks)
			}
			bounded(t, limit, func() { DeriveCIChecks(repo) })
		})
	}
}

// bounded runs read, failing the test if it allocates past limit bytes or outlasts
// the hard stop.
func bounded(t *testing.T, limit uint64, read func()) {
	t.Helper()
	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	done := make(chan struct{})
	go func() {
		defer close(done)
		read()
	}()
	select {
	case <-done:
	case <-time.After(hostileReadSeconds * time.Second):
		t.Fatalf("the read was still going after %d seconds", hostileReadSeconds)
	}
	runtime.ReadMemStats(&after)
	got := after.TotalAlloc - before.TotalAlloc
	t.Logf("allocated %d KiB", got>>10)
	if got > limit {
		t.Errorf("the read allocated %d MiB, past its %d MiB bound", got>>20, limit>>20)
	}
}

// TestRequiredChecksReadsTheTriggerFromTheParsedFile: a trigger reads the same however
// its YAML is laid out, so a workflow is judged by what GitHub reads, whichever column
// its keys start in and whatever its text says that looks like a key. One with no
// `on:` at all is named rather than dropped.
func TestRequiredChecksReadsTheTriggerFromTheParsedFile(t *testing.T) {
	repo := writeWorkflows(t, map[string]string{
		"json.yml":     `{"on": "pull_request", "jobs": {"json": {"runs-on": "x"}}}`,
		"bom.yml":      "\ufeffon: pull_request\njobs:\n  bom:\n    runs-on: x\n",
		"anchored.yml": "&t on: pull_request\njobs:\n  anchored:\n    runs-on: x\n",
		"aliased.yml":  "name: &e pull_request\non: *e\njobs:\n  aliased:\n    runs-on: x\n",
		"indented.yml": "  on: pull_request\n  jobs:\n    indented:\n      runs-on: x\n",
		"decoy.yml":    "name: \"a\n\n  on: push\n\n  b\"\non: pull_request\njobs:\n  decoy:\n    runs-on: x\n",
		"push.yml":     "\ufeffon: push\njobs:\n  not-a-gate:\n    runs-on: x\n",
		"no-on.yml":    "jobs:\n  no-on:\n    runs-on: x\n",
	})
	checks, notes := requiredChecksOf(t, repo, false)
	if want := []string{"aliased", "anchored", "bom", "decoy", "indented", "json"}; !slices.Equal(checks, want) {
		t.Errorf("checks = %q, want %q; notes %q", checks, want, notes)
	}
	if !strings.Contains(strings.Join(notes, "\n"), "no-on.yml") {
		t.Errorf("the notes do not name the file with no trigger: %q", notes)
	}
}

// TestRequiredChecksRefusesAKeyDefinedTwice: GitHub refuses a workflow that defines a
// key twice in any mapping, so the read does too, wherever in the file it is.
func TestRequiredChecksRefusesAKeyDefinedTwice(t *testing.T) {
	for name, body := range map[string]string{
		"on":        "on: pull_request\non: push\njobs:\n  a:\n    runs-on: x\n",
		"event":     "on:\n  pull_request:\n  pull_request:\njobs:\n  a:\n    runs-on: x\n",
		"filter":    "on:\n  pull_request:\n    types: [opened]\n    types: [opened, synchronize, reopened]\njobs:\n  a:\n    runs-on: x\n",
		"job":       "on: pull_request\njobs:\n  a:\n    runs-on: x\n  a:\n    runs-on: y\n",
		"job's key": "on: pull_request\njobs:\n  a:\n    if: always()\n    if: false\n    runs-on: x\n",
		"matrix":    "on: pull_request\njobs:\n  a:\n    strategy:\n      matrix:\n        os: [a]\n        os: [b]\n",
		"a step's":  "on: pull_request\njobs:\n  a:\n    runs-on: x\n    steps:\n      - run: a\n        run: b\n",
		"an unused": "x: {a: 1, a: 2}\non: pull_request\njobs:\n  a:\n    runs-on: x\n",
	} {
		t.Run(name, func(t *testing.T) {
			if checks, _, err := RequiredChecks(writeWorkflows(t, map[string]string{"twice.yml": body}), "main", false); err == nil {
				t.Errorf("the read went ahead, with checks %q", checks)
			} else if !strings.Contains(err.Error(), "twice.yml") {
				t.Errorf("the refusal does not name the file: %v", err)
			}
		})
	}
}

// TestRequiredChecksEncodesWhatAFileCarries: a file's name and what the parser says
// about it reach a JSON surface as well as the terminal, so neither carries a rune a
// reader cannot see, on a note or on a refusal.
func TestRequiredChecksEncodesWhatAFileCarries(t *testing.T) {
	const hidden = "\u202e"
	name := "a" + hidden + "b.yml"
	repo := writeWorkflows(t, map[string]string{name: "on:\n  pull_request:\n    paths: [x]\njobs:\n  a:\n    runs-on: x\n"})
	_, notes := requiredChecksOf(t, repo, false)
	if joined := strings.Join(notes, "\n"); !strings.Contains(joined, "b.yml") || strings.Contains(joined, hidden) {
		t.Errorf("the note should name the file with its hidden rune encoded: %q", joined)
	}

	for body, why := range map[string]string{
		"on: [\n":              "a file that does not parse",
		"\"\u009b2J\u202e\"\n": "a file that is one hidden-rune scalar",
	} {
		repo := writeWorkflows(t, map[string]string{name: body})
		_, _, err := RequiredChecks(repo, "main", false)
		if err == nil {
			t.Fatalf("%s: the read went ahead", why)
		}
		if strings.ContainsAny(err.Error(), "\u202e\u009b") {
			t.Errorf("%s: the refusal carries a hidden rune: %q", why, err)
		}
		if !strings.Contains(err.Error(), "b.yml") {
			t.Errorf("%s: the refusal does not name the file: %q", why, err)
		}
	}
}

// TestALargeOrdinaryWorkflowIsRead: the limits leave a large real workflow alone: a
// thousand jobs, an `if:` and a `needs:` shared through anchors within their limit, and
// a matrix as long as GitHub allows.
func TestALargeOrdinaryWorkflowIsRead(t *testing.T) {
	var b strings.Builder
	b.WriteString("x-if: &cond ${{ always() }}\nx-needs: &gates [first]\non: pull_request\njobs:\n  first:\n    runs-on: x\n")
	want := []string{"first"}
	for i := range 1000 {
		fmt.Fprintf(&b, "  job-%d:\n    runs-on: x\n    steps:\n      - run: make\n", i)
		want = append(want, fmt.Sprintf("job-%d", i))
		if i < maxAliasFollows {
			b.WriteString("    if: *cond\n    needs: *gates\n")
		}
	}
	b.WriteString("  wide:\n    strategy:\n      matrix:\n        n: [")
	for i := range maxMatrixValues {
		fmt.Fprintf(&b, "%d,", i)
		want = append(want, fmt.Sprintf("wide (%d)", i))
	}
	b.WriteString("]\n")
	checks, notes := requiredChecksOf(t, writeWorkflows(t, map[string]string{"ci.yml": b.String()}), false)
	slices.Sort(want)
	if !slices.Equal(checks, want) {
		t.Errorf("read %d checks, want %d; notes %q", len(checks), len(want), notes)
	}

	// One value past GitHub's limit is a matrix abcd does not expand.
	over := strings.Replace(b.String(), "n: [", "n: [over,", 1)
	_, notes = requiredChecksOf(t, writeWorkflows(t, map[string]string{"ci.yml": over}), false)
	if !strings.Contains(strings.Join(notes, "\n"), "job wide ") {
		t.Errorf("a matrix past the limit is not named: %q", notes)
	}
}

// FuzzReadWorkflow: whatever the file holds, the read answers, and a job it reports
// has an id GitHub accepts.
func FuzzReadWorkflow(f *testing.F) {
	f.Add("on: pull_request\njobs:\n  a:\n    needs: [b]\n    if: always()\n    strategy:\n      matrix:\n        os: [x, y]\n  b: {}\n")
	f.Add("x: &a [*a]\non: &o {pull_request: *o}\njobs: &j {a: *j, <<: *j}\n")
	f.Add("? [a, b]\n: c\n*x: 1\n")
	f.Add("\ufeff%YAML 1.1\n---\non: !!binary x\n--- \njobs: !t\n")
	f.Fuzz(func(t *testing.T, src string) {
		if len(src) > maxWorkflowBytes {
			return
		}
		wf, err := readWorkflow(src)
		if err != nil {
			return
		}
		for _, job := range wf.jobs {
			if !jobIDRe.MatchString(job.id) {
				t.Errorf("read a job with the id %q", job.id)
			}
		}
		if skippable, err := skippableJobs(wf.jobs); err == nil && len(skippable) != len(wf.jobs) {
			t.Errorf("judged %d of %d jobs", len(skippable), len(wf.jobs))
		}
	})
}
