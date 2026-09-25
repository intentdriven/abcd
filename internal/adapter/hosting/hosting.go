// Package hosting is the provider seam `abcd site setup` routes a rendered site
// through (itd-2609061543533170, spc-2609212141407459 scope 2). It is the second
// seam under internal/adapter beside the scanners, and it follows their shape:
// the core consumes the interface, never a vendor, and a provider is one
// implementation of it.
//
// One provider ships (cloudflare, the host abcd's own site uses). A second is a
// new implementation of Adapter and one line in the core's provider list, never
// a change to the verb.
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
package hosting

import "context"

// Site is what a provider hosts: the host's name, and the custom domain routed
// to it, if any. Both are validated by the caller before they reach a provider
// and again by the provider before they reach a request.
type Site struct {
	Name   string
	Domain string
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
	// repo-relative, and its bytes for s.
	HostConfig(s Site) (rel string, data []byte)
	// Connect returns a Provider acting with token. The token stays inside it.
	Connect(token string) Provider
}
