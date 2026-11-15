package lint_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	releaseGateDetectorRel = ".abcd/development/release-gate/brief-surface-crosscheck.js"
	releaseGateManifestRel = ".abcd/development/release-gate/manifest.json"
)

// The release-gate detector is what turns the pinned context's ${commands} and
// ${agents} into names: the manifest keeps the rosters outside the prompt text
// (iss-2610150526192138), so a detector that sent the context unfilled would
// hand every checker a literal ${commands} and a run would attest a scope no
// checker was told about. The check on that used to be two substrings of the
// script's source, which a script with the substitution commented out still
// held (iss-2610151050390126).
//
// This test runs the detector instead. The committed script is executed by
// node over the committed manifest, with the harness's agent() replaced by a
// recorder, and every prompt a checker would have received is read: there is
// one per brief doc and one per surface, none carries a ${...} placeholder,
// and each opens with the pinned context naming exactly the two rosters.
func TestReleaseGateDetectorFillsEveryCheckerPrompt(t *testing.T) {
	requireNode(t)
	root := filepath.Join("..", "..", "..")
	script := readRepoFile(t, root, releaseGateDetectorRel)
	manifest := []byte(readRepoFile(t, root, releaseGateManifestRel))
	for _, p := range detectorPromptProblems(t, script, manifest) {
		t.Error(p)
	}
}

// detectorFillCall is the statement in the detector that fills the context's
// roster placeholders. The test below disables it, and refuses a script it is
// no longer found in rather than mutate nothing and pass.
const detectorFillCall = `const CTX = fill(input.prompt.context, {
  commands: input.commands.join(', '),
  agents: input.agents.join(', '),
})`

// The check above is only worth having if it fails on a detector that does not
// fill. Each case breaks the committed script's fill one way and requires a
// problem to be reported. The first is the issue's own: the substitution
// commented out, both of the old check's substrings still in the source.
func TestReleaseGateDetectorWithItsFillBrokenIsReported(t *testing.T) {
	requireNode(t)
	root := filepath.Join("..", "..", "..")
	script := readRepoFile(t, root, releaseGateDetectorRel)
	manifest := []byte(readRepoFile(t, root, releaseGateManifestRel))
	if strings.Count(script, detectorFillCall) != 1 {
		t.Fatalf("%s no longer holds, exactly once, the fill statement this test breaks; update detectorFillCall to the statement that fills the context:\n%s", releaseGateDetectorRel, detectorFillCall)
	}
	for _, tc := range []struct {
		name, replacement, want string
	}{
		{
			name:        "the fill commented out",
			replacement: "const CTX = input.prompt.context\n// " + strings.ReplaceAll(detectorFillCall, "\n", "\n// "),
			want:        "placeholder",
		},
		{
			name:        "the agents roster not filled",
			replacement: "const CTX = fill(input.prompt.context, {\n  commands: input.commands.join(', '),\n})",
			want:        "${agents}",
		},
		{
			name:        "the commands roster cut short",
			replacement: "const CTX = fill(input.prompt.context, {\n  commands: input.commands.slice(1).join(', '),\n  agents: input.agents.join(', '),\n})",
			want:        "commands roster",
		},
		{
			name:        "the agents roster filled with the commands",
			replacement: "const CTX = fill(input.prompt.context, {\n  commands: input.commands.join(', '),\n  agents: input.commands.join(', '),\n})",
			want:        "agents roster",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ps := detectorPromptProblems(t, strings.Replace(script, detectorFillCall, tc.replacement, 1), manifest)
			if len(ps) == 0 {
				t.Fatal("a detector with its fill broken is reported as sound")
			}
			if !slices.ContainsFunc(ps, func(p string) bool { return strings.Contains(p, tc.want) }) {
				t.Errorf("no reported problem mentions %q:\n%s", tc.want, strings.Join(ps, "\n"))
			}
		})
	}
}

// requireNode skips where node is absent, and fails on a CI runner, where the
// check must not go missing unnoticed.
func requireNode(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("node"); err != nil {
		if os.Getenv("CI") != "" {
			t.Fatal("node is required to run the release-gate detector's fill and is missing on a CI runner")
		}
		t.Skip("node not on PATH; the release-gate detector's fill is checked by running the script")
	}
}

// detectorHarness runs a detector script as the agent harness would: the
// script is a workflow body (top-level await and return, with args, agent,
// parallel and log in scope), so it becomes the body of an async function.
// agent() records the prompt it was given and returns an empty finding, so no
// checker is spawned and the merge still runs.
const detectorHarness = `import { readFileSync } from 'node:fs'
const [scriptPath, manifestPath] = process.argv.slice(2)
const src = readFileSync(scriptPath, 'utf8')
const body = src.replace(/^export (const meta\b)/m, '$1')
if (body === src) throw new Error('the script has no "export const meta" line to unwrap')
const AsyncFunction = (async () => {}).constructor
const prompts = []
const agent = async (prompt, opts) => {
  prompts.push({ label: opts.label, prompt })
  return { item: opts.label, discrepancies: [] }
}
const parallel = thunks => Promise.all(thunks.map(f => f()))
const run = new AsyncFunction('args', 'agent', 'parallel', 'log', body)
await run(JSON.parse(readFileSync(manifestPath, 'utf8')), agent, parallel, () => {})
process.stdout.write(JSON.stringify(prompts))
`

type detectorPrompt struct {
	Label  string `json:"label"`
	Prompt string `json:"prompt"`
}

// runDetector executes script over manifest through node and returns the
// prompt each checker would have been started with.
func runDetector(t *testing.T, script string, manifest []byte) []detectorPrompt {
	t.Helper()
	dir := t.TempDir()
	write := func(name string, data []byte) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, data, 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	harness := write("harness.mjs", []byte(detectorHarness))
	scriptPath := write("detector.js", []byte(script))
	manifestPath := write("manifest.json", manifest)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "node", harness, scriptPath, manifestPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("run the detector through node: %v\n%s", err, stderr.String())
	}
	var prompts []detectorPrompt
	if err := json.Unmarshal(stdout.Bytes(), &prompts); err != nil {
		t.Fatalf("the detector harness did not print the prompts as JSON: %v\n%s", err, stdout.String())
	}
	return prompts
}

// rosterNameSeparator splits a filled roster into its names, whatever the
// detector joins them with: a name is a command or agent page's basename.
var rosterNameSeparator = regexp.MustCompile(`[^A-Za-z0-9_.-]+`)

// detectorPromptProblems runs script over manifest and reports every way the
// prompts it would start checkers with fall short: a checker count other than
// one per brief doc plus one per surface, a ${...} placeholder left in a
// prompt, a prompt that does not open with the pinned context, or a context
// whose filled rosters are not exactly the manifest's. Each kind of problem is
// reported once, naming the first checker that showed it and how many did.
func detectorPromptProblems(t *testing.T, script string, manifest []byte) []string {
	t.Helper()
	m, err := parseReleaseGateManifest(manifest)
	if err != nil {
		t.Fatalf("parse the manifest: %v", err)
	}
	prompts := runDetector(t, script, manifest)

	var order []string
	first, count := map[string]string{}, map[string]int{}
	report := func(label, problem string) {
		if count[problem] == 0 {
			order = append(order, problem)
			first[problem] = label
		}
		count[problem]++
	}

	// The pinned context with each roster placeholder turned into a capture:
	// what a filled prompt must open with, and where its rosters sit.
	pattern := regexp.QuoteMeta(m.Prompt.Context)
	for _, roster := range []string{"commands", "agents"} {
		pattern = strings.Replace(pattern, regexp.QuoteMeta("${"+roster+"}"), "(?P<"+roster+">.*?)", 1)
	}
	filled := regexp.MustCompile(`(?s)\A` + pattern + `\n\n`)
	placeholder := regexp.MustCompile(`\$\{[^}]*\}`)

	for _, p := range prompts {
		if left := placeholder.FindAllString(p.Prompt, -1); len(left) > 0 {
			slices.Sort(left)
			report(p.Label, fmt.Sprintf("the prompt still carries the placeholder(s) %s", strings.Join(slices.Compact(left), ", ")))
		}
		match := filled.FindStringSubmatch(p.Prompt)
		if match == nil {
			report(p.Label, "the prompt does not open with the manifest's pinned context")
			continue
		}
		for _, roster := range []struct {
			name string
			want []string
		}{{"commands", m.Commands}, {"agents", m.Agents}} {
			got := rosterNameSeparator.Split(match[filled.SubexpIndex(roster.name)], -1)
			got = slices.DeleteFunc(got, func(s string) bool { return s == "" })
			if !slices.Equal(got, roster.want) {
				report(p.Label, fmt.Sprintf("the context names %v where the manifest's %s roster is %v", got, roster.name, roster.want))
			}
		}
	}

	var out []string
	if want := len(m.BriefDocs) + len(m.Surfaces); len(prompts) != want {
		out = append(out, fmt.Sprintf("the detector started %d checkers; the manifest's briefDocs (%d) and surfaces (%d) make %d", len(prompts), len(m.BriefDocs), len(m.Surfaces), want))
	}
	for _, problem := range order {
		out = append(out, fmt.Sprintf("checker %s (and %d more of %d): %s", first[problem], count[problem]-1, len(prompts), problem))
	}
	return out
}
