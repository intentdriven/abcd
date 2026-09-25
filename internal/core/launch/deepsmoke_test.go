package launch

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// inProcessRunner answers the deep tier's pages the way the subprocess does,
// through RenderPageHelp, and records the root it was handed.
func inProcessRunner(roots *[]string) PageRunner {
	return func(root string, pages []PageRef) ([]PageHelp, error) {
		*roots = append(*roots, root)
		out := make([]PageHelp, 0, len(pages))
		for _, p := range pages {
			out = append(out, RenderPageHelp(root, p))
		}
		return out, nil
	}
}

// deepFixture is a materialised payload with one command page of each fate.
func deepFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeLockstepTree(t, root, "", "", "")
	writeFile(t, root, "commands/good.md", "---\nname: good\ndescription: \"A good page\"\nargument-hint: \"[--flag]\"\n---\n# Good\n")
	writeFile(t, root, "commands/unclosed.md", "---\nname: unclosed\ndescription: never closed\n# body\n")
	return root
}

// TestRenderPageHelp pins what "loads" means for one page: its help renders,
// and each way a page resolves on disk but would not load is named.
func TestRenderPageHelp(t *testing.T) {
	root := t.TempDir()
	pages := map[string]string{
		"commands/good.md":        "---\nname: good\ndescription: \"A good page\"\nargument-hint: \"[--flag]\"\nallowed-tools:\n- Bash\n---\n# Good\n",
		"commands/bare.md":        "Plain help line\n\nmore\n",
		"commands/unclosed.md":    "---\ndescription: x\n",
		"commands/dup.md":         "---\ndescription: one\ndescription: two\n---\nbody\n",
		"commands/notmapping.md":  "---\ndescription: x\nthis is not yaml\n---\nbody\n",
		"commands/empty.md":       "---\nname: empty\n---\n\n",
		"commands/binary.md":      "---\ndescription: \xff\xfe\n---\n",
		"skills/s/SKILL.md":       "---\nname: s\ndescription: a skill\n---\nbody\n",
		"skills/nodesc/SKILL.md":  "---\nname: nodesc\n---\nbody\n",
		"agents/README.md":        "# Agents\n\nprose\n",
		"commands/folded.md":      "---\ndescription: >\n  folded text\n  over lines\n---\nbody\n",
		"commands/nested/deep.md": "---\ndescription: nested\n---\n",
	}
	for rel, body := range pages {
		writeFile(t, root, rel, body)
	}
	cases := []struct {
		kind    SurfaceKind
		path    string
		wantErr string
		wantDes string
	}{
		{SurfaceCommand, "commands/good.md", "", "A good page"},
		{SurfaceCommand, "commands/bare.md", "", "Plain help line"},
		{SurfaceCommand, "commands/unclosed.md", "never closed", ""},
		{SurfaceCommand, "commands/dup.md", "duplicate", ""},
		{SurfaceCommand, "commands/notmapping.md", "line 3", ""},
		{SurfaceCommand, "commands/empty.md", "no help", ""},
		{SurfaceCommand, "commands/binary.md", "UTF-8", ""},
		{SurfaceSkill, "skills/s/SKILL.md", "", "a skill"},
		{SurfaceSkill, "skills/nodesc/SKILL.md", "description", ""},
		{SurfaceAgent, "agents/README.md", "", "# Agents"},
		{SurfaceCommand, "commands/folded.md", "", "folded text over lines"},
		{SurfaceCommand, "commands/nested/deep.md", "", "nested"},
		{SurfaceCommand, "commands/absent.md", "not readable", ""},
		{SurfaceCommand, "../escape.md", "not readable", ""},
	}
	for _, tc := range cases {
		got := RenderPageHelp(root, PageRef{Kind: tc.kind, Path: tc.path})
		if tc.wantErr != "" {
			if !strings.Contains(got.Error, tc.wantErr) {
				t.Errorf("%s: want an error naming %q, got %+v", tc.path, tc.wantErr, got)
			}
			continue
		}
		if got.Error != "" || !strings.HasPrefix(got.Description, tc.wantDes) {
			t.Errorf("%s: want help %q, got %+v", tc.path, tc.wantDes, got)
		}
	}
	if got := RenderPageHelp(root, PageRef{Kind: SurfaceCommand, Path: "commands/good.md"}); got.Name != "good" || got.ArgumentHint != "[--flag]" {
		t.Errorf("the help must carry the name and the argument hint, got %+v", got)
	}
	if got := RenderPageHelp(root, PageRef{Kind: SurfaceCommand, Path: "commands/bare.md"}); got.Name != "bare" {
		t.Errorf("a page with no frontmatter is named for its file, got %+v", got)
	}
}

// TestSmokeDeepCatchesAPageThatResolvesButDoesNotLoad is AC4: the light tier
// passes the fixture (every declared path is carried), and the deep tier,
// running rooted at the materialised tree, fails it on the page that would not
// load.
func TestSmokeDeepCatchesAPageThatResolvesButDoesNotLoad(t *testing.T) {
	root := deepFixture(t)
	if light := SmokeLight(NewDirTree(root)); !light.OK {
		t.Fatalf("the light tier must pass the fixture, or this test proves nothing: %+v", light.Findings)
	}
	var roots []string
	rep := SmokeDeep(root, inProcessRunner(&roots))
	if rep.OK || rep.Tier != SmokeTierDeep {
		t.Fatalf("an unloadable page must fail the deep tier, got %+v", rep)
	}
	if rep.Checked != 2 || len(rep.Pages) != 2 {
		t.Errorf("both command pages must be checked, got %d / %+v", rep.Checked, rep.Pages)
	}
	if len(rep.Findings) != 1 || rep.Findings[0].Path != "commands/unclosed.md" || rep.Findings[0].Kind != findingPageUnloadable {
		t.Errorf("exactly the unclosed page must be named, got %+v", rep.Findings)
	}
	if len(roots) != 1 || roots[0] != root {
		t.Errorf("the runner must be rooted at the rendered tree %s, got %v", root, roots)
	}
}

// TestSmokeDeepFailsClosed: a runner that cannot answer, or answers for fewer
// pages than it was asked about, is a failed tier, never a pass.
func TestSmokeDeepFailsClosed(t *testing.T) {
	root := deepFixture(t)
	if err := os.Remove(filepath.Join(root, "commands", "unclosed.md")); err != nil {
		t.Fatal(err)
	}
	broken := func(string, []PageRef) ([]PageHelp, error) { return nil, errors.New("the subprocess died") }
	if rep := SmokeDeep(root, broken); rep.OK || len(rep.Findings) == 0 || rep.Findings[0].Kind != findingDeepSmokeUnavailable {
		t.Errorf("a runner failure must fail the tier, got %+v", rep)
	}
	silent := func(string, []PageRef) ([]PageHelp, error) { return nil, nil }
	if rep := SmokeDeep(root, silent); rep.OK {
		t.Errorf("a page the runner never answered for must fail the tier, got %+v", rep)
	}
	if rep := SmokeDeep(root, nil); rep.OK {
		t.Errorf("no runner at all must fail the tier, got %+v", rep)
	}
	var roots []string
	if rep := SmokeDeep(root, inProcessRunner(&roots)); !rep.OK {
		t.Errorf("a payload whose every page loads must pass, got %+v", rep.Findings)
	}
}

// TestDryRunDeepSmokeIsOptInAndLeavesNoResidue: the preview runs the deep tier
// only when handed a runner, over a materialised copy it removes.
func TestDryRunDeepSmokeIsOptInAndLeavesNoResidue(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	writeFile(t, root, "commands/unclosed.md", "---\ndescription: never closed\n")
	repo.Commit("an unloadable page")

	rep, err := DryRun(DryRunRequest{RepoRoot: root, Version: "0.2.0", ExistingTags: []Semver{}})
	if err != nil {
		t.Fatal(err)
	}
	if rep.DeepSmoke != nil || gateRan(rep.Gates, "installability-smoke-deep") {
		t.Fatalf("the deep tier is opt-in in the preview, got %+v", rep.DeepSmoke)
	}

	var roots []string
	rep, err = DryRun(DryRunRequest{RepoRoot: root, Version: "0.2.0", ExistingTags: []Semver{}, DeepSmoke: inProcessRunner(&roots)})
	if err != nil {
		t.Fatal(err)
	}
	if rep.DeepSmoke == nil || rep.DeepSmoke.OK || !gateRan(rep.Gates, "installability-smoke-deep") {
		t.Fatalf("the opted-in deep tier must run and fail the unloadable page, got %+v", rep.DeepSmoke)
	}
	if !containsSubstring(rep.WouldRefuseOn, "commands/unclosed.md") {
		t.Errorf("the deep finding must be on the would-refuse list, got %v", rep.WouldRefuseOn)
	}
	if len(roots) != 1 || roots[0] == root || strings.HasPrefix(roots[0], root+string(filepath.Separator)) {
		t.Fatalf("the deep tier must run over a materialised copy outside the repository, got %v", roots)
	}
	if _, err := os.Stat(roots[0]); !os.IsNotExist(err) {
		t.Errorf("the materialised copy must be removed, stat says %v", err)
	}
	if out := repo.Git("status", "--porcelain"); out != "" {
		t.Errorf("the deep tier left the source tree dirty:\n%s", out)
	}
}

// TestPrecheckRunsTheDeepTierAndParityWhenAsked: the cut's precheck refuses an
// unloadable page and an unreadable baseline before anything is written, and
// runs neither when not asked (the archive gate's render path).
func TestPrecheckRunsTheDeepTierAndParityWhenAsked(t *testing.T) {
	repo := parityRepo(t)
	root := repo.Root()
	writeFile(t, root, "commands/unclosed.md", "---\ndescription: never closed\n")
	repo.Commit("an unloadable page")
	dest := filepath.Join(t.TempDir(), "payload")

	pre, err := PrecheckPayload(root, dest, PrecheckOptions{})
	if err != nil {
		t.Fatalf("without the deep tier the precheck passes: %v", err)
	}
	if pre.DeepSmoke != nil || pre.Parity != nil {
		t.Fatalf("neither runs unless asked: %+v %+v", pre.DeepSmoke, pre.Parity)
	}

	var roots []string
	pre, err = PrecheckPayload(root, dest, PrecheckOptions{DeepSmoke: inProcessRunner(&roots), Parity: &ParityInput{Baseline: "v9.9.9"}})
	var refusal *PrecheckRefusal
	if !errors.As(err, &refusal) {
		t.Fatalf("the precheck must refuse, got %v", err)
	}
	if !errors.Is(err, ErrPayloadUninstallable) || !errors.Is(err, ErrParityBaselineUnreadable) {
		t.Errorf("both refusals must be matchable, got %v", err)
	}
	if pre.DeepSmoke == nil || pre.Parity == nil || !pre.Parity.Refused {
		t.Errorf("the precheck must carry both reports, got %+v %+v", pre.DeepSmoke, pre.Parity)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("the precheck must not create the destination, stat says %v", err)
	}
	rep := pre.PreflightReport(testReportInstant, "")
	if rep.Parity == nil || rep.DeepSmoke == nil {
		t.Errorf("the cut's pre-flight report must carry both, got %+v", rep)
	}

	pre, err = PrecheckPayload(root, dest, PrecheckOptions{Parity: &ParityInput{Baseline: "v0.1.0"}})
	if err != nil {
		t.Fatalf("a readable baseline refuses nothing: %v", err)
	}
	if pre.Parity == nil || pre.Parity.Added != 2 {
		t.Errorf("the precheck must carry the diff (c.md and unclosed.md added), got %+v", pre.Parity)
	}
}

// testReportInstant pins the clock a pre-flight report is stamped with.
var testReportInstant = time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC)
