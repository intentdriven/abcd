package tools

import (
	"os"
	"runtime"
	"strings"
)

// Explanation is what a verb says when it finds a tool missing (criterion 1):
// the tool, whether it is optional or required for this capability, what works
// without it, what it would do, and the exact install step, all from the
// registry. It is structured so each front door renders it its own way (the
// plugin page reads it from a verb's --json); Lines is the plain-text rendering
// every front door shares.
type Explanation struct {
	Tool           string      `json:"tool"`
	Known          bool        `json:"known"`
	Capability     Capability  `json:"capability"`
	CapabilityName string      `json:"capability_name,omitempty"`
	Requirement    Requirement `json:"requirement,omitempty"`
	What           string      `json:"what,omitempty"`
	Homepage       string      `json:"homepage,omitempty"`
	Does           string      `json:"does,omitempty"`
	WithoutIt      string      `json:"without_it,omitempty"`
	NativeDefault  string      `json:"native_default,omitempty"`
	OnDecline      string      `json:"on_decline,omitempty"`
	// Download is the checked download Install would make on this platform:
	// the pinned version, its full URL and sha256, where the program goes and
	// how far it is checked. nil when the registry names none (an unknown
	// tool, or a platform abcd's release holds no download for).
	Download *Download `json:"download,omitempty"`
	// Plan is what Install found on this machine before it asked: the copy
	// already installed and whether it is compatible, and which copy will
	// run. Set only on the explanation Install hands its Confirm.
	Plan    *Plan    `json:"plan,omitempty"`
	Effects string   `json:"effects,omitempty"`
	Verify  []string `json:"verify,omitempty"`
	// RegistryGap is set for a tool the registry does not know: the gap is
	// abcd's own, and this names the capture that records it.
	RegistryGap string `json:"registry_gap,omitempty"`
}

// Explain renders the explanation for name as capability uses it, on the
// running platform, into the running account's bin folder.
func Explain(name string, capability Capability) Explanation {
	home, _ := os.UserHomeDir()
	return explainFor(name, capability, runtime.GOOS, runtime.GOARCH, userBinDir(os.Getenv, home), home)
}

// explanation is Explain for this installer's platform, environment and home.
func (in *Installer) explanation(name string, capability Capability) Explanation {
	return explainFor(name, capability, in.GOOS, in.GOARCH, userBinDir(in.getenv, in.Home), in.Home)
}

// explainFor is Explain with the platform, the bin folder and the home named,
// so a test can reach every platform's download from any host.
func explainFor(name string, capability Capability, goos, goarch, binDir, home string) Explanation {
	tool, ok := registry[name]
	if !ok {
		return unknown(name, capability)
	}
	e := Explanation{
		Tool:       name,
		Known:      true,
		Capability: capability,
		What:       tool.What,
		Homepage:   tool.Homepage,
		Effects:    tool.Effects,
		Verify:     append([]string(nil), tool.Verify...),
	}
	if use, ok := tool.Uses[capability]; ok {
		e.CapabilityName = use.Capability
		e.Requirement = use.Requirement
		e.Does = use.Does
		e.WithoutIt = use.WithoutIt
		e.NativeDefault = use.NativeDefault
		e.OnDecline = use.OnDecline
	} else {
		// A known tool asked about for a capability its entry does not name is
		// a registry gap too, but a narrower one: what the tool is and how it is
		// installed are still known, so only the use is generic.
		e.CapabilityName = string(capability)
		e.Requirement = Optional
		e.Does = "abcd's registry does not say what this capability uses it for"
		e.WithoutIt = "abcd's registry does not say"
		e.OnDecline = "nothing was installed"
		e.RegistryGap = gapCapture(name, capability)
	}
	if asset, ok := tool.Release.assetFor(goos, goarch); ok && binDir != "" {
		dest := binDir + string(os.PathSeparator) + name
		e.Download = &Download{
			Version: tool.Release.Version, URL: asset.URL, SHA256: asset.SHA256, Member: asset.Member,
			Dest: displayIn(home, dest), Checks: tool.Release.Checks(), ChecksumOnly: tool.Release.Signer == nil, dest: dest,
		}
	}
	return e
}

// unknown is the generic explanation for a tool the registry does not know
// (criterion 3). It carries no download: abcd installs only what it can
// explain, and a download it does not hold is never composed from the tool's
// name.
func unknown(name string, capability Capability) Explanation {
	return Explanation{
		Tool:           name,
		Capability:     capability,
		CapabilityName: string(capability),
		What: "abcd's tool registry has no entry for it, so abcd cannot say what it is, why this capability " +
			"wants it, or how to install it safely",
		Does:        "unknown to abcd",
		WithoutIt:   "unknown to abcd",
		OnDecline:   "nothing was installed",
		RegistryGap: gapCapture(name, capability),
	}
}

// gapCapture is the capture that records a registry gap. The gap is abcd's own
// (its curated registry lacks an entry), so it is recorded in abcd's ledger by
// whoever meets it, and never written into the repository the verb ran in.
func gapCapture(name string, capability Capability) string {
	return `abcd's tool registry has no entry for ` + name + ` as ` + string(capability) +
		` uses it; record the gap in abcd's own ledger with: abcd capture "the tool registry has no entry for ` +
		name + ` (` + string(capability) + `)" --category future-work-seed`
}

// StepText is the install as a person reads it in the install question, or
// the reason there is none. It never names a step that needs admin rights.
func (e Explanation) StepText() string {
	if e.Download == nil {
		if !e.Known {
			return "none known to abcd (look the tool up before installing it)"
		}
		return "none known to abcd for this platform (see " + e.Homepage + ")"
	}
	d := e.Download
	return "download " + e.Tool + " " + d.Version + " into " + d.Dest + ", " + d.Checks
}

// ManualText is the same install for a person to make by hand: the exact
// URL, the sha256 to check it against, and where the program goes.
func (e Explanation) ManualText() string {
	if e.Download == nil {
		return e.StepText()
	}
	d := e.Download
	return "download " + d.URL + ", check its sha256 is " + d.SHA256 + ", and put " + d.Member +
		" from it at " + d.Dest + " (no admin rights needed)"
}

// Lines is the plain-text explanation, one line per part, the first naming the
// tool and whether it is optional or required.
func (e Explanation) Lines() []string {
	if !e.Known {
		return []string{
			e.Tool + " — " + e.What,
			"  install: " + e.StepText(),
			"  registry gap: " + e.RegistryGap,
		}
	}
	lines := []string{
		e.Tool + " — " + string(e.Requirement) + " for " + e.CapabilityName,
		"  what it is: " + e.What + " (" + e.Homepage + ")",
		"  what abcd uses it for: " + e.Does,
		"  without it: " + e.WithoutIt,
	}
	if e.Download != nil {
		lines = append(lines,
			"  install: "+e.StepText(),
			"  from: "+e.Download.URL+" (no admin rights; nothing is written until the download passes its check)")
	} else {
		lines = append(lines, "  install: "+e.StepText())
	}
	if e.Plan != nil && e.Plan.Existing != nil {
		lines = append(lines, "  already installed: "+e.Plan.Existing.Path+" — "+e.Plan.Existing.Why+"; abcd leaves that copy as it is")
	}
	if e.Plan != nil && e.Plan.Runs != "" {
		lines = append(lines, "  which copy runs: "+e.Plan.Runs)
	}
	lines = append(lines, "  what the install does: "+e.Effects)
	if e.RegistryGap != "" {
		lines = append(lines, "  registry gap: "+e.RegistryGap)
	}
	return lines
}

// MissingError is a verb's refusal for a missing tool with the registry's
// explanation appended: the refusal stands exactly as it was (its cause stays
// reachable through errors.Is), and only its message grows to say what the
// tool is, whether this capability needs it, and the exact install step.
type MissingError struct {
	Cause       error
	Explanation Explanation
}

// Missing wraps cause with the explanation for name as capability uses it.
func Missing(cause error, name string, capability Capability) error {
	return &MissingError{Cause: cause, Explanation: Explain(name, capability)}
}

func (m *MissingError) Error() string {
	return m.Cause.Error() + "\n" + strings.Join(m.Explanation.Lines(), "\n")
}

func (m *MissingError) Unwrap() error { return m.Cause }
