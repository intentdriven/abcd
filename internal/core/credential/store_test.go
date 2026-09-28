package credential

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The credential store proper (itd-2609221017023290): one Resolve and one Set
// over three homes, the walkthrough that chooses one, and the refusals that
// keep a value out of the tree and the harness's settings.
//
// No test here touches the real keychain: the keychain home runs a fake,
// which is this test binary re-executed (TestMain), keeping each item as a
// file under a temporary directory and logging every argv it was given.

const (
	fakeKeychainDirEnv = "ABCD_TEST_FAKE_KEYCHAIN_DIR"
	fakeKeychainKind   = "ABCD_TEST_FAKE_KEYCHAIN_KIND"
)

func TestMain(m *testing.M) {
	if dir := os.Getenv(fakeKeychainDirEnv); dir != "" && len(os.Args) > 1 && os.Args[1] != "" && !strings.HasPrefix(os.Args[1], "-test.") {
		os.Exit(fakeKeychain(dir, os.Getenv(fakeKeychainKind), os.Args[1:]))
	}
	os.Exit(m.Run())
}

// fakeKeychain emulates the two platform commands closely enough to exercise
// the argv and stdin the store hands them.
func fakeKeychain(dir, kind string, args []string) int {
	logf, _ := os.OpenFile(filepath.Join(dir, "argv.log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	fmt.Fprintln(logf, strings.Join(args, " "))
	logf.Close()
	item := func(account string) string { return filepath.Join(dir, "item-"+account) }
	flag := func(args []string, name string) string {
		for i := 0; i+1 < len(args); i++ {
			if args[i] == name {
				return args[i+1]
			}
		}
		return ""
	}
	switch kind {
	case keychainMacOS:
		switch args[0] {
		case "find-generic-password":
			if flag(args, "-s") != keychainService {
				return 2
			}
			b, err := os.ReadFile(item(flag(args, "-a")))
			if err != nil {
				fmt.Fprintln(os.Stderr, "security: SecKeychainSearchCopyNext: The specified item could not be found in the keychain.")
				return 44
			}
			fmt.Println(string(b))
			return 0
		case "-i":
			in, _ := io.ReadAll(os.Stdin)
			fields := strings.Fields(string(in))
			if len(fields) == 0 || fields[0] != "add-generic-password" || flag(fields, "-s") != keychainService {
				return 0 // interactive mode reports a failed command on stderr, not in its exit
			}
			v, err := hex.DecodeString(flag(fields, "-X"))
			if err != nil {
				return 0
			}
			_ = os.WriteFile(item(flag(fields, "-a")), v, 0o600)
			return 0
		}
	case keychainSecretService:
		switch args[0] {
		case "lookup":
			b, err := os.ReadFile(item(args[len(args)-1]))
			if err != nil {
				return 1
			}
			os.Stdout.Write(b)
			return 0
		case "store":
			v, _ := io.ReadAll(os.Stdin)
			_ = os.WriteFile(item(args[len(args)-1]), v, 0o600)
			return 0
		}
	}
	return 2
}

// withFakeKeychain points the keychain home at the fake for one test.
func withFakeKeychain(t *testing.T, kind string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(fakeKeychainDirEnv, dir)
	t.Setenv(fakeKeychainKind, kind)
	old := locateKeychain
	locateKeychain = func() (keychainTool, error) { return keychainTool{kind: kind, path: os.Args[0]}, nil }
	t.Cleanup(func() { locateKeychain = old })
	return dir
}

func withNoKeychain(t *testing.T) {
	t.Helper()
	old := locateKeychain
	locateKeychain = func() (keychainTool, error) { return keychainTool{}, errKeychainAbsent }
	t.Cleanup(func() { locateKeychain = old })
}

// assertNowhere fails when value appears in any file under root, except the
// paths allowed (absolute).
func assertNowhere(t *testing.T, root, value string, allowed ...string) {
	t.Helper()
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		for _, a := range allowed {
			if p == a {
				return nil
			}
		}
		b, _ := os.ReadFile(p)
		if strings.Contains(string(b), value) {
			t.Errorf("%s carries the value", p)
		}
		return nil
	})
}

func TestStoreResolvesEveryHome(t *testing.T) {
	for _, kind := range []string{keychainMacOS, keychainSecretService} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			withFakeKeychain(t, kind)
			t.Setenv("ABCD_TEST_TOKEN_FOR_STORE", "env-"+secretValue)
			tool := filepath.Join(home, ".config", "tool")
			if err := os.MkdirAll(tool, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(tool, "auth.json"), []byte(`{"auth":{"token":"file-`+secretValue+`"}}`), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, c := range []struct {
				name string
				ch   Choice
				want string
			}{
				{"svc.abcd", Choice{Home: HomeABCD, Value: "abcd-" + secretValue}, "abcd-" + secretValue},
				{"svc.keychain", Choice{Home: HomeKeychain, Value: "kc-" + secretValue}, "kc-" + secretValue},
				{"svc.env", Choice{Home: HomeExternal, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_FOR_STORE"}}, "env-" + secretValue},
				{"svc.file", Choice{Home: HomeExternal, Pointer: Pointer{File: "~/.config/tool/auth.json", Field: "auth.token"}}, "file-" + secretValue},
			} {
				changed, err := Set(home, c.name, c.ch)
				if err != nil || !changed {
					t.Fatalf("%s: Set = %v, %v", c.name, changed, err)
				}
				got, err := Store(home).Resolve(c.name)
				if err != nil || got != c.want {
					t.Fatalf("%s: Resolve = %v; want the value set", c.name, err)
				}
				where, err := Where(home, c.name)
				if err != nil || where != c.ch.Home {
					t.Fatalf("%s: Where = %q, %v; want %s", c.name, where, err, c.ch.Home)
				}
			}
		})
	}
}

// TestAnUnsetNameRefusesNamingTheWalkthrough (criterion 1): a name no home
// holds is a refusal that is ErrNotSet and names the walkthrough.
func TestAnUnsetNameRefusesNamingTheWalkthrough(t *testing.T) {
	home := t.TempDir()
	_, err := Store(home).Resolve("hosting.cloudflare")
	if !errors.Is(err, ErrNotSet) {
		t.Fatalf("err = %v, want ErrNotSet", err)
	}
	if !strings.Contains(err.Error(), "abcd ahoy credential hosting.cloudflare") {
		t.Fatalf("the refusal does not name the walkthrough: %v", err)
	}
	t.Setenv("ABCD_TEST_UNSET_VAR", "")
	if _, err := Set(home, "svc.env", Choice{Home: HomeExternal, Pointer: Pointer{Env: "ABCD_TEST_UNSET_VAR"}}); err == nil {
		t.Fatal("a pointer at an empty variable was stored")
	}
}

// TestAWriteIntoAWorkingTreeIsRefused (criterion 3): a home whose ~/.abcd
// sits inside a git working tree would put the store where a commit can reach
// it, so every home's write is refused and nothing is written.
func TestAWriteIntoAWorkingTreeIsRefused(t *testing.T) {
	withFakeKeychain(t, keychainMacOS)
	for _, ch := range []Choice{
		{Home: HomeABCD, Value: secretValue},
		{Home: HomeKeychain, Value: secretValue},
		{Home: HomeExternal, Pointer: Pointer{Env: "HOME"}},
	} {
		home := t.TempDir()
		if err := os.Mkdir(filepath.Join(home, ".git"), 0o700); err != nil {
			t.Fatal(err)
		}
		_, err := Set(home, "svc", ch)
		if err == nil || !strings.Contains(err.Error(), "working tree") {
			t.Fatalf("%s: err = %v, want a working-tree refusal", ch.Home, err)
		}
		if _, statErr := os.Stat(filepath.Join(home, ".abcd")); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("%s: ~/.abcd was created", ch.Home)
		}
	}
	// A tool file inside a working tree is refused as a pointer too.
	home := t.TempDir()
	repo := filepath.Join(home, "repo")
	if err := os.MkdirAll(filepath.Join(repo, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "auth.json"), []byte(`{"token":"`+secretValue+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Set(home, "svc", Choice{Home: HomeExternal, Pointer: Pointer{File: "~/repo/auth.json", Field: "token"}}); err == nil ||
		!strings.Contains(err.Error(), "working tree") {
		t.Fatalf("a pointer into a working tree: err = %v", err)
	}
}

// TestTheIndexWriteRunsTheScanner (criterion 3): the one file the store writes
// that must never hold a secret is scanned before it is written, and a
// secret-shaped pointer is refused without echoing it.
func TestTheIndexWriteRunsTheScanner(t *testing.T) {
	home := t.TempDir()
	awsShaped := "AKIA" + strings.Repeat("Q", 16)
	t.Setenv(awsShaped, "some-value-000")
	_, err := Set(home, "svc", Choice{Home: HomeExternal, Pointer: Pointer{Env: awsShaped}})
	if err == nil || !strings.Contains(err.Error(), "scanner") {
		t.Fatalf("err = %v, want the scanner's refusal", err)
	}
	if strings.Contains(err.Error(), awsShaped) {
		t.Fatal("the refusal echoes the secret-shaped pointer")
	}
	if _, statErr := os.Stat(filepath.Join(home, ".abcd", IndexFileName)); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatal("the index was written")
	}
}

// TestNeitherTheTreeNorTheHarnessCarriesTheValue (criterion 3): after a set in
// each home, no file under the home (a repository and the harness's settings
// inside it) carries the value; the abcd home's own file is the one allowed
// holder, and the keychain's values live outside the home altogether.
func TestNeitherTheTreeNorTheHarnessCarriesTheValue(t *testing.T) {
	for _, h := range Homes() {
		t.Run(h, func(t *testing.T) {
			home := t.TempDir()
			withFakeKeychain(t, keychainMacOS)
			for _, p := range []string{".claude/settings.json", "work/repo/.claude/settings.json", "work/repo/README.md"} {
				if err := os.MkdirAll(filepath.Dir(filepath.Join(home, p)), 0o700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(home, p), []byte("{}\n"), 0o600); err != nil {
					t.Fatal(err)
				}
			}
			value := h + "-" + secretValue
			ch := Choice{Home: h, Value: value}
			if h == HomeExternal {
				t.Setenv("ABCD_TEST_TOKEN_FOR_STORE", value)
				ch = Choice{Home: h, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_FOR_STORE"}}
			}
			if _, err := Set(home, "svc", ch); err != nil {
				t.Fatal(err)
			}
			assertNowhere(t, home, value, filepath.Join(home, ".abcd", StoreFileName))
		})
	}
}

// TestTheKeychainValueNeverReachesAnArgv: a process listing shows argv, so the
// value is handed to the keychain command on stdin, never as an argument.
func TestTheKeychainValueNeverReachesAnArgv(t *testing.T) {
	for _, kind := range []string{keychainMacOS, keychainSecretService} {
		home := t.TempDir()
		dir := withFakeKeychain(t, kind)
		if _, err := Set(home, "svc", Choice{Home: HomeKeychain, Value: secretValue}); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
		log, _ := os.ReadFile(filepath.Join(dir, "argv.log"))
		if len(log) == 0 {
			t.Fatalf("%s: the keychain command was never run", kind)
		}
		if strings.Contains(string(log), secretValue) || strings.Contains(string(log), hex.EncodeToString([]byte(secretValue))) {
			t.Fatalf("%s: an argv carries the value:\n%s", kind, log)
		}
	}
}

// TestAPlatformWithoutAKeychainOffersTheOtherHomes (scope condition): the
// keychain home refuses naming the other two, and they still work.
func TestAPlatformWithoutAKeychainOffersTheOtherHomes(t *testing.T) {
	home := t.TempDir()
	withNoKeychain(t)
	_, err := Set(home, "svc", Choice{Home: HomeKeychain, Value: secretValue})
	if err == nil || !strings.Contains(err.Error(), "external") || !strings.Contains(err.Error(), "abcd") {
		t.Fatalf("err = %v, want a refusal naming the other homes", err)
	}
	if _, err := Set(home, "svc", Choice{Home: HomeABCD, Value: secretValue}); err != nil {
		t.Fatalf("the abcd home without a keychain: %v", err)
	}
}

// TestAStoredSecretIsNeverReplaced: a name held in one home is refused in
// another, and a different value in the same home is refused; the same value
// again is no change.
func TestAStoredSecretIsNeverReplaced(t *testing.T) {
	home := t.TempDir()
	withFakeKeychain(t, keychainMacOS)
	if _, err := Set(home, "svc", Choice{Home: HomeKeychain, Value: secretValue}); err != nil {
		t.Fatal(err)
	}
	if changed, err := Set(home, "svc", Choice{Home: HomeKeychain, Value: secretValue}); err != nil || changed {
		t.Fatalf("the same value again = %v, %v; want no change", changed, err)
	}
	for _, ch := range []Choice{{Home: HomeKeychain, Value: "other-" + secretValue}, {Home: HomeABCD, Value: secretValue}} {
		_, err := Set(home, "svc", ch)
		if err == nil || !strings.Contains(err.Error(), "never replaces") {
			t.Fatalf("%s: err = %v, want a refusal", ch.Home, err)
		}
		if strings.Contains(err.Error(), secretValue) {
			t.Fatal("the refusal echoes the value")
		}
	}
	if v, _ := Store(home).Resolve("svc"); v != secretValue {
		t.Fatal("the stored value changed")
	}
}

// TestANameInTwoHomesIsRefused: a hand edit that puts one name in the abcd
// file and the index is ambiguous, and it is refused rather than guessed.
func TestANameInTwoHomesIsRefused(t *testing.T) {
	home := t.TempDir()
	withFakeKeychain(t, keychainMacOS)
	if _, err := Set(home, "svc", Choice{Home: HomeKeychain, Value: secretValue}); err != nil {
		t.Fatal(err)
	}
	writeStore(t, home, `{"svc": "other-value-000"}`, 0o600)
	if _, err := Store(home).Resolve("svc"); err == nil || errors.Is(err, ErrNotSet) || !strings.Contains(err.Error(), "two homes") {
		t.Fatalf("err = %v, want the two-homes refusal", err)
	}
}

func testService(verify func(context.Context, string) error) Service {
	return Service{
		Name:      "svc",
		Unlocks:   "calls to the example service",
		WithoutIt: "everything but those calls",
		Verify:    verify,
	}
}

// TestTheWalkthroughVerifiesBeforeItStores (criterion 2): the adapter's own
// call is made with the value, and only its success stores it; the result
// never carries the value.
func TestTheWalkthroughVerifiesBeforeItStores(t *testing.T) {
	withFakeKeychain(t, keychainMacOS)
	for _, h := range Homes() {
		home := t.TempDir()
		ch := Choice{Home: h, Value: secretValue}
		if h == HomeExternal {
			t.Setenv("ABCD_TEST_TOKEN_FOR_STORE", secretValue)
			ch = Choice{Home: h, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_FOR_STORE"}}
		}
		var seen string
		_, err := Walk(context.Background(), home, testService(func(_ context.Context, v string) error {
			seen = v
			return errors.New("the service refused the credential")
		}), ch)
		if err == nil || seen != secretValue {
			t.Fatalf("%s: a failed verification = %v (verified with the value: %v)", h, err, seen == secretValue)
		}
		if _, rerr := Store(home).Resolve("svc"); !errors.Is(rerr, ErrNotSet) {
			t.Fatalf("%s: a failed verification stored the credential", h)
		}
		res, err := Walk(context.Background(), home, testService(func(context.Context, string) error { return nil }), ch)
		if err != nil || !res.Verified || res.Home != h || res.Name != "svc" {
			t.Fatalf("%s: Walk = %+v, %v", h, res, err)
		}
		if v, _ := Store(home).Resolve("svc"); v != secretValue {
			t.Fatalf("%s: the verified credential does not resolve", h)
		}
		enc, _ := json.Marshal(res)
		if strings.Contains(string(enc), secretValue) {
			t.Fatalf("%s: the result carries the value", h)
		}
	}
	if _, err := Walk(context.Background(), t.TempDir(), testService(nil), Choice{Home: HomeABCD, Value: secretValue}); err == nil {
		t.Fatal("a walkthrough with no verification call stored the credential")
	}
}

// TestTheWalkthroughExplainsFirst (criterion 2): what it unlocks, what works
// without it, then the three homes, the keychain recommended in the prose and
// never as a marked option.
func TestTheWalkthroughExplainsFirst(t *testing.T) {
	lines := testService(nil).Explain()
	joined := strings.Join(lines, "\n")
	for _, want := range []string{"calls to the example service", "everything but those calls", HomesProse} {
		if !strings.Contains(joined, want) {
			t.Fatalf("the explanation lacks %q:\n%s", want, joined)
		}
	}
	if !strings.Contains(HomesProse, "keychain") || !strings.Contains(HomesProse, "recommend") {
		t.Fatalf("the prose does not recommend the keychain: %s", HomesProse)
	}
	if strings.Index(joined, "unlocks") > strings.Index(joined, HomesProse) {
		t.Fatal("the homes come before what the credential unlocks")
	}
	for _, l := range lines {
		if strings.Contains(l, "(recommended)") || strings.Contains(l, "*") {
			t.Fatalf("a home is marked: %q", l)
		}
	}
	if got := Homes(); strings.Join(got, ",") != "external,abcd,keychain" {
		t.Fatalf("homes = %v", got)
	}
}

// TestTheWalkthroughRefusesBeforeItsCall: a name held elsewhere, or a pointer
// other than the one kept, is refused before the verification call, so a
// setup that cannot store its credential is never verified for nothing.
func TestTheWalkthroughRefusesBeforeItsCall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ABCD_TEST_TOKEN_A", secretValue)
	t.Setenv("ABCD_TEST_TOKEN_B", secretValue)
	if _, err := Set(home, "svc", Choice{Home: HomeExternal, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_A"}}); err != nil {
		t.Fatal(err)
	}
	called := false
	svc := testService(func(context.Context, string) error { called = true; return nil })
	for _, ch := range []Choice{
		{Home: HomeExternal, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_B"}},
		{Home: HomeABCD, Value: secretValue},
	} {
		if _, err := Walk(context.Background(), home, svc, ch); err == nil {
			t.Fatalf("%s: a second home or pointer was accepted", ch.Home)
		}
		if called {
			t.Fatalf("%s: the verification call was made before the refusal", ch.Home)
		}
	}
	if res, err := Walk(context.Background(), home, svc, Choice{Home: HomeExternal, Pointer: Pointer{Env: "ABCD_TEST_TOKEN_A"}}); err != nil || res.Changed {
		t.Fatalf("the same pointer again = %+v, %v; want no change", res, err)
	}
}
