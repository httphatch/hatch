// Package health provides health checking and status reporting
// for upstream services proxied by Hatch.
package health

import "time"

const (
	// DefaultInterval is the time between health check cycles.
	DefaultInterval = 10 * time.Second

	// DefaultTimeout is the TCP dial timeout for each check.
	DefaultTimeout = 2 * time.Second
)

// Status represents the health state of a service.
type Status int

const (
	StatusUnknown   Status = iota // Not yet checked
	StatusHealthy                 // TCP dial succeeded
	StatusUnhealthy               // TCP dial failed
)

// String returns a human-readable representation of the status.
func (s Status) String() string {
	switch s {
	case StatusHealthy:
		return "healthy"
	case StatusUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// ServiceKey uniquely identifies a service within a project.
type ServiceKey struct {
	Project string
	Service string
}

// ServiceStatus holds the current health state of a service.
type ServiceStatus struct {
	Status    Status
	Addr      string    // host:port being checked
	Since     time.Time // when current status was first observed
	LastCheck time.Time // when the last check completed
}
