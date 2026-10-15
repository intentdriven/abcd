package ahoy

import "github.com/intentdriven/abcd/internal/core/guard"

// The forge-text half of the outbound switch (iss-2610142210511831). `ahoy
// install --outbound` records `outbound.forge: true` in config.json beside
// `outbound.hook`, and the shell guard reads that key (guard.ForgeTextOptedIn,
// its one reader) to judge the text a posting gh command hands the forge. This
// half writes no hook and names no binary, because the guard is the abcd
// already running.

// stepOutboundForge records the forge-text half for a repository that asked for
// the switch, or whose recorded half this run completes, under the config lock,
// refusing a file it cannot parse and surfacing a failure as a change-note,
// exactly as recordOutboundOptIn does. The explicit flag is its own approval;
// a completion is held to the approval its gap asks for.
func (a *applyCtx) stepOutboundForge() {
	if !a.outbound && !a.completingOutbound() {
		return
	}
	wrote := false
	err := withConfigLock(a.cwd, func() error {
		cfgMap, err := readConfig(a.cwd)
		if err != nil {
			return err
		}
		if cfgMap == nil {
			cfgMap = map[string]any{}
		}
		if v, ok := boolVal(subMap(cfgMap, "outbound"), "forge"); ok && v {
			return nil
		}
		setSub(cfgMap, "outbound", "forge", true)
		if err := writeConfig(a.cwd, cfgMap); err != nil {
			return err
		}
		wrote = true
		return nil
	})
	if err != nil {
		a.changes = append(a.changes, receiptPath(a.cwd, "forge-text opt-in not persisted ("+err.Error()+")"))
		return
	}
	if wrote {
		a.note(writeOutboundForge, configPath(a.cwd))
	}
}

// outboundHalfGap reports a repository that recorded one half of the outbound
// switch and not the other, which only the two retired flags could produce
// (iss-2610152024327989). The switch is one choice, so a half is the choice
// made and half carried out: the gap is required, and install completes it by
// recording the other half — and, for a repository that recorded only the
// forge-text half, writing the commit-msg hook — under the settings approval.
// It never writes abcd.binary: that is the explicit opt-in's alone.
func outboundHalfGap(cwd string) []Gap {
	hook, forge := outboundOptedIn(cwd), guard.ForgeTextOptedIn(cwd)
	if hook == forge {
		return nil
	}
	on, off := "the commit-message check (outbound.hook)", "the forge-text check (outbound.forge)"
	completes := "records outbound.forge: true in .abcd/config.json"
	if forge {
		on, off = off, on
		completes = "records outbound.hook: true in .abcd/config.json and writes the commit-msg hook " + OutboundHookRelPath
	}
	return []Gap{{
		ID: outboundHalfGapID, Category: ConfigChange, Scope: "repo",
		Title:    "the outbound switch is half on: " + on + " is recorded and " + off + " is not",
		Detail:   "One opt-in, `abcd ahoy install --outbound`, turns on both outbound checks. This repository recorded one of them under a flag that has since been retired, so " + off + " is still off.",
		FixHint:  "abcd ahoy install completes it: it " + completes + ".",
		Required: true, Resolvable: true,
	}}
}

// completingOutbound reports whether this run completes a half-on outbound
// switch: detection raised the gap and the settings it changes were approved.
func (a *applyCtx) completingOutbound() bool {
	return a.has(outboundHalfGapID) && a.approved[ConfigChange]
}
