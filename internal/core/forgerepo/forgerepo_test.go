package forgerepo

import "testing"

func TestKeysMatchAcrossURLForms(t *testing.T) {
	same := []string{"https://github.com/O/R", "https://github.com/o/r.git", "git@github.com:o/r.git", "ssh://git@github.com:22/o/r", "https://user@GitHub.com/o/r/"}
	want := Key(same[0])
	for _, u := range same[1:] {
		if got := Key(u); got != want || got == "" {
			t.Errorf("Key(%q) = %q, want %q", u, got, want)
		}
	}
	if Key("https://github.com/o/other") == want || Key("https://gitlab.com/o/r") == want {
		t.Error("a different repository matched")
	}
	if Key("/tmp/x/origin.git") != Key("file:///tmp/x/origin") {
		t.Error("a local path and its file URL differ")
	}
	// A port other than the scheme's own is another server.
	for _, u := range []string{"https://github.com:8443/o/r", "ssh://git@github.com:2222/o/r"} {
		if Key(u) == want {
			t.Errorf("Key(%q) = %q matched the default port's repository", u, Key(u))
		}
	}
	if Key("https://github.com:443/o/r") != want {
		t.Error("https's own port named another server")
	}
}

func TestSplitKey(t *testing.T) {
	for k, want := range map[string][2]string{
		"github.com:/o/r":      {"github.com", "/o/r"},
		"github.com:8443:/o/r": {"github.com:8443", "/o/r"},
		":/tmp/x/origin":       {"", "/tmp/x/origin"},
	} {
		if h, p := SplitKey(k); h != want[0] || p != want[1] {
			t.Errorf("SplitKey(%q) = %q, %q; want %q, %q", k, h, p, want[0], want[1])
		}
	}
}

func TestStrangerIsTheFirstAddressOfAnotherRepository(t *testing.T) {
	keys := Keys("https://github.com/o/r", "git@github.com:o/r.git", "")
	if len(keys) != 1 {
		t.Fatalf("Keys = %v; want the one repository's key", keys)
	}
	if _, _, found := Stranger([]string{"https://github.com/o/r.git", "ssh://git@github.com/O/R"}, keys); found {
		t.Fatal("an address of the forge's repository was a stranger")
	}
	u, k, found := Stranger([]string{"https://github.com/o/r", "https://github.com/o/fork", "x"}, keys)
	if !found || u != "https://github.com/o/fork" || k != "github.com:/o/fork" {
		t.Fatalf("Stranger = %q, %q, %v; want the fork", u, k, found)
	}
	if _, k, found := Stranger([]string{""}, keys); !found || k != "" {
		t.Fatalf("an unreadable address: key %q, found %v", k, found)
	}
}
