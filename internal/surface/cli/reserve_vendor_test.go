package cli

import (
	"errors"
	"strings"
	"testing"
)

// TestReserveUnderVendorIdentityShowsItNowhere is the product thinker's
// ruling of 2026-10-15 at the front door: a hosted container's clone comes
// with an AI vendor's git identity while its forge token is the person's.
// abcd works out the person from the token, and the vendor identity appears
// in no output, no commit, no trailer and no ref name. This is the one file
// the vendor identity may be written in: it proves it appears nowhere else.
func TestReserveUnderVendorIdentityShowsItNowhere(t *testing.T) {
	const vendorName, vendorMail = "Claude", "noreply@anthropic.com"
	w := newReserveWorld(t)
	p := w.clone(t, "p", vendorMail)
	gitCmd(t, p, "config", "user.name", vendorName)
	calls := fakeForge(t, "octo-person", 4242, nil)

	var shown strings.Builder
	for _, args := range [][]string{
		{"reserve", "--holder"},
		{"reserve", "--holder", "--json"},
		{"reserve", "iss-5", "--footprint", "a.go"},
		{"reserve", "iss-6", "--footprint", "b.go", "--json"},
		{"reserve", "iss-6", "--release"},
		{"reserve", "iss-7", "--footprint", "../escape"}, // a refusal
	} {
		code, stdout, stderr := reserveRun(t, p, args...)
		want := 0
		if args[1] == "iss-7" {
			want = 2
		}
		if code != want {
			t.Fatalf("%v: exit %d; want %d\n%s%s", args, code, want, stdout, stderr)
		}
		shown.WriteString(stdout + stderr)
	}
	if calls.n == 0 {
		t.Fatal("the forge was never asked; the holder cannot have come from the token")
	}
	// The token cannot answer, and its error names the vendor identity: the
	// refusals still show it nowhere.
	q := w.clone(t, "q", vendorMail)
	gitCmd(t, q, "config", "user.name", vendorName)
	fakeForge(t, "", 0, errors.New("gh: HTTP 401 for "+vendorName+" <"+vendorMail+">"))
	for _, args := range [][]string{{"reserve", "--holder"}, {"reserve", "iss-8", "--footprint", "c.go"}} {
		code, stdout, stderr := reserveRun(t, q, args...)
		if code != 2 || !strings.Contains(stderr, "could not work out whose GitHub account this session uses") {
			t.Fatalf("%v with a failing token: exit %d; want the token refusal\n%s%s", args, code, stdout, stderr)
		}
		shown.WriteString(stdout + stderr)
	}
	tip := w.remoteTip(t, "iss-5")
	if tip == "" {
		t.Fatal("the reservation did not reach origin")
	}
	const person = "octo-person <4242+octo-person@users.noreply.github.com>"
	for _, id := range []string{"iss-5", "iss-6"} {
		who := gitCmd(t, w.origin, "log", "-1", "--format=%an <%ae>%n%cn <%ce>", "refs/heads/abcd-reserved/"+id)
		if who != person+"\n"+person {
			t.Fatalf("%s's reservation is authored and committed by\n%s\nwant %s in both roles", id, who, person)
		}
		shown.WriteString(gitCmd(t, w.origin, "log", "--format=%an%n%ae%n%cn%n%ce%n%B", "refs/heads/abcd-reserved/"+id))
	}
	for _, dir := range []string{p, q, w.origin} {
		shown.WriteString(gitCmd(t, dir, "for-each-ref", "--format=%(refname)"))
	}
	all := strings.ToLower(shown.String())
	for _, s := range []string{vendorName, vendorMail, "anthropic"} {
		if strings.Contains(all, strings.ToLower(s)) {
			t.Fatalf("the vendor identity %q is shown:\n%s", s, shown.String())
		}
	}
	if !strings.Contains(shown.String(), "Abcd-Holder: octo-person") {
		t.Fatalf("the reservation does not name the token's account as holder:\n%s", shown.String())
	}
}
