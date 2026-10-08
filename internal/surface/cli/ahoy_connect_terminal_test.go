package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"github.com/intentdriven/abcd/internal/abcdhome"
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

	"github.com/intentdriven/abcd/internal/core/layered"
	"github.com/intentdriven/abcd/internal/core/question"
	"github.com/intentdriven/abcd/internal/surface/cli/ask"
	"github.com/spf13/cobra"
)

// The terminal step of guided connect (spc-2610031241482088, step 2): at a
// terminal `abcd ahoy connect` reads the key on hidden input, and with no
// --model it lists the service's models with that key, lets the person pick
// one, verifies the pick with a real completion, and only then writes. These
// tests replace the terminal seams; the pseudo-terminal tests in
// ahoy_connect_pty_test.go run the real hidden read.

// atTerminal makes every stream a terminal for one test, the hidden read
// answer key (counting its calls), and the picker pick (recording what it was
// offered). A nil pick leaves the real picker seam in place.
type terminalFake struct {
	mu       sync.Mutex
	reads    int
	offered  [][]string
	pickHost string
}

func atTerminal(t *testing.T, key string, pick func([]string) (string, error)) *terminalFake {
	t.Helper()
	f := &terminalFake{}
	oldIs, oldRead, oldPick := connectIsTerminal, connectReadHidden, connectPick
	t.Cleanup(func() { connectIsTerminal, connectReadHidden, connectPick = oldIs, oldRead, oldPick })
	connectIsTerminal = func(any) bool { return true }
	connectReadHidden = func(in io.Reader, out io.Writer) (string, error) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.reads++
		return key, nil
	}
	if pick != nil {
		connectPick = func(_ *cobra.Command, _ layered.Roots, baseURL string) func(context.Context, []string) (string, error) {
			return func(_ context.Context, ids []string) (string, error) {
				f.mu.Lock()
				f.offered = append(f.offered, append([]string(nil), ids...))
				f.pickHost = baseURL
				f.mu.Unlock()
				return pick(ids)
			}
		}
	}
	return f
}

// listingService is a stand-in that answers its model list (401 keyless,
// listing ids keyed) and its completions (chatCode), echoing the
// Authorization header into every error body, and remembers what each request
// carried.
type listingService struct {
	srv      *httptest.Server
	ids      []string
	chatCode int

	mu        sync.Mutex
	listAuth  []string
	chatAuth  []string
	chatModel []string
}

func newListingService(t *testing.T, ids []string, chatCode int) *listingService {
	t.Helper()
	s := &listingService{ids: ids, chatCode: chatCode}
	s.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		auth := r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		s.mu.Lock()
		defer s.mu.Unlock()
		echo := func(code int) {
			w.WriteHeader(code)
			b, _ := json.Marshal(map[string]any{"error": map[string]any{"message": "refused, you sent " + auth}})
			_, _ = w.Write(b)
		}
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/models"):
			s.listAuth = append(s.listAuth, auth)
			if auth == "" {
				echo(http.StatusUnauthorized)
				return
			}
			data := make([]map[string]string, len(s.ids))
			for i, id := range s.ids {
				data[i] = map[string]string{"id": id}
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/chat/completions"):
			var req struct {
				Model string `json:"model"`
			}
			_ = json.Unmarshal(raw, &req)
			s.chatAuth = append(s.chatAuth, auth)
			s.chatModel = append(s.chatModel, req.Model)
			if s.chatCode != http.StatusOK {
				echo(s.chatCode)
				return
			}
			_, _ = io.WriteString(w, completionReply(req.Model))
		default:
			echo(http.StatusNotFound)
		}
	}))
	t.Cleanup(s.srv.Close)
	return s
}

func (s *listingService) base() string { return s.srv.URL + "/api/v1" }

func (s *listingService) seen() (listAuth, chatAuth, chatModel []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.listAuth...), append([]string(nil), s.chatAuth...), append([]string(nil), s.chatModel...)
}

// runConnect runs `ahoy connect` with stdin and returns stdout, stderr, the
// error and the exit code the front door would end with.
func runConnect(t *testing.T, stdin string, args ...string) (stdout, stderr string, err error, code int) {
	t.Helper()
	cmd := NewRootCommand()
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetIn(strings.NewReader(stdin))
	cmd.SetArgs(append([]string{"ahoy", "connect"}, args...))
	err = cmd.Execute()
	var coded interface{ ExitCode() int }
	switch {
	case err == nil:
	case errors.As(err, &coded):
		code = coded.ExitCode()
	default:
		code = 1
	}
	return out.String(), errb.String(), err, code
}

// machineWrites lists the files under the fake home's ~/.abcd.
func machineWrites(t *testing.T) []string {
	t.Helper()
	var got []string
	entries, _ := os.ReadDir(abcdhome.Path(os.Getenv("HOME")))
	for _, e := range entries {
		if e.Name() == "config.json" || e.Name() == "credentials.json" || e.Name() == "credential-homes.json" {
			got = append(got, e.Name())
		}
	}
	return got
}

// TestConnectReadsTheKeyHidden (G8's hidden-input half): at a terminal the
// key is read through the hidden reader, after one line on stderr naming the
// provider, and never from the pipe; the key the reader returns is the one
// the verification sends and the one stored, and it is never printed. The
// keychain home reads it the same way.
func TestConnectReadsTheKeyHidden(t *testing.T) {
	for _, home := range []string{"abcd", "keychain"} {
		t.Run(home, func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			fake := atTerminal(t, connectKey, nil)
			base, calls, auth := fakeProvider(t, 200, completionReply("typesafe/jev-1.13"))
			if home == "keychain" {
				// The verification is refused, so the run never reaches the
				// platform keychain: what is held here is the read.
				base, calls, auth = fakeProvider(t, 401, `{"error":{"message":"refused"}}`)
			}
			stdout, stderr, err, _ := runConnect(t, "a-piped-value-that-is-not-read\n", "openrouter",
				"--base-url", base, "--model", "typesafe/jev-1.13", "--home", home)
			if fake.reads != 1 {
				t.Fatalf("the hidden reader was called %d time(s), want once", fake.reads)
			}
			if !strings.Contains(stderr, "Paste the key for openrouter and press Enter. It is not shown.") {
				t.Errorf("no prompt line on stderr:\n%s", stderr)
			}
			if calls.Load() != 1 || auth.Load() != "Bearer "+connectKey {
				t.Fatalf("verification: %d call(s), auth %v; want the hidden key", calls.Load(), auth.Load())
			}
			msg := stdout + stderr
			if err != nil {
				msg += err.Error()
			}
			if strings.Contains(msg, connectKey) {
				t.Fatal("the key reached the output")
			}
			if home == "abcd" {
				if err != nil {
					t.Fatalf("ahoy connect: %v\n%s", err, msg)
				}
				raw, _ := os.ReadFile(abcdhome.Path(os.Getenv("HOME"), "credentials.json"))
				if !strings.Contains(string(raw), connectKey) {
					t.Fatal("the hidden key was not the one stored")
				}
			}
		})
	}
}

// TestConnectRefusesAnEmptyHiddenKey: Enter on nothing at the terminal is
// refused as an empty pipe is, before any call.
func TestConnectRefusesAnEmptyHiddenKey(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	atTerminal(t, "", nil)
	base, calls, _ := fakeProvider(t, 200, completionReply("m"))
	_, _, err, code := runConnect(t, "", "openrouter", "--base-url", base, "--model", "m", "--home", "abcd")
	if err == nil || code != 2 || !strings.Contains(err.Error(), "none was pasted") || calls.Load() != 0 {
		t.Fatalf("an empty hidden key = %v (exit %d, %d call(s))", err, code, calls.Load())
	}
}

// TestConnectPicksAfterAKeyedListing (G5): with no --model at a terminal, a
// service that answers its keyless list 401 is listed with the key, the
// picker is offered the listed ids, the completion asks for the model picked,
// and both writes land.
func TestConnectPicksAfterAKeyedListing(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	svc := newListingService(t, []string{"vendor/coder-large", "vendor/coder-small"}, http.StatusOK)
	fake := atTerminal(t, connectKey, func(ids []string) (string, error) { return ids[1], nil })
	stdout, stderr, err, _ := runConnect(t, "", "example", "--base-url", svc.base(), "--home", "abcd")
	if err != nil {
		t.Fatalf("ahoy connect: %v\n%s%s", err, stdout, stderr)
	}
	// Two listings: the picker's, and the one the verification call reads
	// before it is sent, to judge the request's size; both carry the key.
	listAuth, chatAuth, chatModel := svc.seen()
	if !reflect.DeepEqual(listAuth, []string{"Bearer " + connectKey, "Bearer " + connectKey}) {
		t.Fatalf("list requests carried %q, want two (the picker's and the size check's), with the key", listAuth)
	}
	if !reflect.DeepEqual(fake.offered, [][]string{{"vendor/coder-large", "vendor/coder-small"}}) || fake.pickHost != svc.base() {
		t.Fatalf("the picker was offered %q for %s", fake.offered, fake.pickHost)
	}
	if !reflect.DeepEqual(chatModel, []string{"vendor/coder-small"}) || !reflect.DeepEqual(chatAuth, []string{"Bearer " + connectKey}) {
		t.Fatalf("the completion asked for %q with %q, want the model picked, with the key", chatModel, chatAuth)
	}
	if got := machineWrites(t); !reflect.DeepEqual(got, []string{"config.json", "credentials.json"}) {
		t.Fatalf("writes = %q, want the provider block and the key", got)
	}
	raw, _ := os.ReadFile(abcdhome.Path(os.Getenv("HOME"), "config.json"))
	if !strings.Contains(string(raw), `"vendor/coder-small"`) || strings.Contains(string(raw), "coder-large") {
		t.Fatalf("the provider block does not hold the model picked:\n%s", raw)
	}
	for _, want := range []string{"wrote: " + abcdhome.Display("credentials.json"), "wrote: " + abcdhome.Display("config.json"), `"example/vendor/coder-small"`} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the result does not say %q:\n%s", want, stdout)
		}
	}
}

// TestConnectWritesNothingWhenTheCompletionFails (G5): a service that lists
// with the key and fails the completion leaves no provider block and no
// stored key, and the refusal names the verification.
func TestConnectWritesNothingWhenTheCompletionFails(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	svc := newListingService(t, []string{"vendor/coder-large"}, http.StatusInternalServerError)
	atTerminal(t, connectKey, func(ids []string) (string, error) { return ids[0], nil })
	stdout, stderr, err, code := runConnect(t, "", "example", "--base-url", svc.base(), "--home", "abcd")
	if err == nil || code != 2 || !strings.Contains(err.Error(), "the verification call failed, so nothing was written") {
		t.Fatalf("ahoy connect = %v (exit %d)\n%s%s", err, code, stdout, stderr)
	}
	if strings.Contains(stdout+stderr+err.Error(), connectKey) {
		t.Fatal("the key reached the output or the error")
	}
	if _, _, chatModel := svc.seen(); !reflect.DeepEqual(chatModel, []string{"vendor/coder-large"}) {
		t.Fatalf("completions = %q", chatModel)
	}
	if got := machineWrites(t); len(got) != 0 {
		t.Fatalf("a failed completion wrote %q", got)
	}
}

// TestConnectPickInterruptedOrDeferredWritesNothing: Ctrl-C at the picker
// exits 130, and the picker's decide later is refused naming it; neither makes
// a completion or writes anything.
func TestConnectPickInterruptedOrDeferredWritesNothing(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		code int
		want string
	}{
		{"ctrl-c", ask.ErrInterrupted, ask.ExitInterrupted, "interrupted"},
		{"decide later", errPickLater, 2, "no model was picked"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			svc := newListingService(t, []string{"vendor/coder-large"}, http.StatusOK)
			atTerminal(t, connectKey, func([]string) (string, error) { return "", tc.err })
			_, _, err, code := runConnect(t, "", "example", "--base-url", svc.base(), "--home", "abcd")
			if err == nil || code != tc.code || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "nothing was written") {
				t.Fatalf("ahoy connect = %v (exit %d), want exit %d saying %q and that nothing was written", err, code, tc.code, tc.want)
			}
			if _, _, chatModel := svc.seen(); len(chatModel) != 0 {
				t.Fatalf("a completion was made: %q", chatModel)
			}
			if got := machineWrites(t); len(got) != 0 {
				t.Fatalf("wrote %q", got)
			}
		})
	}
}

// TestConnectWithoutModelRefusesOffATerminal (G5): piped, with no --model,
// the setup is refused before anything is read or sent, naming both ways on.
func TestConnectWithoutModelRefusesOffATerminal(t *testing.T) {
	hermeticEnv(t)
	t.Chdir(t.TempDir())
	svc := newListingService(t, []string{"vendor/coder-large"}, http.StatusOK)
	_, _, err, code := runConnect(t, connectKey+"\n", "example", "--base-url", svc.base(), "--home", "abcd")
	if err == nil || code != 2 {
		t.Fatalf("ahoy connect with no --model off a terminal = %v (exit %d)", err, code)
	}
	for _, want := range []string{"--model", "run the command in a terminal"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal does not name %q: %v", want, err)
		}
	}
	if listAuth, _, chatModel := svc.seen(); len(listAuth)+len(chatModel) != 0 {
		t.Fatalf("a refused setup sent %d request(s)", len(listAuth)+len(chatModel))
	}
	if got := machineWrites(t); len(got) != 0 {
		t.Fatalf("wrote %q", got)
	}
}

// TestPickedConnectKeepsTheCanaryInItsHome (the terminal half of G6): a
// canary key pasted at the terminal, sent to a stand-in that echoes the
// Authorization header into its error bodies and lists one id carrying the
// key and one carrying a shell character, appears after a picked run only in
// the abcd home's file: never on stdout or stderr, never in an error, never in
// any other file under the home, and never offered to the picker.
func TestPickedConnectKeepsTheCanaryInItsHome(t *testing.T) {
	const canary = "canary-7f3e9a1c5b2d4e6f8a0b-not-a-key"
	for _, tc := range []struct {
		name     string
		chatCode int
	}{{"the run succeeds", http.StatusOK}, {"the completion fails, echoing the key", http.StatusUnauthorized}} {
		t.Run(tc.name, func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			svc := newListingService(t, []string{"vendor/" + canary, "vendor/m;rm", "vendor/coder-large"}, tc.chatCode)
			fake := atTerminal(t, canary, func(ids []string) (string, error) { return ids[0], nil })
			stdout, stderr, err, _ := runConnect(t, "", "example", "--base-url", svc.base(), "--home", "abcd")
			if (err == nil) != (tc.chatCode == http.StatusOK) {
				t.Fatalf("ahoy connect = %v\n%s%s", err, stdout, stderr)
			}
			if !reflect.DeepEqual(fake.offered, [][]string{{"vendor/coder-large"}}) {
				t.Fatalf("the picker was offered %q, want only the clean id", fake.offered)
			}
			said := stdout + stderr
			if err != nil {
				said += err.Error()
			}
			if strings.Contains(said, canary) {
				t.Fatalf("the canary reached the output or the error:\n%s", said)
			}
			store := abcdhome.Path(os.Getenv("HOME"), "credentials.json")
			found := map[string]bool{}
			_ = filepath.WalkDir(os.Getenv("HOME"), func(p string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return nil
				}
				if raw, rerr := os.ReadFile(p); rerr == nil && strings.Contains(string(raw), canary) {
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
}

// TestConnectRefusesBeforeTheKeyIsRead (security review finding 7): a setup
// that would be refused on what needs no key (the base URL, a reserved or
// malformed provider name, a provider already configured)
// is refused before the person is asked to paste the key, so nobody pastes
// for nothing; nothing is sent and nothing written.
func TestConnectRefusesBeforeTheKeyIsRead(t *testing.T) {
	for _, tc := range []struct {
		name     string
		provider string
		base     func(svc string) string
		existing bool
		want     string
	}{
		{"a base URL over plain http to another machine", "example", func(string) string { return "http://192.0.2.1/v1" }, false, "plain http to another machine"},
		{"the reserved provider name", "harness", func(s string) string { return s }, false, "reserved"},
		{"a malformed provider name", "Not;A-Name", func(s string) string { return s }, false, "is not lower case"},
		{"a provider already configured", "example", func(s string) string { return s }, true, "already configured"},
	} {
		for _, model := range [][]string{{"--model", "vendor/coder-large"}, nil} {
			t.Run(tc.name+map[bool]string{true: ", --model", false: ", picking"}[model != nil], func(t *testing.T) {
				hermeticEnv(t)
				t.Chdir(t.TempDir())
				if tc.existing {
					dir := abcdhome.Path(os.Getenv("HOME"))
					if err := os.MkdirAll(dir, 0o700); err != nil {
						t.Fatal(err)
					}
					body := `{"oracle":{"api":{"example":{"base_url":"https://api.example.com/v1","models":["m"]}}}}`
					if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o600); err != nil {
						t.Fatal(err)
					}
				}
				svc := newListingService(t, []string{"vendor/coder-large"}, http.StatusOK)
				fake := atTerminal(t, connectKey, func(ids []string) (string, error) { return ids[0], nil })
				args := append([]string{tc.provider, "--base-url", tc.base(svc.base()), "--home", "abcd"}, model...)
				_, stderr, err, code := runConnect(t, "", args...)
				if err == nil || code != 2 || !strings.Contains(err.Error(), tc.want) {
					t.Fatalf("ahoy connect = %v (exit %d), want exit 2 saying %q", err, code, tc.want)
				}
				if fake.reads != 0 || strings.Contains(stderr, "Paste the key") {
					t.Fatalf("the key was asked for (%d read(s)) before a refusal that needs no key:\n%s", fake.reads, stderr)
				}
				if listAuth, _, chatModel := svc.seen(); len(listAuth)+len(chatModel) != 0 {
					t.Fatalf("a refused setup sent %d request(s)", len(listAuth)+len(chatModel))
				}
				if got := machineWrites(t); len(got) != 0 && !(tc.existing && reflect.DeepEqual(got, []string{"config.json"})) {
					t.Fatalf("wrote %q", got)
				}
			})
		}
	}
}

// TestConnectPicksOnlyWhenEveryStreamIsATerminal (security review finding
// 3): picking draws a list, so a run with no --model needs stdin, stdout and
// stderr all terminals (adr-49). With any one of them not a terminal the run
// is refused as off a terminal: the key is not asked for, nothing is sent,
// nothing written.
func TestConnectPicksOnlyWhenEveryStreamIsATerminal(t *testing.T) {
	for _, notTerminal := range []string{"stdin", "stdout", "stderr"} {
		t.Run(notTerminal+" is not a terminal", func(t *testing.T) {
			hermeticEnv(t)
			t.Chdir(t.TempDir())
			svc := newListingService(t, []string{"vendor/coder-large"}, http.StatusOK)
			fake := atTerminal(t, connectKey, func(ids []string) (string, error) { return ids[0], nil })
			cmd := NewRootCommand()
			in := strings.NewReader("")
			var out, errb bytes.Buffer
			cmd.SetIn(in)
			cmd.SetOut(&out)
			cmd.SetErr(&errb)
			streams := map[string]any{"stdin": in, "stdout": &out, "stderr": &errb}
			connectIsTerminal = func(s any) bool { return s != streams[notTerminal] }
			cmd.SetArgs([]string{"ahoy", "connect", "example", "--base-url", svc.base(), "--home", "abcd"})
			err := cmd.Execute()
			var coded interface{ ExitCode() int }
			if err == nil || !errors.As(err, &coded) || coded.ExitCode() != 2 || !strings.Contains(err.Error(), "run the command in a terminal") {
				t.Fatalf("ahoy connect with %s not a terminal = %v, want the off-a-terminal refusal", notTerminal, err)
			}
			if fake.reads != 0 || len(fake.offered) != 0 {
				t.Fatalf("the key was read %d time(s), the picker offered %d list(s)", fake.reads, len(fake.offered))
			}
			if listAuth, _, chatModel := svc.seen(); len(listAuth)+len(chatModel) != 0 {
				t.Fatalf("a refused setup sent %d request(s)", len(listAuth)+len(chatModel))
			}
			if got := machineWrites(t); len(got) != 0 {
				t.Fatalf("wrote %q", got)
			}
		})
	}
}

// TestPickQuestionKeepsOneChoicePerId: a service that lists one id twice
// still gets a question the list accepts, with each id offered once in the
// service's order, and the count the question gives is the ids offered.
func TestPickQuestionKeepsOneChoicePerId(t *testing.T) {
	a := pickQuestion("api.example.com", []string{"vendor/a", "vendor/b", "vendor/a", "vendor/c", "vendor/b"})
	if findings := question.Check(a); len(findings) != 0 {
		t.Fatalf("the pick question fails its check: %v", findings)
	}
	var got []string
	for _, c := range a.Questions[0].List.Choices {
		got = append(got, c.Value)
	}
	if want := []string{"vendor/a", "vendor/b", "vendor/c"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("choices = %q, want %q", got, want)
	}
	if text := a.Questions[0].Material[0].Text; !strings.Contains(text, "api.example.com lists 3 models") {
		t.Fatalf("material = %q, want the host and the 3 ids offered", text)
	}
}
