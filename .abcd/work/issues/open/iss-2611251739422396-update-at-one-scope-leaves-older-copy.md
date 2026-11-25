---
schema_version: 1
id: "iss-2611251739422396"
slug: "update-at-one-scope-leaves-older-copy"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "manual build of 25 November, logged as a test of what abcd relies on"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/ahoy"
remedy: "Have `abcd ahoy` and the session-start notice name every installed plugin copy with its scope and version, say which one this session runs, and when another copy is newer print the update command with the scope it needs. Grounds: the host lists both copies and the running one is the plugin root the hook is handed, so nothing is inferred."
refines: [iss-2610241228359748]
related_intents: [itd-2611251834407931]
---

After the plugin was updated to 0.17.1 at project scope, the host went on loading the user-scope copy, 0.16.0, in every session started in that project, and `abcd ahoy` says only that the plugin root is resolved. On 2026-11-25 the person believed the update was in force. The session ran hooks, agents and command pages five releases old: the rule for a person leaving was missing, and the prompt hook printed 42,791 bytes where 0.17.1 prints 7,936 for the same prompt, so the host cut the rules to a preview. Refines iss-2610241228359748, which waits for a ruling on precedence: this part needs none.

2026-11-25: The first half of the ideate verdict dcn-2611251807556458 is this: the copies compared from disk, the one that runs named, and the one step that mends each. It waits on no ruling.

2026-11-25, tested on host 2.1.332 in a scratch configuration: One plugin installed at user scope (1.0.0) and then at project scope (1.1.0), and a session in the project loaded 1.0.0. After an update at user scope it loaded 1.1.0.
