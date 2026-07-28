package morpheus

import "fmt"

// KnownPorts is the originating project port registry — prevents port conflicts across services.
var KnownPorts = map[string]int{
	"internal-app":                    3000,
	"payment-service":              3001,
	"notification-service-api":     8080,
	"notification-service-worker":  8081,
	"generation-service-api":       8082,
	"generation-service-worker":    8083,
	"webhook-service-api":          8084,
	"webhook-service-worker":       8085,
}

// SuggestNextApiPort finds the next available even port >= 8086.
// API ports are even; worker ports are odd (api + 1).
func SuggestNextApiPort() int {
	port := 8086
	for {
		if !isPortTaken(port) {
			return port
		}
		port += 2 // API ports are even
	}
}

func isPortTaken(port int) bool {
	for _, p := range KnownPorts {
		if p == port {
			return true
		}
	}
	return false
}

// PortConflict describes a port conflict with an existing service.
type PortConflict struct {
	Conflict        bool
	ExistingService string
}

// IsPortConflict checks if a port is already used by another originating project service.
// Normalizes service names (strips -api/-worker suffix) to allow editing existing service ports.
func IsPortConflict(port int, currentServiceName string) PortConflict {
	for name, p := range KnownPorts {
		if p == port {
			// Normalize: strip -api/-worker suffix for comparison
			normalizedExisting := stripServiceSuffix(name)
			normalizedCurrent := stripServiceSuffix(currentServiceName)
			if normalizedExisting != normalizedCurrent {
				return PortConflict{Conflict: true, ExistingService: name}
			}
		}
	}
	return PortConflict{Conflict: false}
}

func stripServiceSuffix(name string) string {
	suffixes := []string{"-api", "-worker"}
	for _, s := range suffixes {
		if len(name) > len(s) && name[len(name)-len(s):] == s {
			return name[:len(name)-len(s)]
		}
	}
	return name
}

// ValidatePort checks if a port number is valid and not conflicting.
func ValidatePort(port int, currentServiceName string, isInternal bool) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if isInternal {
		conflict := IsPortConflict(port, currentServiceName)
		if conflict.Conflict {
			return fmt.Errorf("port %d is already used by %s", port, conflict.ExistingService)
		}
	}
	return nil
}
