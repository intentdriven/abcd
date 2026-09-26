package tools

import (
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
	StepManager    string      `json:"step_manager,omitempty"`
	// Step is the exact argv Install would run on this platform; empty when
	// the registry names none (an unknown tool, or a platform without a step).
	Step    []string `json:"step,omitempty"`
	Effects string   `json:"effects,omitempty"`
	Verify  []string `json:"verify,omitempty"`
	// RegistryGap is set for a tool the registry does not know: the gap is
	// abcd's own, and this names the capture that records it.
	RegistryGap string `json:"registry_gap,omitempty"`
}

// Explain renders the explanation for name as capability uses it, on the
// running platform.
func Explain(name string, capability Capability) Explanation {
	return explainFor(name, capability, runtime.GOOS)
}

// explainFor is Explain with the platform named, so a test can reach every
// platform's step from any host.
func explainFor(name string, capability Capability, goos string) Explanation {
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
	if step, ok := tool.Install[goos]; ok {
		e.StepManager = step.Manager
		e.Step = append([]string(nil), step.Argv...)
	}
	return e
}

// unknown is the generic explanation for a tool the registry does not know
// (criterion 3). It carries no step: abcd installs only what it can explain,
// and a step it does not hold is never composed from the tool's name.
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

// StepText is the install step as a person would type it, or the reason there
// is none.
func (e Explanation) StepText() string {
	if len(e.Step) == 0 {
		if !e.Known {
			return "none known to abcd (look the tool up before installing it)"
		}
		return "none known to abcd for this platform (see " + e.Homepage + ")"
	}
	return strings.Join(e.Step, " ")
}

// Lines is the plain-text explanation, one line per part, the first naming the
// tool and whether it is optional or required.
func (e Explanation) Lines() []string {
	if !e.Known {
		return []string{
			e.Tool + " — " + e.What,
			"  install step: " + e.StepText(),
			"  registry gap: " + e.RegistryGap,
		}
	}
	lines := []string{
		e.Tool + " — " + string(e.Requirement) + " for " + e.CapabilityName,
		"  what it is: " + e.What + " (" + e.Homepage + ")",
		"  what abcd uses it for: " + e.Does,
		"  without it: " + e.WithoutIt,
	}
	if len(e.Step) > 0 {
		lines = append(lines, "  install step ("+e.StepManager+"): "+e.StepText())
	} else {
		lines = append(lines, "  install step: "+e.StepText())
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
