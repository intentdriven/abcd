package site

// credential.go is the site setup's half of the credential walkthrough
// (itd-2609221017023290): what the hosting credential unlocks, what works
// without it, and the hosting adapter's own verification call. The store and
// the walkthrough itself are internal/core/credential's; the setup reads the
// credential through credential.Store by name and never any other way.

import (
	"context"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/core/credential"
)

// CredentialService is the walkthrough's service for name when adapter reads
// it: false for a name the site setup does not read.
func CredentialService(name string, adapter hosting.Adapter) (credential.Service, bool) {
	if adapter == nil || name != adapter.CredentialName() {
		return credential.Service{}, false
	}
	return credential.Service{
		Name: name,
		Unlocks: "`abcd site setup`'s host stage on " + adapter.Name() + ": creating the site's host, routing its " +
			"domain to it and reporting the live address",
		WithoutIt: "the rest of `abcd site setup` (the files it writes and the forge environments) runs, and the host " +
			"stage stops at no_credential, contacting nothing and naming what remains",
		Verify: func(ctx context.Context, token string) error {
			err := adapter.Connect(token).Verify(ctx)
			if err != nil && token != "" && strings.Contains(err.Error(), token) {
				return scrubbed{err.Error(), token}
			}
			return err
		},
	}, true
}

// CredentialServiceFor is CredentialService over every hosting provider the
// site setup knows.
func CredentialServiceFor(name string) (credential.Service, bool) {
	for _, a := range adapters {
		if svc, ok := CredentialService(name, a); ok {
			return svc, true
		}
	}
	return credential.Service{}, false
}

// CredentialNames are the credentials the site setup reads, one per hosting
// provider.
func CredentialNames() []string {
	out := make([]string, 0, len(adapters))
	for _, a := range adapters {
		out = append(out, a.CredentialName())
	}
	return out
}

// scrubbed is an error whose message had the credential replaced.
type scrubbed struct{ msg, token string }

func (s scrubbed) Error() string { return strings.ReplaceAll(s.msg, s.token, "[credential]") }
