package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The bootstrap holds the download to the release its own Go build metadata
// claims, the main-module line Go
// stamps into every build, read from the bytes and never by running them.

// claimingBody is a fixture binary whose bytes carry the main-module line a
// Go build of abcd at tag carries.
func claimingBody(tags ...string) []byte {
	b := "#!/bin/sh\n"
	for _, tag := range tags {
		b += "# mod\tgithub.com/intentdriven/abcd\t" + tag + "\t\n"
	}
	return []byte(b + "exit 0\n")
}

func TestBootstrapRefusesAnotherReleasesBuild(t *testing.T) {
	root := bootstrapRoot(t)
	body := claimingBody("v0.13.4")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 {
		t.Errorf("another release's build must fail loudly, got exit 0 (output %q)", out)
	}
	if !strings.Contains(out, "names release v0.13.4, not "+bootstrapTag) {
		t.Errorf("the refusal must name both releases; output %q", out)
	}
	assertNothingInstalled(t, root, out)
}

func TestBootstrapRefusesABuildClaimingSeveralReleases(t *testing.T) {
	root := bootstrapRoot(t)
	body := claimingBody(bootstrapTag, "v0.13.4")
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 {
		t.Errorf("bytes claiming several releases must fail loudly, got exit 0 (output %q)", out)
	}
	assertNothingInstalled(t, root, out)
}

func TestBootstrapInstallsABuildClaimingItsRelease(t *testing.T) {
	root := bootstrapRoot(t)
	body := claimingBody(bootstrapTag)
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrap(t, root, fx, "")
	if code != 0 {
		t.Fatalf("a build claiming the release it is published as installs: exit %d (output %q)", code, out)
	}
	if got, err := os.ReadFile(filepath.Join(root, "abcd")); err != nil || string(got) != string(body) {
		t.Errorf("the installed binary must be the verified download; got %q (%v)", got, err)
	}
}

// TestBootstrapReadsARealBuildsClaim holds the shell reader to a real Go build,
// not only to fixture bytes written in the shape it expects: the release
// build freshInstallTag made, published as bootstrapTag, is refused by name.
func TestBootstrapReadsARealBuildsClaim(t *testing.T) {
	body := buildRealAbcd(t)
	root := bootstrapRoot(t)
	fx := bootstrapServer(t, body, bootstrapManifest(body))

	out, code := runBootstrap(t, root, fx, "")
	if code == 0 || !strings.Contains(out, "names release "+freshInstallTag+", not "+bootstrapTag) {
		t.Errorf("a real build of %s published as %s must be refused, naming both; exit %d, output %q", freshInstallTag, bootstrapTag, code, out)
	}
	assertNothingInstalled(t, root, out)
}
