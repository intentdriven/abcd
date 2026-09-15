package intent

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

// nativeIntentIDRe is the shape of a native itd id: the family tag, a 12-digit
// UTC second stamp and a 4-digit suffix (adr-45; mechanics per spc-33).
var nativeIntentIDRe = regexp.MustCompile(`^itd-[0-9]{16}$`)

// nativeSpecIDRe is the same shape for the spec the lifecycle mints.
var nativeSpecIDRe = regexp.MustCompile(`^spc-[0-9]{16}$`)

// TestCreateFromTextMintsPastTheOrdinalsWithoutCounting is adr-45 ruling 2 at
// the intent store: a checkout whose tree already holds itd-200 mints a
// timestamp-shaped id, never itd-201. The mint reads no maximum, so the highest
// ordinal in the tree has no bearing on the id it hands out — which is what
// keeps two checkouts that share the same view from converging on one id.
func TestCreateFromTextMintsPastTheOrdinalsWithoutCounting(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, draftsDir+"/itd-200-alpha.md", draftWithAC("itd-200", "alpha"))
	writeFile(t, root, plannedDir+"/itd-9-beta.md",
		"---\nid: itd-9\nslug: beta\nspec_id: spc-1\nkind: standalone\n---\n# beta\n")

	it, err := CreateFromText(root, "another product intent", "", "")
	if err != nil {
		t.Fatalf("CreateFromText: %v", err)
	}
	if it.ID == "itd-201" {
		t.Fatalf("minted %s: the mint counted the tree's maximum instead of stamping the clock", it.ID)
	}
	if !nativeIntentIDRe.MatchString(it.ID) {
		t.Fatalf("minted id %q is not native-shaped (itd-<yymmddHHMMSS><rrrr>)", it.ID)
	}
	// The written record spells the same id in its frontmatter and its filename.
	if !strings.HasPrefix(filepath.Base(it.Path), it.ID+"-") {
		t.Fatalf("filename %q does not carry the minted id %s", it.Path, it.ID)
	}
	data, err := os.ReadFile(filepath.Join(root, it.Path))
	if err != nil {
		t.Fatalf("created file unreadable: %v", err)
	}
	fields := frontmatter.Fields(strings.Split(string(data), "\n"))
	if fields["id"].Value != it.ID {
		t.Fatalf("frontmatter id = %q, want %s", fields["id"].Value, it.ID)
	}
	// The ordinals stay exactly as minted (adr-45 ruling 2): the tree holds three
	// intents, the two seeded ones untouched.
	c, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"itd-200", "itd-9", it.ID} {
		if _, ok := c.Lookup(id); !ok {
			t.Errorf("%s missing from the corpus after the mint", id)
		}
	}
}

// TestPlanMintsATimestampSpecForATimestampDraft is the lifecycle on a native
// id: planning a draft whose id is timestamp-shaped mints a timestamp-shaped
// spec and writes both sides of the link, so a record minted under adr-45
// travels the same path an ordinal one does.
func TestPlanMintsATimestampSpecForATimestampDraft(t *testing.T) {
	root := t.TempDir()
	const draftID = "itd-2608221126066632"
	writeFile(t, root, draftsDir+"/"+draftID+"-alpha.md", draftWithAC(draftID, "alpha"))

	res, err := Plan(root, draftID, "")
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !nativeSpecIDRe.MatchString(res.Spec.ID) {
		t.Fatalf("planned spec id %q is not native-shaped (spc-<yymmddHHMMSS><rrrr>)", res.Spec.ID)
	}
	if res.Spec.Intent != draftID {
		t.Fatalf("spec intent = %q, want %s", res.Spec.Intent, draftID)
	}
	if res.Intent.Bucket != BucketPlanned || res.Intent.SpecID != res.Spec.ID {
		t.Fatalf("planned intent = %+v, want bucket planned linked to %s", res.Intent, res.Spec.ID)
	}
	// Both link sides are on disk, each naming the other.
	body, err := os.ReadFile(filepath.Join(root, plannedDir, draftID+"-alpha.md"))
	if err != nil {
		t.Fatalf("planned file should exist: %v", err)
	}
	if f := frontmatter.Fields(strings.Split(string(body), "\n")); f["spec_id"].Value != res.Spec.ID {
		t.Fatalf("planned intent spec_id = %q, want %s", f["spec_id"].Value, res.Spec.ID)
	}
	sbody, err := os.ReadFile(filepath.Join(root, res.Spec.Path))
	if err != nil {
		t.Fatalf("spec file should exist at %s: %v", res.Spec.Path, err)
	}
	if f := frontmatter.Fields(strings.Split(string(sbody), "\n")); f["intent"].Value != draftID {
		t.Fatalf("spec intent = %q, want %s", f["intent"].Value, draftID)
	}
}

// setMinter swaps the package mint seam for the test's lifetime.
func setMinter(t *testing.T, m recordid.Minter) {
	t.Helper()
	prev := minter
	minter = m
	t.Cleanup(func() { minter = prev })
}

// TestCreateFromTextInTwoCheckoutsNeverCollides is the live recurrence of
// 2026-08-22 (iss-2608221126066632) as a test: two current checkouts of the
// same tree — same records, same maximum, neither stale — each mint an intent in
// the same second. Under a maximum-counting allocator both allocate the same id
// by construction, because an advisory lock scoped to one checkout cannot see
// the other. Under the timestamp-numeric mint the ids differ with both clocks
// pinned to one instant: only the entropy separates them, so the draws are
// scripted (42, then 4369) and the test cannot blame a correct mint for the
// spec's accepted same-suffix residue.
func TestCreateFromTextInTwoCheckoutsNeverCollides(t *testing.T) {
	instant := time.Date(2026, 8, 22, 11, 26, 6, 0, time.UTC)
	setMinter(t, recordid.Minter{
		Now:     func() time.Time { return instant },
		Entropy: bytes.NewReader([]byte{0x00, 0x2A, 0x11, 0x11}),
	})

	checkoutA, checkoutB := t.TempDir(), t.TempDir()
	for _, root := range []string{checkoutA, checkoutB} {
		writeFile(t, root, draftsDir+"/itd-200-alpha.md", draftWithAC("itd-200", "alpha"))
	}

	itA, err := CreateFromText(checkoutA, "the first session's intent", "", "")
	if err != nil {
		t.Fatalf("checkout A CreateFromText: %v", err)
	}
	itB, err := CreateFromText(checkoutB, "the second session's intent", "", "")
	if err != nil {
		t.Fatalf("checkout B CreateFromText: %v", err)
	}
	if itA.ID == itB.ID {
		t.Fatalf("two checkouts minted one id in the same second: %s", itA.ID)
	}
	for _, id := range []string{itA.ID, itB.ID} {
		if id == "itd-201" {
			t.Fatalf("minted %s: the id came from the tree's maximum, which both checkouts share", id)
		}
		if !nativeIntentIDRe.MatchString(id) {
			t.Fatalf("minted id %q is not native-shaped", id)
		}
	}
	if itA.ID != "itd-2608221126060042" || itB.ID != "itd-2608221126064369" {
		t.Fatalf("ids = %s, %s: want the pinned instant with the scripted suffixes 0042 and 4369", itA.ID, itB.ID)
	}
}
