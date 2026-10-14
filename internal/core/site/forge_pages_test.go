package site

// The github-pages provider's forge transport and the forge's visibility read
// (itd-2610091010348892), over in-process fakes of `gh api`.

import (
	"context"
	"reflect"
	"testing"
)

func TestTheForgeReadsWhetherTheRepositoryIsPrivate(t *testing.T) {
	f := newFakeGHAPI()
	for _, want := range []bool{true, false} {
		f.objects["repos/"+f.repo] = map[string]any{"default_branch": "main", "private": want}
		got, err := f.forge().Private(context.Background())
		if err != nil || got != want {
			t.Fatalf("private = %v, %v; want %v", got, err, want)
		}
	}
	f.objects["repos/"+f.repo] = map[string]any{"default_branch": "main"}
	if _, err := f.forge().Private(context.Background()); err == nil {
		t.Fatal("a repository response that does not say whether it is private was read as public")
	}
}

func TestThePagesProviderSpeaksThroughGhAsThePerson(t *testing.T) {
	type call struct {
		stdin string
		args  []string
	}
	var calls []call
	api := ghPagesAPI(func(_ string, stdin []byte, args ...string) ([]byte, error) {
		calls = append(calls, call{string(stdin), args})
		return []byte("{}"), nil
	})
	if _, err := api(context.Background(), "GET", "repos/acme/widgets/pages", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := api(context.Background(), "POST", "repos/acme/widgets/pages", []byte(`{"build_type":"workflow"}`)); err != nil {
		t.Fatal(err)
	}
	head := []string{"api", "--hostname", "github.com", "-H", "Accept: application/vnd.github+json"}
	want := []call{
		{"", append(append([]string{}, head...), "repos/acme/widgets/pages")},
		{`{"build_type":"workflow"}`, append(append([]string{}, head...), "--method", "POST", "repos/acme/widgets/pages", "--input", "-")},
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %q\nwant  %q", calls, want)
	}
}
