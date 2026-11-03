package layered

import (
	"fmt"
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

// TestInterviewRedraftsSetting holds interview.redrafts (iss-2611011402496418):
// three when nothing sets it, the repository's figure from its own file, and
// every fault refused loudly naming the file; the list's reader claims the key
// too, so a repository that sets it is never refused its list.
func TestInterviewRedraftsSetting(t *testing.T) {
	t.Run("bundled default is three", func(t *testing.T) {
		f := newFixture(t)
		v, err := InterviewRedrafts(f.roots)
		if err != nil {
			t.Fatal(err)
		}
		if v.V != 3 || v.Layer != Bundled {
			t.Errorf("got %d from %s, want 3 from bundled", v.V, v.Layer)
		}
	})
	t.Run("the repository's file sets it", func(t *testing.T) {
		for _, n := range []int{0, 5, InterviewRedraftsMax} {
			f := newFixture(t)
			f.repoFile(Config, fmt.Sprintf(`{"interview": {"redrafts": %d}}`, n))
			v, err := InterviewRedrafts(f.roots)
			if err != nil {
				t.Fatal(err)
			}
			if v.V != n || v.Layer != Repo || v.Origin != ".abcd/config.json" {
				t.Errorf("got %+v, want %d from .abcd/config.json", v, n)
			}
		}
	})
	t.Run("the list's reader takes a repository that sets it", func(t *testing.T) {
		f := newFixture(t)
		f.repoFile(Config, `{"interview": {"redrafts": 2}}`)
		f.machineFile(Config, `{"interview": {"list": "numbered"}}`)
		v, err := InterviewList(f.roots)
		if err != nil || v.V != InterviewListNumbered {
			t.Fatalf("got %+v, %v; want numbered and no refusal", v, err)
		}
	})
	for _, tc := range []struct {
		name, repo, machine string
		want                []string
	}{
		{
			name:    "the machine may not set it",
			machine: `{"interview": {"redrafts": 2}}`,
			want:    []string{abcdhome.Display("config.json"), ".abcd/config.json", "interview.redrafts"},
		},
		{
			name: "above the ceiling",
			repo: `{"interview": {"redrafts": 11}}`,
			want: []string{".abcd/config.json", "interview.redrafts", "11", "0 to 10"},
		},
		{
			name: "below the floor",
			repo: `{"interview": {"redrafts": -1}}`,
			want: []string{".abcd/config.json", "interview.redrafts", "-1", "0 to 10"},
		},
		{
			name: "not a whole number",
			repo: `{"interview": {"redrafts": "three"}}`,
			want: []string{".abcd/config.json", "interview.redrafts", "three"},
		},
		{
			name: "an unknown key under interview",
			repo: `{"interview": {"redraft": 2}}`,
			want: []string{".abcd/config.json", "interview.redraft", "unknown key", "list, redrafts"},
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
			_, err := InterviewRedrafts(f.roots)
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
	t.Run("the list's reader refuses it in the machine's file", func(t *testing.T) {
		f := newFixture(t)
		f.machineFile(Config, `{"interview": {"list": "numbered", "redrafts": 2}}`)
		if _, err := InterviewList(f.roots); err == nil || !strings.Contains(err.Error(), "interview.redrafts") {
			t.Fatalf("err = %v, want the machine's interview.redrafts refused", err)
		}
	})
}
