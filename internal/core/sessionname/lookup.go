package sessionname

import "github.com/intentdriven/abcd/internal/core/machineload"

// lookup.go reads the register on behalf of the launch ledger
// (itd-2611031342011261), which names a launch's session by id and persona
// name. Both reads are peek's: no lock, nothing written, and a machine with no
// register, or one that cannot be read, simply names nothing, because a name
// is a courtesy on an entry and never a reason to refuse a launch.

// NameOf is the persona name the register holds for sessionID, "" when it
// holds none.
func NameOf(m Machine, sessionID string) string {
	dir, reg, err := m.peek()
	if err != nil || dir == nil {
		return ""
	}
	defer dir.Close()
	for i := len(reg.Live) - 1; i >= 0; i-- {
		if reg.Live[i].SessionID == sessionID {
			return reg.Live[i].Name
		}
	}
	return ""
}

// SessionAbove is the session whose host process m.From descends from, and its
// name; "" and "" when no ancestor is a session's host. It walks every ancestor
// rather than stopping where FindHost does: a launching process can sit under
// a make or a terminal multiplexer, which is no host of a session and does not
// hide one above it.
func SessionAbove(m Machine) (sessionID, name string) {
	dir, reg, err := m.peek()
	if err != nil || dir == nil || m.ReadTable == nil {
		return "", ""
	}
	defer dir.Close()
	tab, err := m.ReadTable()
	if err != nil {
		return "", ""
	}
	byPID := make(map[int]machineload.Proc, len(tab.Procs))
	for _, p := range tab.Procs {
		byPID[p.PID] = p
	}
	pid := m.From
	for range maxWalk {
		p, ok := byPID[pid]
		if !ok || pid <= 1 {
			return "", ""
		}
		if i := reg.onHost(tab.host(p)); i >= 0 {
			return reg.Live[i].SessionID, reg.Live[i].Name
		}
		pid = p.PPID
	}
	return "", ""
}
