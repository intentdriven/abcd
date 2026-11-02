package layered

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
	"github.com/intentdriven/abcd/internal/core/question"
)

// machineOrigin is how every refusal and source names the person's own file.
var machineOrigin = abcdhome.Display("config.json")

// sourcesOf is the provenance QuestionLimits reports, one origin per key in
// question.Bounds' order.
func sourcesOf(v QuestionLimitsValue) []string {
	out := make([]string, len(v.Sources))
	for i, s := range v.Sources {
		out[i] = s.Origin
	}
	return out
}

// refusal asserts err is a *KeyRefusedError and returns it.
func refusal(t *testing.T, err error) *KeyRefusedError {
	t.Helper()
	var kr *KeyRefusedError
	if !errors.As(err, &kr) {
		t.Fatalf("err = %v (%T), want a *KeyRefusedError", err, err)
	}
	var u *UnreadError
	if errors.As(err, &u) {
		t.Fatalf("a refused value read as an unread layer: %v", err)
	}
	return kr
}

// TestQuestionLimitsDefaultWithNothingSet: no file sets a key, so the limits
// are question.Default, bundled, every key's source bundled.
func TestQuestionLimitsDefaultWithNothingSet(t *testing.T) {
	for name, repo := range map[string]string{
		"no file":            "",
		"no question key":    `{"pace": {"work_minutes": 90}}`,
		"an empty namespace": `{"question": {}}`,
	} {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			if repo != "" {
				f.repoFile(Config, repo)
			}
			v, err := QuestionLimits(f.roots)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(v.V, question.Default) || v.Layer != Bundled || v.Origin != "bundled" {
				t.Errorf("got %+v from %s %q, want question.Default from bundled", v.V, v.Layer, v.Origin)
			}
			if got := sourcesOf(v); !reflect.DeepEqual(got, []string{"bundled", "bundled", "bundled"}) {
				t.Errorf("sources = %v, want every key bundled", got)
			}
			if v.SetByFile() {
				t.Error("SetByFile is true with nothing set")
			}
		})
	}
}

// TestQuestionLimitsReplaceThreeFieldsOnly: a repository's three values
// replace Rows, LabelWords and MeaningSentences and nothing else, and each
// key's source names the repository's file.
func TestQuestionLimitsReplaceThreeFieldsOnly(t *testing.T) {
	f := newFixture(t)
	f.repoFile(Config, `{"question": {"rows": 40, "label_words": 6, "description_sentences": 3}}`)
	v, err := QuestionLimits(f.roots)
	if err != nil {
		t.Fatal(err)
	}
	want := question.Default
	want.Rows, want.LabelWords, want.MeaningSentences = 40, 6, 3
	if !reflect.DeepEqual(v.V, want) {
		t.Errorf("limits = %+v\nwant     %+v", v.V, want)
	}
	if v.Layer != Repo || v.Origin != Config.RepoOrigin() {
		t.Errorf("value from %s %q, want the repo layer's %q", v.Layer, v.Origin, Config.RepoOrigin())
	}
	if got := sourcesOf(v); !reflect.DeepEqual(got, []string{".abcd/config.json", ".abcd/config.json", ".abcd/config.json"}) {
		t.Errorf("sources = %v, want every key the repository's file", got)
	}
	// One key set leaves the other two bundled.
	g := newFixture(t)
	g.repoFile(Config, `{"question": {"label_words": 7}}`)
	v, err = QuestionLimits(g.roots)
	if err != nil {
		t.Fatal(err)
	}
	if v.V.Rows != question.Default.Rows || v.V.LabelWords != 7 || v.V.MeaningSentences != question.Default.MeaningSentences {
		t.Errorf("limits = %v, want only label words replaced", v.V.Configurable())
	}
	if got := sourcesOf(v); !reflect.DeepEqual(got, []string{"bundled", ".abcd/config.json", "bundled"}) {
		t.Errorf("sources = %v", got)
	}
}

// TestQuestionLimitsRefuseAnUnknownKey: the resolver claims the namespace, so
// a key it does not take is refused naming the key and the file, and no value
// comes back.
func TestQuestionLimitsRefuseAnUnknownKey(t *testing.T) {
	for _, key := range []string{"options", "columns", "tabs", "row"} {
		for _, layer := range []string{"repo", "machine"} {
			t.Run(key+"/"+layer, func(t *testing.T) {
				f := newFixture(t)
				body := fmt.Sprintf(`{"question": {"rows": 40, %q: 3}}`, key)
				file := Config.RepoOrigin()
				if layer == "repo" {
					f.repoFile(Config, body)
				} else {
					f.machineFile(Config, strings.Replace(body, "40", "30", 1))
					file = machineOrigin
				}
				v, err := QuestionLimits(f.roots)
				if err == nil {
					t.Fatalf("admitted question.%s: %+v", key, v)
				}
				var u *UnreadError
				if errors.As(err, &u) {
					t.Fatalf("an unknown key read as an unread layer: %v", err)
				}
				for _, w := range []string{"question." + key, file, "unknown key"} {
					if !strings.Contains(err.Error(), w) {
						t.Errorf("refusal %q does not name %q", err, w)
					}
				}
				if !reflect.DeepEqual(v, QuestionLimitsValue{}) {
					t.Errorf("a refusal returned a value: %+v", v)
				}
			})
		}
	}
}

// TestQuestionLimitsLetThePersonTighten: a personal value at or below the
// limit otherwise in force is the limit, reported as the machine file's.
func TestQuestionLimitsLetThePersonTighten(t *testing.T) {
	for _, tc := range []struct {
		name, repo, machine string
		rows                int
	}{
		{"below the repository's", `{"question": {"rows": 40}}`, `{"question": {"rows": 24}}`, 24},
		{"below the default", "", `{"question": {"rows": 24}}`, 24},
		{"equal to the repository's", `{"question": {"rows": 40}}`, `{"question": {"rows": 40}}`, 40},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.repo != "" {
				f.repoFile(Config, tc.repo)
			}
			f.machineFile(Config, tc.machine)
			v, err := QuestionLimits(f.roots)
			if err != nil {
				t.Fatal(err)
			}
			if v.V.Rows != tc.rows {
				t.Errorf("rows = %d, want %d", v.V.Rows, tc.rows)
			}
			if v.Sources[0].Layer != Machine || v.Sources[0].Origin != machineOrigin {
				t.Errorf("rows source = %+v, want the machine file", v.Sources[0])
			}
			if v.Sources[1].Origin != "bundled" || v.Sources[2].Origin != "bundled" {
				t.Errorf("sources = %v, want the other two bundled", sourcesOf(v))
			}
		})
	}
	// The value's own layer is the loosest that set anything: the repository's
	// where it set a key a person did not tighten.
	f := newFixture(t)
	f.repoFile(Config, `{"question": {"rows": 40, "label_words": 6}}`)
	f.machineFile(Config, `{"question": {"rows": 30}}`)
	v, err := QuestionLimits(f.roots)
	if err != nil {
		t.Fatal(err)
	}
	if got := v.V.Configurable(); got != [3]int{30, 6, question.Default.MeaningSentences} {
		t.Errorf("limits = %v, want rows 30, label words 6", got)
	}
	if v.Layer != Repo {
		t.Errorf("value layer = %s, want repo", v.Layer)
	}
	if got := sourcesOf(v); !reflect.DeepEqual(got, []string{machineOrigin, ".abcd/config.json", "bundled"}) {
		t.Errorf("sources = %v", got)
	}
}

// TestQuestionLimitsRefuseAPersonWhoLoosens: a personal value above the limit
// in force is refused naming the key, the value, the limit and the file that
// set each, and neither it nor the repository's value is in force.
func TestQuestionLimitsRefuseAPersonWhoLoosens(t *testing.T) {
	for _, tc := range []struct {
		name, repo  string
		limit       int
		limitOrigin string
	}{
		{"over the repository's", `{"question": {"rows": 32}}`, 32, ".abcd/config.json"},
		{"over the default", "", question.Default.Rows, "bundled"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.repo != "" {
				f.repoFile(Config, tc.repo)
			}
			f.machineFile(Config, `{"question": {"rows": 40}}`)
			v, err := QuestionLimits(f.roots)
			kr := refusal(t, err)
			if !kr.Loosens() || kr.Key != question.RowsKey || kr.Value != "40" || kr.Limit != tc.limit ||
				kr.LimitOrigin != tc.limitOrigin || kr.Origin != machineOrigin || kr.Layer != Machine {
				t.Errorf("refusal = %+v", kr)
			}
			want := fmt.Sprintf("%s (machine layer): question.rows is 40: looser than %d (%s); a personal value may only tighten",
				machineOrigin, tc.limit, tc.limitOrigin)
			if err.Error() != want {
				t.Errorf("refusal reads\n%q\nwant\n%q", err, want)
			}
			if !reflect.DeepEqual(v, QuestionLimitsValue{}) {
				t.Errorf("a refusal returned a value: %+v", v)
			}
		})
	}
}

// TestQuestionLimitsHoldTheBounds: each key at its floor and its ceiling is
// admitted, in either file; one below the floor, one above the ceiling and a
// non-integer are refused naming the key, the value and the bounds. A
// personal value below the floor is refused on the bounds, never the limit.
func TestQuestionLimitsHoldTheBounds(t *testing.T) {
	for _, b := range question.Bounds {
		short := strings.TrimPrefix(b.Key, "question.")
		for _, layer := range []Layer{Repo, Machine} {
			for _, tc := range []struct {
				raw        string
				ok         bool
				notInteger bool
			}{
				{fmt.Sprint(b.Floor), true, false},
				{fmt.Sprint(b.Ceiling), true, false},
				{fmt.Sprint(b.Floor - 1), false, false},
				{fmt.Sprint(b.Ceiling + 1), false, false},
				{fmt.Sprintf("%d.5", b.Floor), false, true},
				{`"` + fmt.Sprint(b.Floor) + `"`, false, true},
				{"null", false, true},
			} {
				t.Run(fmt.Sprintf("%s/%s/%s", short, layer, tc.raw), func(t *testing.T) {
					f := newFixture(t)
					body := fmt.Sprintf(`{"question": {%q: %s}}`, short, tc.raw)
					origin := Config.RepoOrigin()
					if layer == Repo {
						f.repoFile(Config, body)
					} else {
						// The repository sets the ceiling, so a personal value
						// at the ceiling tightens nothing and is admitted.
						f.repoFile(Config, fmt.Sprintf(`{"question": {%q: %d}}`, short, b.Ceiling))
						f.machineFile(Config, body)
						origin = machineOrigin
					}
					v, err := QuestionLimits(f.roots)
					if tc.ok {
						if err != nil {
							t.Fatalf("refused %s: %v", tc.raw, err)
						}
						idx := map[string]int{question.RowsKey: 0, question.LabelWordsKey: 1, question.DescriptionSentencesKey: 2}[b.Key]
						if got := fmt.Sprint(v.V.Configurable()[idx]); got != tc.raw {
							t.Errorf("%s in force = %s, want %s", b.Key, got, tc.raw)
						}
						return
					}
					kr := refusal(t, err)
					if kr.Key != b.Key || kr.Value != tc.raw || kr.Bounds != b || kr.Origin != origin || kr.NotInteger != tc.notInteger || kr.Loosens() {
						t.Errorf("refusal = %+v", kr)
					}
					for _, w := range []string{b.Key, tc.raw, b.String(), origin} {
						if !strings.Contains(err.Error(), w) {
							t.Errorf("refusal %q does not name %q", err, w)
						}
					}
				})
			}
		}
	}
	// The repository's out-of-bounds form, word for word.
	f := newFixture(t)
	f.repoFile(Config, `{"question": {"rows": 100}}`)
	_, err := QuestionLimits(f.roots)
	if want := ".abcd/config.json (repo layer): question.rows is 100: outside 24 to 60"; err == nil || err.Error() != want {
		t.Errorf("refusal = %v, want %q", err, want)
	}
	// Below the floor under a repository's higher value: the bounds, not the
	// limit, are named.
	g := newFixture(t)
	g.repoFile(Config, `{"question": {"rows": 40}}`)
	g.machineFile(Config, `{"question": {"rows": 20}}`)
	_, err = QuestionLimits(g.roots)
	if kr := refusal(t, err); kr.Loosens() || !strings.Contains(err.Error(), "outside 24 to 60") {
		t.Errorf("a personal value below the floor was refused as %q, want the bounds named", err)
	}
}

// TestQuestionLimitsReadUnreadableLayersAsUnread: a repository file that is
// not a regular file, and a machine file writable by others, are
// *UnreadError, returned with the defaults, so a caller fails open on them
// and never mistakes them for a refused value.
func TestQuestionLimitsReadUnreadableLayersAsUnread(t *testing.T) {
	assertUnread(t, map[string]func(t *testing.T, f *fixture) string{
		"a repository file that is a directory": func(t *testing.T, f *fixture) string {
			if err := os.MkdirAll(filepath.Join(f.roots.Repo, filepath.FromSlash(Config.RepoRel)), 0o700); err != nil {
				t.Fatal(err)
			}
			return Config.RepoOrigin()
		},
		"a machine file writable by others": func(t *testing.T, f *fixture) string {
			p := f.machineFile(Config, `{"pace": {"work_minutes": 5}}`)
			if err := os.Chmod(p, 0o666); err != nil {
				t.Fatal(err)
			}
			return machineOrigin
		},
	})
}

// assertUnread lays out each case and asserts QuestionLimits reads it as an
// *UnreadError naming the origin the case returns, with the defaults.
func assertUnread(t *testing.T, cases map[string]func(t *testing.T, f *fixture) string) {
	t.Helper()
	for name, lay := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t)
			origin := lay(t, f)
			v, err := QuestionLimits(f.roots)
			var u *UnreadError
			if !errors.As(err, &u) {
				t.Fatalf("err = %v (%T), want an *UnreadError", err, err)
			}
			if u.Origin != origin || !strings.Contains(err.Error(), origin) {
				t.Errorf("unread origin %q, error %q; want %q named", u.Origin, err, origin)
			}
			var kr *KeyRefusedError
			if errors.As(err, &kr) {
				t.Error("an unread layer read as a refused value")
			}
			if !reflect.DeepEqual(v.V, question.Default) {
				t.Errorf("an unread layer returned %+v, want the defaults", v.V)
			}
		})
	}
}

// TestUnreadErrorKeepsLoadsMessage: wrapping a layer's fault in *UnreadError
// changes no caller's message.
func TestUnreadErrorKeepsLoadsMessage(t *testing.T) {
	f := newFixture(t)
	f.repoFile(Config, `{"pace": `)
	_, err := Load(Config, f.roots)
	var u *UnreadError
	if !errors.As(err, &u) || !strings.HasPrefix(err.Error(), ".abcd/config.json (repo layer): it is not valid JSON") {
		t.Errorf("err = %v (%T), want the repo layer's parse fault as an *UnreadError", err, err)
	}
	_, err = Load(Config, Roots{Repo: f.roots.Repo})
	if !errors.As(err, &u) || !strings.Contains(err.Error(), "the home directory is unresolved") {
		t.Errorf("err = %v (%T), want the unresolved home as an *UnreadError", err, err)
	}
}
