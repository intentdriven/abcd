---
schema_version: 1
id: "iss-2608261133210491"
slug: "memory-storelock-wrong-ifmt-mask"
severity: "nitpick"
category: "tech-debt"
source: "agent-finding"
found_during: "bughunt-round-8"
found_at: "internal/core/memory/writer.go:70"
resolution: "The store-lock fstat guard compares the file type under S_IFMT (lockModeIsRegular), so socket and symlink modes are refused; the flock consolidation itself stays with iss-129."
impact: internal
resolved_by:
  commit: "cdf5a434"
---

the memory store-lock guard tests mode AND S_IFREG nonzero instead of masking with S_IFMT, so its regular-file assertion also accepts symlink and socket modes; dead defence shielded by O_NOFOLLOW, fold into the iss-129 flock consolidation

## Grounds

- pursued: the guard admits only S_IFREG under the S_IFMT mask; a socket or symlink mode accepted by lockModeIsRegular would show it wrong
