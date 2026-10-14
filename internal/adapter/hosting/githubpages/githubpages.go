// Package githubpages is the hosting provider that publishes a project's site
// on its own GitHub Pages (itd-2610091010348892): the project's release
// workflow builds the about page and development record at each release, and
// GitHub hosts them at owner.github.io/<repo>/, or at a custom domain.
//
// The repository half holds no secret. The deploy job runs in the
// `github-pages` environment with the run's own short-lived token and three
// permissions (contents: read, pages: write, id-token: write), and deploys the
// rendered site through actions/upload-pages-artifact and actions/deploy-pages,
// each pinned to a full commit. There is no host configuration file.
//
// The host half speaks the GitHub REST API through the forge, as the person
// who invoked the verb — the way the site setup's forge stage does, so abcd
// holds no GitHub token and resolves no credential:
//
//	GET  repos/{owner}/{repo}         is it public, and has it a Pages site
//	GET  repos/{owner}/{repo}/pages   the site's source, domain and address
//	POST repos/{owner}/{repo}/pages   create the site, built by a workflow
//	PUT  repos/{owner}/{repo}/pages   route a custom domain to it
//
// These calls are written against GitHub's published REST reference and are
// exercised in this repository only against githubpagestest's in-process fake.
//
// Public repositories only, for now: Pages on a private repository is
// access-controlled only on GitHub's enterprise plan, so a private repository
// is refused rather than published.
package githubpages

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
)

// The deploy step's pins: full commits of the two actions' releases, each at
// least two weeks old when it was pinned.
const (
	uploadAction = "actions/upload-pages-artifact@fc324d3547104276b827a68afc52ff2a11cc49c9 # v5.0.0"
	deployAction = "actions/deploy-pages@368f82528645a54fb793d4d04e342629a3f51346 # v5.0.1"
)

// Environment is the forge environment GitHub Pages deploys through.
const Environment = "github-pages"

// maxHostMessage bounds how much of a forge error reaches a returned error.
const maxHostMessage = 300

// API makes one GitHub REST call through the forge, as the person who invoked
// the verb: method, a repo-relative API path ("repos/acme/widgets/pages") and a
// JSON body or nil. A non-2xx answer is an error naming its HTTP status.
type API func(ctx context.Context, method, path string, body []byte) ([]byte, error)

// Adapter is the github-pages provider. API is the forge; nil reaches nothing.
type Adapter struct {
	API API
}

var _ hosting.Adapter = Adapter{}

// Name is the manifest key.
func (Adapter) Name() string { return "github-pages" }

// CredentialName is empty: the provider acts through the forge as the person.
func (Adapter) CredentialName() string { return "" }

// Secrets: none. The deploy job uses the run's own token.
func (Adapter) Secrets() []string { return nil }

// DeployStep hands the unpacked ./site to Pages and deploys it.
func (Adapter) DeployStep() string {
	return `      - name: Hand the rendered site to GitHub Pages
        uses: ` + uploadAction + `
        with:
          path: site

      - name: Deploy to GitHub Pages
        id: deployment
        uses: ` + deployAction + `
`
}

// HostConfig: the provider reads no configuration file.
func (Adapter) HostConfig(hosting.Site) (string, []byte) { return "", nil }

// Profile: the `github-pages` environment, the run's token with Pages and
// OIDC rights, the switch, public repositories only, and the project path
// GitHub serves the site below.
func (Adapter) Profile(s hosting.Site) hosting.Profile {
	return hosting.Profile{
		Environment: Environment,
		Permissions: []string{"contents: read", "pages: write", "id-token: write"},
		Switched:    true,
		PublicOnly:  true,
		BasePath:    basePath(s),
		Address:     address(s),
	}
}

// address is where GitHub will serve the site: the custom domain, or the
// owner's github.io host below the base path.
func address(s hosting.Site) string {
	owner, _, ok := splitRepo(s.Repo)
	switch {
	case s.Domain != "":
		return "https://" + s.Domain + "/"
	case !ok:
		return ""
	}
	return "https://" + strings.ToLower(owner) + ".github.io" + basePath(s)
}

// basePath is where GitHub serves the site: the root of a custom domain, the
// root of an owner's own owner.github.io repository, and /<repo>/ for any other
// repository.
func basePath(s hosting.Site) string {
	owner, name, ok := splitRepo(s.Repo)
	if !ok || s.Domain != "" || strings.EqualFold(name, owner+".github.io") {
		return "/"
	}
	return "/" + name + "/"
}

// Connect returns the host half. The provider holds no token, so the one it is
// handed is ignored.
func (a Adapter) Connect(string) hosting.Provider { return &client{api: a.API} }

var (
	// nameRe is a GitHub owner or repository name, as ahoy's remote parser
	// accepts one; "." and ".." are refused separately.
	nameRe   = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,100}$`)
	domainRe = regexp.MustCompile(`^([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]([a-z0-9-]{0,61}[a-z0-9])?$`)
	// addrRe is the address Pages reports: http or https, a plain host, a
	// plain path. Address accepts http only for the site's own domain.
	addrRe = regexp.MustCompile(`^https?://[a-z0-9.-]+(/[A-Za-z0-9_.~-]+)*/?$`)
)

func splitRepo(repo string) (owner, name string, ok bool) {
	owner, name, ok = strings.Cut(repo, "/")
	for _, p := range []string{owner, name} {
		if !ok || !nameRe.MatchString(p) || p == "." || p == ".." {
			return "", "", false
		}
	}
	return owner, name, true
}

func checkSite(s hosting.Site) error {
	if _, _, ok := splitRepo(s.Repo); !ok {
		return errors.New("github-pages: the site names no plain owner/name repository on GitHub, so nothing is sent")
	}
	if s.Domain != "" && !domainRe.MatchString(s.Domain) {
		return errors.New("github-pages: the domain is not a plain domain name, so it will not be sent")
	}
	return nil
}

type client struct {
	api API
}

// call makes one call; what names it in an error.
func (c *client) call(ctx context.Context, method, what, path string, body any) ([]byte, error) {
	if c.api == nil {
		return nil, fmt.Errorf("github-pages: %s: the forge is not reachable from here", what)
	}
	var raw []byte
	if body != nil {
		var err error
		if raw, err = json.Marshal(body); err != nil {
			return nil, err
		}
	}
	out, err := c.api(ctx, method, path, raw)
	if err != nil {
		msg := err.Error()
		if len(msg) > maxHostMessage {
			msg = msg[:maxHostMessage] + "…"
		}
		return nil, fmt.Errorf("github-pages: %s: %s", what, msg)
	}
	return out, nil
}

type pagesSite struct {
	BuildType string  `json:"build_type"`
	CNAME     *string `json:"cname"`
	HTMLURL   string  `json:"html_url"`
}

func (c *client) pages(ctx context.Context, s hosting.Site) (pagesSite, error) {
	out, err := c.call(ctx, "GET", "read the Pages site", "repos/"+s.Repo+"/pages", nil)
	if err != nil {
		return pagesSite{}, err
	}
	var p pagesSite
	if json.Unmarshal(out, &p) != nil {
		return pagesSite{}, errors.New("github-pages: read the Pages site: the response is not a Pages site")
	}
	return p, nil
}

// Verify: there is no credential to verify; the forge stage proves the forge.
func (c *client) Verify(context.Context) error {
	return errors.New("github-pages: the provider acts through the forge as you and keeps no credential to verify")
}

// Inspect reads whether the repository is public, whether its Pages site
// exists and is built by a workflow, and whether the domain is routed to it.
// An existing site is named by its address and build type.
func (c *client) Inspect(ctx context.Context, s hosting.Site) (hosting.State, error) {
	if err := checkSite(s); err != nil {
		return hosting.State{}, err
	}
	out, err := c.call(ctx, "GET", "read the repository", "repos/"+s.Repo, nil)
	if err != nil {
		return hosting.State{}, err
	}
	var repo struct {
		Private  *bool `json:"private"`
		HasPages bool  `json:"has_pages"`
	}
	if json.Unmarshal(out, &repo) != nil || repo.Private == nil {
		return hosting.State{}, errors.New("github-pages: read the repository: the response does not say whether it is public")
	}
	if *repo.Private {
		return hosting.State{}, fmt.Errorf("github-pages: %s is private, and the project site is published for public repositories only", s.Repo)
	}
	st := hosting.State{Routed: s.Domain == ""}
	if !repo.HasPages {
		return st, nil
	}
	p, err := c.pages(ctx, s)
	if err != nil {
		return hosting.State{}, err
	}
	switch p.BuildType {
	case "workflow":
	case "legacy":
		return hosting.State{}, fmt.Errorf("github-pages: %s already publishes GitHub Pages from a branch; abcd will not move a live site "+
			"off it, so set its Pages source to GitHub Actions in the repository's settings and re-run", s.Repo)
	default:
		// An empty or unknown build type says nothing about how the live site
		// is built, so it is neither called a branch site nor published over.
		return hosting.State{}, fmt.Errorf("github-pages: %s does not say how its Pages site is built (no build type abcd knows), so abcd "+
			"will not publish over it; check its Pages source in the repository's settings, set it to GitHub Actions, and re-run", s.Repo)
	}
	st.Exists = true
	addr := p.HTMLURL
	if !addrRe.MatchString(addr) {
		addr = "an address that is not a plain http or https URL"
	}
	st.Existing = addr + " (build type " + p.BuildType + ")"
	if p.CNAME != nil && *p.CNAME != "" {
		cname := *p.CNAME
		printable := domainRe.MatchString(cname)
		if !printable {
			cname = "a domain with an unprintable name"
		}
		switch {
		case s.Domain == "" && printable:
			// The workflow would build for /<repo>/ while GitHub serves the
			// site at the domain's root, so every asset would 404.
			// The domain is declared in one of two places: the flag, on the
			// run that writes the hosting block, or the block's own field
			// once it exists (setup refuses the flag then).
			return hosting.State{}, fmt.Errorf("github-pages: %s already serves its Pages site at %s; abcd will not move a live domain, so declare %s "+
				"as the site's domain: pass --domain %s when setup writes the hosting block, or set the hosting block's domain to %s "+
				"when the composition already has one", s.Repo, cname, cname, cname, cname)
		case *p.CNAME != s.Domain:
			return hosting.State{}, fmt.Errorf("github-pages: %s already serves its Pages site at %s; abcd will not move a live domain", s.Repo, cname)
		}
		st.Routed = true
	}
	return st, nil
}

// Create creates the Pages site, built by the release workflow.
func (c *client) Create(ctx context.Context, s hosting.Site) error {
	if err := checkSite(s); err != nil {
		return err
	}
	_, err := c.call(ctx, "POST", "create the Pages site", "repos/"+s.Repo+"/pages", map[string]string{"build_type": "workflow"})
	return err
}

// Route sets the custom domain. A site with no domain is a no-op.
func (c *client) Route(ctx context.Context, s hosting.Site) error {
	if err := checkSite(s); err != nil {
		return err
	}
	if s.Domain == "" {
		return nil
	}
	_, err := c.call(ctx, "PUT", "route the domain", "repos/"+s.Repo+"/pages", map[string]string{"cname": s.Domain})
	return err
}

// httpsPending is the note on a custom domain GitHub still serves over http.
const httpsPending = "HTTPS is not enforced yet; GitHub issues the certificate after the domain resolves, which can take up to a day, " +
	"and then Enforce HTTPS can be switched on in the repository's Pages settings"

// Address is the address Pages reports for the site, always as https.
//
// A custom domain is reported as http until GitHub has issued its certificate
// and HTTPS is enforced: GitHub requests the certificate only once the domain's
// DNS resolves to Pages, DNS changes can take up to 24 hours, and "it can take
// up to 24 hours before" Enforce HTTPS is available
// (https://docs.github.com/en/pages/configuring-a-custom-domain-for-your-github-pages-site/managing-a-custom-domain-for-your-github-pages-site,
// https://docs.github.com/en/pages/getting-started-with-github-pages/securing-your-github-pages-site-with-https).
// So an http address on the site's own domain is the site's: host and path
// are what identify it, the scheme only says how far along the certificate
// is, and the note says so. A github.io address is served over https from the
// start, so http there, or on any other host, is refused.
func (c *client) Address(ctx context.Context, s hosting.Site) (string, string, error) {
	if err := checkSite(s); err != nil {
		return "", "", err
	}
	p, err := c.pages(ctx, s)
	if err != nil {
		return "", "", err
	}
	u, err := url.Parse(p.HTMLURL)
	if !addrRe.MatchString(p.HTMLURL) || err != nil {
		return "", "", errors.New("github-pages: the Pages site reports no plain https address")
	}
	if u.Scheme == "https" {
		return p.HTMLURL, "", nil
	}
	if s.Domain == "" || u.Host != s.Domain {
		return "", "", errors.New("github-pages: the Pages site reports an http address that is not the site's own domain")
	}
	return "https://" + u.Host + u.Path, httpsPending, nil
}
