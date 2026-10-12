package ahoy

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// Setup writes the stored README parts, the flag answers the pair, and
// uninstall removes them (spc-2610031704338417 step 3).

const mitLicence = "MIT License\n\nCopyright (c) 2026 The Owner\n"

// setReadmeConfig sets keys under the readme section of the repository's
// .abcd/config.json, as a stored answer would.
func setReadmeConfig(t *testing.T, repo string, kv map[string]any) {
	t.Helper()
	path := configPath(repo)
	cfg := readJSONFile(t, path)
	sub, _ := cfg["readme"].(map[string]any)
	if sub == nil {
		sub = map[string]any{}
	}
	for k, v := range kv {
		sub[k] = v
	}
	cfg["readme"] = sub
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// storedReadme is the readme section of the repository's config.
func storedReadme(t *testing.T, repo string) map[string]any {
	t.Helper()
	sub, _ := readJSONFile(t, configPath(repo))["readme"].(map[string]any)
	return sub
}

// storedReadmeParts is readme.parts as stored, and whether it is stored.
func storedReadmeParts(t *testing.T, repo string) ([]string, bool) {
	t.Helper()
	v, ok := storedReadme(t, repo)["parts"]
	if !ok {
		return nil, false
	}
	list, isList := v.([]any)
	if !isList {
		t.Fatalf("readme.parts is not a list: %#v", v)
	}
	out := []string{}
	for _, e := range list {
		out = append(out, e.(string))
	}
	return out, true
}

func readmeInstall(t *testing.T, repo string, opts InstallOptions) InstallResult {
	t.Helper()
	res, err := Install(repo, opts, RefusingPrompter{})
	if err != nil {
		t.Fatal(err)
	}
	return res
}

func noteContaining(notes []string, want string) bool {
	for _, n := range notes {
		if strings.Contains(n, want) {
			return true
		}
	}
	return false
}

// TestInstallWritesChosenReadmePartsOnce (R1): with the licence the only part
// chosen and a LICENSE whose first line is "MIT License", the first setup run
// writes one licence part naming MIT after the heading and leaves every other
// line as it was; the second writes nothing, so the file's bytes and
// modification time are those the first run left.
func TestInstallWritesChosenReadmePartsOnce(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "README.md", "# Project\n\nThe owner's prose.\n", 0o644)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	setReadmeConfig(t, repo, map[string]any{"parts": []string{"licence"}})

	first := readmeInstall(t, repo, installOpts())
	readme := filepath.Join(repo, "README.md")
	want := "# Project\n\n" + readmePart("licence", "Licensed under the MIT License; see [LICENSE](LICENSE).") + "\n\nThe owner's prose.\n"
	if got := readReadme(t, readme); got != want {
		t.Fatalf("README after the first run:\n%q\nwant\n%q\n(notes %q)", got, want, first.Notes)
	}
	if !containsString(first.Writes, "README.md") {
		t.Errorf("the first run does not report writing README.md: %v", first.Writes)
	}

	old := backdate(t, readme)
	second := readmeInstall(t, repo, installOpts())
	if containsString(second.Writes, "README.md") {
		t.Errorf("the second run reports writing README.md: %v", second.Writes)
	}
	if got := readReadme(t, readme); got != want {
		t.Errorf("the second run changed the bytes:\n%q", got)
	}
	if mt := modTime(t, readme); !mt.Equal(old) {
		t.Errorf("the second run changed the modification time: %v, want %v", mt, old)
	}
}

// TestInstallLeavesHandEditedOrBrokenReadme (R2): a licence part edited by
// hand, two licence opening marks, and an opening mark with no closing mark
// each leave README.md byte-identical, and the result's notes name the part,
// the reason and the line. A broken mark refuses every part's write, so the
// links part chosen beside the licence is not written either.
func TestInstallLeavesHandEditedOrBrokenReadme(t *testing.T) {
	const body = "Licensed under the MIT License; see [LICENSE](LICENSE)."
	part := readmePart("licence", body)
	cases := []struct {
		name, content, note string
		parts               []string
	}{
		{"edited by hand", "# Project\nprose\n\n" + handEdit("licence", body, "Licensed as I please.") + "\n",
			"licence part edited by hand at line 4; left as it is", []string{"licence"}},
		{"duplicated", "# Project\nprose\n\n" + part + "\n\nmore\n\n" + part + "\n",
			"licence part duplicated at lines 4 and 10; nothing written", []string{"licence", "links"}},
		{"unclosed", "# Project\nprose\n\n" + strings.SplitN(part, "\n", 2)[0] + "\n" + body + "\n",
			"licence part's opening mark at line 4 has no closing mark; nothing written", []string{"licence", "links"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupHermetic(t)
			repo := installedRepo(t)
			writeFixture(t, repo, "README.md", c.content, 0o644)
			writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
			writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
			setReadmeConfig(t, repo, map[string]any{"parts": c.parts})

			res := readmeInstall(t, repo, installOpts())
			if got := readReadme(t, filepath.Join(repo, "README.md")); got != c.content {
				t.Fatalf("README.md changed:\n%q\nwant\n%q", got, c.content)
			}
			if !containsString(res.Notes, c.note) {
				t.Errorf("notes = %q, want %q", res.Notes, c.note)
			}
		})
	}
}

// TestInstallCreatesReadmeWithOnlyChosenParts (R4): with no README.md and the
// licence and links parts chosen, setup creates README.md holding only those
// two parts in their marked places, no title and no prose, beginning with the
// licence part's opening mark, and records that abcd made the file.
func TestInstallCreatesReadmeWithOnlyChosenParts(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
	setReadmeConfig(t, repo, map[string]any{"parts": []string{"licence", "links"}})

	res := readmeInstall(t, repo, installOpts())
	got := readReadme(t, filepath.Join(repo, "README.md"))
	licence := readmePart("licence", "Licensed under the MIT License; see [LICENSE](LICENSE).")
	links := readmePart("links", "- [Changelog](CHANGELOG.md)")
	if want := licence + "\n\n" + links + "\n"; got != want {
		t.Fatalf("README.md:\n%q\nwant\n%q\n(notes %q)", got, want, res.Notes)
	}
	if !strings.HasPrefix(got, "<!-- BEGIN ABCD:licence ") {
		t.Errorf("README.md does not begin with the licence part's opening mark")
	}
	if storedReadme(t, repo)["created"] != true {
		t.Errorf("readme.created is not recorded: %v", storedReadme(t, repo))
	}
	if !containsString(res.Writes, "README.md") {
		t.Errorf("the created file is not reported: %v", res.Writes)
	}

	t.Run("no part derivable creates nothing and says why", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		setReadmeConfig(t, repo, map[string]any{"parts": []string{"licence", "links"}})
		res := readmeInstall(t, repo, installOpts())
		if _, err := os.Lstat(filepath.Join(repo, "README.md")); !os.IsNotExist(err) {
			t.Fatalf("README.md was created with no part to hold: %v", err)
		}
		for _, want := range []string{"licence omitted: no LICENSE file", "links omitted: none of CHANGELOG.md"} {
			if !noteContaining(res.Notes, want) {
				t.Errorf("notes %q do not say %q", res.Notes, want)
			}
		}
		if _, ok := storedReadme(t, repo)["created"]; ok {
			t.Errorf("readme.created recorded for a file never made")
		}
	})
}

// TestUninstallRemovesOnlyCurrentReadmeParts (R5): an owner's README.md
// holding no abcd part is byte-identical after uninstall; one holding a
// current part and a part edited by hand loses the current part only, and the
// receipt reads "links removed; licence edited by hand, left". A README.md
// abcd created is deleted once nothing but blank lines remains; an owner's
// file emptied the same way is kept.
func TestUninstallRemovesOnlyCurrentReadmeParts(t *testing.T) {
	const body = "Licensed under the MIT License; see [LICENSE](LICENSE)."
	links := readmePart("links", "- [Changelog](CHANGELOG.md)")

	t.Run("an owner's README with no part is untouched", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		owner := "# Project\n\nThe owner's prose.\n"
		writeFixture(t, repo, "README.md", owner, 0o644)
		path := filepath.Join(repo, "README.md")
		old := backdate(t, path)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if got := readReadme(t, path); got != owner {
			t.Fatalf("README.md changed:\n%q", got)
		}
		if mt := modTime(t, path); !mt.Equal(old) {
			t.Errorf("modification time changed")
		}
		if len(receipt.Readme.Removed) != 0 || len(receipt.Readme.Left) != 0 {
			t.Errorf("receipt = %+v, want nothing removed or left", receipt.Readme)
		}
	})

	t.Run("only the current part is removed, the edited one named", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		content := "# Project\n\n" + links + "\n\n" + handEdit("licence", body, "Licensed as I please.") + "\n\nprose\n"
		writeFixture(t, repo, "README.md", content, 0o644)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		want := "# Project\n\n" + handEdit("licence", body, "Licensed as I please.") + "\n\nprose\n"
		if got := readReadme(t, filepath.Join(repo, "README.md")); got != want {
			t.Fatalf("README.md:\n%q\nwant\n%q", got, want)
		}
		if receipt.Readme.Note != "links removed; licence edited by hand, left" {
			t.Errorf("receipt note = %q", receipt.Readme.Note)
		}
		if !reflect.DeepEqual(receipt.Readme.Removed, []string{"links"}) {
			t.Errorf("removed = %v", receipt.Readme.Removed)
		}
		if len(receipt.Readme.Left) != 1 || receipt.Readme.Left[0] != (ReadmePartLeft{Part: "licence", Reason: "edited by hand", Line: 7}) {
			t.Errorf("left = %+v", receipt.Readme.Left)
		}
	})

	t.Run("a README abcd created is deleted once empty", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
		setReadmeConfig(t, repo, map[string]any{"parts": []string{"links"}})
		readmeInstall(t, repo, installOpts())
		if got := readReadme(t, filepath.Join(repo, "README.md")); got != links+"\n" {
			t.Fatalf("precondition: README.md = %q", got)
		}
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Lstat(filepath.Join(repo, "README.md")); !os.IsNotExist(err) {
			t.Fatalf("README.md abcd made and emptied was kept: %v", err)
		}
		if !receipt.Readme.Deleted || receipt.Readme.Note != "links removed; README.md, which abcd created, deleted" {
			t.Errorf("receipt = %+v", receipt.Readme)
		}
		if _, ok := storedReadme(t, repo)["created"]; ok {
			t.Errorf("readme.created outlived the file it vouched for")
		}
	})

	t.Run("an owner's README emptied by the removal is kept", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		writeFixture(t, repo, "README.md", links+"\n", 0o644)
		if _, err := Uninstall(repo, ""); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(repo, "README.md"))
		if err != nil {
			t.Fatalf("an owner's README.md was deleted: %v", err)
		}
		if strings.TrimSpace(string(got)) != "" {
			t.Errorf("README.md = %q, want the part removed", got)
		}
	})
}

// TestInstallYesLeavesReadmeAndNamesFlag (core half of R7): a --yes setup with
// no README flag leaves README.md untouched, or absent, lists the README
// offer under optional_skipped, and the summary says the question was not
// asked and names the flag that answers it.
func TestInstallYesLeavesReadmeAndNamesFlag(t *testing.T) {
	for name, readme := range map[string]string{"absent": "", "present": "# Project\n\nprose\n"} {
		t.Run(name, func(t *testing.T) {
			setupHermetic(t)
			repo := t.TempDir()
			if err := os.Mkdir(filepath.Join(repo, ".git"), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
			if readme != "" {
				writeFixture(t, repo, "README.md", readme, 0o644)
			}
			res := readmeInstall(t, repo, installOpts())
			got, err := os.ReadFile(filepath.Join(repo, "README.md"))
			switch {
			case readme == "" && !os.IsNotExist(err):
				t.Errorf("README.md was created: %q %v", got, err)
			case readme != "" && string(got) != readme:
				t.Errorf("README.md changed: %q", got)
			}
			if !containsString(res.OptionalSkipped, ReadmeOfferGapID) {
				t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, ReadmeOfferGapID)
			}
			said := false
			for _, it := range res.Summary {
				text := it.What + " " + it.Why + " " + it.Action
				if strings.Contains(text, "The README question was not asked") && strings.Contains(text, "--readme-parts licence|links|contents|release-badge|build-badge|abcd-badge|none") {
					said = true
				}
			}
			if !said {
				t.Errorf("the summary does not say the README question was not asked and name the flag: %+v", res.Summary)
			}
			if _, ok := storedReadmeParts(t, repo); ok {
				t.Errorf("--yes stored a README answer")
			}
		})
	}
}

// TestReadmePartsFlagStoresTheAnswer: --readme-parts answers the pair in any
// run, --yes included. It stores the parts in the fixed order, echoes a change
// from the stored answer, makes an otherwise current repository do the work,
// and `none` stores an empty list, which is never asked again. An unknown
// name, or none beside a name, is refused naming the known names, and nothing
// is written.
func TestReadmePartsFlagStoresTheAnswer(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
	readme := filepath.Join(repo, "README.md")
	withFlag := func(v string) InstallOptions {
		o := installOpts()
		o.ReadmeParts = v
		return o
	}

	res := readmeInstall(t, repo, withFlag("links, licence"))
	if parts, _ := storedReadmeParts(t, repo); !reflect.DeepEqual(parts, []string{"licence", "links"}) {
		t.Fatalf("readme.parts = %v, want [licence links] (notes %q)", parts, res.Notes)
	}
	if !containsString(res.Changes, "readme.parts: not set -> licence,links") {
		t.Errorf("changes = %q", res.Changes)
	}
	if _, err := os.Stat(readme); err != nil {
		t.Fatalf("the flag's parts were not written: %v", err)
	}
	det, err := Detect(repo)
	if err != nil {
		t.Fatal(err)
	}
	if hasGap(det.Gaps, ReadmeOfferGapID) {
		t.Errorf("an answered pair is still offered")
	}

	same := readmeInstall(t, repo, withFlag("licence,links"))
	if same.Status != "already_up_to_date" || len(same.Changes) != 0 {
		t.Errorf("the same answer again: status %q changes %q", same.Status, same.Changes)
	}

	narrowed := readmeInstall(t, repo, withFlag("licence"))
	if !containsString(narrowed.Changes, "readme.parts: licence,links -> licence") {
		t.Errorf("changes = %q", narrowed.Changes)
	}

	none := readmeInstall(t, repo, withFlag("none"))
	if parts, ok := storedReadmeParts(t, repo); !ok || len(parts) != 0 {
		t.Fatalf("none stored %v (stored %v)", parts, ok)
	}
	if !containsString(none.Changes, "readme.parts: licence -> none") {
		t.Errorf("changes = %q", none.Changes)
	}
	if got := readReadme(t, readme); strings.Contains(got, "ABCD:") {
		t.Errorf("none left abcd's parts in README.md:\n%s", got)
	}

	for _, bad := range []string{"licence,wiki", "none,licence", " , "} {
		t.Run("refuses "+bad, func(t *testing.T) {
			before, _ := os.ReadFile(configPath(repo))
			res := readmeInstall(t, repo, withFlag(bad))
			if res.Status != "refused" {
				t.Fatalf("status = %q, want refused", res.Status)
			}
			if !noteContaining(res.Notes, "release-badge, build-badge, abcd-badge, licence, links, contents") {
				t.Errorf("the refusal does not name the known parts: %q", res.Notes)
			}
			if after, _ := os.ReadFile(configPath(repo)); string(after) != string(before) {
				t.Errorf("a refused flag changed the config")
			}
		})
	}
}

// TestNarrowedReadmePartsRemoveDropped: a part dropped from the chosen set is
// removed when it matches its hash, and named and left when it was edited by
// hand; the parts still chosen stay as they were.
func TestNarrowedReadmePartsRemoveDropped(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
	writeFixture(t, repo, "SECURITY.md", "# Security\n", 0o644)
	setReadmeConfig(t, repo, map[string]any{"parts": []string{"licence", "links", "contents"}})
	writeFixture(t, repo, "README.md", "# Project\n\nprose\n\n## Install\n", 0o644)
	readmeInstall(t, repo, installOpts())
	path := filepath.Join(repo, "README.md")
	licence := readmePart("licence", "Licensed under the MIT License; see [LICENSE](LICENSE).")
	links := readmePart("links", "- [Changelog](CHANGELOG.md)\n- [Security policy](SECURITY.md)")
	if got := readReadme(t, path); !strings.Contains(got, licence) || !strings.Contains(got, links) || !strings.Contains(got, "ABCD:contents") {
		t.Fatalf("precondition: README.md = %q", got)
	}
	// The owner edits the contents part by hand.
	edited := strings.Replace(readReadme(t, path), "- [Install](#install)", "- [Install it](#install)", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}

	o := installOpts()
	o.ReadmeParts = "licence"
	res := readmeInstall(t, repo, o)
	got := readReadme(t, path)
	if strings.Contains(got, "ABCD:links") {
		t.Errorf("the dropped links part was kept:\n%s", got)
	}
	if !strings.Contains(got, licence) {
		t.Errorf("the licence part still chosen was not kept:\n%s", got)
	}
	if !strings.Contains(got, "- [Install it](#install)") {
		t.Errorf("the dropped part edited by hand was not left:\n%s", got)
	}
	if !noteContaining(res.Notes, "contents part edited by hand at line") {
		t.Errorf("the edited part is not named: %q", res.Notes)
	}
	if !noteContaining(res.Notes, "links part removed") {
		t.Errorf("the removal is not said: %q", res.Notes)
	}
}

// TestReadmeOfferKeepsPipedAnswerOrder: a scripted `ahoy install` answers its
// questions positionally, so the README offer must not shift the stream. A
// stream of the answer count a repository with the pair answered is asked
// gets the same questions in the same order, nothing is written into
// README.md, no answer is stored, and optional_skipped names the offer.
func TestReadmeOfferKeepsPipedAnswerOrder(t *testing.T) {
	var control []string
	t.Run("control: the pair is answered", func(t *testing.T) {
		setupHermetic(t)
		repo := installedRepo(t)
		setReadmeConfig(t, repo, map[string]any{"parts": []string{}})
		p := &pipedPrompter{answers: make([]bool, 64)}
		for i := range p.answers {
			p.answers[i] = true
		}
		if _, err := Install(repo, InstallOptions{}, p); err != nil {
			t.Fatal(err)
		}
		control = append(control, p.asked...)
	})
	if len(control) == 0 {
		t.Fatal("the control install asked nothing; the stream has nothing to shift")
	}
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "README.md", "# Project\n", 0o644)
	writeFixture(t, repo, "LICENSE", mitLicence, 0o644)
	p := &pipedPrompter{answers: make([]bool, len(control))}
	for i := range p.answers {
		p.answers[i] = true
	}
	res, err := Install(repo, InstallOptions{}, p)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(p.asked, "\n") != strings.Join(control, "\n") {
		t.Errorf("the piped stream was shifted:\n got %q\nwant %q", p.asked, control)
	}
	for _, q := range p.asked {
		if strings.Contains(strings.ToLower(q), "readme") {
			t.Errorf("a pipe was asked about the README: %q", q)
		}
	}
	if got := readReadme(t, filepath.Join(repo, "README.md")); got != "# Project\n" {
		t.Errorf("README.md changed: %q", got)
	}
	if _, ok := storedReadmeParts(t, repo); ok {
		t.Errorf("a piped stream stored a README answer")
	}
	if !containsString(res.OptionalSkipped, ReadmeOfferGapID) {
		t.Errorf("optional_skipped = %v, want it to name %s", res.OptionalSkipped, ReadmeOfferGapID)
	}
}

// TestReadmeCreatedRecordDiesWithTheFile: readme.created vouches for the one
// README.md abcd made, so a run that finds README.md gone clears it, and a
// later README.md, which is the owner's, is never deleted on its word. The
// reviewer's probe: abcd creates README.md, the owner answers none (the file
// is left blank, abcd's own), deletes it, and later writes a placeholder
// holding a newline; uninstall keeps the placeholder, provided an abcd run
// saw the file gone in between, an install here, an uninstall below.
func TestReadmeCreatedRecordDiesWithTheFile(t *testing.T) {
	links := readmePart("links", "- [Changelog](CHANGELOG.md)")
	createThenNone := func(t *testing.T) (string, string) {
		t.Helper()
		setupHermetic(t)
		repo := installedRepo(t)
		writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
		o := installOpts()
		o.ReadmeParts = "links"
		readmeInstall(t, repo, o)
		path := filepath.Join(repo, "README.md")
		if got := readReadme(t, path); got != links+"\n" || storedReadme(t, repo)["created"] != true {
			t.Fatalf("precondition: README.md = %q, readme = %v", got, storedReadme(t, repo))
		}
		o.ReadmeParts = "none"
		readmeInstall(t, repo, o)
		if got := readReadme(t, path); strings.TrimSpace(got) != "" || storedReadme(t, repo)["created"] != true {
			t.Fatalf("precondition: after none, README.md = %q, readme = %v", got, storedReadme(t, repo))
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		return repo, path
	}

	t.Run("an install that finds README.md gone clears the record", func(t *testing.T) {
		repo, path := createThenNone(t)
		readmeInstall(t, repo, installOpts())
		if _, ok := storedReadme(t, repo)["created"]; ok {
			t.Fatalf("readme.created outlived the file after an install: %v", storedReadme(t, repo))
		}
		writeFixture(t, repo, "README.md", "\n", 0o644)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("the owner's placeholder README.md was deleted: %v (receipt %+v)", err, receipt.Readme)
		}
		if string(got) != "\n" || receipt.Readme.Deleted {
			t.Errorf("README.md = %q, receipt %+v; want the placeholder kept", got, receipt.Readme)
		}
	})

	t.Run("an uninstall that finds README.md gone clears the record", func(t *testing.T) {
		repo, _ := createThenNone(t)
		receipt, err := Uninstall(repo, "")
		if err != nil {
			t.Fatal(err)
		}
		if receipt.Readme.Note != "no README.md" {
			t.Errorf("receipt note = %q", receipt.Readme.Note)
		}
		if _, ok := storedReadme(t, repo)["created"]; ok {
			t.Errorf("readme.created outlived the file after an uninstall: %v", storedReadme(t, repo))
		}
	})
}

// TestUninstallSaysWhenTheCreatedRecordStands: uninstall deletes a README.md
// abcd made and emptied, then clears readme.created; a record it could not
// clear is said in the receipt's note, never dropped.
func TestUninstallSaysWhenTheCreatedRecordStands(t *testing.T) {
	setupHermetic(t)
	repo := installedRepo(t)
	writeFixture(t, repo, "CHANGELOG.md", "# Changelog\n", 0o644)
	o := installOpts()
	o.ReadmeParts = "links"
	readmeInstall(t, repo, o)
	dir := filepath.Dir(configPath(repo))
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	receipt, err := Uninstall(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Readme.Deleted {
		t.Fatalf("receipt = %+v, want README.md deleted", receipt.Readme)
	}
	if want := "README.md, which abcd created, deleted; the record that abcd created it was not cleared"; !strings.Contains(receipt.Readme.Note, want) {
		t.Errorf("receipt note = %q, want it to say %q", receipt.Readme.Note, want)
	}
}
