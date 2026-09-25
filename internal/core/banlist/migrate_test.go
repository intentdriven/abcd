package banlist

import (
	"errors"
	"strings"
	"testing"
)

// TestMigratePrivateKeysEveryLegacyLineAsTheGuardAlreadyNamesIt is the migration a
// legacy store is offered (iss-2609252007433563): the format declaration goes on
// line 1, every whole-line pattern keeps its exact bytes under the key the guard
// already prints for it (entry-<its original line>), and every comment and blank
// line survives. A second run finds a keyed store and writes nothing.
func TestMigratePrivateKeysEveryLegacyLineAsTheGuardAlreadyNamesIt(t *testing.T) {
	root := t.TempDir()
	legacy := "\xef\xbb\xbf# my own notes\nwidgetworks\n\t spaced out pattern \r\n\nfoo[.]bar\n"
	writePrivate(t, root, legacy)

	before, keyed, err := parse([]byte(legacy))
	if err != nil || keyed {
		t.Fatalf("fixture is not a legacy store: %v %v", keyed, err)
	}
	res, err := MigratePrivate(root)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Migrated || res.Entries != 3 || res.Path != PrivateRelPath {
		t.Fatalf("result = %+v", res)
	}
	body := readStore(t, root)
	if !strings.HasPrefix(body, privateFormatDecl+"\n# my own notes\n") {
		t.Fatalf("the declaration is not line 1, or the header comment was lost:\n%q", body)
	}
	after, keyed, err := parse([]byte(body))
	if err != nil || !keyed {
		t.Fatalf("migrated store does not read as keyed: %v %v", keyed, err)
	}
	if len(after) != len(before) {
		t.Fatalf("entries: %d before, %d after", len(before), len(after))
	}
	for i := range before {
		if after[i].unparsed || after[i].key != before[i].key || after[i].pattern != before[i].pattern {
			t.Errorf("entry %d: before (%s), after (%s, unparsed=%v); the key or the pattern moved",
				i, before[i].key, after[i].key, after[i].unparsed)
		}
	}

	res, err = MigratePrivate(root)
	if err != nil || res.Migrated {
		t.Fatalf("second migration: %+v %v", res, err)
	}
	if readStore(t, root) != body {
		t.Fatal("a migration of a keyed store rewrote it")
	}
}

// TestMigratePrivateRefusals: there is nothing to migrate in an absent store, and a
// damaged declaration is the parser's refusal, never a migration.
func TestMigratePrivateRefusals(t *testing.T) {
	if _, err := MigratePrivate(t.TempDir()); !errors.Is(err, ErrNoStore) {
		t.Fatalf("absent store: %v, want ErrNoStore", err)
	}
	root := t.TempDir()
	damaged := " " + privateFormatDecl + "\nkey widgetworks\n"
	writePrivate(t, root, damaged)
	if _, err := MigratePrivate(root); !errors.Is(err, ErrMalformedStore) {
		t.Fatalf("damaged declaration: %v, want ErrMalformedStore", err)
	}
	if readStore(t, root) != damaged {
		t.Fatal("a refused migration wrote the store")
	}
}

// TestLegacyRefusalNamesTheMigration: every verb that refuses a legacy store names
// the command that migrates it, rather than asking for every line to be keyed by hand.
func TestLegacyRefusalNamesTheMigration(t *testing.T) {
	root := t.TempDir()
	writePrivate(t, root, "widgetworks\n")
	_, err := AddPrivate(AddPrivateRequest{RepoRoot: root, Key: "k", Pattern: "other"})
	if !errors.Is(err, ErrLegacyStore) || !strings.Contains(err.Error(), "abcd banlist migrate") {
		t.Fatalf("legacy refusal: %v", err)
	}
	if strings.Contains(err.Error(), "widgetworks") {
		t.Fatalf("the refusal quotes a pattern: %v", err)
	}
}

// TestTheGuardRefusesTheSameKeyBeforeAndAfterMigration drives the committed guard
// over one legacy store and its migration: the same staged text is refused, under
// the same key, both times.
func TestTheGuardRefusesTheSameKeyBeforeAndAfterMigration(t *testing.T) {
	legacy := "# notes\nwidgetworks\n"
	blocked, out := hookRun(t, legacy, "the widgetworks draft\n")
	if !blocked || !strings.Contains(out, "entry-2") {
		t.Fatalf("the legacy store does not refuse under entry-2\n%s", out)
	}
	root := t.TempDir()
	writePrivate(t, root, legacy)
	if _, err := MigratePrivate(root); err != nil {
		t.Fatal(err)
	}
	blocked, out = hookRun(t, readStore(t, root), "the widgetworks draft\n")
	if !blocked || !strings.Contains(out, "entry-2") || !strings.Contains(out, "keyed store") {
		t.Fatalf("the migrated store does not refuse under entry-2 as a keyed store\n%s", out)
	}
}
