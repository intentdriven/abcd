package mode_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/core/mode"
)

// TestParseStateVocabularyIsClosed pins the whole of the vocabulary: the three
// words parse (with the trailing newline the writer emits, and with the
// surrounding whitespace a shell argument or a hand edit carries), and everything
// else is refused with an error that names all three. A fourth accepted word is a
// badge whose meaning is set by whoever last wrote the file.
func TestParseStateVocabularyIsClosed(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		raw  string
		want mode.State
		ok   bool
	}{
		{"managed", "managed", mode.Managed, true},
		{"facilitator", "facilitator", mode.Facilitator, true},
		{"product thinker", "product-thinker", mode.ProductThinker, true},
		{"stored line carries the writer's newline", "facilitator\n", mode.Facilitator, true},
		{"surrounding whitespace is trimmed", "  product-thinker \t\n", mode.ProductThinker, true},
		{"empty is not absent", "", "", false},
		{"whitespace only is not absent", " \n\t ", "", false},
		{"a fourth word is refused", "waiting", "", false},
		{"case is not folded", "Facilitator", "", false},
		{"upper case is not folded", "PRODUCT-THINKER", "", false},
		{"two words are refused", "managed facilitator", "", false},
		{"two lines are refused", "managed\nfacilitator\n", "", false},
		{"a near miss is refused", "product_thinker", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got, err := mode.ParseState(tc.raw)
			if tc.ok {
				if err != nil {
					t.Fatalf("ParseState(%q): unexpected error: %v", tc.raw, err)
				}
				if got != tc.want {
					t.Fatalf("ParseState(%q) = %q, want %q", tc.raw, got, tc.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ParseState(%q) = %q, want a refusal", tc.raw, got)
			}
			if !errors.Is(err, mode.ErrUnknownState) {
				t.Fatalf("ParseState(%q) error %v does not wrap ErrUnknownState", tc.raw, err)
			}
			if got != "" {
				t.Fatalf("ParseState(%q) returned %q alongside its refusal, want the zero state", tc.raw, got)
			}
			for _, s := range mode.States() {
				if !strings.Contains(err.Error(), string(s)) {
					t.Fatalf("ParseState(%q) refusal %q does not name %q", tc.raw, err, s)
				}
			}
		})
	}
}

// TestParseStateRefusalIsTerminalSafe pins that a refusal quoting a hand-mangled
// store back at a human carries no raw escape: the file's content is untrusted,
// and left raw an ANSI sequence in it rewrites the report that reports it.
func TestParseStateRefusalIsTerminalSafe(t *testing.T) {
	t.Parallel()
	_, err := mode.ParseState("\x1b[31mmanaged\x1b[0m")
	if err == nil {
		t.Fatal("ParseState of an escape-bearing word: want a refusal")
	}
	if strings.ContainsRune(err.Error(), 0x1b) {
		t.Fatalf("refusal carries a raw ESC: %q", err.Error())
	}
}

// TestStatesEnumeratesTheVocabularyOnce pins that States() is the one
// enumeration, in the documented order, and that every member of it is Valid.
func TestStatesEnumeratesTheVocabularyOnce(t *testing.T) {
	t.Parallel()
	want := []mode.State{mode.Managed, mode.Facilitator, mode.ProductThinker}
	got := mode.States()
	if len(got) != len(want) {
		t.Fatalf("States() = %v, want %v", got, want)
	}
	for i, s := range want {
		if got[i] != s {
			t.Fatalf("States()[%d] = %q, want %q", i, got[i], s)
		}
		if !s.Valid() {
			t.Fatalf("%q is in States() but not Valid()", s)
		}
		if s.String() != string(s) {
			t.Fatalf("String() of %q = %q", s, s.String())
		}
	}
	if mode.State("waiting").Valid() {
		t.Fatal(`Valid() accepted "waiting"`)
	}
	if mode.State("").Valid() {
		t.Fatal("Valid() accepted the zero state")
	}
}

// TestStorePathsSitInTheLocalTier pins the store's placement. The tier is
// gitignored and per worktree, and the managed-only property rests on the file
// being inside it — a path that drifted out of the tier would be a committed
// candidate and would read the same in an unmanaged repository.
func TestStorePathsSitInTheLocalTier(t *testing.T) {
	t.Parallel()
	if mode.TierRelPath != ".abcd/.work.local" {
		t.Fatalf("TierRelPath = %q, want the local-ephemeral tier", mode.TierRelPath)
	}
	if mode.FileRelPath != ".abcd/.work.local/mode" {
		t.Fatalf("FileRelPath = %q", mode.FileRelPath)
	}
	if !strings.HasPrefix(mode.FileRelPath, mode.TierRelPath+"/") {
		t.Fatalf("FileRelPath %q is not inside TierRelPath %q", mode.FileRelPath, mode.TierRelPath)
	}
}

// TestAddresseeNamesWhoseAnswerIsOwed: the two role states each name a person
// in plain words, and the quiet state names nobody — which is what lets a
// front door print "waiting on the <addressee>" for a parked loop and nothing
// at all for `managed`, without inventing the words itself (ac-7).
func TestAddresseeNamesWhoseAnswerIsOwed(t *testing.T) {
	cases := map[mode.State]string{
		mode.Managed:        "",
		mode.Facilitator:    "facilitator",
		mode.ProductThinker: "product thinker",
	}
	for st, want := range cases {
		if got := st.Addressee(); got != want {
			t.Errorf("%s.Addressee() = %q, want %q", st, got, want)
		}
	}
	if got := mode.State("bogus").Addressee(); got != "" {
		t.Errorf("an unknown state names an addressee: %q", got)
	}
}
