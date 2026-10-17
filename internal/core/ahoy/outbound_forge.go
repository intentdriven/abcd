package ahoy

import (
	"os"

	"github.com/intentdriven/abcd/internal/core/guard"
)

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
// switch and left the other key absent, which only the two retired flags could
// produce (iss-2610152024327989). The switch is one choice, so a half is the
// choice made and half carried out: the gap is required, and install completes
// it by recording the other half — and, for a repository that recorded only the
// forge-text half, writing the commit-msg hook where nothing occupies its path —
// under the settings approval. It never writes abcd.binary: that is the explicit
// opt-in's alone.
//
// A key the repository set itself, to false or to anything but true, is the
// person's choice and never a half to complete: it raises outboundHalfChosenGapID,
// a note nothing resolves, and install changes neither key.
func outboundHalfGap(cwd string) []Gap {
	hook, forge := outboundOptedIn(cwd), guard.ForgeTextOptedIn(cwd)
	if hook == forge {
		return nil
	}
	cfg, err := readConfig(cwd)
	if err != nil {
		return nil
	}
	section := subMap(cfg, "outbound")
	on, off, offKey := "the commit-message check (outbound.hook)", "the forge-text check (outbound.forge)", "forge"
	if forge {
		on, off, offKey = off, on, "hook"
	}
	if _, set := section[offKey]; set {
		return []Gap{{
			ID: outboundHalfChosenGapID, Category: PluginOwned, Scope: "repo",
			Title:      "the outbound switch is half on by this repository's choice: " + on + " is on and " + off + " is set to something other than true",
			Detail:     "One opt-in, `abcd ahoy install --outbound`, turns on both outbound checks. This repository set outbound." + offKey + " itself, so abcd leaves it as it is and " + off + " stays off.",
			FixHint:    "nothing to do if that is the choice; `abcd ahoy install --outbound` turns both on, and setting both keys to false turns the switch off.",
			Required:   false,
			Resolvable: false,
		}}
	}
	completes := "records outbound.forge: true in .abcd/config.json"
	if forge {
		completes = "records outbound.hook: true in .abcd/config.json and writes the commit-msg hook " + OutboundHookRelPath
		if root, err := os.OpenRoot(cwd); err == nil {
			state, runsCheck := classifyOutboundHook(root, OutboundHookRelPath)
			root.Close()
			if left := outboundHookLeft(state, runsCheck); left != "" {
				completes = "records outbound.hook: true in .abcd/config.json and does not write the commit-msg hook: " + left
			} else if state == HookInstalled {
				completes = "records outbound.hook: true in .abcd/config.json; the commit-msg hook " + OutboundHookRelPath + " is already abcd's"
			}
		}
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
