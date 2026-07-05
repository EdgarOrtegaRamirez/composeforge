package validator

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

// Severity represents the severity of a validation issue
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
	SeverityCritical
)

func (s Severity) String() string {
	switch s {
	case SeverityInfo:
		return "INFO"
	case SeverityWarning:
		return "WARN"
	case SeverityError:
		return "ERROR"
	case SeverityCritical:
		return "CRITICAL"
	}
	return "UNKNOWN"
}

// Issue represents a validation issue
type Issue struct {
	Severity Severity `json:"severity"`
	Category string   `json:"category"`
	Message  string   `json:"message"`
	Service  string   `json:"service,omitempty"`
	Line     int      `json:"line,omitempty"`
}

// ValidationResult contains all validation issues
type ValidationResult struct {
	Issues       []Issue `json:"issues"`
	ErrorCount   int     `json:"error_count"`
	WarningCount int     `json:"warning_count"`
	InfoCount    int     `json:"info_count"`
}

// Validate performs comprehensive validation on a compose file
func Validate(cf *parser.ComposeFile) *ValidationResult {
	result := &ValidationResult{}
	
	validateServices(cf, result)
	validateNetworks(cf, result)
	validateVolumes(cf, result)
	validateSecrets(cf, result)
	validateCrossReferences(cf, result)
	
	return result
}

func addIssue(result *ValidationResult, severity Severity, category, message, service string) {
	issue := Issue{
		Severity: severity,
		Category: category,
		Message:  message,
		Service:  service,
	}
	result.Issues = append(result.Issues, issue)
	switch severity {
	case SeverityError, SeverityCritical:
		result.ErrorCount++
	case SeverityWarning:
		result.WarningCount++
	case SeverityInfo:
		result.InfoCount++
	}
}

func validateServices(cf *parser.ComposeFile, result *ValidationResult) {
	if len(cf.Services) == 0 {
		addIssue(result, SeverityError, "structure", "no services defined", "")
		return
	}
	
	for name, svc := range cf.Services {
		validateService(name, svc, cf, result)
	}
}

func validateService(name string, svc parser.Service, cf *parser.ComposeFile, result *ValidationResult) {
	// Must have either image or build
	if svc.Image == "" && svc.Build == nil {
		addIssue(result, SeverityError, "missing-field", "service must have either 'image' or 'build'", name)
	}
	
	// Container name should not be set in swarm mode
	if svc.ContainerName != "" && svc.Deploy != nil {
		addIssue(result, SeverityWarning, "swarm", "container_name is ignored in swarm mode", name)
	}
	
	// Restart policy validation
	validateRestartPolicy(name, svc.Restart, result)
	
	// Security checks
	validateServiceSecurity(name, svc, result)
	
	// Port validation
	validatePorts(name, svc.Ports, result)
	
	// Health check validation
	if svc.HealthCheck != nil {
		validateHealthCheck(name, svc.HealthCheck, result)
	}
	
	// Build validation
	if svc.Build != nil {
		validateBuild(name, svc.Build, result)
	}
	
	// Environment variable validation
	validateEnvironment(name, svc.Environment, result)
	
	// Volume mount validation
	validateVolumeMounts(name, svc.Volumes, result)
	
	// Deploy validation
	if svc.Deploy != nil {
		validateDeploy(name, svc.Deploy, result)
	}
}

func validateRestartPolicy(name string, restart string, result *ValidationResult) {
	validPolicies := map[string]bool{
		"":            true,
		"no":          true,
		"always":      true,
		"unless-stopped": true,
		"on-failure":  true,
	}
	if !validPolicies[restart] {
		addIssue(result, SeverityError, "invalid-value",
			fmt.Sprintf("invalid restart policy '%s' (valid: no, always, unless-stopped, on-failure)", restart), name)
	}
}

func validateServiceSecurity(name string, svc parser.Service, result *ValidationResult) {
	// Privileged mode
	if svc.Privileged {
		addIssue(result, SeverityCritical, "security",
			"service runs in privileged mode — gives full host access", name)
	}
	
	// Running as root
	if svc.User == "" {
		addIssue(result, SeverityWarning, "security",
			"no user specified — container runs as root by default", name)
	} else if svc.User == "root" || svc.User == "0" {
		addIssue(result, SeverityWarning, "security",
			"service explicitly runs as root", name)
	}
	
	// Host network via extra_hosts
	if len(svc.ExtraHosts) > 0 {
		for _, host := range svc.ExtraHosts {
			if strings.Contains(host, "host.docker.internal") {
				addIssue(result, SeverityInfo, "security",
					"service uses host.docker.internal to access host", name)
			}
		}
	}
	
	// Writable root filesystem
	if !svc.ReadOnly {
		addIssue(result, SeverityInfo, "security",
			"root filesystem is writable (consider read_only: true)", name)
	}
	
	// Capabilities
	for _, cap := range svc.CapAdd {
		if cap == "ALL" || cap == "SYS_ADMIN" || cap == "NET_ADMIN" {
			addIssue(result, SeverityCritical, "security",
				fmt.Sprintf("dangerous capability added: %s", cap), name)
		}
	}
	
	// Security options
	for _, opt := range svc.SecurityOpt {
		if opt == "apparmor:unconfined" || strings.HasPrefix(opt, "seccomp:unconfined") {
			addIssue(result, SeverityCritical, "security",
				fmt.Sprintf("security profile disabled: %s", opt), name)
		}
	}
}

var portRegex = regexp.MustCompile(`^(\d+)(?::(\d+))?(?:/(tcp|udp))?$`)

func validatePorts(name string, ports []string, result *ValidationResult) {
	for _, port := range ports {
		// Handle range syntax
		if strings.Contains(port, "-") {
			parts := strings.SplitN(port, "-", 2)
			if len(parts) == 2 {
				// Check if it's a host:container-port range
				hostPart := parts[0]
				// Find the last colon before the range
				lastColon := strings.LastIndex(hostPart, ":")
				if lastColon >= 0 {
					hostPort := hostPart[lastColon+1:]
					if _, err := fmt.Sscanf(hostPort, "%d", &struct{}{}); err == nil {
						continue
					}
				}
			}
		}
		
		if !portRegex.MatchString(port) {
			addIssue(result, SeverityError, "invalid-port",
				fmt.Sprintf("invalid port mapping: '%s'", port), name)
		}
	}
}

func validateHealthCheck(name string, hc *parser.HealthCheck, result *ValidationResult) {
	if hc.Test == nil {
		addIssue(result, SeverityError, "healthcheck",
			"healthcheck must have a 'test' command", name)
	}
	
	if hc.Retries < 0 {
		addIssue(result, SeverityError, "healthcheck",
			"retries must be non-negative", name)
	}
	
	if hc.Retries == 0 {
		addIssue(result, SeverityWarning, "healthcheck",
			"retries is 0 — health check will never mark container as unhealthy", name)
	}
}

func validateBuild(name string, build *parser.BuildConfig, result *ValidationResult) {
	if build.Context == "" && build.Dockerfile == "" {
		addIssue(result, SeverityInfo, "build",
			"build has no context or dockerfile specified — defaults to current directory", name)
	}
	
	if build.Dockerfile != "" && !strings.HasSuffix(build.Dockerfile, "Dockerfile") &&
		!strings.Contains(build.Dockerfile, ".dockerfile") &&
		!strings.HasPrefix(build.Dockerfile, "Dockerfile.") {
		addIssue(result, SeverityInfo, "build",
			fmt.Sprintf("dockerfile '%s' does not follow naming convention (Dockerfile.*)", build.Dockerfile), name)
	}
}

func validateEnvironment(name string, env interface{}, result *ValidationResult) {
	envMap := parser.NormalizeEnvironment(env)
	
	// Check for common secrets in environment
	secretPatterns := []string{
		"PASSWORD", "SECRET", "TOKEN", "API_KEY", "PRIVATE_KEY",
		"CREDENTIAL", "AUTH",
	}
	
	for key := range envMap {
		upperKey := strings.ToUpper(key)
		for _, pattern := range secretPatterns {
			if strings.Contains(upperKey, pattern) {
				addIssue(result, SeverityWarning, "security",
					fmt.Sprintf("environment variable '%s' may contain a secret — consider using Docker secrets", key), name)
				break
			}
		}
	}
}

func validateVolumeMounts(name string, volumes []string, result *ValidationResult) {
	for _, vol := range volumes {
		// Check for sensitive host paths
		sensitivePaths := []string{
			"/etc/shadow", "/etc/passwd", "/root/.ssh", "/var/run/docker.sock",
			"/proc", "/sys", "/dev",
		}
		
		for _, sensitive := range sensitivePaths {
			if strings.Contains(vol, sensitive) {
				addIssue(result, SeverityCritical, "security",
					fmt.Sprintf("volume mounts sensitive host path: %s", vol), name)
			}
		}
		
		// Check for writable host mounts
		if strings.Contains(vol, ":rw") || (!strings.Contains(vol, ":ro") && !strings.Contains(vol, ":")) {
			// This is a simplified check
		}
	}
}

func validateDeploy(name string, deploy *parser.DeployConfig, result *ValidationResult) {
	if deploy.Replicas < 0 {
		addIssue(result, SeverityError, "deploy",
			"replicas must be non-negative", name)
	}
	
	if deploy.Resources != nil && deploy.Resources.Limits != nil {
		if deploy.Resources.Limits.Memory == "" {
			addIssue(result, SeverityWarning, "resources",
				"no memory limit set — container can consume unlimited memory", name)
		}
		if deploy.Resources.Limits.Cpus == "" {
			addIssue(result, SeverityWarning, "resources",
				"no CPU limit set — container can consume unlimited CPU", name)
		}
	}
}

func validateNetworks(cf *parser.ComposeFile, result *ValidationResult) {
	for name, net := range cf.Networks {
		if net.Driver != "" {
			validDrivers := map[string]bool{
				"bridge": true, "host": true, "overlay": true,
				"macvlan": true, "none": true, "ipvlan": true,
			}
			if !validDrivers[net.Driver] {
				addIssue(result, SeverityError, "network",
					fmt.Sprintf("invalid network driver '%s'", net.Driver), "")
			}
		}
		
		if net.Driver == "host" {
			addIssue(result, SeverityWarning, "security",
				fmt.Sprintf("network '%s' uses host driver — shares host network namespace", name), "")
		}
	}
}

func validateVolumes(cf *parser.ComposeFile, result *ValidationResult) {
	for name, vol := range cf.Volumes {
		if vol.Driver != "" {
			// Note: third-party drivers are also valid
			if vol.Driver == "local" || vol.Driver == "" {
				// OK
			} else {
				addIssue(result, SeverityInfo, "volume",
					fmt.Sprintf("volume '%s' uses custom driver '%s'", name, vol.Driver), "")
			}
		}
	}
}

func validateSecrets(cf *parser.ComposeFile, result *ValidationResult) {
	for name, sec := range cf.Secrets {
		if sec.External && sec.Name == "" {
			addIssue(result, SeverityError, "secret",
				fmt.Sprintf("external secret '%s' must have a name", name), "")
		}
	}
}

func validateCrossReferences(cf *parser.ComposeFile, result *ValidationResult) {
	// Check depends_on references exist
	for name, svc := range cf.Services {
		deps := parser.NormalizeDependsOn(svc.DependsOn)
		for _, dep := range deps {
			if _, exists := cf.Services[dep]; !exists {
				addIssue(result, SeverityError, "reference",
					fmt.Sprintf("depends_on references non-existent service '%s'", dep), name)
			}
		}
	}
	
	// Check network references exist
	for name, svc := range cf.Services {
		if svc.Networks == nil {
			continue
		}
		switch nets := svc.Networks.(type) {
		case []interface{}:
			for _, n := range nets {
				netName, ok := n.(string)
				if !ok {
					continue
				}
				if _, exists := cf.Networks[netName]; !exists && netName != "default" {
					addIssue(result, SeverityError, "reference",
						fmt.Sprintf("references non-existent network '%s'", netName), name)
				}
			}
		case map[string]interface{}:
			for netName := range nets {
				if _, exists := cf.Networks[netName]; !exists && netName != "default" {
					addIssue(result, SeverityError, "reference",
						fmt.Sprintf("references non-existent network '%s'", netName), name)
				}
			}
		}
	}
	
	// Check volume references exist
	for name, svc := range cf.Services {
		for _, vol := range svc.Volumes {
			// Extract volume name from mount string
			parts := strings.SplitN(vol, ":", 2)
			if len(parts) == 0 {
				continue
			}
			volName := parts[0]
			
			// Skip if it's a bind mount (starts with / or .)
			if strings.HasPrefix(volName, "/") || strings.HasPrefix(volName, ".") || strings.HasPrefix(volName, "~") {
				continue
			}
			
			// Skip if it uses a short syntax with driver option
			if strings.HasPrefix(volName, "name=") {
				volName = strings.TrimPrefix(volName, "name=")
			}
			
			if _, exists := cf.Volumes[volName]; !exists {
				addIssue(result, SeverityWarning, "reference",
					fmt.Sprintf("references non-existent volume '%s'", volName), name)
			}
		}
	}
}
