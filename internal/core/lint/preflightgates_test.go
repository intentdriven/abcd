package lint_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/issueschema"
	"github.com/intentdriven/abcd/internal/core/spec"
)

// preflight's prerequisite list is a DERIVED value, and every surface that
// restates it must name the same set (iss-2608242043243131).
//
// The Makefile declares the list once. The surfaces in the list below restate
// it by hand, and nothing derived them from the recipe, so they drifted every
// time the recipe changed — twice in two releases, each time caught only by a
// host-run semantic reviewer refusing a release:
//
//   - v0.6.4: `lint-issues` made preflight four gates. The commit that added it
//     updated AGENTS.md alone; the install guide, CONTRIBUTING.md and the
//     Makefile's own comment still said three.
//   - v0.6.5: `site-render` made it five. The same three surfaces still said four
//     — including the two just corrected.
//
// The repository already solved this exact shape once, for a different derived
// value: TestInstallGuideDocumentsTheInstallAndUpdatePath reads the module path
// out of go.mod and asserts the documented install command contains it. This is
// that move applied to the gate list.
//
// Deliberately a containment check rather than an equality one. These are
// sentences, not lists — "together with the lint-reviews, lint-issues, … gates"
// — so requiring an exact rendering would fail on a comma. What must hold is
// that every prerequisite the recipe declares is NAMED where the list is
// restated. The reverse direction — a surface naming a gate the recipe no
// longer has — is not checked here; a retired gate's mention is a review
// concern, since a phantom NAME has no recipe line to derive a check from.
func TestPreflightGateListIsNotRestatedWrongly(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	declared := preflightPrereqs(t, root)
	if len(declared) < 2 {
		t.Fatalf("parsed %d preflight prerequisites from the Makefile; the parser or the recipe changed shape", len(declared))
	}

	// Every surface that enumerates the gates. A file joins this list when it
	// starts restating them — which is the moment it becomes able to drift.
	for _, rel := range []string{
		"Makefile",               // the recipe's own comment, above the recipe
		"docs/how-to/install.md", // the build section a contributor copies
		"CONTRIBUTING.md",        // the local-gates paragraph
		"AGENTS.md",              // the definition-of-done list
		"CLAUDE.md",              // AGENTS.md's committed mirror
		".githooks/pre-push",     // the hook that invokes the recipe
	} {
		t.Run(rel, func(t *testing.T) {
			prose := readRepoFile(t, root, rel)
			// The Makefile is checked against the COMMENT BLOCK above the recipe,
			// not the whole file: the recipe line is where `declared` comes from,
			// so a whole-file containment check passes on its own input and can
			// never fail.
			if rel == "Makefile" {
				prose = commentBlockAbovePreflight(prose)
			}
			// No skip for a file that stops enumerating. An earlier draft skipped
			// when "lint-reviews" was absent, which made the gate defeatable by
			// exactly the drift it targets: rewriting a sentence to name two of
			// five gates removes the sentinel along with the gates, and the
			// subtest passes by skipping. The file list is hand-curated — a
			// surface that genuinely stops enumerating leaves this list in the
			// same change.
			for _, gate := range declared {
				if !strings.Contains(prose, gate) {
					t.Errorf("%s enumerates the preflight gates but omits %q.\n\n"+
						"The Makefile declares: %s\n"+
						"A restated list that has fallen behind the recipe understates what "+
						"guards a push, and this drift has reached a release twice.",
						rel, gate, strings.Join(declared, " "))
				}
			}
		})
	}
}

// TestPreflightRunsEveryTaggedEvalLane holds the position every tagged eval
// lane is supposed to occupy (iss-2608311632382737).
//
// Framework 8.6 makes the read-block eval the only component capable of
// falsifying the assembler's firewall, and framework 8.7 makes the amnesia eval
// the only check on what a re-run is handed. Both live behind a build tag, so
// `go test ./...` — preflight's own test step — compiles neither, and preflight
// ran neither target: a defect in the component that falsifies the firewall
// passed every local gate and surfaced, if at all, in CI, where the job that
// executes it is not a required status check. That is not hypothetical — a
// path-elision defect in the amnesia eval's own guard was unsatisfiable wherever
// the process temp directory is the Linux one, so it landed green locally and
// was found by an adversarial review rather than by any gate.
//
// The roster is DERIVED rather than listed here: a Makefile target is an eval
// lane when its recipe runs `go test` with `-tags`. So a third lane joins the
// push gate by existing, and cannot be added to the Makefile while staying
// invisible to it — which a hand-written pair of names could not promise. The
// restatement gate above then carries whatever this finds into every surface
// that enumerates the gates.
func TestPreflightRunsEveryTaggedEvalLane(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	lanes := taggedEvalLanes(t, root)
	if len(lanes) == 0 {
		t.Fatal("parsed no `go test -tags ...` lane from the Makefile; the parser or the recipes changed shape")
	}
	declared := preflightPrereqs(t, root)

	for _, lane := range lanes {
		if slices.Contains(declared, lane) {
			continue
		}
		t.Errorf("the Makefile declares the tagged eval lane %q, and preflight does not run it "+
			"(it declares: %s).\n\n"+
			"`go test ./...` does not compile a tagged file, so a lane preflight omits is "+
			"guarded by nothing a push can fail on, and the eval that certifies the read-block "+
			"then guards nothing that blocks anything.", lane, strings.Join(declared, " "))
	}
}

// taggedEvalLanes returns the Makefile targets whose recipe runs `go test` with
// a `-tags` selector — hand-parsed, for the reason preflightPrereqs is.
func taggedEvalLanes(t *testing.T, root string) []string {
	t.Helper()
	target := regexp.MustCompile(`^([a-z][a-z-]*):`)
	var lanes []string
	var current string
	for _, line := range strings.Split(readRepoFile(t, root, "Makefile"), "\n") {
		if m := target.FindStringSubmatch(line); m != nil {
			current = m[1]
			continue
		}
		if !strings.HasPrefix(line, "\t") || current == "" {
			continue
		}
		body := strings.TrimSpace(strings.TrimPrefix(line, "\t"))
		if strings.HasPrefix(body, "go test ") && strings.Contains(body, "-tags ") &&
			!slices.Contains(lanes, current) {
			lanes = append(lanes, current)
		}
	}
	return lanes
}

// TestColdReadingEvalsIsARequiredStatusCheck holds the other half of that gate
// (iss-2608311051046981): the CI job that runs the read-block eval (framework
// 8.6) blocks a merge rather than merely reporting on one.
//
// Three properties, and each is a way the guarantee can be lost. The workflow
// defines the job, so a rename cannot leave a required context that never
// arrives and wedges the queue. The job stands down on no event — no `if:` and
// no `needs:` — so its context arrives on every event the other required checks
// arrive on, which is what makes requiring it safe. And the committed ruleset
// mirror requires the context, which is the tree's own record of what the live
// ruleset is set to.
func TestColdReadingEvalsIsARequiredStatusCheck(t *testing.T) {
	const job = "cold-reading-evals"
	root := filepath.Join("..", "..", "..")

	workflow := readRepoFile(t, root, ".github/workflows/ci.yml")
	block, ok := workflowJobBlock(workflow, job)
	if !ok {
		t.Fatalf(".github/workflows/ci.yml defines no %q job; a required status check whose "+
			"job does not exist never reports, and a required context that never arrives "+
			"wedges the merge queue", job)
	}
	for _, standDown := range []string{"if:", "needs:"} {
		if !strings.Contains(block, standDown) {
			continue
		}
		t.Errorf("the %q job carries a %q, so it can stand down on some event; a required "+
			"check must report on every event the others do, and a stood-down job that "+
			"still reports its context green is a green for work that did not happen",
			job, standDown)
	}

	const mirror = ".abcd/work/rulesets/main-protection.json"
	var ruleset struct {
		Rules []struct {
			Type       string `json:"type"`
			Parameters struct {
				RequiredStatusChecks []struct {
					Context string `json:"context"`
				} `json:"required_status_checks"`
			} `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, mirror)), &ruleset); err != nil {
		t.Fatalf("decoding %s: %v", mirror, err)
	}
	var required []string
	for _, rule := range ruleset.Rules {
		for _, c := range rule.Parameters.RequiredStatusChecks {
			required = append(required, c.Context)
		}
	}
	if len(required) == 0 {
		t.Fatalf("%s declares no required status checks at all; the mirror is the tree's "+
			"record of the live ruleset, so an empty list here reads as an unprotected branch", mirror)
	}
	if !slices.Contains(required, job) {
		t.Fatalf("%s does not require the %q context (it requires: %s).\n\n"+
			"The point of the always-run lane is that a record-only pull request cannot reach "+
			"main with warm content in included material, and an unrequired check does not "+
			"stop one.", mirror, job, strings.Join(required, ", "))
	}
	if !slices.IsSorted(required) {
		t.Errorf("%s lists its required contexts out of order (%s); the mirror is refreshed "+
			"by hand from a sorted `jq -S` rendering, and an unsorted list means it was not",
			mirror, strings.Join(required, ", "))
	}
}

// workflowJobBlock returns one job's block from a workflow: the `  <name>:`
// line and everything indented under it, up to the next job. Hand-parsed for
// the reason preflightPrereqs is: this repository carries no YAML parser for
// its own workflows and adds none for one job.
func workflowJobBlock(workflow, job string) (string, bool) {
	lines := strings.Split(workflow, "\n")
	at := -1
	for i, l := range lines {
		if l == "  "+job+":" {
			at = i
			break
		}
	}
	if at < 0 {
		return "", false
	}
	nextJob := regexp.MustCompile(`^  [A-Za-z]`)
	end := len(lines)
	for i := at + 1; i < len(lines); i++ {
		if nextJob.MatchString(lines[i]) {
			end = i
			break
		}
	}
	return strings.Join(lines[at:end], "\n"), true
}

// preflightPrereqs reads the prerequisites off the `preflight:` recipe line.
// Hand-parsed for the same reason the workflow contracts beside it are: this
// repository carries no Makefile parser and adds none for one line.
func preflightPrereqs(t *testing.T, root string) []string {
	t.Helper()
	for _, line := range strings.Split(readRepoFile(t, root, "Makefile"), "\n") {
		rest, ok := strings.CutPrefix(line, "preflight:")
		if !ok {
			continue
		}
		var out []string
		for _, f := range strings.Fields(rest) {
			// Only the lint-style prerequisites are restated in prose; a future
			// non-gate prerequisite should not force itself into a sentence.
			if regexp.MustCompile(`^[a-z][a-z-]*$`).MatchString(f) {
				out = append(out, f)
			}
		}
		return out
	}
	t.Fatal("Makefile declares no `preflight:` recipe")
	return nil
}

// commentBlockAbovePreflight returns the contiguous `#` comment block directly
// above the `preflight:` recipe, which is the part that restates the gate list.
func commentBlockAbovePreflight(makefile string) string {
	lines := strings.Split(makefile, "\n")
	at := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "preflight:") {
			at = i
			break
		}
	}
	if at < 0 {
		return ""
	}
	start := at
	for start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "#") {
		start--
	}
	return strings.Join(lines[start:at], "\n")
}

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(b)
}

// The issue-resolution gate scopes its git pathspecs to the ledger's STATUS
// directories, and the shell cannot import Go — so the script holds the second
// and last spelling of the list issueschema.StatusDirs owns. This reads the
// array the script declares and holds it to that one value.
//
// It matters because the ledger tree gained sibling record families (readings/,
// dispositions/) whose files are not issue records: RS002 and RS003 scan every
// `.md` under the pathspec and would read a `commit:`-shaped line out of one of
// them. Scoping is what keeps them out, and a scope that drifts from the
// canonical list either re-admits them or drops a real status folder — the
// second of which fails open, silently.
func TestIssueResolutionGateScopesToStatusDirs(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	script := readRepoFile(t, root, "scripts/check-issue-resolution.sh")

	m := regexp.MustCompile(`(?m)^STATUS_DIRS=\(([^)]*)\)`).FindStringSubmatch(script)
	if m == nil {
		t.Fatalf("scripts/check-issue-resolution.sh declares no STATUS_DIRS=(...) array;\n" +
			"the gate must scope to the ledger's status directories, and the array is where it says which")
	}
	got := strings.Fields(m[1])
	want := issueschema.StatusDirs
	if !slices.Equal(got, want) {
		t.Fatalf("scripts/check-issue-resolution.sh declares STATUS_DIRS=%v, want %v (issueschema.StatusDirs)",
			got, want)
	}
}

// RS005 (itd-2609111003026787) reads the intent store and the spec store from
// the same shell script, and the shell cannot import Go either — so the script
// holds the second spelling of three values the stores own: the intent-store
// root, its bucket list, and the spec-store root. Each is held to its one Go
// value here. A root that drifts makes every declared intent read as "no record
// anywhere", and a bucket list that drops one sends a real record to the same
// verdict, which is a refusal with the wrong diagnosis on every delivery.
func TestIssueResolutionGateReadsTheIntentAndSpecStores(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	script := readRepoFile(t, root, "scripts/check-issue-resolution.sh")

	for _, pin := range []struct{ name, want string }{
		{"INTENTS_DIR", intent.IntentsRelDir},
		{"SPECS_DIR", spec.SpecsRelDir},
	} {
		m := regexp.MustCompile(`(?m)^` + pin.name + `="([^"]*)"`).FindStringSubmatch(script)
		if m == nil {
			t.Fatalf("scripts/check-issue-resolution.sh declares no %s=\"...\"; RS005 must name the store it reads", pin.name)
		}
		if m[1] != pin.want {
			t.Fatalf("scripts/check-issue-resolution.sh declares %s=%q, want %q", pin.name, m[1], pin.want)
		}
	}

	m := regexp.MustCompile(`(?m)^INTENT_BUCKETS=\(([^)]*)\)`).FindStringSubmatch(script)
	if m == nil {
		t.Fatalf("scripts/check-issue-resolution.sh declares no INTENT_BUCKETS=(...) array;\n" +
			"RS005 must scope its intent lookups to the store's buckets, and the array is where it says which")
	}
	if got := strings.Fields(m[1]); !slices.Equal(got, intent.Buckets) {
		t.Fatalf("scripts/check-issue-resolution.sh declares INTENT_BUCKETS=%v, want %v (intent.Buckets)", got, intent.Buckets)
	}
}

// TestFormatGateResolvesThroughTheDeclaredToolchain holds the format gate to one
// definition, resolved through the toolchain go.mod declares
// (iss-2609081953452204).
//
// The gate used to be a bare `gofmt -l .`, written out by hand in three places:
// CI's own step, AGENTS.md's command table, and CONTRIBUTING.md's local-gates
// paragraph. A bare `gofmt` is whatever the caller has on PATH, and gofmt's
// rules move between releases — go 1.27 re-indents a multi-value return whose
// operands are composite literals, which this tree contains at
// internal/core/ahoy/remote.go:419. So on a go1.27 machine the documented gate
// named that file against an unmodified checkout of main, while CI's pinned
// 1.26.7 gofmt called the same bytes correct. The developer "fixes" the file
// the gate names and pushes something CI then rejects in the other direction,
// and neither direction is visible: the output names a file and never says
// which toolchain judged it.
//
// Three properties, and each is a way the trap comes back. CI must invoke the
// Makefile target rather than carry its own copy of the command, so there is one
// definition to pin the toolchain in
// (.abcd/development/principles/one-canonical-primitive.md). That target must
// resolve gofmt from the version go.mod declares, read from go.mod rather than
// spelled again in the Makefile — a second spelling is a second thing to bump.
// And no developer-facing surface may instruct a bare `gofmt` invocation, which
// is the form that reads PATH.
//
// The target name is DERIVED from CI's step rather than written here: CI is what
// actually runs, so a rename that reaches the workflow drags the Makefile and
// every prose surface along with it instead of failing on a literal this test
// would have to hold too.
func TestFormatGateResolvesThroughTheDeclaredToolchain(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	// The CI step is the authority on what the gate command is.
	step, ok := workflowStepBlock(readRepoFile(t, root, ".github/workflows/ci.yml"), "Format (gofmt)")
	if !ok {
		t.Fatal(".github/workflows/ci.yml defines no `Format (gofmt)` step; the format gate is the " +
			"one gate outside `make preflight`, and a CI job that stops running it leaves nothing enforcing it")
	}
	m := regexp.MustCompile(`\bmake ([a-z][a-z-]*)\b`).FindStringSubmatch(step)
	if m == nil {
		t.Fatalf("the `Format (gofmt)` step in .github/workflows/ci.yml runs no `make <target>`:\n\n%s\n\n"+
			"CI carrying its own copy of the gate command is how the developer's gate and CI's gate "+
			"came to be resolved through different toolchains. One definition, invoked from both.", step)
	}
	target := m[1]

	if bare := bareGofmtInvocation(step); bare != "" {
		t.Errorf("the `Format (gofmt)` step in .github/workflows/ci.yml still invokes a bare gofmt (%q); "+
			"a bare gofmt is whichever one is on PATH, which is the skew this gate exists to close", bare)
	}

	// The Makefile must declare that target, and resolve gofmt from go.mod.
	makefile := readRepoFile(t, root, "Makefile")
	recipe, ok := makeRecipe(makefile, target)
	if !ok {
		t.Fatalf("CI's format step runs `make %s`, and the Makefile declares no `%s:` target; "+
			"a workflow calling a target that does not exist fails the job with a make error "+
			"rather than a format report", target, target)
	}
	if !strings.Contains(recipe, "GOTOOLCHAIN=go") {
		t.Errorf("the `%s:` recipe does not resolve a pinned toolchain (no GOTOOLCHAIN=go...):\n\n%s\n\n"+
			"gofmt must come from the toolchain go.mod declares, or the gate judges the tree "+
			"by whatever version the caller happens to have", target, recipe)
	}
	if bare := bareGofmtInvocation(recipe); bare != "" {
		t.Errorf("the `%s:` recipe invokes a bare gofmt (%q); it must run the gofmt inside the "+
			"resolved toolchain's GOROOT, not the one on PATH", target, bare)
	}

	// The declared version is READ from go.mod, not spelled again in the Makefile.
	goMod := readRepoFile(t, root, "go.mod")
	dm := regexp.MustCompile(`(?m)^go ([0-9][0-9.]*)$`).FindStringSubmatch(goMod)
	if dm == nil {
		t.Fatal("go.mod declares no `go <version>` line; the format gate has no toolchain to resolve")
	}
	declared := dm[1]
	if !strings.Contains(makefile, "go.mod") {
		t.Errorf("the Makefile never reads go.mod, so the format gate's toolchain is not derived from "+
			"the declaration; go.mod says go %s", declared)
	}
	if strings.Contains(makefile, declared) {
		t.Errorf("the Makefile spells the Go version %q itself; it must read it out of go.mod, or the "+
			"gate keeps judging the tree by the old toolchain after the declaration moves", declared)
	}

	// No developer-facing surface may still instruct a bare gofmt, and each must
	// name the target CI runs — otherwise the human runs a different gate.
	for _, rel := range []string{
		"AGENTS.md",                        // the command table and the definition-of-done list
		"CLAUDE.md",                        // AGENTS.md's committed mirror
		"CONTRIBUTING.md",                  // the local-gates paragraph
		".github/PULL_REQUEST_TEMPLATE.md", // the verification prompt
		".githooks/pre-push",               // the hook's header, which tells the developer what CI adds
	} {
		t.Run(rel, func(t *testing.T) {
			prose := readRepoFile(t, root, rel)
			if bare := bareGofmtInvocation(prose); bare != "" {
				t.Errorf("%s instructs a bare gofmt invocation (%q).\n\n"+
					"That command reads the developer's PATH, so on a machine newer than go %s it "+
					"names a file CI considers correctly formatted — and the 'fix' it invites is a "+
					"file CI then rejects the other way.", rel, bare, declared)
			}
			if !strings.Contains(prose, "make "+target) {
				t.Errorf("%s documents the format gate but never names `make %s`, the command CI runs; "+
					"a documented gate that differs from the enforced one is the drift this closes",
					rel, target)
			}
		})
	}
}

// formatStepName is the name every format-gate step carries, in this repo's
// workflows and in the one a managed repo is scaffolded. It is also the first
// entry of the release-gate runbook's deterministic-gate list, which the
// gate_lockstep rule holds to the workflow — so the name is load-bearing in two
// directions and is spelled once here.
const formatStepName = "Format (gofmt)"

// TestNoShippedWorkflowRunsTheGofmtOnPATH extends the property above to the two
// surfaces its hand-written roster never read (iss-2609091128354325).
//
// TestFormatGateResolvesThroughTheDeclaredToolchain names ci.yml and four prose
// files. That left release.yml's `verify` job inlining `gofmt -l .` and telling
// the reader to run `gofmt -w .` — the command the pinned target replaced — and
// left the scaffold template carrying the identical block, which is the block
// `abcd launch scaffold` writes into every managed repo. So a claim AGENTS.md
// makes about the format gate as such was true of one of its three homes, and
// the release lane — the one lane whose output people download — was the home it
// was false of.
//
// The roster here is DERIVED: every workflow under .github/workflows and every
// scaffold template. A workflow that grows a format step joins by existing,
// which is what the hand-written list could not promise. Two floors make a
// broken sweep fail rather than pass quietly — the roster must be non-empty, and
// it must find at least the three format gates the tree carries (ci.yml,
// release.yml, and the template both are checked against).
//
// A managed repo has no Makefile of ours to invoke, so "pinned" cannot mean
// `make fmt-check` everywhere. What it means is the negative property the helper
// already encodes: no gofmt resolved by PATH. The template satisfies it by
// running the gofmt inside the GOROOT the module's own toolchain resolves to,
// which is the same binary its `go build` step uses and which follows the go
// directive with no workflow edit.
func TestNoShippedWorkflowRunsTheGofmtOnPATH(t *testing.T) {
	root := filepath.Join("..", "..", "..")

	var roster []string
	for _, dir := range []string{
		".github/workflows",                       // what this repo runs
		"internal/core/launch/scaffold/templates", // what every managed repo is handed
	} {
		entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(dir)))
		if err != nil {
			t.Fatalf("read %s: %v", dir, err)
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			switch filepath.Ext(e.Name()) {
			case ".yml", ".yaml", ".tmpl":
				roster = append(roster, dir+"/"+e.Name())
			}
		}
	}
	if len(roster) == 0 {
		t.Fatal("the workflow/template sweep matched no files; it would pass by finding nothing")
	}

	gates := 0
	for _, rel := range roster {
		text := readRepoFile(t, root, rel)
		if _, ok := workflowStepBlock(text, formatStepName); ok {
			gates++
		}
		if bare := bareGofmtInvocation(text); bare != "" {
			t.Errorf("%s invokes a bare gofmt (%q).\n\n"+
				"That is whichever gofmt the runner's PATH resolves, which is the skew the "+
				"format gate exists to close. This repo's workflows invoke `make fmt-check`; a "+
				"scaffolded repo, which has no Makefile of ours, runs the gofmt under the GOROOT "+
				"its own declared toolchain resolves to.", rel, bare)
		}
	}
	const minFormatGates = 3 // ci.yml, release.yml, release.yml.tmpl
	if gates < minFormatGates {
		t.Fatalf("found %d %q steps across %d shipped workflows and templates, want at least %d; "+
			"the step-name parser or the step name itself changed, and a sweep that cannot see the "+
			"format gates cannot report an unpinned one",
			gates, formatStepName, len(roster), minFormatGates)
	}
}

// bareGofmtInvocation returns the first unqualified `gofmt <flag>` invocation in
// text, or "" if there is none. Qualified is the point: `"$GOROOT/bin/gofmt" -l`
// names a specific binary, `gofmt -l` names whatever PATH resolves. Prose that
// merely mentions the word (a step name, an error prefix) carries no flag after
// it and does not match.
func bareGofmtInvocation(text string) string {
	m := regexp.MustCompile(`(?:^|[^/\w.-])(gofmt +-[a-zA-Z]+[^\n]*)`).FindStringSubmatch(text)
	if m == nil {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// makeRecipe returns one Makefile target's recipe: every tab-indented line under
// `<target>:`. Hand-parsed for the reason preflightPrereqs is — this repository
// carries no Makefile parser and adds none for one target.
func makeRecipe(makefile, target string) (string, bool) {
	lines := strings.Split(makefile, "\n")
	at := -1
	for i, l := range lines {
		if strings.HasPrefix(l, target+":") {
			at = i
			break
		}
	}
	if at < 0 {
		return "", false
	}
	var body []string
	for i := at + 1; i < len(lines); i++ {
		if !strings.HasPrefix(lines[i], "\t") {
			break
		}
		body = append(body, lines[i])
	}
	// A `define`d block invoked by the recipe is part of what the target runs, so
	// fold every define in the file into what is inspected: the resolution logic
	// lives there when two targets share it.
	for _, l := range body {
		for _, name := range regexp.MustCompile(`\$\(call ([a-z_]+)`).FindAllStringSubmatch(l, -1) {
			if block, ok := makeDefine(makefile, name[1]); ok {
				body = append(body, block)
			}
		}
	}
	return strings.Join(body, "\n"), true
}

// makeDefine returns the body of a `define <name> ... endef` block.
func makeDefine(makefile, name string) (string, bool) {
	lines := strings.Split(makefile, "\n")
	at := -1
	for i, l := range lines {
		if strings.HasPrefix(l, "define "+name) {
			at = i
			break
		}
	}
	if at < 0 {
		return "", false
	}
	for i := at + 1; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "endef") {
			return strings.Join(lines[at+1:i], "\n"), true
		}
	}
	return "", false
}

// workflowStepBlock returns one step's block from a workflow: the `- name: <n>`
// line and everything under it up to the next step at the same indent.
func workflowStepBlock(workflow, name string) (string, bool) {
	lines := strings.Split(workflow, "\n")
	at, indent := -1, ""
	stepName := regexp.MustCompile(`^(\s*)- name: "?` + regexp.QuoteMeta(name) + `"?\s*$`)
	for i, l := range lines {
		if m := stepName.FindStringSubmatch(l); m != nil {
			at, indent = i, m[1]
			break
		}
	}
	if at < 0 {
		return "", false
	}
	next := regexp.MustCompile(`^` + indent + `- `)
	end := len(lines)
	for i := at + 1; i < len(lines); i++ {
		if next.MatchString(lines[i]) {
			end = i
			break
		}
	}
	return strings.Join(lines[at:end], "\n"), true
}
