package site

// `abcd site setup` (itd-2609061543533170, spc-2609212141407459 scope 1): one
// verb that takes a managed repository's site from the checkout to a live
// address.
//
// It works in three stages, in this order, and each stage's outcome is
// reported whether or not the next one runs:
//
//  1. THE REPOSITORY. The composition (.abcd/site.json, derived from the
//     identity block and the documentation), the site's static inputs under
//     site-src/, the render-on-release-then-deploy workflow, and the
//     provider's host configuration. Written through the launch scaffold's
//     writer, so a file already current is not rewritten, a file the
//     repository owns once it exists (the composition and the static inputs)
//     is kept, and machinery that drifted refuses the whole stage — with
//     nothing written and nothing remote attempted — unless it is confirmed.
//  2. THE FORGE. The two deployment environments the workflow runs in,
//     restricted to the default branch and release tags, created or corrected
//     through the forge's API as the person who invoked the verb (adr-44: a
//     remote write only through a dedicated verb, invoked AND confirmed).
//  3. THE HOST. With a hosting credential on this machine, the provider
//     adapter creates the host, routes the domain to it and reports the live
//     address, again only after a confirmation that names the changes.
//     Without one, the stage stops and says exactly what remains.
//
// What never happens: a secret's value is read or written anywhere (the
// forge's environment secrets are the person's step, printed as the exact
// command), the credential is written into the repository or the report, or a
// file outside the list above is touched.

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"github.com/intentdriven/abcd/internal/adapter/hosting"
	"github.com/intentdriven/abcd/internal/core/ahoy"
	"github.com/intentdriven/abcd/internal/core/credential"
	"github.com/intentdriven/abcd/internal/core/launch/scaffold"
	"github.com/intentdriven/abcd/internal/core/positioning"
	"github.com/intentdriven/abcd/internal/gitutil"
)

//go:embed setupsrc/ui.json setupsrc/site.css setupsrc/site.js setupsrc/record.js setupsrc/headers setupsrc/site.yml.tmpl
var setupSources embed.FS

// seedNames are the static inputs a managed repository is seeded with under
// site-src/: byte copies of abcd's own (TestTheSeedSourcesAreAbcdsOwn). The
// redirects map is not among them: it is abcd's own URL history.
var seedNames = []string{"ui.json", "site.css", "site.js", "record.js", "headers"}

// SiteWorkflowRelPath is the workflow setup writes.
const SiteWorkflowRelPath = ".github/workflows/site.yml"

// The two deployment environments the workflow runs in.
const (
	EnvRender = "site-render"
	EnvDeploy = "site"
)

// abcdReleaseRepo is where the workflow downloads the abcd binary it renders
// with, and whose release workflow must have signed it.
const abcdReleaseRepo = "intentdriven/abcd"

// Overall statuses.
const (
	// StatusChanged: something was written, locally or remotely.
	StatusChanged = "changed"
	// StatusNoChange: nothing was written, because every file, environment and
	// host was already current or is left to a remaining step (an existing
	// environment setup does not rewrite).
	StatusNoChange = "no_change"
	// StatusDeclined: a confirmation was declined, so a remote write did not
	// happen.
	StatusDeclined = "declined"
	// StatusRefused: a gate or a failure stopped a stage.
	StatusRefused = "refused"
)

// Remote outcome statuses (an environment).
const (
	RemoteCurrent     = "current"
	RemoteWritten     = "written"
	RemoteDeclined    = "declined"
	RemoteRefused     = "refused"
	RemoteUnreachable = "unreachable"
	RemoteNotReached  = "not_reached"
	// RemoteUnrestricted: the environment exists and admits more than the
	// default branch and release tags. The forge's environment write replaces
	// an environment's whole protection set (its required reviewers and wait
	// timer with it), so setup never rewrites one that exists: restricting it
	// is a remaining step.
	RemoteUnrestricted = "unrestricted"
)

// Host outcome statuses.
const (
	HostCurrent      = "current"
	HostWritten      = "written"
	HostDeclined     = "declined"
	HostRefused      = "refused"
	HostNoCredential = "no_credential"
	HostNotReached   = "not_reached"
)

// ErrNotManaged refuses a folder abcd does not manage.
var ErrNotManaged = errors.New("site setup: this is not a repository abcd manages (run `abcd ahoy install` first)")

// Asker confirms a remote write. A nil Asker declines.
type Asker interface {
	Confirm(question string) bool
}

// SetupRequest is one run of the verb.
type SetupRequest struct {
	// RepoRoot is any directory inside the repository.
	RepoRoot string
	// Name and Domain seed the hosting block when the composition has none.
	// Name defaults to the repository's own name.
	Name, Domain string
	// Confirm replaces machinery that drifted from what setup writes.
	Confirm bool
	// Asker confirms each remote write; nil declines them all.
	Asker Asker
	// Forge is the repository's forge; nil resolves GitHub through gh.
	Forge Forge
	// Credentials resolves the hosting credential by name; nil is this
	// machine's store.
	Credentials credential.Source
	// Adapter overrides the provider the composition names (tests point it at
	// a fake); nil resolves the composition's provider from the list.
	Adapter hosting.Adapter
	Context context.Context
}

// EnvironmentOutcome is one environment's result.
type EnvironmentOutcome struct {
	Name    string   `json:"name"`
	Status  string   `json:"status"`
	Changes []string `json:"changes,omitempty"`
}

// HostOutcome is the host stage's result.
type HostOutcome struct {
	Provider string   `json:"provider"`
	Name     string   `json:"name"`
	Domain   string   `json:"domain,omitempty"`
	Status   string   `json:"status"`
	Changes  []string `json:"changes,omitempty"`
	Address  string   `json:"address,omitempty"`
	Detail   string   `json:"detail,omitempty"`
}

// SetupResult is what the verb did, stage by stage, and what remains.
type SetupResult struct {
	Status       string                 `json:"status"`
	Repo         string                 `json:"repo,omitempty"`
	Files        []scaffold.FileOutcome `json:"files"`
	Environments []EnvironmentOutcome   `json:"environments"`
	Host         HostOutcome            `json:"host"`
	// Remaining are the exact steps left for the person, in order.
	Remaining []string `json:"remaining,omitempty"`
	// Notes say what the verb deliberately did not do, and why.
	Notes []string `json:"notes,omitempty"`
}

// Setup runs the verb. It returns an error only for a gate that stops it before
// any stage runs; every later failure is a refused stage in the result.
func Setup(req SetupRequest) (SetupResult, error) {
	ctx := req.Context
	if ctx == nil {
		ctx = context.Background()
	}
	root, err := setupRoot(req.RepoRoot)
	if err != nil {
		return SetupResult{}, err
	}
	det, err := ahoy.Detect(root)
	if err != nil {
		return SetupResult{}, fmt.Errorf("site setup: could not classify this folder: %w", err)
	}
	if det.FolderKind != ahoy.ManagedRepo {
		return SetupResult{}, ErrNotManaged
	}

	manifest, manifestBytes, err := setupComposition(root, req)
	if err != nil {
		return SetupResult{}, err
	}
	hostingBlock := *manifest.Hosting
	adapter := req.Adapter
	if adapter == nil {
		a, ok := adapterNamed(hostingBlock.Provider)
		if !ok {
			return SetupResult{}, fmt.Errorf("site setup: hosting.provider %q is not a provider abcd ships (%s)", hostingBlock.Provider, providerList())
		}
		adapter = a
	}
	s := hosting.Site{Name: hostingBlock.Name, Domain: hostingBlock.Domain}
	branch, _ := scaffold.DeriveRepoFacts(root)

	res := SetupResult{Host: HostOutcome{Provider: adapter.Name(), Name: s.Name, Domain: s.Domain, Status: HostNotReached}}

	// Stage 1: the repository.
	planned, err := plannedFiles(root, manifest, manifestBytes, branch, adapter, s)
	if err != nil {
		return SetupResult{}, err
	}
	outcomes, wrote, _, werr := scaffold.WriteFiles(root, planned, req.Confirm)
	res.Files = outcomes
	if werr != nil {
		res.Status = StatusRefused
		if errors.Is(werr, scaffold.ErrScaffoldBlocked) {
			res.Notes = append(res.Notes, "a file setup owns was edited by hand, so nothing was written and nothing remote was attempted; "+
				"re-run with --confirm to replace it")
		} else {
			res.Notes = append(res.Notes, "a write failed, so nothing remote was attempted: "+scrubRoot(werr, root))
		}
		for _, env := range []string{EnvRender, EnvDeploy} {
			res.Environments = append(res.Environments, EnvironmentOutcome{Name: env, Status: RemoteNotReached})
		}
		return res, nil
	}
	changed := wrote > 0
	declined, refused := false, false

	// Stage 2: the forge.
	forge := req.Forge
	if forge == nil {
		f, ferr := GitHubForge(root)
		if ferr != nil {
			res.Notes = append(res.Notes, "the forge is not reachable from this checkout, so the environments were not created: "+ferr.Error())
		} else {
			forge = f
		}
	}
	envOK := true
	// Without this step ahead of the secret steps, a person following them
	// sets the token first, and the workflow's first run creates the deploy
	// environment with no policy at all: deployable from any branch.
	createStep := fmt.Sprintf("create the forge environments %s and %s, each admitting only branch %s and tags v*, "+
		"as the workflow's header describes, before setting any deploy secret", EnvRender, EnvDeploy, branch)
	if forge == nil {
		envOK = false
		for _, env := range []string{EnvRender, EnvDeploy} {
			res.Environments = append(res.Environments, EnvironmentOutcome{Name: env, Status: RemoteUnreachable})
		}
		res.Remaining = append(res.Remaining, createStep)
	} else {
		res.Repo = forge.Repo()
		envs, st, note, steps := setupEnvironments(ctx, forge, branch, req.Asker)
		res.Environments = envs
		res.Remaining = append(res.Remaining, steps...)
		switch st {
		case RemoteWritten:
			changed = true
		case RemoteDeclined:
			declined, envOK = true, false
			res.Remaining = append(res.Remaining, "re-run `abcd site setup` and confirm, to create the environments "+EnvRender+" and "+EnvDeploy)
		case RemoteRefused:
			refused, envOK = true, false
			res.Notes = append(res.Notes, note)
			res.Remaining = append(res.Remaining, createStep)
		}
	}

	// Stage 3: the host. A forge failure stops here: the environments are what
	// the deploy runs in, and a host created for a deploy that cannot run is a
	// half-built site.
	if refused {
		res.Host.Detail = "not reached: the forge stage failed"
	} else {
		hc, hdeclined, hrefused := setupHost(ctx, adapter, s, req)
		res.Host = hc
		switch {
		case hrefused:
			refused = true
		case hdeclined:
			declined = true
		case hc.Status == HostWritten:
			changed = true
		}
		if hc.Status == HostNoCredential {
			step := fmt.Sprintf("store a %s API token under the name %s in %s (mode 0600) and re-run `abcd site setup`, "+
				"or create the host %s", adapter.Name(), adapter.CredentialName(), credential.StorePath, s.Name)
			if s.Domain != "" {
				step += " and route " + s.Domain + " to it"
			}
			res.Remaining = append(res.Remaining, step+" in the provider's own console")
		}
		if hdeclined {
			res.Remaining = append(res.Remaining, "re-run `abcd site setup` and confirm, to create and route the host")
		}
	}

	// The deploy environment's secrets: never set by abcd (a value would pass
	// through it), but their presence is readable by name.
	res.Remaining = append(res.Remaining, secretSteps(ctx, forge, envOK, adapter, &res)...)

	if wrote > 0 {
		var paths []string
		for _, f := range res.Files {
			if f.Status == scaffold.StatusWritten {
				paths = append(paths, f.Path)
			}
		}
		res.Remaining = append([]string{"commit the written files and push them to " + branch + ": `git add " +
			strings.Join(paths, " ") + "`"}, res.Remaining...)
	}
	res.Notes = append(res.Notes, "the site renders and deploys on the next published release, whether a person or "+
		"the release workflows publish it; `gh workflow run site.yml` deploys the latest one now")

	switch {
	case refused:
		res.Status = StatusRefused
	case declined:
		res.Status = StatusDeclined
	case changed:
		res.Status = StatusChanged
	default:
		res.Status = StatusNoChange
	}
	return res, nil
}

// setupRoot anchors the run at the working-tree root.
func setupRoot(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	top, err := gitutil.Run(abs, "rev-parse", "--show-toplevel")
	if err != nil || strings.TrimSpace(top) == "" {
		return "", errors.New("site setup: this is not inside a git checkout")
	}
	return strings.TrimSpace(top), nil
}

// scrubRoot keeps the checkout's absolute path out of a message.
func scrubRoot(err error, root string) string {
	return strings.ReplaceAll(err.Error(), root, ".")
}

// setupComposition returns the composition setup works from, and the bytes to
// write when the repository has none yet (nil when it has one).
func setupComposition(root string, req SetupRequest) (Manifest, []byte, error) {
	bad := func(format string, args ...any) error {
		return fmt.Errorf("%w: %s", ErrManifestInvalid, fmt.Sprintf(format, args...))
	}
	m, err := LoadManifest(root)
	switch {
	case err == nil:
		if m.Hosting == nil {
			h := &Hosting{Provider: DefaultProvider, Name: req.Name, Domain: req.Domain}
			if h.Name == "" {
				h.Name = defaultHostName(root)
			}
			if err := h.validate(bad); err != nil {
				return Manifest{}, nil, err
			}
			if req.Domain != "" || req.Name != "" {
				// The composition is the repository's own, so setup does not
				// rewrite it; the domain has to live in it for the next run to
				// route the same one.
				blk, _ := json.Marshal(h)
				return Manifest{}, nil, fmt.Errorf("site setup: %s exists and declares no hosting block; add `\"hosting\": %s` to it rather than passing --name or --domain",
					ManifestRelPath, blk)
			}
			m.Hosting = h
		} else if (req.Name != "" && req.Name != m.Hosting.Name) || (req.Domain != "" && req.Domain != m.Hosting.Domain) {
			return Manifest{}, nil, fmt.Errorf("site setup: %s already names the host %q and domain %q; edit its hosting block to change them",
				ManifestRelPath, m.Hosting.Name, m.Hosting.Domain)
		}
		return m, nil, nil
	case !os.IsNotExist(err):
		return Manifest{}, nil, err
	}

	cfg, ok, cerr := positioning.LoadConfig(root)
	if cerr != nil {
		return Manifest{}, nil, fmt.Errorf("site setup: the identity pointer cannot be read: %w", cerr)
	}
	if !ok {
		return Manifest{}, nil, errors.New("site setup: this repository has recorded no identity block, and the landing page's hero is composed from it; record one first with `abcd identity init`")
	}
	hero := ""
	for _, p := range []string{"docs/README.md", "docs/index.md"} {
		if fi, serr := os.Lstat(filepath.Join(root, filepath.FromSlash(p))); serr == nil && fi.Mode().IsRegular() {
			hero = p
			break
		}
	}
	if hero == "" {
		return Manifest{}, nil, errors.New("site setup: the landing page is composed from the documentation, and this repository has none; write docs/README.md (its first heading and paragraph become the hero) and re-run")
	}
	name := req.Name
	if name == "" {
		name = defaultHostName(root)
	}
	m = Manifest{
		SchemaVersion: 1,
		Purpose: "Composition manifest for this repository's site, written by `abcd site setup`. It names WHERE each block " +
			"of the site comes from and carries no prose. `pages` switches pages of the closed set off; `hosting` names " +
			"where setup puts the site.",
		Identity:  BlockRef{File: cfg.Block.File, Heading: cfg.Block.Heading},
		UIStrings: "site-src/ui.json",
		Home: Home{
			Hero:     Hero{Page: hero, Figure: figureFirstImage},
			Chapters: []Chapter{{Letter: "a", Page: hero, Layout: LayoutProse}},
		},
		Hosting: &Hosting{Provider: DefaultProvider, Name: name, Domain: req.Domain},
	}
	if err := m.validate(); err != nil {
		return Manifest{}, nil, err
	}
	raw, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return Manifest{}, nil, err
	}
	return m, append(raw, '\n'), nil
}

// hostCharRe is what a repository name keeps on its way to a host name.
var hostCharRe = regexp.MustCompile(`[^a-z0-9-]+`)

// defaultHostName is the repository's own name, as a host name: the origin's
// repository segment when it names one, the checkout's directory otherwise.
func defaultHostName(root string) string {
	raw := filepath.Base(root)
	if repo, err := ahoy.GitHubRepo(root); err == nil {
		if _, name, ok := strings.Cut(repo, "/"); ok {
			raw = name
		}
	}
	n := strings.Trim(hostCharRe.ReplaceAllString(strings.ToLower(raw), "-"), "-")
	if len(n) > 63 {
		n = strings.TrimRight(n[:63], "-")
	}
	return n
}

// plannedFiles is the repository half.
func plannedFiles(root string, m Manifest, manifestBytes []byte, branch string, adapter hosting.Adapter, s hosting.Site) ([]scaffold.PlannedFile, error) {
	var planned []scaffold.PlannedFile
	if manifestBytes != nil {
		planned = append(planned, scaffold.PlannedFile{Path: ManifestRelPath, Data: manifestBytes, Seed: true})
	}
	for _, name := range seedNames {
		data, err := setupSources.ReadFile("setupsrc/" + name)
		if err != nil {
			return nil, err
		}
		rel := "site-src/" + name
		if name == "ui.json" {
			rel = m.UIStrings
		}
		planned = append(planned, scaffold.PlannedFile{Path: rel, Data: data, Seed: true})
	}
	wf, err := renderSiteWorkflow(branch, adapter)
	if err != nil {
		return nil, err
	}
	planned = append(planned, scaffold.PlannedFile{Path: SiteWorkflowRelPath, Data: wf})
	rel, data := adapter.HostConfig(s)
	planned = append(planned, scaffold.PlannedFile{Path: rel, Data: data})
	// The writer resolves each path under the root by name, so a symlinked
	// directory on the way would carry a write out of the repository. Refuse
	// the run before any write rather than follow one.
	for _, p := range planned {
		if err := refuseSymlinkedAncestor(root, p.Path); err != nil {
			return nil, err
		}
	}
	return planned, nil
}

// refuseSymlinkedAncestor refuses rel when a directory between root and its
// leaf is a symlink.
func refuseSymlinkedAncestor(root, rel string) error {
	parts := strings.Split(rel, "/")
	cur := root
	for _, part := range parts[:len(parts)-1] {
		cur = filepath.Join(cur, part)
		fi, err := os.Lstat(cur)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("site setup: %s could not be examined", rel)
		}
		if fi.Mode()&os.ModeSymlink != 0 || !fi.IsDir() {
			return fmt.Errorf("site setup: a directory on the way to %s is a symlink or not a directory, so nothing is written through it", rel)
		}
	}
	return nil
}

// renderSiteWorkflow renders the workflow for the repository's default branch
// and the provider's deploy step.
func renderSiteWorkflow(branch string, adapter hosting.Adapter) ([]byte, error) {
	raw, err := setupSources.ReadFile("setupsrc/site.yml.tmpl")
	if err != nil {
		return nil, err
	}
	tmpl, err := template.New("site.yml").Delims("<%", "%>").Option("missingkey=error").Parse(string(raw))
	if err != nil {
		return nil, err
	}
	rel, _ := adapter.HostConfig(hosting.Site{Name: "x"})
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		// The branch comes from scaffold.DeriveRepoFacts, allowlisted there
		// against every YAML metacharacter.
		"DefaultBranch": branch,
		"Provider":      adapter.Name(),
		"Secrets":       adapter.Secrets(),
		"DeployStep":    adapter.DeployStep(),
		"HostConfig":    rel,
		"AbcdRepo":      abcdReleaseRepo,
	})
	return buf.Bytes(), err
}

// desiredPolicies are the refs each environment admits.
func desiredPolicies(branch string) []BranchPolicy {
	return []BranchPolicy{{Name: branch, Type: "branch"}, {Name: "v*", Type: "tag"}}
}

// setupEnvironments reads both environments, asks once for every change, and
// applies them in order, stopping at the first failure. It returns the
// outcomes, the stage's status, a note on a refusal, and the steps left for
// the person.
//
// Only an ABSENT environment is written through the forge's environment
// endpoint: that write replaces the environment's whole protection set, so on
// an existing one it would drop the required reviewers and wait timer a person
// put there. An existing environment on custom policies only gains the
// policies it lacks (their own endpoint, which touches nothing else); one that
// admits more refs than that, through its protection mode or through a custom
// rule beyond the two it needs, is left as it is, and restricting it is a step.
func setupEnvironments(ctx context.Context, forge Forge, branch string, asker Asker) ([]EnvironmentOutcome, string, string, []string) {
	names := []string{EnvRender, EnvDeploy}
	outcomes := make([]EnvironmentOutcome, len(names))
	var steps []string
	fail := func(what string, err error) ([]EnvironmentOutcome, string, string, []string) {
		for i := range outcomes {
			if outcomes[i].Status == "" || outcomes[i].Status == RemoteNotReached {
				outcomes[i] = EnvironmentOutcome{Name: names[i], Status: RemoteRefused, Changes: outcomes[i].Changes}
			}
		}
		return outcomes, RemoteRefused, "the forge refused " + what + ", so the remaining forge and host steps were not attempted: " + err.Error(), steps
	}
	envs, err := forge.Environments(ctx)
	if err != nil {
		return fail("the environment read", err)
	}
	type plan struct {
		put     bool
		missing []BranchPolicy
	}
	plans := make([]plan, len(names))
	var all []string
	for i, env := range names {
		outcomes[i] = EnvironmentOutcome{Name: env, Status: RemoteNotReached}
		st, exists := envs[env]
		if exists && (!st.CustomBranchPolicies || st.ProtectedBranches) {
			outcomes[i].Status = RemoteUnrestricted
			steps = append(steps, fmt.Sprintf("restrict the existing environment %s to branch %s and tags v*: in the forge's settings "+
				"for %s, admit selected branches and tags only, then re-run `abcd site setup` to add the two rules. "+
				"abcd does not rewrite an environment that exists, because the forge's environment write replaces its "+
				"protection rules, required reviewers included", env, branch, env))
			continue
		}
		var have []BranchPolicy
		if exists {
			if have, err = forge.Policies(ctx, env); err != nil {
				return fail("the deployment policy read for "+env, err)
			}
			// A rule of the environment's own beyond the two it needs admits
			// more refs than they do (a branch `*`, for one), so the
			// environment is as open as one on no custom rules at all.
			var extra []string
			for _, h := range have {
				if !containsPolicy(desiredPolicies(branch), h) {
					extra = append(extra, h.Type+" "+h.Name)
				}
			}
			if len(extra) > 0 {
				outcomes[i].Status = RemoteUnrestricted
				steps = append(steps, fmt.Sprintf("restrict the existing environment %s to branch %s and tags v*: in the forge's settings "+
					"for %s, remove its deployment rules for %s, then re-run `abcd site setup` to add any rule it lacks. "+
					"abcd does not remove a rule a person put on an environment that exists", env, branch, env, strings.Join(extra, ", ")))
				continue
			}
		}
		p := plan{put: !exists}
		for _, want := range desiredPolicies(branch) {
			if !containsPolicy(have, want) {
				p.missing = append(p.missing, want)
			}
		}
		plans[i] = p
		if !exists {
			outcomes[i].Changes = append(outcomes[i].Changes, "create "+env+" admitting named branches and tags only")
		}
		for _, m := range p.missing {
			outcomes[i].Changes = append(outcomes[i].Changes, "admit "+m.Type+" "+m.Name+" to "+env)
		}
		all = append(all, outcomes[i].Changes...)
		if len(outcomes[i].Changes) == 0 {
			outcomes[i].Status = RemoteCurrent
		}
	}
	if len(all) == 0 {
		return outcomes, RemoteCurrent, "", steps
	}
	if asker == nil || !asker.Confirm("Change the deployment environments on "+forge.Repo()+"? ("+strings.Join(all, "; ")+")") {
		for i := range outcomes {
			if outcomes[i].Status != RemoteCurrent && outcomes[i].Status != RemoteUnrestricted {
				outcomes[i].Status = RemoteDeclined
			}
		}
		return outcomes, RemoteDeclined, "", steps
	}
	for i, env := range names {
		if outcomes[i].Status == RemoteCurrent || outcomes[i].Status == RemoteUnrestricted {
			continue
		}
		if plans[i].put {
			if err := forge.PutEnvironment(ctx, env); err != nil {
				return fail("the environment write for "+env, err)
			}
		}
		for _, m := range plans[i].missing {
			if err := forge.AddPolicy(ctx, env, m); err != nil {
				return fail("the deployment policy write for "+env, err)
			}
		}
		outcomes[i].Status = RemoteWritten
	}
	return outcomes, RemoteWritten, "", steps
}

func containsPolicy(have []BranchPolicy, want BranchPolicy) bool {
	for _, h := range have {
		if h.Name == want.Name && h.Type == want.Type {
			return true
		}
	}
	return false
}

// setupHost is the host stage.
func setupHost(ctx context.Context, adapter hosting.Adapter, s hosting.Site, req SetupRequest) (out HostOutcome, declined, refused bool) {
	out = HostOutcome{Provider: adapter.Name(), Name: s.Name, Domain: s.Domain}
	src := req.Credentials
	if src == nil {
		src = credential.UserMachine()
	}
	token, err := src.Resolve(adapter.CredentialName())
	if errors.Is(err, credential.ErrNotSet) {
		out.Status = HostNoCredential
		out.Detail = "no " + adapter.CredentialName() + " credential on this machine, so the host was not contacted"
		return out, false, false
	}
	if err != nil {
		out.Status = HostRefused
		out.Detail = err.Error()
		return out, false, true
	}
	p := adapter.Connect(token)
	refuse := func(err error) (HostOutcome, bool, bool) {
		out.Status = HostRefused
		out.Detail = strings.ReplaceAll(err.Error(), token, "[credential]")
		return out, false, true
	}
	st, err := p.Inspect(ctx, s)
	if err != nil {
		return refuse(err)
	}
	if !st.Exists {
		out.Changes = append(out.Changes, "create "+adapter.Name()+" host "+s.Name)
	}
	if !st.Routed {
		out.Changes = append(out.Changes, "route "+s.Domain+" to "+s.Name)
	}
	if len(out.Changes) > 0 {
		if req.Asker == nil || !req.Asker.Confirm("Change the host on "+adapter.Name()+"? ("+strings.Join(out.Changes, "; ")+")") {
			out.Status = HostDeclined
			return out, true, false
		}
		if !st.Exists {
			if err := p.Create(ctx, s); err != nil {
				return refuse(err)
			}
		}
		if !st.Routed {
			if err := p.Route(ctx, s); err != nil {
				return refuse(err)
			}
		}
	}
	addr, err := p.Address(ctx, s)
	if err != nil {
		return refuse(err)
	}
	out.Address = addr
	out.Status = HostCurrent
	if len(out.Changes) > 0 {
		out.Status = HostWritten
	}
	return out, false, false
}

// secretSteps names the deploy environment's secrets that are not set, as the
// exact commands that set them. gh reads the value from the terminal, so it
// never passes through abcd or a shell history.
func secretSteps(ctx context.Context, forge Forge, envOK bool, adapter hosting.Adapter, res *SetupResult) []string {
	want := adapter.Secrets()
	missing := want
	repo := res.Repo
	if forge != nil && envOK {
		names, err := forge.SecretNames(ctx, EnvDeploy)
		if err != nil {
			res.Notes = append(res.Notes, "the "+EnvDeploy+" environment's secret names could not be read, so every one is listed: "+err.Error())
		} else {
			have := map[string]bool{}
			for _, n := range names {
				have[n] = true
			}
			missing = nil
			for _, w := range want {
				if !have[w] {
					missing = append(missing, w)
				}
			}
		}
	}
	sort.Strings(missing)
	var steps []string
	for _, name := range missing {
		cmd := "gh secret set " + name + " --env " + EnvDeploy
		if repo != "" {
			cmd += " --repo " + repo
		}
		steps = append(steps, "set the deploy secret: `"+cmd+"`")
	}
	return steps
}
