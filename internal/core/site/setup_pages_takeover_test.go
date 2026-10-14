package site

// Publishing over a Pages site that is already live, and adding the hosting
// block to a composition that lacks one (itd-2610091010348892). The forge is
// the in-process fake, the Pages API githubpagestest's. Nothing here reaches a
// network.

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages"
	"github.com/intentdriven/abcd/internal/adapter/hosting/githubpages/githubpagestest"
)

const existingPages = "https://example-owner.github.io/example-site/ (build type workflow)"

// A run answered in advance (--yes) never answers the takeover: with a live
// site in the repository it refuses before anything is asked or written, and
// names the site, unless --replace-existing was passed.
func TestAnUnattendedRunNeverPublishesOverALiveSite(t *testing.T) {
	t.Run("refused without --replace-existing", func(t *testing.T) {
		h := newPagesHarness(t)
		h.pages.Repos["example-owner/example-site"].Pages = &githubpagestest.Pages{BuildType: "workflow"}
		req := pagesRequest(t, h)
		req.Unattended = true
		_, err := Setup(req)
		refusedBeforeAnything(t, h, err, existingPages)
		if !strings.Contains(err.Error(), "--replace-existing") {
			t.Fatalf("the refusal does not name the flag that lets it through: %v", err)
		}
	})
	t.Run("refused when the composition is already switched on", func(t *testing.T) {
		h := newPagesHarness(t)
		h.run(t)
		h.repo.Git("add", "-A")
		h.repo.Git("commit", "-q", "-m", "switch the project site on")
		if err := os.Remove(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil {
			t.Fatal(err)
		}
		h.ask.questions = nil
		writes, forgeWrites := len(pagesWrites(h)), len(h.forge.writes)
		req := pagesRequest(t, h)
		req.Unattended = true
		_, err := Setup(req)
		if err == nil || !strings.Contains(err.Error(), existingPages) || !strings.Contains(err.Error(), "--replace-existing") {
			t.Fatalf("setup = %v, want a refusal naming %q and the flag", err, existingPages)
		}
		if len(pagesWrites(h)) != writes || len(h.forge.writes) != forgeWrites || len(h.ask.questions) != 0 {
			t.Fatalf("the refused run wrote %v %v or asked %q", pagesWrites(h)[writes:], h.forge.writes[forgeWrites:], h.ask.questions)
		}
		if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); !os.IsNotExist(err) {
			t.Fatal("the refused run wrote the workflow")
		}
	})
	t.Run("published over with --replace-existing", func(t *testing.T) {
		h := newPagesHarness(t)
		h.pages.Repos["example-owner/example-site"].Pages = &githubpagestest.Pages{BuildType: "workflow"}
		res := h.run(t, func(r *SetupRequest) { r.Unattended, r.ReplaceExisting = true, true })
		if res.Status == StatusRefused || res.Status == StatusDeclined {
			t.Fatalf("status = %q, notes %v", res.Status, res.Notes)
		}
		if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil {
			t.Fatal("the workflow was not written")
		}
		if res.Host.Status != HostCurrent || !strings.Contains(res.Host.Detail, existingPages) {
			t.Fatalf("host = %+v, want the outcome naming %q", res.Host, existingPages)
		}
	})
	// A person at a terminal who passes the flag has answered the takeover
	// already, so only the switch is asked.
	t.Run("--replace-existing answers the takeover", func(t *testing.T) {
		h := newPagesHarness(t)
		h.run(t)
		h.repo.Git("add", "-A")
		h.repo.Git("commit", "-q", "-m", "switch the project site on")
		if err := os.Remove(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil {
			t.Fatal(err)
		}
		h.ask.questions = nil
		res := h.run(t, func(r *SetupRequest) { r.ReplaceExisting = true })
		for _, q := range h.ask.questions {
			if strings.HasPrefix(q, "Publish over") {
				t.Fatalf("the takeover was asked although --replace-existing answered it: %q", q)
			}
		}
		if res.Status == StatusRefused || res.Status == StatusDeclined {
			t.Fatalf("status = %q, notes %v", res.Status, res.Notes)
		}
	})
}

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

// hostingBlockRe finds the block the refusal tells the person to add.
var hostingBlockRe = regexp.MustCompile("`\"hosting\": (\\{[^`]*\\})`")

// A composition with no hosting block is not rewritten; setup names the block
// to add. A block added that way, switch on, is still put to the person as the
// switch question before the workflow is written: the switch is asked at setup
// whoever wrote it.
func TestAHostingBlockAddedByHandIsStillAskedTheSwitch(t *testing.T) {
	h := newPagesHarness(t)
	h.run(t)
	// Back to a repository with a composition but no hosting block, no
	// workflow and no Pages site.
	var m map[string]any
	if err := json.Unmarshal(mustRead(t, h.repo.Root(), ManifestRelPath), &m); err != nil {
		t.Fatal(err)
	}
	delete(m, "hosting")
	raw, _ := json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(h.repo.Root(), ManifestRelPath), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil {
		t.Fatal(err)
	}
	h.pages.Repos["example-owner/example-site"].Pages = nil
	h.repo.Git("add", "-A")
	h.repo.Git("commit", "-q", "-m", "a composition with no hosting block")

	_, err := Setup(pagesRequest(t, h))
	if err == nil {
		t.Fatal("setup accepted --provider on a composition with no hosting block")
	}
	got := hostingBlockRe.FindStringSubmatch(err.Error())
	if got == nil || !strings.Contains(err.Error(), "asks whether to switch the project site on") {
		t.Fatalf("the refusal names no block to add, or does not say the switch is still asked: %v", err)
	}
	var blk map[string]any
	if jerr := json.Unmarshal([]byte(got[1]), &blk); jerr != nil {
		t.Fatalf("the block is not JSON: %v", jerr)
	}
	m["hosting"] = blk
	raw, _ = json.MarshalIndent(m, "", "  ")
	if err := os.WriteFile(filepath.Join(h.repo.Root(), ManifestRelPath), append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	h.repo.Git("add", "-A")
	h.repo.Git("commit", "-q", "-m", "the hosting block, as setup named it")

	h.ask.questions, h.ask.yes = nil, false
	res := h.harness.run(t, func(r *SetupRequest) {
		r.Adapter = githubpages.Adapter{API: h.pages.API}
		r.Credentials = noCredential{t}
		r.Asker = h.ask
	})
	if len(h.ask.questions) != 1 || !strings.HasPrefix(h.ask.questions[0], "Switch on the project site?") {
		t.Fatalf("questions = %q, want the switch alone", h.ask.questions)
	}
	if res.Status != StatusDeclined {
		t.Fatalf("status = %q, want declined", res.Status)
	}
	if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); !os.IsNotExist(err) {
		t.Fatal("a declined switch wrote the workflow")
	}
	if len(changedPaths(t, h.harness)) != 0 {
		t.Fatalf("a declined switch changed %v", changedPaths(t, h.harness))
	}

	h.ask.questions, h.ask.yes = nil, true
	res = h.harness.run(t, func(r *SetupRequest) {
		r.Adapter = githubpages.Adapter{API: h.pages.API}
		r.Credentials = noCredential{t}
		r.Asker = h.ask
	})
	if _, err := os.Stat(filepath.Join(h.repo.Root(), SiteWorkflowRelPath)); err != nil || res.Status != StatusChanged {
		t.Fatalf("a yes did not write the workflow (status %q): %v", res.Status, err)
	}
	// The composition the person wrote is left as they wrote it.
	for _, p := range changedPaths(t, h.harness) {
		if p == ManifestRelPath {
			t.Fatal("setup rewrote the composition the person wrote")
		}
	}
}
