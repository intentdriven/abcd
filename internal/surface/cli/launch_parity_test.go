package cli

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/launch"
	"github.com/intentdriven/abcd/internal/gittest"
)

// parityCLIRepo is shipRenderableRepo shipping its commands too, tagged at
// v0.5.0 with a payload, then moved on: plugin.json changes and a command page
// is added.
func parityCLIRepo(t *testing.T) *gittest.Repo {
	t.Helper()
	r := shipRenderableRepo(t)
	r.Write(".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "CHANGELOG.md", "commands"]}`+"\n")
	r.Write("commands/one.md", "---\nname: one\ndescription: the first command\n---\n# one\n")
	r.Commit("a payload with commands")
	r.Git("tag", "v0.5.0")
	r.Write("commands/two.md", "---\nname: two\ndescription: the second command\n---\n# two\n")
	r.Commit("a second command")
	return r
}

func dryRunJSON(t *testing.T, r *gittest.Repo, args ...string) (launch.DryRunReport, string) {
	t.Helper()
	t.Chdir(r.Root())
	stdout, stderr, err := runCLIPipedStdinSplit(t, "", append([]string{"launch", "--dry-run", "--json"}, args...)...)
	if err != nil {
		t.Fatalf("dry-run %v: %v\n%s\n%s", args, err, stdout, stderr)
	}
	var rep launch.DryRunReport
	if err := json.Unmarshal(stdout, &rep); err != nil {
		t.Fatalf("dry-run JSON: %v\n%s", err, stdout)
	}
	return rep, string(stderr)
}

// TestLaunchDryRunReportsTheParityDiff is AC3 at the preview: every preview
// diffs its payload against the newest release tag's, rendered fresh at that
// tag, and the plain render and the pre-flight report both carry it.
func TestLaunchDryRunReportsTheParityDiff(t *testing.T) {
	r := parityCLIRepo(t)
	rep, _ := dryRunJSON(t, r)
	if rep.Parity == nil {
		t.Fatal("the preview must carry the parity diff")
	}
	if rep.Parity.Baseline != "v0.5.0" || rep.Parity.Source != launch.ParitySourceRenderAtTag {
		t.Fatalf("the baseline must be the newest tag rendered fresh, got %+v", rep.Parity)
	}
	if rep.Parity.Added != 1 || len(rep.Parity.Entries) != 1 || rep.Parity.Entries[0].Path != "commands/two.md" {
		t.Errorf("exactly the added command must be listed, got %+v", rep.Parity.Entries)
	}
	written := readPreflight(t, r.Root(), rep.ReportPath)
	if written.Parity == nil || written.Parity.Added != 1 {
		t.Errorf("the pre-flight report must carry the diff, got %+v", written.Parity)
	}

	plain, err := shipIn(t, r, "launch", "--dry-run")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, plain)
	}
	for _, want := range []string{"parity:", "v0.5.0", "added commands/two.md"} {
		if !strings.Contains(string(plain), want) {
			t.Errorf("the plain preview must say %q:\n%s", want, plain)
		}
	}
}

// TestLaunchDryRunConfiguredBaseline is AC7's configured half: a named
// baseline is used, and a wrong one errors by name rather than reading as a
// first launch.
func TestLaunchDryRunConfiguredBaseline(t *testing.T) {
	r := parityCLIRepo(t)
	rep, _ := dryRunJSON(t, r, "--baseline", "v0.4.0")
	if rep.Parity == nil || rep.Parity.Baseline != "v0.4.0" {
		t.Fatalf("the configured baseline must be used, got %+v", rep.Parity)
	}
	if !strings.Contains(rep.Parity.Note, "no launch payload") || rep.Parity.Added == 0 {
		t.Errorf("v0.4.0 declared no payload, so every path is added and the report says why: %+v", rep.Parity)
	}

	for _, bad := range []string{"v9.9.9", "main"} {
		out, err := shipIn(t, r, "launch", "--dry-run", "--baseline", bad)
		if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), bad) {
			t.Errorf("--baseline %s must exit 2 naming it, got %d: %v\n%s", bad, code, err, out)
		}
	}
}

// fakeReleaseAssets serves one release's assets in memory; no test reaches the
// network.
type fakeReleaseAssets struct{ assets map[string][]byte }

func (f fakeReleaseAssets) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	data, ok := f.assets[name]
	return data, "https://example.com/" + tag + "/" + name, ok, nil
}

// TestLaunchDryRunFetchBaselineReadsTheVerifiedReleaseAsset: --fetch-baseline
// is the explicit ask that lets the preview read the release asset, and every
// fetch is announced where the operator reads it.
func TestLaunchDryRunFetchBaselineReadsTheVerifiedReleaseAsset(t *testing.T) {
	r := parityCLIRepo(t)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		".claude-plugin/plugin.json": `{"name":"abcd","description":"fixture","repository":"` + fixtureRepository + `","version":"0.5.0"}`,
		"CHANGELOG.md":               readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")),
		"commands/one.md":            "---\nname: one\ndescription: the first command\n---\n# one\n",
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	archive := launch.PluginArchiveName("abcd", "0.5.0")
	var origins []string
	orig := newReleaseAssetFetcher
	newReleaseAssetFetcher = func(origin string) (launch.ReleaseAssetFetcher, []string, error) {
		origins = append(origins, origin)
		return fakeReleaseAssets{assets: map[string][]byte{
			"checksums.txt": []byte(hex.EncodeToString(sum[:]) + "  " + archive + "\n"),
			archive:         buf.Bytes(),
		}}, nil, nil
	}
	t.Cleanup(func() { newReleaseAssetFetcher = orig })

	rep, stderr := dryRunJSON(t, r, "--fetch-baseline")
	if len(origins) != 1 || origins[0] != fixtureRepository {
		t.Fatalf("the fetcher must be pinned to the plugin's own repository, got %v", origins)
	}
	if rep.Parity == nil || rep.Parity.Source != launch.ParitySourceReleaseAsset || rep.Parity.Refused {
		t.Fatalf("the baseline must be the verified release asset, got %+v", rep.Parity)
	}
	if rep.Parity.Added != 1 || rep.Parity.Changed+rep.Parity.Removed != 0 {
		t.Errorf("only the added command differs from the published archive, got %+v", rep.Parity.Entries)
	}
	if !strings.Contains(stderr, "fetching") || !strings.Contains(stderr, archive) {
		t.Errorf("every fetch must be announced on stderr, got:\n%s", stderr)
	}

	// Without the flag the preview never constructs a fetcher: disk only.
	origins = nil
	if rep, _ := dryRunJSON(t, r); rep.Parity.Source != launch.ParitySourceRenderAtTag || len(origins) != 0 {
		t.Errorf("without --fetch-baseline the preview must stay disk-only, got %q and %v", rep.Parity.Source, origins)
	}
}

// TestLaunchDryRunTaglessOrShallowCheckoutIsNotAFirstLaunch: a checkout that
// lacks the previous release's tag — a clone made without tags, or a shallow
// one — while its CHANGELOG.md dates a release refuses the parity diff, naming
// the release and the remedy, rather than reading as a first launch that adds
// every path. --fetch-baseline is a real remedy: the verified asset answers. A
// tree whose CHANGELOG.md dates no release stays a first launch
// (iss-2609251902439938).
func TestLaunchDryRunTaglessOrShallowCheckoutIsNotAFirstLaunch(t *testing.T) {
	r := parityCLIRepo(t)
	shallowSrc := r.Root()
	r.Git("tag", "-d", "v0.4.0", "v0.5.0")

	rep, _ := dryRunJSON(t, r)
	if rep.Parity == nil || !rep.Parity.Refused || rep.Parity.Source == launch.ParitySourceNone || rep.Parity.Baseline != "v0.4.0" {
		t.Fatalf("a tagless checkout whose CHANGELOG.md dates 0.4.0 must refuse against v0.4.0, got %+v", rep.Parity)
	}
	for _, want := range []string{"CHANGELOG.md dates release 0.4.0", "git fetch --tags", "--fetch-baseline"} {
		if !strings.Contains(rep.Parity.RefusalReason, want) {
			t.Errorf("the refusal must say %q, got %q", want, rep.Parity.RefusalReason)
		}
	}
	if !containsPrefix(rep.WouldRefuseOn, "payload parity: ") {
		t.Errorf("the refusal must be on the preview's would-refuse list, got %v", rep.WouldRefuseOn)
	}

	// The remedy works: the verified release asset of the named release answers.
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range map[string]string{
		".claude-plugin/plugin.json": `{"name":"abcd","description":"fixture","repository":"` + fixtureRepository + `","version":"0.4.0"}`,
		"CHANGELOG.md":               readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")),
	} {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	archive := launch.PluginArchiveName("abcd", "0.4.0")
	orig := newReleaseAssetFetcher
	newReleaseAssetFetcher = func(string) (launch.ReleaseAssetFetcher, []string, error) {
		return fakeReleaseAssets{assets: map[string][]byte{
			"checksums.txt": []byte(hex.EncodeToString(sum[:]) + "  " + archive + "\n"),
			archive:         buf.Bytes(),
		}}, nil, nil
	}
	t.Cleanup(func() { newReleaseAssetFetcher = orig })
	if rep, _ := dryRunJSON(t, r, "--fetch-baseline"); rep.Parity == nil || rep.Parity.Refused || rep.Parity.Source != launch.ParitySourceReleaseAsset {
		t.Errorf("with --fetch-baseline the verified asset of v0.4.0 is the baseline, got %+v", rep.Parity)
	}

	// A shallow clone refuses too, even when the listing it holds names a tag:
	// the listing holds only what was fetched.
	src := gittest.NewRepo(t)
	dst := filepath.Join(t.TempDir(), "shallow")
	clone := exec.Command("git", "clone", "--quiet", "--depth", "2", "file://"+shallowSrc, dst)
	clone.Env = src.Env()
	if out, err := clone.CombinedOutput(); err != nil {
		t.Fatalf("shallow clone: %v\n%s", err, out)
	}
	t.Chdir(dst)
	stdout, stderr, err := runCLIPipedStdinSplit(t, "", "launch", "--dry-run", "--json")
	if err != nil {
		t.Fatalf("dry-run in the shallow clone: %v\n%s\n%s", err, stdout, stderr)
	}
	var shallow launch.DryRunReport
	if err := json.Unmarshal(stdout, &shallow); err != nil {
		t.Fatalf("dry-run JSON: %v\n%s", err, stdout)
	}
	if shallow.Parity == nil || !shallow.Parity.Refused || !strings.Contains(shallow.Parity.RefusalReason, "shallow") {
		t.Errorf("a shallow clone must refuse the parity diff by saying so, got %+v", shallow.Parity)
	}

	// A tree whose CHANGELOG.md dates no release is a first launch.
	first := parityCLIRepo(t)
	first.Git("tag", "-d", "v0.4.0", "v0.5.0")
	first.Write("CHANGELOG.md", "# Changelog\n\n## [Unreleased]\n")
	first.Commit("no release dated yet")
	if rep, _ := dryRunJSON(t, first); rep.Parity == nil || rep.Parity.Refused || rep.Parity.Source != launch.ParitySourceNone {
		t.Errorf("no dated release and no tag is a first launch, got %+v", rep.Parity)
	}
}

func containsPrefix(lines []string, prefix string) bool {
	for _, l := range lines {
		if strings.HasPrefix(l, prefix) {
			return true
		}
	}
	return false
}

// failingReleaseAssets fails every fetch in transport, as a dial behind a
// mandatory proxy does when the proxy is not honoured.
type failingReleaseAssets struct{}

func (failingReleaseAssets) FetchReleaseAsset(tag, name string) ([]byte, string, bool, error) {
	return nil, "https://example.com/" + tag + "/" + name, false, errors.New("dial tcp 192.0.2.1:443: i/o timeout")
}

// TestLaunchDryRunFetchBaselineNamesTheIgnoredEnvironment: the proxy and CA
// variables the baseline fetch does not honour are named in the plain preview
// and in --json, as `abcd update` names them, and a fetch that fails says so
// in its refusal too (iss-2609251902444497).
func TestLaunchDryRunFetchBaselineNamesTheIgnoredEnvironment(t *testing.T) {
	r := parityCLIRepo(t)
	fetcher := launch.ReleaseAssetFetcher(fakeReleaseAssets{assets: map[string][]byte{}})
	orig := newReleaseAssetFetcher
	newReleaseAssetFetcher = func(string) (launch.ReleaseAssetFetcher, []string, error) {
		return fetcher, []string{"HTTPS_PROXY", "SSL_CERT_FILE"}, nil
	}
	t.Cleanup(func() { newReleaseAssetFetcher = orig })

	rep, _ := dryRunJSON(t, r, "--fetch-baseline")
	if rep.Parity == nil || strings.Join(rep.Parity.EnvIgnored, ",") != "HTTPS_PROXY,SSL_CERT_FILE" {
		t.Fatalf("--json must carry the ignored names, got %+v", rep.Parity)
	}
	plain, err := shipIn(t, r, "launch", "--dry-run", "--fetch-baseline")
	if err != nil || !strings.Contains(string(plain), "ignored from the environment: HTTPS_PROXY, SSL_CERT_FILE") {
		t.Errorf("the plain preview must name the ignored variables, got %v:\n%s", err, plain)
	}

	fetcher = failingReleaseAssets{}
	plain, err = shipIn(t, r, "launch", "--dry-run", "--fetch-baseline")
	if err != nil || !strings.Contains(string(plain), "ignored HTTPS_PROXY, SSL_CERT_FILE from the environment") {
		t.Errorf("a failed fetch must say what it ignored, got %v:\n%s", err, plain)
	}

	// Without the flag nothing is fetched and nothing is named.
	if rep, _ := dryRunJSON(t, r); rep.Parity == nil || len(rep.Parity.EnvIgnored) != 0 {
		t.Errorf("a disk-only preview ignores nothing, got %+v", rep.Parity)
	}
}

// TestLaunchDryRunDeepSmokeRunsInAnIsolatedSubprocess is AC4 at the preview,
// through the real runner: the page is rendered by a child process rooted at a
// materialised copy of the payload, and a page that resolves but does not load
// is named. Without the flag the preview stays at the light tier.
func TestLaunchDryRunDeepSmokeRunsInAnIsolatedSubprocess(t *testing.T) {
	r := parityCLIRepo(t)
	r.Write("commands/broken.md", "---\nname: broken\ndescription: never closed\n")
	r.Commit("a page that does not load")

	rep, _ := dryRunJSON(t, r)
	if rep.DeepSmoke != nil {
		t.Fatalf("the deep tier is opt-in in the preview, got %+v", rep.DeepSmoke)
	}
	if !rep.Smoke.OK {
		t.Fatalf("the light tier passes this payload, or the test proves nothing: %+v", rep.Smoke.Findings)
	}

	rep, _ = dryRunJSON(t, r, "--deep-smoke")
	if rep.DeepSmoke == nil || rep.DeepSmoke.OK || rep.DeepSmoke.Checked != 3 {
		t.Fatalf("the deep tier must render the three pages and fail one, got %+v", rep.DeepSmoke)
	}
	if len(rep.DeepSmoke.Findings) != 1 || rep.DeepSmoke.Findings[0].Path != "commands/broken.md" {
		t.Errorf("exactly the broken page must be named, got %+v", rep.DeepSmoke.Findings)
	}
	var sawGood bool
	for _, p := range rep.DeepSmoke.Pages {
		if p.Path == "commands/one.md" && p.Description == "the first command" && p.Error == "" {
			sawGood = true
		}
	}
	if !sawGood {
		t.Errorf("a page that loads must render its help, got %+v", rep.DeepSmoke.Pages)
	}
	if out := r.Git("status", "--porcelain", "--ignored", "--", "commands", ".claude-plugin"); out != "" {
		t.Errorf("the deep tier left residue in the repository:\n%s", out)
	}
}

// TestSubprocessPageRunnerChecksTheChildsRoot: the runner checks that the
// child ran where it was told to, so a runner that ran elsewhere never passes.
func TestSubprocessPageRunnerChecksTheChildsRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "commands"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "commands", "a.md"), []byte("---\ndescription: a\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	answers, err := subprocessPageRunner(root, []launch.PageRef{{Kind: launch.SurfaceCommand, Path: "commands/a.md"}})
	if err != nil {
		t.Fatalf("the real runner must answer: %v", err)
	}
	if len(answers) != 1 || answers[0].Description != "a" {
		t.Fatalf("the child must render the page, got %+v", answers)
	}
	if _, err := subprocessPageRunner(filepath.Join(root, "absent"), nil); err == nil {
		t.Error("a root that does not exist must fail the runner")
	}
}

// TestLaunchShipRunsTheDeepTierAndParity is the cut's half: the deep tier is
// always on and refuses an unloadable page before anything is written, and a
// clean cut carries the parity diff in its report and its pre-flight record.
func TestLaunchShipRunsTheDeepTierAndParity(t *testing.T) {
	r := shipRenderableRepo(t)
	r.Write(".abcd/config/launch-payload.json", `{"includes": [".claude-plugin", "CHANGELOG.md", "commands"]}`+"\n")
	r.Write("commands/broken.md", "---\nname: broken\ndescription: never closed\n")
	r.Commit("a page that does not load")
	before := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md"))

	dest := filepath.Join(t.TempDir(), "payload")
	payload := composedPayload(t, t.TempDir(), "v0.4.1", "itd-73")
	out, err := shipIn(t, r, "launch", "ship", "--changelog-json", payload, "--payload-dir", dest)
	if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "commands/broken.md") {
		t.Fatalf("the cut must refuse the unloadable page by name, got %d: %v\n%s", code, err, out)
	}
	if after := readFileString(t, filepath.Join(r.Root(), "CHANGELOG.md")); after != before {
		t.Error("a refused cut must write nothing")
	}

	r.Write("commands/broken.md", "---\nname: broken\ndescription: now closed\n---\n")
	r.Commit("the page loads")
	t.Chdir(r.Root())
	stdout, stderr, err := runCLIPipedStdinSplit(t, "", "launch", "ship", "--changelog-json", payload, "--payload-dir", dest, "--json")
	if code := exitCodeOf(err); code != 0 {
		t.Fatalf("a clean cut must pass, got %d: %v\n%s\n%s", code, err, stdout, stderr)
	}
	var res struct {
		Preflight string                  `json:"preflight_report"`
		Parity    *launch.ParityReport    `json:"parity"`
		DeepSmoke *launch.DeepSmokeReport `json:"deep_smoke"`
	}
	if err := json.Unmarshal(stdout, &res); err != nil {
		t.Fatalf("ship JSON: %v\n%s", err, stdout)
	}
	if res.DeepSmoke == nil || !res.DeepSmoke.OK || res.DeepSmoke.Checked != 1 {
		t.Errorf("the cut must carry a passing deep tier, got %+v", res.DeepSmoke)
	}
	if res.Parity == nil || res.Parity.Baseline != "v0.4.0" || res.Parity.Added == 0 {
		t.Errorf("the cut must carry the diff against its anchor tag, got %+v", res.Parity)
	}
	written := readPreflight(t, r.Root(), res.Preflight)
	if written.Parity == nil || written.DeepSmoke == nil {
		t.Errorf("the cut's pre-flight report must carry both, got %+v", written)
	}
}

// TestLaunchShipFetchBaselineNeedsARender: the flag reads a baseline only a
// render compares against, so asking the emit step for it is an operand error.
func TestLaunchShipFetchBaselineNeedsARender(t *testing.T) {
	r := shipRenderableRepo(t)
	out, err := shipIn(t, r, "launch", "ship", "--fetch-baseline")
	if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "--fetch-baseline") {
		t.Errorf("--fetch-baseline without an ingest must exit 2 naming the flag, got %d: %v\n%s", code, err, out)
	}
}
