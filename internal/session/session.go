package session

import (
	"fmt"
	"time"
)

// SessionState represents the lifecycle of a session.
type SessionState int

const (
	StateStarting SessionState = iota
	StateRunning
	StateStopping
	StateStopped
)

func (s SessionState) String() string {
	switch s {
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateStopping:
		return "stopping"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// SessionID uniquely identifies a session within a project.
type SessionID struct {
	Project string
	Name    string
}

func (id SessionID) String() string {
	return id.Project + "/" + id.Name
}

// QualifiedProject returns the project name used to register session
// processes and routes. The tilde separator cannot appear in normal
// project names or hostname labels, so it avoids collisions.
func (id SessionID) QualifiedProject() string {
	return id.Project + "~" + id.Name
}

// Session holds the full state of a single session instance.
type Session struct {
	ID         SessionID
	State      SessionState
	BaseDomain string            // the parent project's base domain
	Ports      map[string]int    // service name -> allocated port
	Domains    map[string]string // service name -> FQDN
	CreatedAt  time.Time
	LastActive time.Time
	TTL        time.Duration
}

// Domain returns the subdomain for a session service. For services with an
// existing subdomain in the base config, the format is "session--subdomain.domain".
// For services without a subdomain, the format is "session.domain".
func Domain(sessionName, baseDomain, serviceSubdomain string) string {
	if serviceSubdomain != "" {
		return fmt.Sprintf("%s--%s.%s", sessionName, serviceSubdomain, baseDomain)
	}
	return fmt.Sprintf("%s.%s", sessionName, baseDomain)
}
