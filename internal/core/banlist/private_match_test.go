package banlist

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
)

// TestMatchPrivateMasksAMatchingName: the names the private layer matches are
// reported, case-insensitively and by the guard's own engine, and nothing else.
func TestMatchPrivateMasksAMatchingName(t *testing.T) {
	if _, err := exec.LookPath("grep"); err != nil {
		t.Skip("grep unavailable: the enforcement engine cannot be driven")
	}
	root := t.TempDir()
	writePrivate(t, root, privateFormatDecl+"\nclient-a widget[0-9]+\nhost-b ^zeta-host$\n")

	got, err := MatchPrivate(root, []string{"yes", "WIDGET42", "zeta-host", "zeta-hostname", "plain widget"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []bool{false, true, true, false, false}; !slices.Equal(got, want) {
		t.Fatalf("matches = %v, want %v", got, want)
	}

	// A legacy store matches its whole-line patterns the same way.
	writePrivate(t, root, "secretname\n")
	got, err = MatchPrivate(root, []string{"a-secretname-b", "other"})
	if err != nil || !slices.Equal(got, []bool{true, false}) {
		t.Fatalf("legacy store: %v, %v", got, err)
	}
}

// TestMatchPrivateAbsentStoreMatchesNothing: a machine that has not opted in has
// nothing to mask, and that is not an error.
func TestMatchPrivateAbsentStoreMatchesNothing(t *testing.T) {
	got, err := MatchPrivate(t.TempDir(), []string{"anything"})
	if err != nil || !slices.Equal(got, []bool{false}) {
		t.Fatalf("absent store: %v, %v", got, err)
	}
	if got, err := MatchPrivate(t.TempDir(), nil); err != nil || len(got) != 0 {
		t.Fatalf("no names: %v, %v", got, err)
	}
}

// TestMatchPrivateFailsClosed: a store that cannot be read for what it is, a
// pattern the engine refuses, or no engine at all is an error, never a clean
// answer, so the caller withholds every name.
func TestMatchPrivateFailsClosed(t *testing.T) {
	grep, lookErr := exec.LookPath("grep")
	cases := map[string]func(t *testing.T, root string){
		"unparsed keyed line": func(t *testing.T, root string) {
			writePrivate(t, root, privateFormatDecl+"\nonly-a-key\n")
		},
		"damaged declaration": func(t *testing.T, root string) {
			writePrivate(t, root, "  "+privateFormatDecl+"\nk v\n")
		},
		"unreadable store": func(t *testing.T, root string) {
			path := writePrivate(t, root, "x\n")
			if err := os.Remove(path); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join(root, "elsewhere"), path); err != nil {
				t.Fatal(err)
			}
		},
	}
	if lookErr == nil {
		cases["pattern the engine refuses"] = func(t *testing.T, root string) {
			writePrivate(t, root, privateFormatDecl+"\nbad [a-z-.\n")
		}
	}
	for name, setup := range cases {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			setup(t, root)
			if got, err := matchPrivateWith(root, []string{"name"}, grep); err == nil {
				t.Fatalf("no error; matches = %v", got)
			}
		})
	}

	t.Run("no grep", func(t *testing.T) {
		root := t.TempDir()
		writePrivate(t, root, privateFormatDecl+"\nk v\n")
		_, err := matchPrivateWith(root, []string{"name"}, "")
		if !errors.Is(err, ErrNoEngine) {
			t.Fatalf("err = %v, want ErrNoEngine", err)
		}
		_, err = matchPrivateWith(root, []string{"name"}, filepath.Join(root, "no-such-grep"))
		if err == nil {
			t.Fatal("an engine that cannot be run answered")
		}
	})
}
