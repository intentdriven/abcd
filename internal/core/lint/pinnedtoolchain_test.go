package lint_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// The declared-toolchain resolver, scripts/pinned-toolchain.sh, driven against a
// stub `go` so every branch runs without a network or a second toolchain on
// the machine. The stub answers the two queries the resolver makes of the go on
// PATH (`env GOVERSION`, `env GOROOT`); the fake GOROOT it reports holds a gofmt
// and a go whose `version` names the declared release.

const resolverStubGo = `#!/bin/sh
case "$1 $2" in
"env GOVERSION") echo go1.27.1 ;;
"env GOROOT")
	[ -n "$STUB_PROGRESS" ] && echo "$STUB_PROGRESS" >&2
	if [ -n "$STUB_FAIL" ]; then echo "$STUB_FAIL" >&2; exit 1; fi
	echo "$STUB_GOROOT"
	;;
*) echo "stub go: unexpected arguments: $*" >&2; exit 3 ;;
esac
`

type resolverRun struct {
	stdout, stderr string
	code           int
}

// runResolver runs the resolver with the stub go first on PATH. reported is the
// release the fake GOROOT's go claims to be.
func runResolver(t *testing.T, version, reported string, env ...string) (resolverRun, string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the resolver is a POSIX shell script; the gates run on macOS and Linux")
	}
	root := filepath.Join("..", "..", "..")
	script, err := filepath.Abs(filepath.Join(root, "scripts", "pinned-toolchain.sh"))
	if err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	stubBin := filepath.Join(dir, "stub")
	goroot := filepath.Join(dir, "goroot")
	for _, d := range []string{stubBin, filepath.Join(goroot, "bin")} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	writeExec(t, filepath.Join(stubBin, "go"), resolverStubGo)
	writeExec(t, filepath.Join(goroot, "bin", "gofmt"), "#!/bin/sh\nexit 0\n")
	writeExec(t, filepath.Join(goroot, "bin", "go"), "#!/bin/sh\necho \"go version "+reported+" stub/arch\"\n")

	cmd := exec.Command("bash", script, version)
	cmd.Env = append([]string{
		"PATH=" + stubBin + string(os.PathListSeparator) + "/usr/bin" + string(os.PathListSeparator) + "/bin",
		"STUB_GOROOT=" + goroot,
		"HOME=" + dir,
	}, env...)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	code := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running the resolver: %v", err)
		}
		code = ee.ExitCode()
	}
	return resolverRun{out.String(), errb.String(), code}, goroot
}

func writeExec(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

// TestPinnedToolchainResolverIgnoresDownloadProgress is iss-2609090951287096's
// detector. On the first run on a machine without the declared toolchain
// cached, `go env GOROOT` prints the download notice on stderr before it
// reports the root on stdout. The format gate used to capture both streams as
// the root, so the path became two lines, the executable test on it failed,
// and the gate refused — saying the fetch needed network, on the run where the
// fetch had just succeeded.
func TestPinnedToolchainResolverIgnoresDownloadProgress(t *testing.T) {
	got, goroot := runResolver(t, "1.26.7", "go1.26.7",
		"STUB_PROGRESS=go: downloading go1.26.7 (darwin/arm64)")
	if got.code != 0 {
		t.Fatalf("the resolver refused a fetch that succeeded and merely printed progress (exit %d).\n"+
			"stderr:\n%s", got.code, got.stderr)
	}
	if strings.TrimSuffix(got.stdout, "\n") != goroot {
		t.Fatalf("the resolver printed %q, want exactly the GOROOT %q: stdout is the value a caller runs "+
			"binaries from, so anything else on it corrupts the path", got.stdout, goroot)
	}
}

// A fetch that FAILS refuses, exit 2, naming the declared release and the one
// on PATH — the skew — and carrying go's own error, and never prints a root a
// caller could run a fallback from.
func TestPinnedToolchainResolverRefusesAFailedFetchNamingTheSkew(t *testing.T) {
	got, _ := runResolver(t, "1.26.7", "go1.26.7",
		"STUB_FAIL=go: download go1.26.7: dial tcp: lookup proxy.golang.org: no such host")
	if got.code != 2 {
		t.Fatalf("a failed fetch exited %d, want the refusal's 2.\nstderr:\n%s", got.code, got.stderr)
	}
	if got.stdout != "" {
		t.Errorf("a refusal printed %q on stdout; a caller would take it for a GOROOT", got.stdout)
	}
	for _, want := range []string{"REFUSING", "go1.26.7", "go1.27.1", "no such host"} {
		if !strings.Contains(got.stderr, want) {
			t.Errorf("the refusal does not name %q:\n%s", want, got.stderr)
		}
	}
}

// A switch that silently did not happen — the resolved go reports another
// release — refuses rather than judging the tree with it.
func TestPinnedToolchainResolverRefusesAToolchainThatDidNotSwitch(t *testing.T) {
	got, _ := runResolver(t, "1.26.7", "go1.27.1")
	if got.code != 2 || !strings.Contains(got.stderr, "reports go1.27.1") {
		t.Fatalf("a resolved toolchain reporting the wrong release exited %d, want 2 with the mismatch named.\n"+
			"stderr:\n%s", got.code, got.stderr)
	}
}

// go.mod without a go directive leaves the caller an empty version; that
// refuses before any go runs.
func TestPinnedToolchainResolverRefusesAnEmptyDeclaration(t *testing.T) {
	got, _ := runResolver(t, "", "go1.26.7")
	if got.code != 2 || !strings.Contains(got.stderr, "declares no") {
		t.Fatalf("an empty declaration exited %d, want 2 naming the missing go line.\nstderr:\n%s",
			got.code, got.stderr)
	}
}

// TestPreflightRunsTheDeclaredToolchain holds every Go step of `make preflight`
// to the toolchain go.mod declares (iss-2609261850045839). CI's setup-go
// installs that release and every Go step runs on it, while preflight ran the
// go on PATH — so on a newer machine a test asserting standard-library wording
// passed preflight and failed CI (pull request 728). Preflight exports
// GOTOOLCHAIN from the same go.mod read the format gate uses, and runs the
// format gate — whose resolver refuses, naming the skew, when the toolchain
// cannot be fetched — before any other gate, so nothing is judged on a
// toolchain that was never resolved. One resolver, not a second: the Makefile
// resolves a GOROOT nowhere but through the script.
func TestPreflightRunsTheDeclaredToolchain(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	makefile := readRepoFile(t, root, "Makefile")
	workflow := readRepoFile(t, root, ".github/workflows/ci.yml")

	const export = "preflight: export GOTOOLCHAIN := go$(GO_TOOLCHAIN_VERSION)"
	if !strings.Contains(makefile, export) {
		t.Errorf("the Makefile does not declare %q.\n\n"+
			"Without it preflight builds, vets and tests on the go on PATH while CI uses the "+
			"release go.mod declares, and a green preflight vouches for nothing CI's toolchain "+
			"would say differently.", export)
	}
	if n := strings.Count(makefile, "go env GOROOT"); n != 0 {
		t.Errorf("the Makefile resolves a GOROOT itself (%d `go env GOROOT`); the resolver is "+
			"scripts/pinned-toolchain.sh alone, or the format gate and the Go steps can come to "+
			"disagree about which toolchain is declared", n)
	}

	// The format gate CI runs, derived from CI's own step as the format test does.
	step, ok := workflowStepBlock(workflow, formatStepName)
	if !ok {
		t.Fatalf(".github/workflows/ci.yml defines no %q step", formatStepName)
	}
	m := regexp.MustCompile(`\bmake ([a-z][a-z-]*)\b`).FindStringSubmatch(step)
	if m == nil {
		t.Fatalf("the %q step runs no `make <target>`", formatStepName)
	}
	format := m[1]
	recipe, ok := makeRecipe(makefile, format)
	if !ok || !strings.Contains(recipe, "scripts/pinned-toolchain.sh") {
		t.Errorf("the `%s:` recipe does not resolve its toolchain through scripts/pinned-toolchain.sh:\n\n%s",
			format, recipe)
	}
	prereqs := preflightPrereqs(t, root)
	if len(prereqs) < 2 || prereqs[1] != format {
		t.Errorf("preflight's prerequisites are %v; the format gate %q must come second, straight "+
			"after the load check (a warning that judges nothing): it is CI's gate too, and its "+
			"resolver is what refuses, naming the skew, before any gate runs on a toolchain that "+
			"could not be fetched", prereqs, format)
	}
}

// TestPreflightArmsRecordLintAsCIDoes holds preflight's record-lint to the range
// CI's step arms it with (iss-2609021152026246). CI passes `-agent-diff
// <base>...HEAD`, which arms agent_contract's unbumped-edit check; preflight
// passed nothing, so a changed agent prompt that bumped no prompt_version
// passed three green preflights and was refused in the merge queue (pull
// request 606).
func TestPreflightArmsRecordLintAsCIDoes(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	makefile := readRepoFile(t, root, "Makefile")
	workflow := readRepoFile(t, root, ".github/workflows/ci.yml")

	ciStep, ok := workflowStepBlock(workflow, "Record-lint (drift gate)")
	if !ok {
		t.Fatal(".github/workflows/ci.yml defines no `Record-lint (drift gate)` step")
	}
	if !strings.Contains(ciStep, `-agent-diff "${BASE_SHA}...HEAD"`) {
		t.Fatalf("CI's record-lint step no longer arms `-agent-diff \"${BASE_SHA}...HEAD\"`; "+
			"re-derive what preflight must match:\n\n%s", ciStep)
	}
	lintRecipe, ok := makeRecipe(makefile, "record-lint")
	if !ok {
		t.Fatal("the Makefile declares no `record-lint:` target")
	}
	if !strings.Contains(lintRecipe, "go run ./cmd/record-lint -agent-diff origin/main...HEAD") {
		t.Errorf("the `record-lint:` recipe does not arm agent_contract over origin/main...HEAD, "+
			"the merge-base range CI's step passes as ${BASE_SHA}...HEAD:\n\n%s\n\n"+
			"Unarmed, the unbumped-prompt check is a no-op locally and fires first in the merge "+
			"queue.", lintRecipe)
	}
}
