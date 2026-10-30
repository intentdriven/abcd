package claim

import (
	"errors"
	"slices"
	"strings"
	"testing"
)

// TestNoPushedTextNamesAPairedLogin: with a declaration pairing maya's id
// with maya's login in octo's home, nothing octo's acts push or post names
// that login: no commit message or trailer, no ref name and no comment body,
// across a reservation, its renewal and release, a take-over of maya's
// reservation, a release's reservation with its withdrawals, an admission
// and a release with its re-arms, a settle of a tagged release, and an
// announcement.
func TestNoPushedTextNamesAPairedLogin(t *testing.T) {
	w := newWorld(t)
	f := commentedForge(t)
	maya := w.clone("5678+maya@users.noreply.github.com", "")
	keepHolderID(t, maya.home, mayaB)
	octo := w.clone("1234+octo@users.noreply.github.com", "")
	if _, err := octo.reserve("iss-1", "a.go"); err != nil {
		t.Fatal(err)
	}
	declareHolderIDs(t, octo.home, mayaB+" maya\n")
	if _, err := maya.reserve("iss-2", "b.go"); err != nil {
		t.Fatal(err)
	}
	steps := []struct {
		name string
		do   func() error
	}{
		{"renew", func() error { _, err := octo.do(Request{Record: "iss-1", Renew: true}); return err }},
		{"release", func() error { _, err := octo.do(Request{Record: "iss-1", Release: true}); return err }},
		{"take over", func() error {
			_, err := octo.do(Request{Record: "iss-2", Footprint: []string{"b.go"}, TakeOver: true})
			return err
		}},
		{"release reservation", func() error { _, err := octo.do(Request{Record: "rel-v0.16.0"}); return err }},
		{"admit", func() error { _, err := octo.do(Request{Record: "rel-v0.16.0", Admit: 12}); return err }},
		{"release of the cut", func() error { _, err := octo.do(Request{Record: "rel-v0.16.0", Release: true}); return err }},
		{"second cut", func() error { _, err := octo.do(Request{Record: "rel-v0.17.0"}); return err }},
		{"settle of the tagged cut", func() error {
			w.tagOrigin("v0.17.0")
			res, err := octo.reserve("iss-3", "c.go")
			if err == nil && !slices.ContainsFunc(res.Settles, func(s Settle) bool { return s.Subject == "rel-v0.17.0" && s.Commit != "" }) {
				return errors.New("the tagged cut was not settled")
			}
			return err
		}},
		{"announce", func() error { _, err := octo.announce("merging", "#1012"); return err }},
	}
	for _, s := range steps {
		if err := s.do(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
	}
	if len(f.posted()) == 0 {
		t.Fatal("no comment was posted to check")
	}
	texts := map[string]string{
		"commit messages": runGit(t, w.bare, "log", "--all", "--format=%B"),
		"ref names":       runGit(t, w.bare, "for-each-ref", "--format=%(refname)"),
		"comments":        strings.Join(f.posted(), "\n"),
	}
	for what, text := range texts {
		if strings.Contains(strings.ToLower(text), "maya") {
			t.Errorf("the %s name the paired login:\n%s", what, text)
		}
	}
}
