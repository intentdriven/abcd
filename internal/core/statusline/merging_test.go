package statusline

import (
	"context"
	"testing"
)

// The merging element (itd-2610150602547271, spc-2610161609350616 step 7).
// While the repository's facilitator.mode is auto, the row says merging is
// autonomous; under human, or where the mode cannot be read, it says nothing.
// The mode is read through MergingMode, which the front door sets to
// landing.AutonomousMerging: this package cannot import landing, so the
// reading itself — the default branch's committed .abcd/config.json and the
// machine layer, never the working tree's copy — is proved in landing's tests.
// What is proved here is the element's wiring: what the row does with the
// reader's answer, with no reader, and with the element switched off.

// withMergingMode sets MergingMode to read for the rest of the test, and
// restores whatever was there after it.
func withMergingMode(t *testing.T, read func(ctx context.Context, root string) (bool, string)) {
	t.Helper()
	prev := MergingMode
	MergingMode = read
	t.Cleanup(func() { MergingMode = prev })
}

// mergingAnswer is a reader that gives auto and note whatever it is asked,
// and counts the roots it was asked about.
func mergingAnswer(auto bool, note string, asked *[]string) func(context.Context, string) (bool, string) {
	return func(_ context.Context, root string) (bool, string) {
		if asked != nil {
			*asked = append(*asked, root)
		}
		return auto, note
	}
}

func TestTheRowSaysMergingIsAutonomousUnderAuto(t *testing.T) {
	root := composeRepo(t)
	var asked []string
	withMergingMode(t, mergingAnswer(true, "", &asked))
	res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if len(asked) != 1 || asked[0] != root {
		t.Errorf("the reader was asked about %q, want once about %q", asked, root)
	}
	if got := plainOf(t, res.Row, KeyMerging); got != "autonomous merging" {
		t.Errorf("merging element = %q, want %q", got, "autonomous merging")
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %q, want none", res.Notes)
	}
	branch, merging := -1, -1
	for i, el := range res.Row.Elements {
		switch el.Key {
		case KeyBranch:
			branch = i
		case KeyMerging:
			merging = i
		}
	}
	if branch < 0 || merging != branch+1 {
		t.Errorf("the merging element does not follow the branch: %q", res.Row.Plain())
	}
}

func TestTheRowSaysNothingOfMergingOtherwise(t *testing.T) {
	root := composeRepo(t)
	withMergingMode(t, mergingAnswer(false, "", nil))
	res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if el, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("the row says %q: %q", el.Plain, res.Row.Plain())
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %q, want none", res.Notes)
	}
}

// TestWithNoMergingReaderTheElementDropsSilently: a composition no front door
// handed a reader has nothing it failed to read, so the element drops with no
// note.
func TestWithNoMergingReaderTheElementDropsSilently(t *testing.T) {
	root := composeRepo(t)
	withMergingMode(t, nil)
	res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if el, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("the row says %q with no reader: %q", el.Plain, res.Row.Plain())
	}
	if _, ok := res.Row.Element(KeyPresence); !ok {
		t.Errorf("the badge dropped: %q", res.Row.Plain())
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %q, want none", res.Notes)
	}
}

func TestTheMergingElementIsSwitchable(t *testing.T) {
	// A mode that does not read would be a note: switched off, the mode is
	// not read at all, so there is none.
	root := composeRepo(t)
	var asked []string
	withMergingMode(t, mergingAnswer(false, "statusline: DROPPED the merging element — facilitator.mode could not be read (x)", &asked))
	set := Defaults()
	if !set.Enabled(KeyMerging) {
		t.Fatal("the merging element is off by default")
	}
	set.Elements[KeyMerging] = false
	res, err := ComposeContext(context.Background(), root, Payload{}, set)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("a switched-off element rendered: %q", res.Row.Plain())
	}
	if len(res.Notes) != 0 {
		t.Errorf("notes = %q, want none: a switched-off element reads nothing", res.Notes)
	}

	withMergingMode(t, mergingAnswer(true, "", &asked))
	res, err = ComposeContext(context.Background(), root, Payload{}, set)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("a switched-off element rendered under auto: %q", res.Row.Plain())
	}
	if len(asked) != 0 {
		t.Errorf("a switched-off element asked its reader about %q", asked)
	}
}

// TestAReaderNoteDropsTheMergingElementAndIsPassedOn: a reader that could not
// read the mode returns a note; the row drops the element, keeps the rest,
// and hands the note on, as it is, for the front door's stderr.
func TestAReaderNoteDropsTheMergingElementAndIsPassedOn(t *testing.T) {
	root := composeRepo(t)
	const note = "statusline: DROPPED the merging element — facilitator.mode could not be read (a value that is not a mode)"
	withMergingMode(t, mergingAnswer(false, note, nil))
	res, err := ComposeContext(context.Background(), root, Payload{}, Defaults())
	if err != nil {
		t.Fatal(err)
	}
	if el, ok := res.Row.Element(KeyMerging); ok {
		t.Errorf("the row says %q over a mode that does not read", el.Plain)
	}
	if _, ok := res.Row.Element(KeyPresence); !ok {
		t.Errorf("the badge dropped: %q", res.Row.Plain())
	}
	if len(res.Notes) != 1 || res.Notes[0] != note {
		t.Fatalf("notes = %q, want exactly the reader's note", res.Notes)
	}
}
