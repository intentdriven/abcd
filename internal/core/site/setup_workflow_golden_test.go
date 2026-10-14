package site

import (
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
)

// TestTheCloudflareWorkflowIsUnchanged pins the workflow setup writes for the
// cloudflare provider byte for byte, so a managed repository that set its
// site up before the github-pages provider existed finds its workflow current
// and is not asked to replace it.
func TestTheCloudflareWorkflowIsUnchanged(t *testing.T) {
	wf, err := renderSiteWorkflow("main", cloudflare.Adapter{}, hosting.Site{Name: "x"})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "site-workflow-cloudflare.yml", wf)
}
