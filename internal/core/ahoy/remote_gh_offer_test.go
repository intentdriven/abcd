package ahoy

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/tools"
)

// ghOfferRig is a managed repository whose PATH holds git and nothing else, so
// gh is missing, and whose tool installer is a fake: a "brew install gh" it is
// told to run writes a stand-in gh onto that PATH (answering with body) and
// records the argv, so a test sees exactly what would have run and what the
// verb did once gh was there.
type ghOfferRig struct {
	repo  string
	path  string // the only PATH directory
	calls [][]string
	ghLog string
	fail  bool // the fake brew exits non-zero
}

func newGHOfferRig(t *testing.T, body string) *ghOfferRig {
	t.Helper()
	setupHermetic(t)
	repo := managedRepoWithOrigin(t, "https://github.com/example-org/example-repo")
	gitBin, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git unavailable")
	}
	r := &ghOfferRig{repo: repo, path: t.TempDir()}
	r.ghLog = filepath.Join(t.TempDir(), "gh.log")
	if err := os.Symlink(gitBin, filepath.Join(r.path, "git")); err != nil {
		t.Fatal(err)
	}
	// brew is a real file the installer's admit can resolve; the fake Run never
	// executes it.
	if err := os.WriteFile(filepath.Join(r.path, "brew"), []byte("#!/bin/sh\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	bodyFile := filepath.Join(t.TempDir(), "get.json")
	if err := os.WriteFile(bodyFile, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", r.path)

	prev := newToolInstaller
	newToolInstaller = func(guard string) *tools.Installer {
		in := tools.Default(guard)
		in.Getenv = func(string) string { return "" }
		in.Run = func(_ context.Context, argv []string) ([]byte, error) {
			r.calls = append(r.calls, argv)
			if filepath.Base(argv[0]) != "brew" {
				return []byte("gh version 2.0.0 (fake)\n"), nil
			}
			if r.fail {
				return []byte("Error: gh: download failed\n"), errors.New("exit status 1")
			}
			script := "#!/bin/sh\necho \"$*\" >> '" + r.ghLog + "'\n/bin/cat '" + bodyFile + "'\n"
			if err := os.WriteFile(filepath.Join(r.path, "gh"), []byte(script), 0o755); err != nil {
				return nil, err
			}
			return []byte("==> Pouring gh\n"), nil
		}
		return in
	}
	t.Cleanup(func() { newToolInstaller = prev })
	return r
}

func (r *ghOfferRig) ghRan(t *testing.T) bool {
	t.Helper()
	_, err := os.Stat(r.ghLog)
	return err == nil
}

// asked records what the confirmation was shown and answers ans.
func asked(seen *[]tools.Explanation, ans tools.Answer) tools.Confirm {
	return func(e tools.Explanation) tools.Answer {
		*seen = append(*seen, e)
		return ans
	}
}

// TestRemoteApplyOffersGhAndRunsTheStepOnYes is the gh offer's yes (the DQ3
// ruling, itd-63 criterion 2): a missing gh is put to the question with the
// registry's explanation, the yes runs the registry's step, the verb says what
// it ran, and the settings are then read through the gh it installed.
func TestRemoteApplyOffersGhAndRunsTheStepOnYes(t *testing.T) {
	r := newGHOfferRig(t, bothEnabled)
	var seen []tools.Explanation
	res, err := RemoteApply(r.repo, confirmingPrompter{}, asked(&seen, tools.Answer{Yes: true, Why: "answered yes at the terminal"}))
	if err != nil {
		t.Fatal(err)
	}
	if len(seen) != 1 || seen[0].Tool != "gh" || seen[0].StepText() != "brew install gh" {
		t.Fatalf("the offer was not put to the question with gh's explanation: %+v", seen)
	}
	if len(r.calls) == 0 || strings.Join(r.calls[0][1:], " ") != "install gh" {
		t.Fatalf("the yes did not run the registry's step: %v", r.calls)
	}
	if res.Status == "refused" {
		t.Fatalf("the verb refused after gh was installed: %+v", res)
	}
	if !r.ghRan(t) {
		t.Fatal("the settings were not read through the installed gh")
	}
	notes := strings.Join(res.Notes, "\n")
	for _, want := range []string{"gh: ran brew install gh — installed, and verified with gh --version", "gh auth login"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes lack %q:\n%s", want, notes)
		}
	}
}

// TestRemoteApplyGhDeclinedRunsNothingAndShowsTheStep is every no: an answer
// of no, and a caller that asked no one (off a terminal the front door answers
// no itself). Nothing runs, gh is never reached, and the refusal carries the
// explanation with the exact command to run by hand.
func TestRemoteApplyGhDeclinedRunsNothingAndShowsTheStep(t *testing.T) {
	for name, confirm := range map[string]tools.Confirm{
		"no":     func(tools.Explanation) tools.Answer { return tools.Answer{Why: "answered no at the terminal"} },
		"no-one": nil,
		"no-tty": func(tools.Explanation) tools.Answer { return tools.Answer{Why: "no terminal to ask at"} },
	} {
		t.Run(name, func(t *testing.T) {
			r := newGHOfferRig(t, bothEnabled)
			res, err := RemoteApply(r.repo, confirmingPrompter{}, confirm)
			if err != nil {
				t.Fatal(err)
			}
			if len(r.calls) != 0 || r.ghRan(t) {
				t.Fatalf("a no ran something: installs %v, gh ran %v", r.calls, r.ghRan(t))
			}
			if res.Status != "refused" {
				t.Fatalf("status = %q, want refused", res.Status)
			}
			notes := strings.Join(res.Notes, "\n")
			for _, want := range []string{"gh not installed (", "install step (Homebrew): brew install gh", "the settings on GitHub stay unread and unchanged"} {
				if !strings.Contains(notes, want) {
					t.Errorf("the notes lack %q:\n%s", want, notes)
				}
			}
		})
	}
}

// TestRemoteApplyGhInstallFailureIsLoudAndStops: a failed step is reported with
// its output, and the verb goes no further: no gh call, no mirror written.
func TestRemoteApplyGhInstallFailureIsLoudAndStops(t *testing.T) {
	r := newGHOfferRig(t, bothEnabled)
	r.fail = true
	res, err := RemoteApply(r.repo, confirmingPrompter{}, func(tools.Explanation) tools.Answer { return tools.Answer{Yes: true} })
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "refused" || r.ghRan(t) || len(res.Writes) != 0 {
		t.Fatalf("a failed install went on: status %q, gh ran %v, writes %v", res.Status, r.ghRan(t), res.Writes)
	}
	notes := strings.Join(res.Notes, "\n")
	for _, want := range []string{"ran brew install gh — it failed", "download failed", "the settings on GitHub stay unread and unchanged"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes lack %q:\n%s", want, notes)
		}
	}
}

// TestRemoteReadNeverOffersGh: the read is bare ahoy's, which writes nothing,
// so a missing gh is explained and pointed at the verb that offers it, never
// installed.
func TestRemoteReadNeverOffersGh(t *testing.T) {
	r := newGHOfferRig(t, bothEnabled)
	res, err := RemoteRead(r.repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.calls) != 0 {
		t.Fatalf("the read ran an install: %v", r.calls)
	}
	notes := strings.Join(res.Notes, "\n")
	for _, want := range []string{"brew install gh", "abcd ahoy remote apply offers to install it"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the read's refusal lacks %q:\n%s", want, notes)
		}
	}
}
