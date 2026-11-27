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
		{`echo update v0.7.0 | xargs -J % abcd %`, "abcd-ahoy-shared-folder"},
		{`echo update v0.7.0 | xargs -J% scripts/abcd.sh %`, stopAny},
		{`find update v0.7.0 -maxdepth 0 -exec abcd {} +`, "abcd-ahoy-shared-folder"},
		{`find . -exec scripts/abcd.sh {} +`, stopAny},
		{`echo update v0.7.0 | xargs -I{} sh -c 'abcd {}'`, "abcd-ahoy-shared-folder"},
		{`find . -exec sh -c 'abcd {}' \;`, "abcd-ahoy-shared-folder"},
		{`echo v0.7.0 | xargs -tI% abcd update %`, tag},
		{`echo v0.7.0 | xargs -tI% sh -c 'abcd update %'`, tag},
		{`echo v0.7.0 | xargs -rtI % sh -c 'abcd update %'`, tag},
		{`echo v0.7.0 | xargs -rtI % abcd update %`, tag},
		{`echo v0.7.0 | xargs -0tJ@ abcd update @`, tag},
		{`find . -exec abcd implement {} +`, stop},
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
		{`cat tag.txt | xargs "$OPT" abcd update`, "abcd-update-tag"},
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
		`echo x | xargs -tI% echo %`,
		`echo x | xargs -J % cp % /tmp`,
		`find . -name '*.go' -exec gofmt -l {} +`,
		`echo v0.7.0 | xargs -tn1 -I% abcd explain %`,
		`echo x | xargs -tL1 abcd capture`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
}

// TestAPlaceholderAsTheProgramIsReadAsItsWords: xargs's replacement string standing
// as the program's own word is the words xargs puts there, and GNU's abbreviation
// of --replace names one too.
func TestAPlaceholderAsTheProgramIsReadAsItsWords(t *testing.T) {
	reg := Defaults()
	for _, cmd := range []string{
		`echo abcd | xargs -I{} {} update v0.7.0`,
		`echo abcd | xargs -tI% % update v0.7.0`,
		`echo abcd update | xargs -J % % v0.7.0`,
		`echo abcd | xargs -i {} update v0.7.0`,
		`echo v0.7.0 | xargs --repl=% abcd update %`,
		`echo v0.7.0 | xargs --r=% abcd update %`,
		`echo abcd update | xargs -J "$(echo X)" X v1.2.3`,
		`echo abcd | xargs -I"$(echo X)" X update v1.2.3`,
		`echo abcd update | xargs $O X v1.2.3`,
		`echo abcd | xargs -I "$R" X update v1.2.3`,
		`echo abcd update v1.2.3 | xargs -J % %`,
		`echo ls | xargs -J % % -l`,
		`echo pkill make | xargs -I{} sh -c '{}'`,
		`echo pkill make | xargs -I% bash -c 'echo start; %'`,
		`echo pkill make | xargs sh -c`,
		`echo pkill make | xargs -I{} zsh -c "{}"`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict != VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
	for _, cmd := range []string{
		`echo x | xargs --repl=% echo %`,
		`ls | xargs -I{} sh -c 'echo "$1"' _ {}`,
		`ls | xargs sh -c 'echo "$@"' _`,
	} {
		if d, err := reg.Check(cmd); err != nil || d.Verdict == VerdictBlock {
			t.Errorf("Check(%q) = %q via %q, %v; want it not refused", cmd, d.Verdict, d.EntryID, err)
		}
	}
}
