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
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// dryRunPending is every command with a filled-in input whose practice run is
// step 2 of spc-2610301431131636, not yet built. The list only shrinks: a new
// command with a -json input and no --dry-run fails the test, and so does an
// entry here that has gained one.
var dryRunPending = map[string]bool{
	"abcd disembark graveyard":     true,
	"abcd disembark press-release": true,
	"abcd disembark principles":    true,
	"abcd disembark review":        true,
	"abcd launch ship":             true,
	"abcd library ask":             true,
	"abcd library ingest":          true,
	"abcd reading ingest":          true,
	"abcd scribe ingest":           true,
}

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
