---
schema_version: 1
id: "iss-2610290709017008"
slug: "a-session-start-says-nothing-when-the"
severity: "minor"
category: "ux"
source: "agent-finding"
found_during: "checking the SessionEnd cancellation issue's remedy against what merged"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/skew.go"
refines: ["iss-2610281259462058"]
remedy: "Rule whether a session start may ask for the newest release (today only abcd update --check does) and, if so, add a start-of-session notice naming the installed and newest versions and the update step, within the bootstrap's existing release lookup where it already runs."
---

A session start says nothing when the installed plugin is older than the newest published release, so a fix that has shipped (such as the SessionEnd staging change) never reaches a person whose plugin did not update, and nothing tells them to update. The start-of-session notice that exists compares the plugin surface with the binary serving it, and only reports a plugin newer than its binary; asking for the newest release is left to abcd update --check alone, which a session never runs.

## Evidence 2026-11-21

On 2026-11-20, two days after v0.17.0 was cut, sessions in abcd's own checkout still ran the v0.16.0 plugin, with nothing at their start saying that a newer release existed. One of them ran `/abcd:ideate` from the v0.16.0 page and repeated its adversarial leg, and its rules hook overran the host's cap on each of four prompts over 20 and 21 November although the fix shipped in v0.17.0. The detail is under the same date in iss-2610241228359748.

2026-11-25: The ruling asked for here is weighed in an ideate verdict (dcn-2611251807556458): a check on a standing yes, at most daily, metadata alone, to the person alone, is the shape that survives; accepted on 2026-11-25 as adr-2611251808065076.

## Evidence 2026-11-26: a session behind the machine, not only behind the release

The same silence covers a case closer to home than the newest published release: a running session whose plugin root is older than the newest root already installed on the machine. A session started on 0.16.0 kept running hooks from the 0.16.0 root after 0.17.1 was installed on 25 November. On 26 November its Stop hook still cut notes at 256 bytes, a bound b46125db7 (in v0.17.1) had already raised to 512, while the owned PATH copy beside it was at 0.17.1. This is the host's intended behaviour: a session keeps the plugin root it loaded until `/reload-plugins`, and an updated-away root stays on disk for 14 days so such a session keeps running. So the remedy above has a cheaper half that needs no network: compare the session's root with the newest root, or the owned PATH copy's release, already on disk, and say so naming `/reload-plugins`. It fits implicit checks being disk-only. This notice, together with the precedence ruling iss-2610241228359748 asks for, is also what running one abcd per account would rest on; the detail is under "One abcd per account" in that record.
