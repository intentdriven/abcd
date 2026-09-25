---
schema_version: 1
id: "iss-2609190338409222"
slug: "the-lan-hostname-scan-flags-a-possessive-persona-host-its-ow"
severity: "nitpick"
category: "inconsistency"
source: "agent-observation"
found_during: "Gropius autonomous sweep, session gropiusllm-66, relayed to abcd-17 on 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/adapter/scanner/network.go"
resolution: "Took the record's second option: the net_lan_hostname and net_device_hostname suggestions now name the exact accepted shape, <persona>-<noun> in lower case with no possessive and no capitals; the exemption itself is unchanged, since the capitalised possessive is macOS's default name for a real person's machine."
impact: fix
resolved_by:
  commit: "b868499ac31ea4a95ff40611f301cca835a337ba"
---

The LAN-hostname scan flags a possessive persona host its own suggestion calls acceptable. net_lan_hostname flags "[redacted-hostname]" while its Suggestion reads "replace with a reserved name (example.com, host.test) or a persona-derived fixture host": personaDerivedHost takes the label before the first hyphen and looks it up in personaNames, so "alice-mac.local" passes and "[redacted-hostname]" (the possessive, which is how macOS names a machine by default) does not. The author reads the suggestion, sees a persona-derived host, and cannot tell why theirs was refused. Relayed from the Gropius session gropiusllm-66 on 2026-09-19 at v0.9.0, reproduced by reading internal/adapter/scanner/network.go. Wanted, either: the skip admits a persona name in possessive form (alices, alice's), or the suggestion names the exact shape it accepts (<persona>-<noun>, no possessive).

## Grounds

- pursued: an author refused on a possessive persona host can read the accepted shape in the finding; a suggestion that still offers a shape the skip refuses would show it wrong
