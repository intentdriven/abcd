package machineload

// LoadConfigKeys are every key the machine configuration's `load` object takes.
// Every reader of that object claims the namespace with all of them
// (layered.Stack.Claim), so a key under `load` that nothing reads, a misspelt
// one among them, is named, and none of these is. Most are the load gate's
// (internal/core/implement); go_cache_limit_gib is the Go build cache's trim
// (internal/core/ahoy/worktree), which both import this package to share one list.
var LoadConfigKeys = []string{"preflights", "narrow_runs", "memory_floor_macos", "memory_floor_linux", "go_cache_limit_gib"}
