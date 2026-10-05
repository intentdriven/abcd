package ahoy

// STUB: replaced by lane detect. statusLineEntryTrust is the trust check the
// detect lane adds in statusline_trust.go (owned by the user, binary and its
// folder not writable by others, not inside the working tree, recorded in
// path-entry, not under the versioned plugin cache). This placeholder trusts
// every path so the wiring can be built against the signature; the integrator
// deletes this file at merge.
func statusLineEntryTrust(path string) (ok bool, reason string) {
	_ = path
	return true, ""
}
