package layered

import (
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/abcdhome"
)

// TestInterviewListSetting holds interview.list: arrows when nothing sets it,
// numbered from the machine's file, and every fault refused loudly naming the
// file (spc-2610030911534855, "The numbered fallback").
func TestInterviewListSetting(t *testing.T) {
	t.Run("bundled default is arrows", func(t *testing.T) {
		f := newFixture(t)
		v, err := InterviewList(f.roots)
		if err != nil {
			t.Fatal(err)
		}
		if v.V != InterviewListArrows || v.Layer != Bundled {
			t.Errorf("got %q from %s, want %q from bundled", v.V, v.Layer, InterviewListArrows)
		}
	})
	t.Run("the machine's file selects numbered", func(t *testing.T) {
		f := newFixture(t)
		f.machineFile(Config, `{"interview": {"list": "numbered"}}`)
		v, err := InterviewList(f.roots)
		if err != nil {
			t.Fatal(err)
		}
		if v.V != InterviewListNumbered || v.Layer != Machine || v.Origin != abcdhome.Display("config.json") {
			t.Errorf("got %+v, want numbered from ~/.abcd.noindex/config.json", v)
		}
	})
	for _, tc := range []struct {
		name, repo, machine string
		want                []string
	}{
		{
			name: "a repository may not set it",
			repo: `{"interview": {"list": "numbered"}}`,
			want: []string{".abcd/config.json", abcdhome.Display("config.json"), "interview.list"},
		},
		{
			name:    "a value outside the two",
			machine: `{"interview": {"list": "tabs"}}`,
			want:    []string{abcdhome.Display("config.json"), "interview.list", "arrows", "numbered"},
		},
		{
			name:    "an unknown key under interview",
			machine: `{"interview": {"lst": "numbered"}}`,
			want:    []string{abcdhome.Display("config.json"), "interview.lst", "unknown key"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			if tc.repo != "" {
				f.repoFile(Config, tc.repo)
			}
			if tc.machine != "" {
				f.machineFile(Config, tc.machine)
			}
			_, err := InterviewList(f.roots)
			if err == nil {
				t.Fatal("want a refusal")
			}
			for _, w := range tc.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("refusal %q does not name %q", err, w)
				}
			}
		})
	}
}
