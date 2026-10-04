package cli

import (
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/testsecret"
)

// The guided connect's front door (spc-2610031241482088, step 3): one turn a
// run, the resume object passed back on stdin as the page passes it, the
// done turn's command and paths, and the command, pasted, doing what the
// guide said it would.

// keylessService is a stand-in that lists ids with no key and answers every
// completion, counting the requests it was sent.
type keylessService struct {
	srv *httptest.Server
	mu  sync.Mutex
	got []string
}

func newKeylessService(t *testing.T, ids []string) *keylessService {
	t.Helper()
	s := &keylessService{}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.got = append(s.got, r.Method+" "+r.URL.Path)
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models"):
			data := make([]map[string]string, len(ids))
			for i, id := range ids {
				data[i] = map[string]string{"id": id}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/chat/completions"):
			var req struct {
				Model string `json:"model"`
			}
			_ = json.NewDecoder(r.Body).Decode(&req)
			_, _ = io.WriteString(w, completionReply(req.Model))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *keylessService) base() string { return s.srv.URL + "/v1" }

func (s *keylessService) requests() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.got...)
}

// guideJSON is one turn as the page reads it.
type guideJSON struct {
	Ask     *question.Ask   `json:"ask"`
	Resume  json.RawMessage `json:"resume"`
	Stopped string          `json:"stopped"`
	Done    *struct {
		Command         string   `json:"command"`
		Writes          []string `json:"writes"`
		PicksInTerminal bool     `json:"picks_in_terminal"`
	} `json:"done"`
	Tool *hostQuestionCall `json:"tool"`
}

// guideTurns runs the guide the way the page does: the first turn with
// first, then each answer with the last turn's resume object on stdin. It
// returns every turn's JSON (raw and decoded), the text the last turn prints
// without --json, and everything every run printed, stdout and stderr.
func guideTurns(t *testing.T, first []string, answers ...string) (raw []string, turns []guideJSON, lastText, said string) {
	t.Helper()
	var all strings.Builder
	run := func(stdin string, asJSON bool, args ...string) string {
		if asJSON {
			args = append(args, "--json")
		}
		stdout, stderr, err, _ := runConnect(t, stdin, args...)
		if err != nil {
			t.Fatalf("ahoy connect %q: %v\n%s", args, err, stderr)
		}
		all.WriteString(stdout + stderr)
		return stdout
	}
	out := run("", true, append([]string{"--guide"}, first...)...)
	for i := 0; ; i++ {
		var turn guideJSON
		if err := json.Unmarshal([]byte(out), &turn); err != nil {
			t.Fatalf("turn %d is not JSON: %v\n%s", i, err, out)
		}
		raw, turns = append(raw, out), append(turns, turn)
		if i == len(answers) {
			return raw, turns, lastText, all.String()
		}
		args := []string{"--guide", "--resume", "-", "--answer", answers[i]}
		if i == len(answers)-1 {
			lastText = run(string(turn.Resume), false, args...)
		}
		out = run(string(turn.Resume), true, args...)
	}
}

// commandArgs splits the printed command into the arguments after
// "abcd ahoy connect", undoing the guide's single quotes.
func commandArgs(t *testing.T, command string) []string {
	t.Helper()
	const lead = "abcd ahoy connect "
	if !strings.HasPrefix(command, lead) {
		t.Fatalf("the command does not start %q: %s", lead, command)
	}
	var out []string
	for _, w := range strings.Fields(strings.TrimPrefix(command, lead)) {
		if strings.HasPrefix(w, "'") {
			w = strings.ReplaceAll(strings.Trim(w, "'"), `'\''`, "'")
		}
		out = append(out, w)
	}
	return out
}

// TestGuideCannotRun (G4, decision 3): no flag runs the command. --guide
// with a flag that sets the connection up is refused before anything is
// read, sent or written, and so are --resume and --answer without --guide.
func TestGuideCannotRun(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	svc := newKeylessService(t, []string{"vendor/coder"})
	for _, extra := range [][]string{{"--home", "none"}, {"--home", "abcd"}, {"--model", "vendor/coder"}, {"--key", "k"}, {"--env", "X_API_KEY"}} {
		args := append([]string{"local", "--guide", "--base-url", svc.base()}, extra...)
		_, _, err, code := runConnect(t, "", args...)
		if code != 2 || err == nil || !strings.Contains(err.Error(), "the guide never does") {
			t.Errorf("%q = %v (exit %d), want refused", args, err, code)
		}
	}
	for _, args := range [][]string{{"local", "--resume", "-"}, {"local", "--answer", "x"}} {
		if _, _, err, code := runConnect(t, "{}", args...); code != 2 || err == nil || !strings.Contains(err.Error(), "add --guide") {
			t.Errorf("%q = %v (exit %d), want refused naming --guide", args, err, code)
		}
	}
	if got := svc.requests(); len(got) != 0 {
		t.Fatalf("a refused guide sent %q", got)
	}
	if got := machineWrites(t); len(got) != 0 {
		t.Fatalf("a refused guide wrote %q", got)
	}
}

// TestGuideShowsTheCommandAndEveryPath (G4): for each home the done turn's
// text form prints the command on a line of its own and then each path, and
// the paths are the ones a run of that command, pasted, reports writing.
func TestGuideShowsTheCommandAndEveryPath(t *testing.T) {
	for _, tc := range []struct {
		home    string
		answers []string
	}{
		{"none", []string{"lookup", "vendor/coder", "none"}},
		{"abcd", []string{"lookup", "vendor/coder", "key", "abcd"}},
		{"external", []string{"lookup", "vendor/coder", "key", "external", "GUIDE_TEST_API_KEY"}},
	} {
		t.Run(tc.home, func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			svc := newKeylessService(t, []string{"vendor/coder", "vendor/other"})
			// Only the test's own variable ends in _API_KEY here, whatever
			// the environment the tests run in holds.
			for _, kv := range os.Environ() {
				if n, _, _ := strings.Cut(kv, "="); strings.HasSuffix(n, "_API_KEY") {
					t.Setenv(n, "")
					os.Unsetenv(n)
				}
			}
			const value = "an-external-value-not-a-key"
			t.Setenv("GUIDE_TEST_API_KEY", value)
			raw, turns, text, said := guideTurns(t, []string{"--base-url", svc.base()}, tc.answers...)
			done := turns[len(turns)-1].Done
			if done == nil {
				t.Fatalf("no done turn: %+v", turns[len(turns)-1])
			}
			// The variable's name is offered from this process's environment,
			// and its value reaches no turn, no text and no error.
			if tc.home == "external" {
				q := turns[len(turns)-2].Ask.Questions[0]
				if q.ID != "env" || len(q.Options) != 1 || q.Options[0].Value != "GUIDE_TEST_API_KEY" {
					t.Fatalf("the variable question is %q offering %+v; want GUIDE_TEST_API_KEY offered", q.ID, q.Options)
				}
			}
			if strings.Contains(strings.Join(raw, "\n")+text+said, value) {
				t.Fatal("a variable's value reached a turn, the text or stderr")
			}
			lines := strings.Split(strings.TrimSpace(text), "\n")
			want := append([]string{done.Command, "When it runs, this command writes:"}, func() []string {
				var ps []string
				for _, p := range done.Writes {
					ps = append(ps, "  "+p)
				}
				return ps
			}()...)
			if len(lines) < len(want) || !reflect.DeepEqual(lines[:len(want)], want) {
				t.Fatalf("the text form:\n%s\nwant it to open:\n%s", text, strings.Join(want, "\n"))
			}
			if got := machineWrites(t); len(got) != 0 {
				t.Fatalf("the guide wrote %q", got)
			}

			atTerminal(t, connectKey, nil)
			stdout, stderr, err, _ := runConnect(t, "", append(commandArgs(t, done.Command), "--json")...)
			if err != nil {
				t.Fatalf("the printed command: %v\n%s", err, stderr)
			}
			var res struct {
				Wrote []string `json:"wrote"`
			}
			if err := json.Unmarshal([]byte(stdout), &res); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(res.Wrote, done.Writes) {
				t.Fatalf("the command wrote %q; the guide listed %q", res.Wrote, done.Writes)
			}
		})
	}
}

// TestGuideToolPassesTheQuestionGuard: each turn's tool member, the input the
// page hands the host's question tool, is what the guard's decoder reads and
// passes the asking limits, with the two to four options the host's tool
// takes: a typed question with no other option carries the row pointer, and
// choosing it asks the question again.
func TestGuideToolPassesTheQuestionGuard(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	svc := newKeylessService(t, []string{"vendor/coder"})
	raw, turns, _, _ := guideTurns(t, nil, question.TypedRowLabel, "http://192.0.2.1/v1", svc.base(), "lookup", "cod", "vendor/coder", "key", "external", "GUIDE_API_KEY")
	asked := 0
	for i, turn := range turns {
		if turn.Tool == nil {
			continue
		}
		asked++
		var in struct {
			Tool struct {
				Questions json.RawMessage `json:"questions"`
			} `json:"tool"`
		}
		if err := json.Unmarshal([]byte(raw[i]), &in); err != nil {
			t.Fatal(err)
		}
		fields, err := decodeQuestions(in.Tool.Questions)
		if err != nil {
			t.Fatalf("turn %d: the guard cannot read the tool input: %v", i, err)
		}
		for _, f := range question.CheckLimits(fields, question.Default, question.Addressee{Person: question.Facilitator}) {
			t.Errorf("turn %d: %s", i, f.String())
		}
	}
	if asked < 6 {
		t.Fatalf("only %d turns asked", asked)
	}
	first := turns[0].Tool.Questions[0]
	if first.Options[0].Label != question.TypedRowLabel || len(first.Options) != 2 {
		t.Fatalf("the address question's options = %+v", first.Options)
	}
	if q := turns[1].Ask.Questions[0]; q.ID != "address" || !strings.Contains(q.Material[0].Text, "type the answer itself") {
		t.Fatalf("choosing the row pointer did not ask again: %+v", q)
	}
}

// TestKeyCanaryAppearsOnlyInItsHome (G6): a known key, run through the
// guide and then through the printed command against a stand-in that lists
// only for a key, echoes the Authorization header into its error bodies and
// lists one id carrying the key, appears in no turn, no output, no error and
// no file but the abcd credential file.
func TestKeyCanaryAppearsOnlyInItsHome(t *testing.T) {
	canary := "canary-" + strings.Repeat("9c", 10) + "-not-a-key"
	for _, tc := range []struct {
		name     string
		chatCode int
	}{{"the run succeeds", http.StatusOK}, {"the completion fails, echoing the key", http.StatusUnauthorized}} {
		t.Run(tc.name, func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			svc := newListingService(t, []string{"vendor/" + canary, "vendor/coder-large"}, tc.chatCode)
			raw, turns, text, _ := guideTurns(t, []string{"example", "--base-url", svc.base()}, "lookup", "abcd")
			done := turns[len(turns)-1].Done
			if done == nil || !done.PicksInTerminal || strings.Contains(done.Command, "--model") {
				t.Fatalf("a service listing only for a key ends with %+v", turns[len(turns)-1])
			}
			fake := atTerminal(t, canary, func(ids []string) (string, error) { return ids[0], nil })
			stdout, stderr, err, _ := runConnect(t, "", commandArgs(t, done.Command)...)
			if (err == nil) != (tc.chatCode == http.StatusOK) {
				t.Fatalf("the printed command = %v\n%s%s", err, stdout, stderr)
			}
			if !reflect.DeepEqual(fake.offered, [][]string{{"vendor/coder-large"}}) {
				t.Fatalf("the picker was offered %q; the id carrying the key must be dropped", fake.offered)
			}
			said := strings.Join(raw, "\n") + text + stdout + stderr
			if err != nil {
				said += err.Error()
			}
			if strings.Contains(said, canary) {
				t.Fatal("the canary reached a turn, the output or an error")
			}
			store := filepath.Join(os.Getenv("HOME"), ".abcd", "credentials.json")
			found := map[string]bool{}
			_ = filepath.WalkDir(os.Getenv("HOME"), func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if b, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(b), canary) {
					found[p] = true
				}
				return nil
			})
			want := map[string]bool{}
			if tc.chatCode == http.StatusOK {
				want[store] = true
			}
			if !reflect.DeepEqual(found, want) {
				t.Fatalf("the canary is in %v, want only %v", found, want)
			}
		})
	}
	// The guide itself: a service listing keyless, one of its ids carrying a
	// key-shaped canary. The guide holds no key, so it knows the canary by its
	// shape alone: the look-up's keep drops the id before the resume object
	// carries it, and the canary reaches no turn, no output and no file but
	// the credential file the printed command writes.
	t.Run("the service lists keyless, one id carrying the key", func(t *testing.T) {
		hermeticEnv(t)
		t.Chdir(t.TempDir())
		keyed := "sk-proj-" + testsecret.Synthetic(61, 48)
		svc := newKeylessService(t, []string{"vendor/" + keyed, "vendor/coder-large"})
		raw, turns, text, said := guideTurns(t, []string{"example", "--base-url", svc.base()}, "lookup", "vendor/coder-large", "key", "abcd")
		var st struct {
			Listed struct {
				Models []string `json:"models"`
			} `json:"listed"`
		}
		if err := json.Unmarshal(turns[1].Resume, &st); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(st.Listed.Models, []string{"vendor/coder-large"}) {
			t.Fatalf("the resume object carries %q; the id carrying the key must be dropped", st.Listed.Models)
		}
		done := turns[len(turns)-1].Done
		if done == nil || !strings.Contains(done.Command, "--model vendor/coder-large") {
			t.Fatalf("the guide ends with %+v", turns[len(turns)-1])
		}
		atTerminal(t, keyed, nil)
		stdout, stderr, err, _ := runConnect(t, "", commandArgs(t, done.Command)...)
		if err != nil {
			t.Fatalf("the printed command = %v\n%s%s", err, stdout, stderr)
		}
		if strings.Contains(strings.Join(raw, "\n")+text+said+stdout+stderr, keyed) {
			t.Fatal("the canary reached a turn, the output or an error")
		}
		store := filepath.Join(os.Getenv("HOME"), ".abcd", "credentials.json")
		found := map[string]bool{}
		_ = filepath.WalkDir(os.Getenv("HOME"), func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if b, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(b), keyed) {
				found[p] = true
			}
			return nil
		})
		if !reflect.DeepEqual(found, map[string]bool{store: true}) {
			t.Fatalf("the canary is in %v, want only %s", found, store)
		}
	})
}
