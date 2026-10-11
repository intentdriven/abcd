package lint_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"testing"
)

const queueSweepWorkflow = ".github/workflows/queue-sweep.yml"

// TestQueueSweepCancelsOnlyRunsWhoseQueueRefIsGone runs queue-sweep.yml's own
// script, read from the committed workflow, against a fake gh that plays the
// Actions and git refs APIs (iss-2610110613372021).
//
// The sweep exists to cancel the runs of merge-queue entries the queue has
// already dropped, and the one thing it must never do is cancel a run of a
// LIVE entry: GitHub reads a cancelled required check as a failure and ejects
// the entry (iss-2609020716560392). So the cases are mostly the ones it must
// leave alone — a live ref, a lookup that failed, a branch that is not a queue
// ref, a ref re-created before the force-cancel, and a token that cannot read
// even the default branch, where every 404 proves nothing.
func TestQueueSweepCancelsOnlyRunsWhoseQueueRefIsGone(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	step := findStep(t, readRepoFile(t, root, queueSweepWorkflow), queueSweepWorkflow, "force-cancel")
	script := stepScript(t, step)
	workflows, ok := step.env["WORKFLOWS"]
	if !ok {
		t.Fatalf("%s: the sweep step declares no WORKFLOWS env", queueSweepWorkflow)
	}

	const q = "gh-readonly-queue/main/"
	type world struct {
		refs   map[string][]string // branch -> successive lookup answers ("none": no status line)
		runs   map[string]string   // "<workflow>.<status>" -> TSV listing
		states map[string]string   // run id -> status after the cancel
	}
	base := func() world {
		return world{
			refs: map[string][]string{
				"main":          {"200"},
				q + "pr-1-aaa":  {"200"},
				q + "pr-2-bbb":  {"404"},
				q + "pr-3-ccc":  {"404"},
				q + "pr-4-ddd":  {"500"},
				q + "pr-5-eee":  {"none"},
				q + "pr-6-fff":  {"404", "200"},
				q + "pr-7-x?y":  {"404"},
				"feature/other": {"404"},
			},
			runs: map[string]string{
				"ci.yml.queued": "101\t" + q + "pr-1-aaa\n" + "102\t" + q + "pr-2-bbb\n" + "103\t" + q + "pr-3-ccc\n" +
					"104\t" + q + "pr-4-ddd\n" + "105\t" + q + "pr-5-eee\n" + "106\t" + q + "pr-6-fff\n" +
					"107\t" + q + "pr-7-x?y\n" + "108\tfeature/other\n",
				// The same run listed under a second status is cancelled once.
				"ci.yml.in_progress":            "102\t" + q + "pr-2-bbb\n",
				"attribution.yml.waiting":       "201\t" + q + "pr-2-bbb\n",
				"external-review.yml.requested": "301\t" + q + "pr-1-aaa\n",
			},
			states: map[string]string{"102": "completed", "103": "queued", "106": "queued", "201": "completed"},
		}
	}

	cases := []struct {
		name   string
		world  func() world
		cancel []string
		force  []string
		say    string
	}{
		{"only the runs of gone refs are cancelled", base,
			[]string{"102", "103", "106", "201"}, []string{"103"}, "::notice"},
		{"a token that cannot read the default branch cancels nothing", func() world {
			w := base()
			w.refs["main"] = []string{"404"}
			return w
		}, nil, nil, "::warning"},
		{"nothing to sweep", func() world {
			w := base()
			w.runs = map[string]string{"ci.yml.queued": "101\t" + q + "pr-1-aaa\n"}
			return w
		}, nil, nil, "no run of a dropped queue entry"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := c.world()
			state := t.TempDir()
			for _, d := range []string{"ref", "runs", "status"} {
				if err := os.MkdirAll(filepath.Join(state, d), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for branch, answers := range w.refs {
				writeSweepFile(t, filepath.Join(state, "ref", strings.ReplaceAll(branch, "/", "_")), strings.Join(answers, "\n")+"\n")
			}
			for key, listing := range w.runs {
				writeSweepFile(t, filepath.Join(state, "runs", key), listing)
			}
			for id, st := range w.states {
				writeSweepFile(t, filepath.Join(state, "status", id), st+"\n")
			}
			bin := t.TempDir()
			fakeTool(t, bin, "gh", fakeSweepGH)
			out, code := runScript(t, script, state, bin, map[string]string{
				"STATE":          state,
				"REPO":           "intentdriven/abcd",
				"DEFAULT_BRANCH": "main",
				"WORKFLOWS":      workflows,
				"GRACE_SECONDS":  "0",
				"GH_TOKEN":       "fake",
			})
			if code != 0 {
				t.Fatalf("the sweep exited %d:\n%s", code, out)
			}
			raw, _ := os.ReadFile(filepath.Join(state, "calls"))
			var cancel, force []string
			for _, l := range strings.Split(string(raw), "\n") {
				if m := regexp.MustCompile(`^POST repos/intentdriven/abcd/actions/runs/(\d+)/(cancel|force-cancel)$`).FindStringSubmatch(l); m != nil {
					if m[2] == "cancel" {
						cancel = append(cancel, m[1])
					} else {
						force = append(force, m[1])
					}
				} else if strings.HasPrefix(l, "POST ") {
					t.Errorf("the sweep made an unexpected write: %s", l)
				}
			}
			sort.Strings(cancel)
			sort.Strings(force)
			if !slices.Equal(cancel, c.cancel) {
				t.Errorf("cancelled %q; want %q\n%s", cancel, c.cancel, out)
			}
			if !slices.Equal(force, c.force) {
				t.Errorf("force-cancelled %q; want %q\n%s", force, c.force, out)
			}
			if !strings.Contains(out, c.say) {
				t.Errorf("the sweep's output does not say %q:\n%s", c.say, out)
			}
		})
	}
}

// fakeSweepGH plays the slice of `gh api` the sweep calls. Every call is logged as
// "<METHOD> <path>"; a ref lookup answers the next status queued for that
// branch (the last one repeats), a run listing prints the TSV queued for its
// workflow and status (the sweep's --jq is the fake's to honour, so the
// listing is already in the shape the --jq yields), and a run read prints the
// status the run reached after its cancel.
const fakeSweepGH = `log="$STATE/calls"
[ "$1" = api ] && shift
method=GET; url=""
while [ $# -gt 0 ]; do
  case "$1" in
    -i | --paginate) ;;
    -X) method=$2; shift ;;
    --jq) shift ;;
    *) url=$1 ;;
  esac
  shift
done
echo "$method $url" >>"$log"
case "$url" in
  repos/*/git/ref/heads/*)
    branch=${url#*/git/ref/heads/}
    f="$STATE/ref/$(printf '%s' "$branch" | tr '/' '_')"
    [ -f "$f" ] || { echo "HTTP/2.0 404 Not Found"; exit 1; }
    code=$(head -n 1 "$f")
    if [ "$(wc -l <"$f")" -gt 1 ]; then tail -n +2 "$f" >"$f.next"; cat "$f.next" >"$f"; fi
    [ "$code" = none ] && exit 1
    echo "HTTP/2.0 $code X"
    [ "$code" = 200 ]
    ;;
  repos/*/actions/workflows/*/runs*)
    wf=${url#*/actions/workflows/}; wf=${wf%%/*}
    st=${url#*status=}; st=${st%%&*}
    [ -f "$STATE/runs/$wf.$st" ] && cat "$STATE/runs/$wf.$st"
    exit 0
    ;;
  repos/*/actions/runs/*/cancel | repos/*/actions/runs/*/force-cancel) exit 0 ;;
  repos/*/actions/runs/*)
    id=${url##*/}
    cat "$STATE/status/$id" 2>/dev/null || echo in_progress
    ;;
  *) echo "fake gh: unexpected call $url" >&2; exit 2 ;;
esac
`

func writeSweepFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestQueueSweepIsScopedAndNeverRequired holds the sweep's shape: it runs on
// merge_group (when a rebuild has just happened) and by hand, on an ubuntu
// runner, with a token that can write to Actions and nothing else; its script
// takes event data through env only; it covers every other workflow the queue
// waits on; and no context of it is a required check, so a slow or failed
// sweep can never hold the entry it reports on.
func TestQueueSweepIsScopedAndNeverRequired(t *testing.T) {
	root := filepath.Join("..", "..", "..")
	wf := readRepoFile(t, root, queueSweepWorkflow)

	triggers := workflowTriggers(wf)
	if !slices.Equal(triggers, []string{"merge_group", "workflow_dispatch"}) {
		t.Errorf("%s triggers on %q; want merge_group and workflow_dispatch alone", queueSweepWorkflow, triggers)
	}
	if !regexp.MustCompile(`(?m)^permissions: \{\}\s*$`).MatchString(wf) {
		t.Errorf("%s does not drop every workflow-level permission (`permissions: {}`)", queueSweepWorkflow)
	}
	job, ok := workflowJobBlock(wf, "sweep")
	if !ok {
		t.Fatalf("%s defines no sweep job", queueSweepWorkflow)
	}
	if r, _ := jobKey(job, "runs-on"); r != "ubuntu-latest" {
		t.Errorf("%s: the sweep runs on %q; want ubuntu-latest, never the scarce macOS runner", queueSweepWorkflow, r)
	}
	perms := regexp.MustCompile(`(?m)^      ([a-z-]+): ([a-z]+)\s*$`).FindAllStringSubmatch(job, -1)
	var granted []string
	for _, p := range perms {
		granted = append(granted, p[1]+": "+p[2])
	}
	if !slices.Equal(granted, []string{"actions: write", "contents: read"}) {
		t.Errorf("%s: the sweep job is granted %q; want actions: write and contents: read alone", queueSweepWorkflow, granted)
	}
	step := findStep(t, wf, queueSweepWorkflow, "force-cancel")
	if strings.Contains(stepScript(t, step), "${{") {
		t.Errorf("%s: the sweep interpolates an expression into its script; pass it through env", queueSweepWorkflow)
	}

	// Every other workflow that runs on merge_group is one whose runs the
	// sweep must reach.
	listed := strings.Fields(step.env["WORKFLOWS"])
	entries, err := os.ReadDir(filepath.Join(root, ".github", "workflows"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		name := e.Name()
		if name == filepath.Base(queueSweepWorkflow) || !strings.HasSuffix(name, ".yml") {
			continue
		}
		onQueue := triggersOnMergeGroup(readRepoFile(t, root, ".github/workflows/"+name))
		if onQueue != slices.Contains(listed, name) {
			t.Errorf("%s: %s runs on merge_group=%v but the sweep lists it=%v; the sweep covers exactly "+
				"the workflows the queue starts", queueSweepWorkflow, name, onQueue, !onQueue)
		}
	}

	var ruleset struct {
		Rules []struct {
			Parameters struct {
				RequiredStatusChecks []struct {
					Context string `json:"context"`
				} `json:"required_status_checks"`
			} `json:"parameters"`
		} `json:"rules"`
	}
	if err := json.Unmarshal([]byte(readRepoFile(t, root, rulesetMirror)), &ruleset); err != nil {
		t.Fatal(err)
	}
	for _, r := range ruleset.Rules {
		for _, c := range r.Parameters.RequiredStatusChecks {
			if c.Context == "sweep" || strings.HasPrefix(c.Context, "queue-sweep") {
				t.Errorf("%s requires %q; the sweep must never be a required check", rulesetMirror, c.Context)
			}
		}
	}
}
