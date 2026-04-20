package session

import (
	"fmt"
	"math/rand/v2"
	"net"
	"sync"
)

const (
	DefaultPortMin = 49152
	DefaultPortMax = 65535
)

// PortAllocator manages dynamic port allocation for sessions.
type PortAllocator struct {
	mu       sync.Mutex
	reserved map[int]SessionID
	excluded map[int]bool // ports used by static project configs
	portMin  int
	portMax  int
}

// NewPortAllocator creates a port allocator with the given range.
// If min/max are zero, defaults are used.
func NewPortAllocator(min, max int) *PortAllocator {
	if min <= 0 {
		min = DefaultPortMin
	}
	if max <= 0 {
		max = DefaultPortMax
	}
	return &PortAllocator{
		reserved: make(map[int]SessionID),
		excluded: make(map[int]bool),
		portMin:  min,
		portMax:  max,
	}
}

// ExcludePorts marks ports as unavailable for allocation. Use this to
// exclude ports from static project proxy URLs.
func (a *PortAllocator) ExcludePorts(ports []int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, p := range ports {
		a.excluded[p] = true
	}
}

// Allocate finds n free TCP ports and reserves them for the given session.
func (a *PortAllocator) Allocate(id SessionID, n int) ([]int, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	rangeSize := a.portMax - a.portMin + 1
	if rangeSize < n {
		return nil, fmt.Errorf("port range [%d, %d] too small for %d ports", a.portMin, a.portMax, n)
	}

	offset := rand.IntN(rangeSize)
	var ports []int

	for i := range rangeSize {
		if len(ports) == n {
			break
		}
		port := a.portMin + (offset+i)%rangeSize

		if a.reserved[port] != (SessionID{}) {
			continue
		}
		if a.excluded[port] {
			continue
		}
		if !isPortFree(port) {
			continue
		}

		ports = append(ports, port)
	}

	if len(ports) < n {
		return nil, fmt.Errorf("could not allocate %d free ports in range [%d, %d]", n, a.portMin, a.portMax)
	}

	for _, p := range ports {
		a.reserved[p] = id
	}
	return ports, nil
}

// AllocateSpecific attempts to reserve the given ports for a session.
// Returns an error if any port is unavailable. Used for restoring sessions.
func (a *PortAllocator) AllocateSpecific(id SessionID, ports []int) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for _, p := range ports {
		if p < a.portMin || p > a.portMax {
			return fmt.Errorf("port %d outside allowed range [%d, %d]", p, a.portMin, a.portMax)
		}
		if owner := a.reserved[p]; owner != (SessionID{}) {
			return fmt.Errorf("port %d already reserved by %s", p, owner)
		}
		if a.excluded[p] {
			return fmt.Errorf("port %d is excluded (used by static config)", p)
		}
	}

	for _, p := range ports {
		a.reserved[p] = id
	}
	return nil
}

// Release returns all ports reserved by the given session.
func (a *PortAllocator) Release(id SessionID) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for port, owner := range a.reserved {
		if owner == id {
			delete(a.reserved, port)
		}
	}
}

// Reserved returns a snapshot of all reserved ports.
func (a *PortAllocator) Reserved() map[int]SessionID {
	a.mu.Lock()
	defer a.mu.Unlock()

	out := make(map[int]SessionID, len(a.reserved))
	for k, v := range a.reserved {
		out[k] = v
	}
	return out
}

func isPortFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
