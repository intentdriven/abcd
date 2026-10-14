---
schema_version: 1
id: "iss-2610111855583968"
slug: "guard-cd-three-operands-bash32"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "security fix lane G round 5, 2026-10-11"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/script.go"
remedy: "Take the first operand as bash's candidate for any number of operands, which only adds blocks; prove with a Check test over a planted a/s.sh"
resolution: "bash's reading of a cd or pushd now takes the first operand as entered whatever number of operands follows, as bash 3.2 enters it, so the scripts after cd a b c are read in a (TestCdManyOperandsBashEntersTheFirst); only blocks are added."
impact: fix
---

The guard's cd reader gives bash's reading no directory for three or more operands, but macOS /bin/bash 3.2.57 enters the first operand of `cd a b c` and exits 0 (verified 2026-10-11 with cd and pwd only), so a following `bash s.sh` runs the script in `a` while the guard leaves it unread. Same class as iss-2610111809142536; round 5 added the first operand for two operands only.
