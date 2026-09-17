---
schema_version: 1
id: "iss-2609120446083912"
slug: "identity-redaction-rewrites-by-string-rather-than-by-byte-sp"
severity: "major"
category: "bug"
source: "agent-finding"
found_during: "fixing the reverse-DNS redaction corruption"
origin: researcher-authored
production_mode: hand-written
deferred_after: "v0.8.0"
deferral_reason: "The detection half of the corruption this sits behind is fixed in this cut. This is the rewrite half, and fixing it means making identity masking span-based rather than whole-string, which reverses a recorded design choice in the single write-time sanitiser that history, memory, capture, ideate, intent, decide and launch all write through, and reverses it in the fail-open direction. That is a decision about the sanitiser's contract rather than a patch to one detector, and the same mechanism carries the git-identity and real-name kinds, so the blast radius is every record abcd writes."
found_at: "internal/adapter/scanner/redact.go"
resolution: "Identity masking rewrites exactly the byte spans the detector flagged; a cleared lookalike on the same line survives byte-for-byte"
impact: fix
---

Identity redaction rewrites by string rather than by byte span, so a line carrying both a real leak and a lookalike has the lookalike rewritten too. The detector was taught to leave a dotted namespace component alone, which fixed the corruption of reverse-DNS identifiers. The rewrite did not learn it. Masking for identity kinds replaces every occurrence of the matched string rather than the span the detector found, which is recorded as deliberate because an identity placeholder changes length. So where one line contains a genuine bare mention of the account name and also an identifier that merely begins with it, the first is masked correctly and the second is mangled anyway: a line reading that the crash is in a particular reverse-DNS bundle comes out with the bundle's first component replaced, even though detection refused it. The same mechanism turns an ordinary word containing the account name into a masked fragment mid-word when a real mention appears on the same line, though detection alone leaves that word intact. A whole-string replace cannot express one occurrence masked and another left, so the fix is span-based replacement for identity kinds. That is safe on offsets, because the sealing pass is length-preserving, but it reverses a recorded design choice in the single shared write-time sanitiser that history, memory, capture, ideate, intent, decide and launch all write through, and it reverses it in the fail-open direction. That is a human design decision rather than a patch, which is why it is recorded rather than done. The git-identity and real-name kinds share the mechanism and would corrupt the same way where a name collides with a namespace label.

## Grounds

- pursued: the detector's refusals (a dotted-namespace component, a mid-word collision, an occurrence inside a URL) are only real if the rewrite honours them, so masking by recorded span instead of by whole string stops the corruption; what would show it wrong is a genuine mention the detector does not flag that the whole-string rewrite used to catch — that residue is named in redact.go and pinned by TestSpanMaskingFailsOpenOnClearedLookalikes
