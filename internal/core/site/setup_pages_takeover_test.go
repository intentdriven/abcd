package site

// Publishing over a Pages site that is already live, and adding the hosting
// block to a composition that lacks one (itd-2610091010348892). The forge is
// the in-process fake, the Pages API githubpagestest's. Nothing here reaches a
// network.

import (
	"context"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages/githubpagestest"
)

// GitHub reports a custom domain over http until its certificate is issued;
// the host stage reports the site live at its https address and says HTTPS is
// not enforced yet.
func TestTheHostStageSaysWhenHTTPSIsNotEnforcedYet(t *testing.T) {
	f := githubpagestest.New()
	f.Repos["acme/widgets"].Pages = &githubpagestest.Pages{BuildType: "workflow", CNAME: "docs.example.com", HTMLURL: "http://docs.example.com/"}
	out, declined, refused := setupHost(context.Background(), githubpages.Adapter{API: f.API},
		hosting.Site{Name: "widgets", Repo: "acme/widgets", Domain: "docs.example.com"}, true, SetupRequest{Asker: answer(true)})
	if refused || declined || out.Status != HostCurrent {
		t.Fatalf("host = %+v (declined %v, refused %v), want current", out, declined, refused)
	}
	if out.Address != "https://docs.example.com/" || !strings.Contains(out.Detail, "HTTPS is not enforced yet") {
		t.Fatalf("host = %+v, want the https address and the note", out)
	}
}
