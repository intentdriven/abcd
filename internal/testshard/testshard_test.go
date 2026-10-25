package testshard

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// selfTestEnv runs this package's own binary as a package that opts in, its
// probes recording under the directory it names which process ran them.
const selfTestEnv = "ABCD_TESTSHARD_SELFTEST"

// failEnv makes the failing probe fail.
const failEnv = "ABCD_TESTSHARD_FAIL"

func TestMain(m *testing.M) {
	if os.Getenv(selfTestEnv) != "" {
		if code, ok := Shard(); ok {
			os.Exit(code)
		}
	}
	os.Exit(m.Run())
}

// probes are the tests the self-test deals out.
var probes = []string{"TestProbeA", "TestProbeB", "TestProbeC", "TestProbeD", "TestProbeE", "TestProbeFail"}

// probe records, in self-test mode, that this process ran the test.
func probe(t *testing.T) {
	dir := os.Getenv(selfTestEnv)
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, t.Name()), []byte(strconv.Itoa(os.Getpid())), 0o600); err != nil {
		t.Fatal(err)
	}
	if t.Name() == "TestProbeFail" && os.Getenv(failEnv) != "" {
		t.Fatal("the probe was asked to fail")
	}
}

func TestProbeA(t *testing.T)    { probe(t) }
func TestProbeB(t *testing.T)    { probe(t) }
func TestProbeC(t *testing.T)    { probe(t) }
func TestProbeD(t *testing.T)    { probe(t) }
func TestProbeE(t *testing.T)    { probe(t) }
func TestProbeFail(t *testing.T) { probe(t) }

// selfTest runs this binary as a package that opts in, over three shards,
// with go test's own flags, and returns its output, its exit code, the pid
// each test ran in, and the test log go test would read, failing t when a
// shard's own log is left behind.
func selfTest(t *testing.T, fail bool) (string, int, map[string]string, string) {
	t.Helper()
	dir := t.TempDir()
	logfile := filepath.Join(t.TempDir(), "testlog.txt")
	cmd := exec.Command(os.Args[0], "-test.paniconexit0", "-test.timeout=2m0s", "-test.testlogfile="+logfile)
	cmd.Env = append(os.Environ(), selfTestEnv+"="+dir, ShardsEnv+"=3")
	if fail {
		cmd.Env = append(cmd.Env, failEnv+"=1")
	}
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		code = ee.ExitCode()
	}
	ran := map[string]string{}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		pid, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		ran[e.Name()] = string(pid)
	}
	if left, _ := filepath.Glob(logfile + ".shard*"); len(left) != 0 {
		t.Errorf("shard logs left behind: %v", left)
	}
	log, _ := os.ReadFile(logfile)
	return string(out), code, ran, string(log)
}

// TestEveryTestRunsOnceAcrossTheShards: with the shards asked for, every test
// of the package runs, each once, in more than one process, the run passes,
// and go test's test log is one log, merged from every shard's, with no
// shard's own left behind (iss-2610251004378066).
func TestEveryTestRunsOnceAcrossTheShards(t *testing.T) {
	if os.Getenv(selfTestEnv) != "" {
		return
	}
	out, code, ran, log := selfTest(t, false)
	if code != 0 {
		t.Fatalf("exit %d; want 0:\n%s", code, out)
	}
	for _, p := range probes {
		if _, ok := ran[p]; !ok {
			t.Errorf("%s did not run:\n%s", p, out)
		}
	}
	pids := map[string]bool{}
	for _, pid := range ran {
		pids[pid] = true
	}
	if len(pids) != 3 {
		t.Errorf("the probes ran in %d processes; want the 3 shards asked for: %v", len(pids), ran)
	}
	if n := strings.Count(out, "=== testshard: shard "); n != 3 {
		t.Errorf("the output names %d shards; want 3:\n%s", n, out)
	}
	if !strings.HasPrefix(log, testlogMagic) || strings.Count(log, testlogMagic) != 1 || strings.Count(log, "\nchdir ") != 3 {
		t.Errorf("the test log is not one log of three shards:\n%s", log)
	}
}

// TestAFailingShardFailsTheRun: one test failing in one shard fails the run,
// and its failure is in the output.
func TestAFailingShardFailsTheRun(t *testing.T) {
	if os.Getenv(selfTestEnv) != "" {
		return
	}
	out, code, ran, _ := selfTest(t, true)
	if code == 0 {
		t.Fatalf("exit 0 with a failing test; want a failure:\n%s", out)
	}
	if !strings.Contains(out, "--- FAIL: TestProbeFail") || !regexp.MustCompile(`testshard: shard \d of 3 failed`).MatchString(out) {
		t.Errorf("the output does not carry the failure and name its shard:\n%s", out)
	}
	if len(ran) != len(probes) {
		t.Errorf("ran %d probes; want every one, the failure stopping only its own: %v", len(ran), ran)
	}
}

// TestShardsOnlyForTheWholePackage: a count of one or less, a shard's own
// process, a -run of the caller's own, a profile and -json's stream each run
// the package as one process.
func TestShardsOnlyForTheWholePackage(t *testing.T) {
	env := func(kv ...string) func(string) string {
		return func(k string) string {
			for i := 0; i+1 < len(kv); i += 2 {
				if kv[i] == k {
					return kv[i+1]
				}
			}
			return ""
		}
	}
	goTest := []string{"-test.paniconexit0", "-test.timeout=40m0s", "-test.testlogfile=/x/testlog.txt"}
	for _, c := range []struct {
		name   string
		getenv func(string) string
		args   []string
		want   int
	}{
		{"asked for three", env(ShardsEnv, "3"), goTest, 3},
		{"verbose", env(ShardsEnv, "3"), append(slices.Clone(goTest), "-test.v=true", "-test.count=1"), 3},
		{"unset", env(), goTest, 0},
		{"one", env(ShardsEnv, "1"), goTest, 0},
		{"not a number", env(ShardsEnv, "many"), goTest, 0},
		{"a shard's own process", env(ShardsEnv, "3", childEnv, "1"), goTest, 0},
		{"a -run of its own", env(ShardsEnv, "3"), append(slices.Clone(goTest), "-test.run=^TestX$"), 0},
		{"a profile", env(ShardsEnv, "3"), append(slices.Clone(goTest), "-test.coverprofile=c.out"), 0},
		{"-json", env(ShardsEnv, "3"), append(slices.Clone(goTest), "-test.v=test2json"), 0},
	} {
		n, ok := shards(c.getenv, c.args)
		if !ok {
			n = 0
		}
		if n != c.want {
			t.Errorf("%s: shards %d; want %d", c.name, n, c.want)
		}
	}
}

// TestDealMatchesEachNameInExactlyOneShare: every name is matched by exactly
// one share's pattern, and by none as a prefix of a longer name.
func TestDealMatchesEachNameInExactlyOneShare(t *testing.T) {
	names := []string{"TestA", "TestAB", "TestB", "TestC.x", "ExampleA", "FuzzA", "TestD"}
	pats := deal(names, 3)
	if len(pats) != 3 {
		t.Fatalf("%d shares; want 3", len(pats))
	}
	for _, name := range append(slices.Clone(names), "TestABC", "TestCxx") {
		matched := 0
		for _, p := range pats {
			if regexp.MustCompile(p).MatchString(name) {
				matched++
			}
		}
		want := 1
		if !slices.Contains(names, name) {
			want = 0
		}
		if matched != want {
			t.Errorf("%s is matched by %d shares; want %d (%q)", name, matched, want, pats)
		}
	}
	if got := deal([]string{"TestA"}, 4); len(got) != 1 {
		t.Errorf("one name over four shares made %d shares; want 1", len(got))
	}
}
