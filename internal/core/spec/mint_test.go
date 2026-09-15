package spec

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/intentdriven/abcd/internal/core/frontmatter"
	"github.com/intentdriven/abcd/internal/core/recordid"
)

// nativeSpecIDRe is the shape of a native spc id: the family tag, a 12-digit
// UTC second stamp and a 4-digit suffix (adr-45; mechanics per spc-33).
var nativeSpecIDRe = regexp.MustCompile(`^spc-[0-9]{16}$`)

// TestCreateMintsPastTheOrdinalsWithoutCounting is adr-45 ruling 2 at the spec
// store: a checkout whose tree already holds spc-69, with an intent reserving a
// higher number in its spec_id, mints a timestamp-shaped id — never spc-70 and
// never the reservation plus one. The mint reads no maximum: not the store's,
// not the intents', not the refs'.
func TestCreateMintsPastTheOrdinalsWithoutCounting(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, specsOpen+"/spc-69-existing.md",
		"---\nid: spc-69\nslug: existing\nintent: itd-9\n---\n# ok\n")
	writeFile(t, root, intentsBase+"/planned/itd-20-x.md",
		"---\nid: itd-20\nslug: x\nspec_id: spc-80\nkind: standalone\n---\n# ok\n")

	sp, err := Create(root, "itd-200", "my-feature", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if sp.ID == "spc-70" || sp.ID == "spc-81" {
		t.Fatalf("minted %s: the mint counted a maximum instead of stamping the clock", sp.ID)
	}
	if !nativeSpecIDRe.MatchString(sp.ID) {
		t.Fatalf("minted id %q is not native-shaped (spc-<yymmddHHMMSS><rrrr>)", sp.ID)
	}
	if !strings.HasPrefix(filepath.Base(sp.Path), sp.ID+"-") {
		t.Fatalf("filename %q does not carry the minted id %s", sp.Path, sp.ID)
	}
	data, err := os.ReadFile(filepath.Join(root, sp.Path))
	if err != nil {
		t.Fatalf("spec file unreadable: %v", err)
	}
	fields := frontmatter.Fields(strings.Split(string(data), "\n"))
	if fields["id"].Value != sp.ID || fields["intent"].Value != "itd-200" {
		t.Fatalf("frontmatter = id %q intent %q, want %s / itd-200", fields["id"].Value, fields["intent"].Value, sp.ID)
	}
	// The ordinal stays exactly as minted, next to the native id, and the store
	// resolves both.
	store, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"spc-69", sp.ID} {
		if _, ok := store.Lookup(id); !ok {
			t.Errorf("%s missing from the store after the mint", id)
		}
	}
}

// setMinter swaps the package mint seam for the test's lifetime.
func setMinter(t *testing.T, m recordid.Minter) {
	t.Helper()
	prev := minter
	minter = m
	t.Cleanup(func() { minter = prev })
}

// TestCreateInTwoCheckoutsNeverCollides is the spec store's half of the
// 2026-08-22 recurrence (iss-2608221126066632): two current checkouts of one
// tree each plan an intent in the same second. A maximum-counting allocator
// hands both the same spc id, the checkout-scoped lock notwithstanding; the
// timestamp-numeric mint separates them by entropy alone, so the draws are
// scripted (42, then 4369) and the assertion is exact.
func TestCreateInTwoCheckoutsNeverCollides(t *testing.T) {
	instant := time.Date(2026, 8, 22, 11, 26, 6, 0, time.UTC)
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return instant },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x11, 0x11}),
	})

	checkoutA, checkoutB := t.TempDir(), t.TempDir()
	for _, root := range []string{checkoutA, checkoutB} {
		writeFile(t, root, specsOpen+"/spc-69-existing.md",
			"---\nid: spc-69\nslug: existing\nintent: itd-9\n---\n# ok\n")
	}

	spA, err := Create(checkoutA, "itd-144", "alpha", "")
	if err != nil {
		t.Fatalf("checkout A Create: %v", err)
	}
	spB, err := Create(checkoutB, "itd-145", "beta", "")
	if err != nil {
		t.Fatalf("checkout B Create: %v", err)
	}
	if spA.ID == spB.ID {
		t.Fatalf("two checkouts minted one id in the same second: %s", spA.ID)
	}
	if spA.ID != "spc-2608221126060042" || spB.ID != "spc-2608221126064369" {
		t.Fatalf("ids = %s, %s: want the pinned instant with the scripted suffixes 0042 and 4369", spA.ID, spB.ID)
	}
}
