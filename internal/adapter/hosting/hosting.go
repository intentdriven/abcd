// Package hosting is the provider seam `abcd site setup` routes a rendered site
// through (itd-2609061543533170, spc-2609212141407459 scope 2). It is the second
// seam under internal/adapter beside the scanners, and it follows their shape:
// the core consumes the interface, never a vendor, and a provider is one
// implementation of it.
//
// Two providers ship: cloudflare, the host abcd's own site uses, and
// github-pages, a project's own GitHub Pages, built by its release workflow
// (itd-2610091010348892). Another is a new implementation of Adapter and one
// line in the core's provider list, never a change to the verb.
//
// The seam has two halves because a provider touches the site twice:
//
//   - in the REPOSITORY, it names the secrets its deploy job reads, the deploy
//     step itself, and the host configuration file that step deploys from.
//     These are data, rendered into files the verb writes; nothing here runs.
//   - on the HOST, through its API and the person's credential, it creates the
//     host, routes the domain to it and reports the live address. Every call
//     is made only after the verb's confirmation, and Inspect, the read, is the
//     only call made before it.
//
// The credential is handed to Connect and lives only inside the returned
// Provider. No method returns it, no error carries it, and nothing writes it.
// A provider that names no credential (CredentialName is empty) acts through
// the forge as the person who invoked the verb, the way the forge stage does,
// and Connect is handed nothing.
package hosting

import "context"

// Site is what a provider hosts: the host's name, and the custom domain routed
// to it, if any. Both are validated by the caller before they reach a provider
// and again by the provider before they reach a request.
type Site struct {
	Name   string
	Domain string
	// Repo is the forge repository the site is built from, as owner/name,
	// when the caller knows it. A provider that hosts the site on the forge
	// itself acts on it; the others ignore it.
	Repo string
}

// Profile is what the verb and the workflow it writes need to know about a
// provider beyond its deploy step.
type Profile struct {
	// Environment is the forge environment the deploy job runs in.
	Environment string
	// Permissions are the deploy job's token permissions, one `scope: level`
	// each, in the order the workflow lists them.
	Permissions []string
	// Switched: the provider publishes only while the composition's
	// hosting.publish switch is on. Setup asks for the switch, and the
	// workflow publishes nothing while it is off.
	Switched bool
	// PublicOnly: the provider serves public repositories only, so setup
	// refuses a private one before it writes anything.
	PublicOnly bool
	// BasePath is the path the site is served below on the host, with a
	// leading and a trailing slash; "/" is the root.
	BasePath string
}

// State is what the host holds for a site, read without writing anything.
type State struct {
	// Exists is true when the host already exists.
	Exists bool
	// Routed is true when the domain is already routed to the host, and
	// vacuously true when the site asks for no domain.
	Routed bool
}

// Provider is the live half of the seam: one connected account.
type Provider interface {
	// Verify is the provider's own call proving the credential reaches an
	// account it can act in. It writes nothing; the credential walkthrough
	// makes it before the credential is stored.
	Verify(ctx context.Context) error
	// Inspect reads what the host holds for s. It writes nothing.
	Inspect(ctx context.Context, s Site) (State, error)
	// Create creates the host for s.
	Create(ctx context.Context, s Site) error
	// Route routes s.Domain to the host. A site with no domain is a no-op.
	Route(ctx context.Context, s Site) error
	// Address reports the address the site is served from.
	Address(ctx context.Context, s Site) (string, error)
}

// Adapter is one provider: the repository half as data, and Connect for the
// host half. It is the interface a second provider implements.
type Adapter interface {
	// Name is the provider's key in the composition manifest's hosting block.
	Name() string
	// CredentialName is the name the credential is resolved by on this
	// machine. It names the credential; it never is one.
	CredentialName() string
	// Secrets are the forge environment secrets the deploy job reads, by name.
	Secrets() []string
	// DeployStep is the workflow step, as YAML indented for a job's step list,
	// that deploys the unpacked site from ./site. It reads its credentials from
	// the secrets named by Secrets and nothing else.
	DeployStep() string
	// HostConfig is the committed configuration file the deploy step reads,
	// repo-relative, and its bytes for s; an empty path when the provider
	// reads none.
	HostConfig(s Site) (rel string, data []byte)
	// Profile is the provider's facts for s.
	Profile(s Site) Profile
	// Connect returns a Provider acting with token. The token stays inside it.
	Connect(token string) Provider
}
