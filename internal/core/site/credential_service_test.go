package site

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare"
	"github.com/intentdriven/abcd/internal/adapter/hosting/cloudflare/cloudflaretest"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// The site setup reads its hosting credential through the store by name
// (itd-2609221017023290 criterion 4), and its walkthrough verifies with the
// hosting adapter's own call (criterion 2).

// TestSetupReadsTheHostingCredentialThroughTheStore: a credential kept in the
// external home (an environment variable) reaches the host through
// credential.Store, and the run's record names the credential and never the
// value (criterion 5).
func TestSetupReadsTheHostingCredentialThroughTheStore(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ABCD_TEST_CLOUDFLARE_VALUE", cloudflaretest.Token)
	if _, err := credential.Set(home, cloudflare.CredentialName, credential.Choice{
		Home: credential.HomeExternal, Pointer: credential.Pointer{Env: "ABCD_TEST_CLOUDFLARE_VALUE"},
	}); err != nil {
		t.Fatal(err)
	}
	h := newHarness(t)
	h.cred = credential.Store(home)
	res := h.run(t)
	if res.Host.Status != HostWritten {
		t.Fatalf("host = %+v, want written", res.Host)
	}
	if res.Host.Credential != cloudflare.CredentialName {
		t.Fatalf("the host outcome names credential %q, want %q", res.Host.Credential, cloudflare.CredentialName)
	}
	enc, _ := json.Marshal(res)
	if strings.Contains(string(enc), cloudflaretest.Token) {
		t.Fatal("the result carries the credential's value")
	}
}

// TestTheHostingWalkthroughVerifiesWithTheAdaptersOwnCall: the service the
// walkthrough runs for the hosting credential explains itself and verifies a
// token by the provider's own read, refusing one the provider refuses.
func TestTheHostingWalkthroughVerifiesWithTheAdaptersOwnCall(t *testing.T) {
	srv := cloudflaretest.New(t)
	svc, ok := CredentialService(cloudflare.CredentialName, cloudflare.Adapter{BaseURL: srv.URL})
	if !ok {
		t.Fatal("the hosting credential has no walkthrough")
	}
	if svc.Name != cloudflare.CredentialName || svc.Unlocks == "" || svc.WithoutIt == "" {
		t.Fatalf("service = %+v", svc)
	}
	if err := svc.Verify(context.Background(), cloudflaretest.Token); err != nil {
		t.Fatalf("a good token: %v", err)
	}
	if n := len(srv.CallLog()); n != 1 {
		t.Fatalf("verification made %d calls, want one: %v", n, srv.CallLog())
	}
	const bad = "cf-wrong-token-1111111111111111111111111111"
	err := svc.Verify(context.Background(), bad)
	if err == nil || strings.Contains(err.Error(), bad) {
		t.Fatalf("a refused token: err = %v (never carrying the token)", err)
	}
	if _, ok := CredentialService("openrouter", cloudflare.Adapter{BaseURL: srv.URL}); ok {
		t.Fatal("a name the site does not read has a site walkthrough")
	}
}
