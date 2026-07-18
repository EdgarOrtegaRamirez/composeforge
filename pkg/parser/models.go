package parser

import (
	"fmt"
	"strconv"
	"strings"
)

// ComposeFile represents a complete docker-compose.yml file
type ComposeFile struct {
	Version  string             `yaml:"version" json:"version"`
	Services map[string]Service `yaml:"services" json:"services"`
	Volumes  map[string]Volume  `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Networks map[string]Network `yaml:"networks,omitempty" json:"networks,omitempty"`
	Secrets  map[string]Secret  `yaml:"secrets,omitempty" json:"secrets,omitempty"`
	Configs  map[string]Config  `yaml:"configs,omitempty" json:"configs,omitempty"`
	Name     string             `yaml:"name,omitempty" json:"name,omitempty"`
}

// Service represents a single service definition
type Service struct {
	Image           string            `yaml:"image,omitempty" json:"image,omitempty"`
	Build           *BuildConfig      `yaml:"build,omitempty" json:"build,omitempty"`
	Command         interface{}       `yaml:"command,omitempty" json:"command,omitempty"`
	Entrypoint      interface{}       `yaml:"entrypoint,omitempty" json:"entrypoint,omitempty"`
	Environment     interface{}       `yaml:"environment,omitempty" json:"environment,omitempty"`
	Volumes         []string          `yaml:"volumes,omitempty" json:"volumes,omitempty"`
	Ports           []string          `yaml:"ports,omitempty" json:"ports,omitempty"`
	Expose          []string          `yaml:"expose,omitempty" json:"expose,omitempty"`
	DependsOn       interface{}       `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Networks        interface{}       `yaml:"networks,omitempty" json:"networks,omitempty"`
	Restart         string            `yaml:"restart,omitempty" json:"restart,omitempty"`
	HostName        string            `yaml:"hostname,omitempty" json:"hostname,omitempty"`
	ContainerName   string            `yaml:"container_name,omitempty" json:"container_name,omitempty"`
	EnvFile         interface{}       `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	Labels          map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	HealthCheck     *HealthCheck      `yaml:"healthcheck,omitempty" json:"healthcheck,omitempty"`
	Deploy          *DeployConfig     `yaml:"deploy,omitempty" json:"deploy,omitempty"`
	Privileged      bool              `yaml:"privileged,omitempty" json:"privileged,omitempty"`
	User            string            `yaml:"user,omitempty" json:"user,omitempty"`
	WorkingDir      string            `yaml:"working_dir,omitempty" json:"working_dir,omitempty"`
	StdinOpen       bool              `yaml:"stdin_open,omitempty" json:"stdin_open,omitempty"`
	Tty             bool              `yaml:"tty,omitempty" json:"tty,omitempty"`
	ReadOnly        bool              `yaml:"read_only,omitempty" json:"read_only,omitempty"`
	PidsLimit       interface{}       `yaml:"pids_limit,omitempty" json:"pids_limit,omitempty"`
	ShmSize         interface{}       `yaml:"shm_size,omitempty" json:"shm_size,omitempty"`
	CapAdd          []string          `yaml:"cap_add,omitempty" json:"cap_add,omitempty"`
	CapDrop         []string          `yaml:"cap_drop,omitempty" json:"cap_drop,omitempty"`
	SecurityOpt     []string          `yaml:"security_opt,omitempty" json:"security_opt,omitempty"`
	Sysctls         interface{}       `yaml:"sysctls,omitempty" json:"sysctls,omitempty"`
	DNS             interface{}       `yaml:"dns,omitempty" json:"dns,omitempty"`
	ExtraHosts      []string          `yaml:"extra_hosts,omitempty" json:"extra_hosts,omitempty"`
	Logging         *LoggingConfig    `yaml:"logging,omitempty" json:"logging,omitempty"`
	Ulimits         interface{}       `yaml:"ulimits,omitempty" json:"ulimits,omitempty"`
	Tmpfs           interface{}       `yaml:"tmpfs,omitempty" json:"tmpfs,omitempty"`
	StopGracePeriod string            `yaml:"stop_grace_period,omitempty" json:"stop_grace_period,omitempty"`
	StopSignal      string            `yaml:"stop_signal,omitempty" json:"stop_signal,omitempty"`
}

// BuildConfig represents build configuration
type BuildConfig struct {
	Context    string            `yaml:"context,omitempty" json:"context,omitempty"`
	Dockerfile string            `yaml:"dockerfile,omitempty" json:"dockerfile,omitempty"`
	Args       interface{}       `yaml:"args,omitempty" json:"args,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	ShmSize    interface{}       `yaml:"shm_size,omitempty" json:"shm_size,omitempty"`
	Target     string            `yaml:"target,omitempty" json:"target,omitempty"`
	CacheFrom  []string          `yaml:"cache_from,omitempty" json:"cache_from,omitempty"`
}

// HealthCheck represents container health check configuration
type HealthCheck struct {
	Test        interface{} `yaml:"test,omitempty" json:"test,omitempty"`
	Interval    string      `yaml:"interval,omitempty" json:"interval,omitempty"`
	Timeout     string      `yaml:"timeout,omitempty" json:"timeout,omitempty"`
	Retries     int         `yaml:"retries,omitempty" json:"retries,omitempty"`
	StartPeriod string      `yaml:"start_period,omitempty" json:"start_period,omitempty"`
}

// DeployConfig represents deployment configuration
type DeployConfig struct {
	Replicas      int                   `yaml:"replicas,omitempty" json:"replicas,omitempty"`
	Resources     *ResourceRequirements `yaml:"resources,omitempty" json:"resources,omitempty"`
	Labels        map[string]string     `yaml:"labels,omitempty" json:"labels,omitempty"`
	RestartPolicy *RestartPolicy        `yaml:"restart_policy,omitempty" json:"restart_policy,omitempty"`
	EndpointMode  string                `yaml:"endpoint_mode,omitempty" json:"endpoint_mode,omitempty"`
}

// ResourceRequirements represents resource limits and reservations
type ResourceRequirements struct {
	Limits       *ResourceList `yaml:"limits,omitempty" json:"limits,omitempty"`
	Reservations *ResourceList `yaml:"reservations,omitempty" json:"reservations,omitempty"`
}

// ResourceList represents a list of resources
type ResourceList struct {
	Cpus   string `yaml:"cpus,omitempty" json:"cpus,omitempty"`
	Memory string `yaml:"memory,omitempty" json:"memory,omitempty"`
	Pids   int    `yaml:"pids,omitempty" json:"pids,omitempty"`
}

// RestartPolicy represents restart policy configuration
type RestartPolicy struct {
	Condition   string `yaml:"condition,omitempty" json:"condition,omitempty"`
	Delay       string `yaml:"delay,omitempty" json:"delay,omitempty"`
	MaxAttempts int    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	Window      string `yaml:"window,omitempty" json:"window,omitempty"`
}

// LoggingConfig represents logging configuration
type LoggingConfig struct {
	Driver  string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	Options map[string]string `yaml:"options,omitempty" json:"options,omitempty"`
}

// Volume represents a named volume definition
type Volume struct {
	Driver     string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	External   bool              `yaml:"external,omitempty" json:"external,omitempty"`
	Name       string            `yaml:"name,omitempty" json:"name,omitempty"`
}

// Network represents a network definition
type Network struct {
	Driver     string            `yaml:"driver,omitempty" json:"driver,omitempty"`
	DriverOpts map[string]string `yaml:"driver_opts,omitempty" json:"driver_opts,omitempty"`
	Labels     map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	External   bool              `yaml:"external,omitempty" json:"external,omitempty"`
	Name       string            `yaml:"name,omitempty" json:"name,omitempty"`
	IPAM       *IPAMConfig       `yaml:"ipam,omitempty" json:"ipam,omitempty"`
	Internal   bool              `yaml:"internal,omitempty" json:"internal,omitempty"`
	Attachable bool              `yaml:"attachable,omitempty" json:"attachable,omitempty"`
}

// IPAMConfig represents IP address management configuration
type IPAMConfig struct {
	Driver string                   `yaml:"driver,omitempty" json:"driver,omitempty"`
	Config []map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
}

// Secret represents a secret definition
type Secret struct {
	File     string `yaml:"file,omitempty" json:"file,omitempty"`
	External bool   `yaml:"external,omitempty" json:"external,omitempty"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
}

// Config represents a config definition
type Config struct {
	File     string `yaml:"file,omitempty" json:"file,omitempty"`
	External bool   `yaml:"external,omitempty" json:"external,omitempty"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
}

// ParseMemorySize parses Docker memory size strings (e.g., "512m", "1g", "256k")
func ParseMemorySize(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	s = strings.TrimSpace(strings.ToUpper(s))

	multipliers := map[string]int64{
		"B":  1,
		"K":  1024,
		"KB": 1024,
		"M":  1024 * 1024,
		"MB": 1024 * 1024,
		"G":  1024 * 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
		"T":  1024 * 1024 * 1024 * 1024,
		"TB": 1024 * 1024 * 1024 * 1024,
	}

	for suffix, mult := range multipliers {
		if strings.HasSuffix(s, suffix) {
			numStr := strings.TrimSuffix(s, suffix)
			num, err := strconv.ParseFloat(numStr, 64)
			if err != nil {
				return 0, fmt.Errorf("invalid memory size: %s", s)
			}
			return int64(num * float64(mult)), nil
		}
	}

	// Plain number = bytes
	num, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid memory size: %s", s)
	}
	return num, nil
}

// FormatMemorySize formats bytes into human-readable memory size
func FormatMemorySize(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.1f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// NormalizeEnvironment normalizes environment variables to a map
func NormalizeEnvironment(env interface{}) map[string]string {
	result := make(map[string]string)
	if env == nil {
		return result
	}

	switch v := env.(type) {
	case map[string]interface{}:
		for key, val := range v {
			result[key] = fmt.Sprintf("%v", val)
		}
	case []interface{}:
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				continue
			}
			parts := strings.SplitN(s, "=", 2)
			if len(parts) == 2 {
				result[parts[0]] = parts[1]
			}
		}
	}
	return result
}

// NormalizeDependsOn normalizes depends_on to a list of service names
func NormalizeDependsOn(dep interface{}) []string {
	if dep == nil {
		return nil
	}

	switch v := dep.(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	case map[string]interface{}:
		var result []string
		for name := range v {
			result = append(result, name)
		}
		return result
	}
	return nil
}

// NormalizeVolumes normalizes volumes to a list of strings
func NormalizeVolumes(vols interface{}) []string {
	if vols == nil {
		return nil
	}

	switch v := vols.(type) {
	case []interface{}:
		var result []string
		for _, item := range v {
			if s, ok := item.(string); ok {
				result = append(result, s)
			}
		}
		return result
	}
	return nil
}
