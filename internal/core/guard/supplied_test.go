package guard

import "testing"

// TestSuppliedWordsAreReadAsAnActsWords: a word xargs adds, or puts where its
// replacement string stands, and a path a find puts where `{}` stands, are words
// the line does not fix, read as the act's verb and its operand can be. A command
// whose supplied words can be neither is left alone, and so is another program.
func TestSuppliedWordsAreReadAsAnActsWords(t *testing.T) {
	reg := Defaults()
	const tag, tagAny = "abcd-update-tag", updateTagAnyEntryID
	const stop, stopAny = "abcd-implement-stop", implementStopAnyEntryID
	for _, c := range []struct{ cmd, entry string }{
		{`echo v0.7.0 | xargs abcd update`, tag},
		{`echo v0.7.0 | xargs scripts/abcd.sh update`, tagAny},
		{`echo v0.7.0 | xargs -I{} abcd update {}`, tag},
		{`echo v0.7.0 | xargs -I{} scripts/abcd.sh update {}`, tagAny},
		{`echo v0.7.0 | xargs -J % abcd update %`, tag},
		{`echo v0.7.0 | xargs -i abcd update {}`, tag},
		{`echo v0.7.0 | xargs -i@ abcd update @`, tag},
		{`echo v0.7.0 | xargs --replace=@ scripts/abcd.sh update @`, tagAny},
		{`echo 0.7 | xargs -I@ abcd update v@.0`, tag},
		{`echo v0.7.0 | xargs -0 -r -n1 abcd update --yes`, tag},
		{`xargs -a tag.txt abcd update --json`, tag},
		{`xargs scripts/abcd.sh --json update < tag.txt`, tagAny},
		{`cat tag.txt | xargs go run ./cmd/abcd update`, tag},
		{`cat tag.txt | xargs env X=1 abcd update`, tag},
		{`cat tag.txt | sudo xargs abcd update`, tag},
		{`cat tag.txt | $SUDO xargs abcd update`, tag},
		{`cat tag.txt | /usr/bin/xargs bin/abcd-darwin-arm64 update`, tagAny},
		{`cat tag.txt | xargs bash scripts/abcd.sh update`, tagAny},
		{`cat tag.txt | xargs -I% sh -c 'abcd update %'`, tag},
		{`cat tag.txt | xargs "$OPT" abcd update`, tag},
		{`f() { xargs abcd update; }; f < tag.txt`, tag},
		{`echo update | xargs -I{} abcd {} v0.7.0`, tag},
		{`echo update v0.7.0 | xargs abcd`, "abcd-ahoy-shared-folder"},
		{`echo update v0.7.0 | xargs scripts/abcd.sh`, stopAny},
		{`find . -name tag -exec abcd update {} \;`, tag},
		{`find . -name tag -exec abcd update {} +`, tag},
		{`find . -name tag -execdir scripts/abcd.sh update {} \;`, tagAny},
		{`find . -name tag -ok abcd update --yes {} \;`, tag},
		{`find . -name tag -exec env X=1 abcd update {} \;`, tag},
		{`find . -name tag -exec abcd update v0.7.0 \;`, tag},
		{`find . -name tag -exec sh -c 'abcd update {}' \;`, tag},
		{`find . -name tag -exec sh -c 'abcd update "$1"' _ {} \;`, tag},
		{`find . -name tag -exec bash scripts/abcd.sh update {} \;`, tagAny},
		{`find "$D" -name tag -exec abcd update {} \;`, tag},
		{`find "$D" -name "$N" -exec scripts/abcd.sh update {} \;`, tagAny},
		{`sudo find . -name tag -exec abcd update {} \;`, tag},
		{`find . -name '*.sh' -exec {} \; -exec abcd update {} \;`, tag},
		{`abcd update {}`, tag},
		{`parallel 'abcd update {}' ::: v0.7.0`, tag},

		{`echo stop | xargs abcd implement`, stop},
		{`echo stop | xargs scripts/abcd.sh implement`, stopAny},
		{`echo stop | xargs -I{} abcd implement {} --lift`, stop},
		{`find . -name stop -exec abcd implement {} \;`, stop},
		{`find . -name stop -exec scripts/abcd.sh implement {} \;`, stopAny},
		{`echo shared-folder /x | xargs scripts/abcd.sh ahoy`, sharedFolderAnyEntryID},
	} {
		d, err := reg.Check(c.cmd)
		if err != nil || d.Verdict != VerdictBlock || d.EntryID != c.entry {
			t.Errorf("Check(%q) = %q via %q, %v; want a block via %q", c.cmd, d.Verdict, d.EntryID, err, c.entry)
		}
	}
	for _, cmd := range []string{
		`ls | xargs abcd capture`,
		`ls | xargs abcd explain`,
		`ls | xargs -n1 scripts/abcd.sh docs lint`,
		`ls | xargs -I{} abcd explain {}`,
		`ls | xargs abcd update news`,
		`ls | xargs -I{} abcd update news --answer {}`,
		`ls | xargs -I{} abcd update --check`,
		`ls | xargs -I{} scripts/abcd.sh update --yes --json`,
		`ls | xargs -I% abcd update {}x/y`,
		`ls | xargs npm update`,
		`ls | xargs ./deploy.sh update`,
		`ls | xargs -I{} ./deploy.sh update {}`,
		`ls | xargs grep 'abcd update'`,
		`ls | xargs echo abcd update`,
		`ls | xargs abcd implement receipt`,
		`ls | xargs scripts/abcd.sh implement step --run`,
		`xargs -a files.txt "$TOOL"`,
		`"$WRAP" abcd update --yes`,
		`$SUDO abcd update`,
		`find . -name tag -exec abcd capture {} \;`,
		`find . -name tag -exec abcd update \;`,
		`find . -name tag -exec abcd update --check \;`,
		`find . -name tag -exec abcd update news {} \;`,
		`find . -name tag -exec ./deploy.sh update {} \;`,
		`find . -name tag -exec grep 'abcd update' {} +`,
		`find "$D" -name abcd`,
		`find "$D" -path '*/abcd' -exec ls {} \;`,
		`find . -name tag -exec abcd {} \;`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
}
