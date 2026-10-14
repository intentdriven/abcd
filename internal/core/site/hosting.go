package site

import (
	"regexp"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
)

// Hosting is the manifest's `hosting` block: which provider adapter `abcd site
// setup` routes the site through, the name of the host it creates there, and the
// custom domain it attaches, if any. It holds no credential and no account
// identifier: the credential is the machine's, read by name at setup time, and
// never enters the repository.
type Hosting struct {
	Provider string `json:"provider"`
	Name     string `json:"name"`
	Domain   string `json:"domain,omitempty"`
	// Publish is the project-site switch of a provider that publishes only
	// while it is on (github-pages, itd-2610091010348892): setup asks for it
	// and writes `"publish": true`, and the release workflow publishes nothing
	// while it is anything else. A provider without a switch refuses it.
	Publish *bool `json:"publish,omitempty"`
}

// On reports whether the switch is on.
func (h *Hosting) On() bool { return h != nil && h.Publish != nil && *h.Publish }

// hostNameRe is the host-name charset: lowercase letters, digits and inner
// hyphens, at most 63 characters. It is the narrowest of the providers' rules
// and is applied before the name reaches an API path or a workflow file.
var hostNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// domainRe is a lowercase DNS name of at least two labels, no scheme, no port,
// no path and no trailing dot. It reaches a provider API body and a committed
// config file, so it is held to the plainest spelling.
var domainRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)

// validate refuses a hosting block setup could not act on.
func (h *Hosting) validate(bad func(string, ...any) error) error {
	if h == nil {
		return nil
	}
	if h.Provider == "" {
		return bad("hosting.provider is empty; it names the adapter setup routes the site through")
	}
	a, ok := adapterNamed(h.Provider)
	if !ok {
		return bad("hosting.provider %q is not a provider abcd ships (%s)", h.Provider, providerList())
	}
	if h.Publish != nil && !a.Profile(hosting.Site{}).Switched {
		return bad("hosting.publish is the switch of a provider that publishes only while it is on; %s deploys every release and reads no switch", h.Provider)
	}
	if !hostNameRe.MatchString(h.Name) {
		return bad("hosting.name %q is not a host name (lowercase letters, digits and inner hyphens, at most 63)", h.Name)
	}
	if h.Domain != "" && (len(h.Domain) > 253 || !domainRe.MatchString(h.Domain)) {
		return bad("hosting.domain %q is not a plain lowercase domain name (no scheme, port or path)", h.Domain)
	}
	return nil
}
