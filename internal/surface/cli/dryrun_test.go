package cli

// dryrun_test.go is the tree-wide proof of the practice run
// (itd-2610301428390406, spc-2610301431131636, approach 4). It enumerates the
// live command tree: every visible command with a flag ending -json that it
// reads takes --dry-run, and for each one a valid fixture's practice run
// changes no byte of the repository (its git directory included) or of abcd's
// home and still names what it would write; an invalid fixture's practice run
// refuses exactly as the real run does, also writing nothing; and the paths
// the real run then writes are the ones the practice run named.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/dryrun"
	"github.com/intentdriven/abcd/internal/core/ideate"
	"github.com/intentdriven/abcd/internal/core/intent"
	"github.com/intentdriven/abcd/internal/core/lifeboat"
	"github.com/intentdriven/abcd/internal/core/reading"
	"github.com/intentdriven/abcd/internal/core/scribe"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// dryRunPending is every command with a filled-in input whose practice run is
// not yet built. Step 2 of spc-2610301431131636 emptied it, and it stays empty:
// a new command with a -json input and no --dry-run fails the test, and so
// does an entry here that has gained one.
var dryRunPending = map[string]bool{}

// filledInInputCommands walks the live tree for every visible command with a
// local flag ending -json.
func filledInInputCommands() map[string]*cobra.Command {
	out := map[string]*cobra.Command{}
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		if c.Hidden {
			return
		}
		c.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if strings.HasSuffix(f.Name, "-json") {
				out[c.CommandPath()] = c
			}
		})
		for _, s := range c.Commands() {
			walk(s)
		}
	}
	walk(NewRootCommand())
	return out
}

// TestEveryFilledInInputTakesDryRun: the flag is there on every command that
// reads a filled-in input, and every one of them has a fixture below.
func TestEveryFilledInInputTakesDryRun(t *testing.T) {
	cmds := filledInInputCommands()
	if len(cmds) == 0 {
		t.Fatal("the walk found no command with a -json input")
	}
	for path, c := range cmds {
		has := c.Flags().Lookup("dry-run") != nil
		switch {
		case dryRunPending[path] && has:
			t.Errorf("%s takes --dry-run now: drop it from dryRunPending and give it a fixture", path)
		case dryRunPending[path]:
		case !has:
			t.Errorf("%s reads a filled-in input and takes no --dry-run", path)
		case dryRunCases[path] == nil:
			t.Errorf("%s takes --dry-run and has no fixture in dryRunCases", path)
		}
	}
	for path := range dryRunPending {
		if cmds[path] == nil {
			t.Errorf("dryRunPending names %s, which reads no -json input in the live tree", path)
		}
	}
}

// dryRunFixture is one command's fixture, laid in a temporary repository and
// a temporary home with the working directory in the repository: the
// arguments of a run with a valid input and of one with an invalid input.
type dryRunFixture struct {
	repo, home     string
	valid, invalid []string
}

// dryRunFixtureDir is the fixtures' directory, resolved before any fixture
// moves the working directory into its repository.
var dryRunFixtureDir = func() string {
	dir, err := filepath.Abs(filepath.Join("testdata", "dryrun"))
	if err != nil {
		panic(err)
	}
	return dir
}()

// dryRunTemplate reads a fixture under testdata/dryrun with its placeholders
// filled, and stages it outside the repository, as a host's scratch would be.
func dryRunTemplate(t *testing.T, name string, fill map[string]string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dryRunFixtureDir, name))
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	for k, v := range fill {
		s = strings.ReplaceAll(s, "{{"+k+"}}", v)
	}
	p := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

// requestHashes reads the two policy hashes an emitted request states.
func requestHashes(t *testing.T, repo, requestRel string) map[string]string {
	t.Helper()
	req, err := os.ReadFile(filepath.Join(repo, requestRel))
	if err != nil {
		t.Fatal(err)
	}
	get := func(key string) string {
		m := regexp.MustCompile(`(?m)^- ` + key + `: (\S+)$`).FindStringSubmatch(string(req))
		if m == nil {
			t.Fatalf("the request states no %s:\n%s", key, req)
		}
		return m[1]
	}
	return map[string]string{"RUBRIC": get("rubric_hash"), "PROMPT": get("prompt_hash")}
}

// dryRunCases is one fixture per command that takes --dry-run.
var dryRunCases = map[string]func(t *testing.T) dryRunFixture{
	"abcd ideate record": func(t *testing.T) dryRunFixture {
		repo := ideateRepo(t)
		t.Chdir(repo)
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"ideate", "record", "the-ideate-gate", "--verdict-json", dryRunTemplate(t, "ideate-record.valid.json", nil)},
			invalid: []string{"ideate", "record", "the-ideate-gate", "--verdict-json", dryRunTemplate(t, "ideate-record.invalid.json", nil)},
		}
	},
	"abcd intent prepass": func(t *testing.T) dryRunFixture {
		repo := cliPrepassRepo(t)
		in := cliPrepassInput(t)
		fill := map[string]string{"TYPE": intent.PrepassFindingsType, "DIGEST": in.Digest}
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"intent", "prepass", "itd-500", "--findings-json", dryRunTemplate(t, "intent-prepass.valid.json", fill)},
			invalid: []string{"intent", "prepass", "itd-500", "--findings-json", dryRunTemplate(t, "intent-prepass.invalid.json", fill)},
		}
	},
	"abcd intent consistency ingest": func(t *testing.T) dryRunFixture {
		repo := consistencyCLIRepo(t)
		var em consistencyEmitted
		if err := json.Unmarshal(runCLI(t, "intent", "consistency", "--json"), &em); err != nil {
			t.Fatal(err)
		}
		fill := requestHashes(t, repo, em.RequestPath)
		fill["RECEIPT"] = em.ReceiptID
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"intent", "consistency", "ingest", "--findings-json", dryRunTemplate(t, "intent-consistency-ingest.valid.json", fill)},
			invalid: []string{"intent", "consistency", "ingest", "--findings-json", dryRunTemplate(t, "intent-consistency-ingest.invalid.json", fill)},
		}
	},
	"abcd intent audit ingest": func(t *testing.T) dryRunFixture {
		repo := intentTestRepo(t)
		writeRepoFile(t, repo, ".abcd/development/intents/shipped/itd-10-alpha.md",
			"---\nid: itd-10\nslug: alpha\nspec_id: spc-1\nkind: standalone\n---\n# alpha\n\n"+
				"## Acceptance Criteria\n\n- the ship move parks an OWED stub\n\n## Audit Notes\n")
		var em struct {
			ReceiptID   string `json:"receipt_id"`
			RequestPath string `json:"request_path"`
		}
		if err := json.Unmarshal(runCLI(t, "intent", "audit", "itd-10", "--json"), &em); err != nil {
			t.Fatal(err)
		}
		fill := requestHashes(t, repo, em.RequestPath)
		fill["RECEIPT"] = em.ReceiptID
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"intent", "audit", "ingest", "--verdict-json", dryRunTemplate(t, "intent-audit-ingest.valid.json", fill)},
			invalid: []string{"intent", "audit", "ingest", "--verdict-json", dryRunTemplate(t, "intent-audit-ingest.invalid.json", fill)},
		}
	},
	"abcd disembark graveyard": func(t *testing.T) dryRunFixture {
		t.Setenv("HOME", t.TempDir())
		dir := realPath(t, buildGraveyardLifeboat(t))
		t.Chdir(filepath.Dir(dir))
		return dryRunFixture{repo: dir, home: os.Getenv("HOME"),
			valid:   []string{"disembark", "graveyard", dir, "--lessons-json", dryRunTemplate(t, "disembark-graveyard.valid.json", nil)},
			invalid: []string{"disembark", "graveyard", dir, "--lessons-json", dryRunTemplate(t, "disembark-graveyard.invalid.json", nil)},
		}
	},
	"abcd disembark principles": func(t *testing.T) dryRunFixture {
		t.Setenv("HOME", t.TempDir())
		dir := realPath(t, buildSynthLifeboat(t, synthOpts{}))
		t.Chdir(filepath.Dir(dir))
		return dryRunFixture{repo: dir, home: os.Getenv("HOME"),
			valid:   []string{"disembark", "principles", dir, "--principles-json", dryRunTemplate(t, "disembark-principles.valid.json", nil)},
			invalid: []string{"disembark", "principles", dir, "--principles-json", dryRunTemplate(t, "disembark-principles.invalid.json", nil)},
		}
	},
	"abcd disembark press-release": func(t *testing.T) dryRunFixture {
		t.Setenv("HOME", t.TempDir())
		dir := realPath(t, buildSynthLifeboat(t, synthOpts{briefPR: true}))
		t.Chdir(filepath.Dir(dir))
		return dryRunFixture{repo: dir, home: os.Getenv("HOME"),
			valid:   []string{"disembark", "press-release", dir, "--press-release-json", dryRunTemplate(t, "disembark-press-release.valid.json", nil)},
			invalid: []string{"disembark", "press-release", dir, "--press-release-json", dryRunTemplate(t, "disembark-press-release.invalid.json", nil)},
		}
	},
	"abcd disembark review": func(t *testing.T) dryRunFixture {
		t.Setenv("HOME", t.TempDir())
		dir := realPath(t, buildSynthLifeboat(t, synthOpts{sourceName: "src", coverage: &lifeboat.Summary{Grounded: 7, Blank: 3}}))
		src := realSrcDir(t, "src")
		t.Chdir(filepath.Dir(dir))
		return dryRunFixture{repo: dir, home: os.Getenv("HOME"),
			valid:   []string{"disembark", "review", dir, src, "--review-json", dryRunTemplate(t, "disembark-review.valid.json", nil)},
			invalid: []string{"disembark", "review", dir, src, "--review-json", dryRunTemplate(t, "disembark-review.invalid.json", nil)},
		}
	},
	"abcd launch ship": func(t *testing.T) dryRunFixture {
		t.Setenv("HOME", t.TempDir())
		r := shipReadyRepo(t)
		repo := realPath(t, r.Root())
		t.Chdir(repo)
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"launch", "ship", "--changelog-json", dryRunTemplate(t, "launch-ship.valid.json", nil)},
			invalid: []string{"launch", "ship", "--changelog-json", dryRunTemplate(t, "launch-ship.invalid.json", nil)},
		}
	},
	"abcd library ingest": func(t *testing.T) dryRunFixture {
		repo, _ := memoryStoreFixture(t)
		src, _ := ingestOperands(t, t.TempDir())
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid: []string{"library", "ingest", src, "--keep-original",
				"--pages-json", dryRunTemplate(t, "library-ingest.valid.json", nil)},
			invalid: []string{"library", "ingest", src, "--pages-json", dryRunTemplate(t, "library-ingest.invalid.json", nil)},
		}
	},
	"abcd library ask": func(t *testing.T) dryRunFixture {
		repo, _ := memoryStoreFixture(t)
		// A filed-back answer cites the pages it was synthesised from, so the
		// store is seeded through a real ingest first.
		src, pages := ingestOperands(t, t.TempDir())
		runCLI(t, "library", "ingest", src, "--pages-json", pages)
		const q = "what does the auth rotation page say"
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"library", "ask", q, "--file-back", "--page-json", dryRunTemplate(t, "library-ask.valid.json", nil)},
			invalid: []string{"library", "ask", q, "--file-back", "--page-json", dryRunTemplate(t, "library-ask.invalid.json", nil)},
		}
	},
	"abcd reading ingest": func(t *testing.T) dryRunFixture {
		// The shipped definition is read from this checkout, before the
		// working directory moves into the fixture.
		srcRoot := repoRootFromTest(t)
		t.Setenv("HOME", t.TempDir())
		repo := realPath(t, readingRepo(t))
		t.Chdir(repo)
		runID, manifestHash, def := parkedRunForIngest(t, srcRoot, repo, "detection")
		// The real ingest lays its stage root and the stage's lock before it
		// judges the output, a refused one included, so the fixture starts
		// from a checkout an earlier ingest has already laid them in; the
		// refusal parity below then compares two runs that write nothing.
		stage := filepath.Join(repo, filepath.FromSlash(reading.IngestStageDir))
		if err := os.MkdirAll(stage, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(stage, ".lock"), nil, 0o644); err != nil {
			t.Fatal(err)
		}
		fill := map[string]string{"RUN": runID, "REGIME": def.Regime, "MANIFEST": manifestHash,
			"DEFINITION": def.SHA256, "ASSEMBLER": reading.AssemblerVersion()}
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"reading", "ingest", "--reading-json", dryRunTemplate(t, "reading-ingest.valid.json", fill)},
			invalid: []string{"reading", "ingest", "--reading-json", dryRunTemplate(t, "reading-ingest.invalid.json", fill)},
		}
	},
	"abcd scribe ingest": func(t *testing.T) dryRunFixture {
		repo, item := scribeRepo(t)
		repo = realPath(t, repo)
		t.Chdir(repo)
		ground := "the constraint the reading names is real and binds the verb as shipped"
		disp := filepath.Join(t.TempDir(), "dispositions.md")
		if err := os.WriteFile(disp, []byte(item+": accepted — "+ground+".\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		var asm scribe.AssembleResult
		if err := json.Unmarshal(runCLI(t, "scribe", "assemble", "--run", "rdg-2609250000000001",
			"--dispositions", disp, "--json"), &asm); err != nil {
			t.Fatal(err)
		}
		fill := map[string]string{"TYPE": scribe.OutputType, "CONTEXT": asm.ContextSHA256, "ITEM": item}
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid: []string{"scribe", "ingest", "--scribe-json", dryRunTemplate(t, "scribe-ingest.valid.json", fill),
				"--dispositions", disp},
			invalid: []string{"scribe", "ingest", "--scribe-json", dryRunTemplate(t, "scribe-ingest.invalid.json", fill),
				"--dispositions", disp},
		}
	},
	"abcd docs fidelity record": func(t *testing.T) dryRunFixture {
		t.Setenv("HOME", t.TempDir())
		repo := realPath(t, armedCloseRepo(t))
		t.Chdir(repo)
		return dryRunFixture{repo: repo, home: os.Getenv("HOME"),
			valid:   []string{"docs", "fidelity", "record", "--verdict-json", dryRunTemplate(t, "docs-fidelity-record.valid.json", nil)},
			invalid: []string{"docs", "fidelity", "record", "--verdict-json", dryRunTemplate(t, "docs-fidelity-record.invalid.json", nil)},
		}
	},
}

// dryRunExec runs the binary's own front door (run), so the home is narrowed
// and its ignore file laid exactly as a real invocation does.
func dryRunExec(args ...string) (code int, stdout, stderr string) {
	var out, errb bytes.Buffer
	code = run(args, strings.NewReader(""), &out, &errb)
	return code, out.String(), errb.String()
}

// treeHashes hashes every file under each root, the git directory included.
func treeHashes(t *testing.T, roots ...string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, root := range roots {
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				out[p+"/"] = "dir"
				return nil
			}
			if d.Type()&fs.ModeSymlink != 0 {
				target, _ := os.Readlink(p)
				out[p] = "link:" + target
				return nil
			}
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(data)
			out[p] = hex.EncodeToString(sum[:])
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	return out
}

func hashDiff(before, after map[string]string) []string {
	var out []string
	for p, h := range after {
		if before[p] != h {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// plannedPathRe is a planned path as a pattern over an absolute one: `~/`
// is the home, any other path is under the repository, and dryrun.MintedID
// stands for one minted id.
func plannedPathRe(repo, home, planned string) *regexp.Regexp {
	abs := filepath.Join(repo, filepath.FromSlash(planned))
	if rest, ok := strings.CutPrefix(planned, "~/"); ok {
		abs = filepath.Join(home, filepath.FromSlash(rest))
	}
	parts := strings.Split(abs, dryrun.MintedID)
	for i := range parts {
		parts[i] = regexp.QuoteMeta(parts[i])
	}
	return regexp.MustCompile("^" + strings.Join(parts, "[0-9]{16}") + "$")
}

func TestDryRunWritesNothingAndPlansTheRealWrite(t *testing.T) {
	for path, setup := range dryRunCases {
		t.Run(strings.ReplaceAll(path, " ", "_"), func(t *testing.T) {
			fx := setup(t)
			// Every invocation narrows abcd's home and lays its ignore file on
			// its first run, so the hashes start from a home one command that
			// writes nothing has already settled.
			if code, _, stderr := dryRunExec("--version"); code != 0 {
				t.Fatalf("abcd --version exited %d: %s", code, stderr)
			}
			roots := []string{fx.repo, fx.home}
			before := treeHashes(t, roots...)

			// The valid input: a practice run that names what it would write
			// and writes nothing.
			code, stdout, stderr := dryRunExec(append(append([]string{}, fx.valid...), "--dry-run", "--json")...)
			if code != 0 {
				t.Fatalf("the practice run exited %d:\n%s%s", code, stdout, stderr)
			}
			var got struct {
				DryRun     bool            `json:"dry_run"`
				WouldWrite []dryrun.Write  `json:"would_write"`
				WouldDo    []dryrun.Effect `json:"would_do"`
			}
			if err := json.Unmarshal([]byte(stdout), &got); err != nil {
				t.Fatalf("the practice run's --json is not an object: %v\n%s", err, stdout)
			}
			if !got.DryRun || len(got.WouldWrite)+len(got.WouldDo) == 0 {
				t.Fatalf("the practice run plans nothing, or does not say it is one:\n%s", stdout)
			}
			if d := hashDiff(before, treeHashes(t, roots...)); len(d) != 0 {
				t.Fatalf("the practice run changed %v", d)
			}
			if code, text, stderr := dryRunExec(append(append([]string{}, fx.valid...), "--dry-run")...); code != 0 ||
				!strings.Contains(text, "practice run") || !strings.Contains(text, "would ") {
				t.Fatalf("the practice run's text exited %d and reads:\n%s%s", code, text, stderr)
			}
			if d := hashDiff(before, treeHashes(t, roots...)); len(d) != 0 {
				t.Fatalf("the practice run's text form changed %v", d)
			}

			// The invalid input: the practice run refuses as the real run does,
			// in the same words and with the same exit code, and neither writes.
			dcode, dout, derr := dryRunExec(append(append([]string{}, fx.invalid...), "--dry-run")...)
			if d := hashDiff(before, treeHashes(t, roots...)); len(d) != 0 {
				t.Fatalf("the refused practice run changed %v", d)
			}
			rcode, rout, rerr := dryRunExec(fx.invalid...)
			if dcode == 0 || dcode != rcode || derr != rerr || dout != rout {
				t.Fatalf("the practice run's refusal is not the real run's:\npractice %d: %s%s\nreal %d: %s%s", dcode, dout, derr, rcode, rout, rerr)
			}
			if d := hashDiff(before, treeHashes(t, roots...)); len(d) != 0 {
				t.Fatalf("the refused real run changed %v", d)
			}

			// The real write lays exactly the paths the practice run planned.
			if code, out, stderr := dryRunExec(fx.valid...); code != 0 {
				t.Fatalf("the real run exited %d:\n%s%s", code, out, stderr)
			}
			gitDir := filepath.Join(fx.repo, ".git") + string(filepath.Separator)
			var wrote []string
			for _, p := range hashDiff(before, treeHashes(t, roots...)) {
				if !strings.HasSuffix(p, "/") && !strings.HasPrefix(p, gitDir) {
					wrote = append(wrote, p)
				}
			}
			matched := make([]bool, len(got.WouldWrite))
			for _, p := range wrote {
				// A path the real run removed is planned as a remove effect.
				if _, err := os.Lstat(p); os.IsNotExist(err) {
					hit := false
					for _, e := range got.WouldDo {
						if e.Effect == "remove" && plannedPathRe(fx.repo, fx.home, e.Target).MatchString(p) {
							hit = true
						}
					}
					if !hit {
						t.Errorf("the real run removed %s, which the practice run did not plan (planned %+v)", p, got.WouldDo)
					}
					continue
				}
				hit := false
				for i, w := range got.WouldWrite {
					if !matched[i] && plannedPathRe(fx.repo, fx.home, w.Path).MatchString(p) {
						matched[i], hit = true, true
						break
					}
				}
				if !hit {
					t.Errorf("the real run wrote %s, which the practice run did not plan (planned %+v)", p, got.WouldWrite)
				}
			}
			for i, ok := range matched {
				if !ok {
					t.Errorf("the practice run planned %s, which the real run did not write (wrote %v)", got.WouldWrite[i].Path, wrote)
				}
			}
		})
	}
}

// TestDryRunIdeatePublicPlansTheRecordAndItsPointer: a public verdict's
// practice run names both committed artefacts, the pointer with the id the
// decision writer mints, and shows the record's opening lines.
func TestDryRunIdeatePublicPlansTheRecordAndItsPointer(t *testing.T) {
	repo := ideateRepo(t)
	t.Chdir(repo)
	before := cliTreeHash(t, repo)
	out := string(runCLI(t, "ideate", "record", "the-ideate-gate", "--public", "--dry-run",
		"--verdict-json", writeVerdict(t, ideateVerdictJSON("itd-104"))))
	for _, want := range []string{
		"abcd ideate record — practice run",
		"would create " + ideate.ResearchRelDir + "/",
		"would create .abcd/work/decisions/dcn-" + dryrun.MintedID + "-ideate-the-ideate-gate",
		"would open:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the practice run does not say %q:\n%s", want, out)
		}
	}
	if d := cliTreeDiff(before, cliTreeHash(t, repo)); len(d) != 0 {
		t.Fatalf("the practice run wrote %v", d)
	}
}

// TestDryRunPrepassBareIsRefused: the bare pre-pass writes nothing already,
// so --dry-run without an input to check is refused rather than ignored.
func TestDryRunPrepassBareIsRefused(t *testing.T) {
	cliPrepassRepo(t)
	_, err := runCLIErr(t, "intent", "prepass", "itd-500", "--dry-run")
	if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), "--findings-json") {
		t.Fatalf("bare prepass --dry-run = %d %v; want exit 2 naming --findings-json", code, err)
	}
}

// TestDryRunStep2OperandRefusals (spc-2610301431131636 step 2): a --dry-run
// beside a run that reads no filled-in input, or beside a flag that would
// write, is refused with exit 2 rather than ignored, and writes nothing.
func TestDryRunStep2OperandRefusals(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	r := shipReadyRepo(t)
	repo := realPath(t, r.Root())
	t.Chdir(repo)
	payload := dryRunTemplate(t, "launch-ship.valid.json", nil)
	before := treeHashes(t, repo)
	for name, tc := range map[string]struct {
		args []string
		want string
	}{
		"ship's emit step":             {[]string{"launch", "ship", "--dry-run"}, "--changelog-json"},
		"ship with a fetch":            {[]string{"launch", "ship", "--changelog-json", payload, "--dry-run", "--fetch-baseline"}, "--fetch-baseline"},
		"an ask filing nothing":        {[]string{"library", "ask", "a question", "--dry-run"}, "--file-back"},
		"a reading sent to a provider": {[]string{"reading", "ingest", "--dispatch", "rdg-2609250000000001", "--dry-run"}, "--reading-json"},
	} {
		_, err := runCLIErr(t, tc.args...)
		if code := exitCodeOf(err); code != 2 || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: %v = exit %d (%v); want exit 2 naming %s", name, tc.args, code, err, tc.want)
		}
	}
	if d := hashDiff(before, treeHashes(t, repo)); len(d) != 0 {
		t.Fatalf("a refused --dry-run changed %v", d)
	}
}
