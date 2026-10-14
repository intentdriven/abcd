package ahoy

import "github.com/intentdriven/abcd/internal/core/guard"

// The forge-text opt-in (iss-2610142210511831). `ahoy install
// --outbound-forge` records `outbound.forge: true` in config.json, and the
// shell guard reads that key (guard.ForgeTextOptedIn, its one reader) to judge
// the text a posting gh command hands the forge. Nothing else is written: no
// hook, no binary path, because the guard is the abcd already running.

// outboundForgeWouldChange reports work the gap set cannot see: an explicit
// --outbound-forge against a repository that has not recorded the opt-in.
func outboundForgeWouldChange(cwd string, opts InstallOptions) bool {
	return opts.OutboundForge && !guard.ForgeTextOptedIn(cwd)
}

// stepOutboundForge records the opt-in for a repository that asked for it,
// under the config lock, refusing a file it cannot parse and surfacing a
// failure as a change-note, exactly as recordOutboundOptIn does. The explicit
// flag is its own approval.
func (a *applyCtx) stepOutboundForge() {
	if !a.outboundForge {
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
