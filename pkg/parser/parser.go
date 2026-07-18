package parser

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// ParseFile reads and parses a docker-compose YAML file
func ParseFile(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return ParseBytes(data)
}

// ParseBytes parses YAML bytes into a ComposeFile
func ParseBytes(data []byte) (*ComposeFile, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}
	return convertRaw(raw)
}

// ParseString parses a YAML string into a ComposeFile
func ParseString(s string) (*ComposeFile, error) {
	return ParseBytes([]byte(s))
}

// convertRaw converts the raw parsed YAML into our typed model
func convertRaw(raw map[string]interface{}) (*ComposeFile, error) {
	cf := &ComposeFile{
		Services: make(map[string]Service),
		Volumes:  make(map[string]Volume),
		Networks: make(map[string]Network),
		Secrets:  make(map[string]Secret),
		Configs:  make(map[string]Config),
	}

	// Parse version
	if v, ok := raw["version"].(string); ok {
		cf.Version = v
	}

	// Parse name
	if v, ok := raw["name"].(string); ok {
		cf.Name = v
	}

	// Parse services
	if servicesRaw, ok := raw["services"].(map[string]interface{}); ok {
		for name, svcRaw := range servicesRaw {
			svcMap, ok := svcRaw.(map[string]interface{})
			if !ok {
				continue
			}
			svc := convertService(svcMap)
			cf.Services[name] = svc
		}
	}

	// Parse volumes
	if volumesRaw, ok := raw["volumes"].(map[string]interface{}); ok {
		for name, volRaw := range volumesRaw {
			if volMap, ok := volRaw.(map[string]interface{}); ok {
				cf.Volumes[name] = convertVolume(volMap)
			} else {
				// Simple volume definition (e.g., just external: true)
				cf.Volumes[name] = Volume{}
			}
		}
	}

	// Parse networks
	if networksRaw, ok := raw["networks"].(map[string]interface{}); ok {
		for name, netRaw := range networksRaw {
			if netMap, ok := netRaw.(map[string]interface{}); ok {
				cf.Networks[name] = convertNetwork(netMap)
			} else {
				cf.Networks[name] = Network{}
			}
		}
	}

	// Parse secrets
	if secretsRaw, ok := raw["secrets"].(map[string]interface{}); ok {
		for name, secRaw := range secretsRaw {
			if secMap, ok := secRaw.(map[string]interface{}); ok {
				cf.Secrets[name] = convertSecret(secMap)
			}
		}
	}

	// Parse configs
	if configsRaw, ok := raw["configs"].(map[string]interface{}); ok {
		for name, cfgRaw := range configsRaw {
			if cfgMap, ok := cfgRaw.(map[string]interface{}); ok {
				cf.Configs[name] = convertConfig(cfgMap)
			}
		}
	}

	return cf, nil
}

func convertService(raw map[string]interface{}) Service {
	svc := Service{}

	if v, ok := raw["image"].(string); ok {
		svc.Image = v
	}
	if v, ok := raw["hostname"].(string); ok {
		svc.HostName = v
	}
	if v, ok := raw["container_name"].(string); ok {
		svc.ContainerName = v
	}
	if v, ok := raw["restart"].(string); ok {
		svc.Restart = v
	}
	if v, ok := raw["user"].(string); ok {
		svc.User = v
	}
	if v, ok := raw["working_dir"].(string); ok {
		svc.WorkingDir = v
	}
	if v, ok := raw["stdin_open"].(bool); ok {
		svc.StdinOpen = v
	}
	if v, ok := raw["tty"].(bool); ok {
		svc.Tty = v
	}
	if v, ok := raw["read_only"].(bool); ok {
		svc.ReadOnly = v
	}
	if v, ok := raw["privileged"].(bool); ok {
		svc.Privileged = v
	}
	if v, ok := raw["stop_signal"].(string); ok {
		svc.StopSignal = v
	}
	if v, ok := raw["stop_grace_period"].(string); ok {
		svc.StopGracePeriod = v
	}

	svc.Command = raw["command"]
	svc.Entrypoint = raw["entrypoint"]
	svc.Environment = raw["environment"]
	svc.DependsOn = raw["depends_on"]
	svc.Networks = raw["networks"]
	svc.EnvFile = raw["env_file"]
	svc.PidsLimit = raw["pids_limit"]
	svc.ShmSize = raw["shm_size"]
	svc.Sysctls = raw["sysctls"]
	svc.DNS = raw["dns"]
	svc.Ulimits = raw["ulimits"]
	svc.Tmpfs = raw["tmpfs"]

	svc.Volumes = NormalizeVolumes(raw["volumes"])
	svc.Ports = NormalizeStringList(raw["ports"])
	svc.Expose = NormalizeStringList(raw["expose"])
	svc.ExtraHosts = NormalizeStringList(raw["extra_hosts"])
	svc.CapAdd = NormalizeStringList(raw["cap_add"])
	svc.CapDrop = NormalizeStringList(raw["cap_drop"])
	svc.SecurityOpt = NormalizeStringList(raw["security_opt"])

	if v, ok := raw["labels"].(map[string]interface{}); ok {
		svc.Labels = convertLabels(v)
	}

	if v, ok := raw["build"].(map[string]interface{}); ok {
		svc.Build = convertBuild(v)
	}

	if v, ok := raw["healthcheck"].(map[string]interface{}); ok {
		svc.HealthCheck = convertHealthCheck(v)
	}

	if v, ok := raw["deploy"].(map[string]interface{}); ok {
		svc.Deploy = convertDeploy(v)
	}

	if v, ok := raw["logging"].(map[string]interface{}); ok {
		svc.Logging = convertLogging(v)
	}

	return svc
}

func convertBuild(raw map[string]interface{}) *BuildConfig {
	bc := &BuildConfig{}
	if v, ok := raw["context"].(string); ok {
		bc.Context = v
	}
	if v, ok := raw["dockerfile"].(string); ok {
		bc.Dockerfile = v
	}
	if v, ok := raw["target"].(string); ok {
		bc.Target = v
	}
	bc.Args = raw["args"]
	bc.CacheFrom = NormalizeStringList(raw["cache_from"])
	if v, ok := raw["labels"].(map[string]interface{}); ok {
		bc.Labels = convertLabels(v)
	}
	return bc
}

func convertHealthCheck(raw map[string]interface{}) *HealthCheck {
	hc := &HealthCheck{}
	hc.Test = raw["test"]
	if v, ok := raw["interval"].(string); ok {
		hc.Interval = v
	}
	if v, ok := raw["timeout"].(string); ok {
		hc.Timeout = v
	}
	if v, ok := raw["retries"].(int); ok {
		hc.Retries = v
	}
	if v, ok := raw["start_period"].(string); ok {
		hc.StartPeriod = v
	}
	return hc
}

func convertDeploy(raw map[string]interface{}) *DeployConfig {
	dc := &DeployConfig{}
	if v, ok := raw["replicas"].(int); ok {
		dc.Replicas = v
	}
	if v, ok := raw["endpoint_mode"].(string); ok {
		dc.EndpointMode = v
	}
	if v, ok := raw["labels"].(map[string]interface{}); ok {
		dc.Labels = convertLabels(v)
	}
	if v, ok := raw["resources"].(map[string]interface{}); ok {
		dc.Resources = convertResources(v)
	}
	if v, ok := raw["restart_policy"].(map[string]interface{}); ok {
		dc.RestartPolicy = convertRestartPolicy(v)
	}
	return dc
}

func convertResources(raw map[string]interface{}) *ResourceRequirements {
	rr := &ResourceRequirements{}
	if v, ok := raw["limits"].(map[string]interface{}); ok {
		rr.Limits = convertResourceList(v)
	}
	if v, ok := raw["reservations"].(map[string]interface{}); ok && v != nil {
		rr.Reservations = convertResourceList(v)
	}
	return rr
}

func convertResourceList(raw map[string]interface{}) *ResourceList {
	rl := &ResourceList{}
	if v, ok := raw["cpus"].(string); ok {
		rl.Cpus = v
	} else if v, ok := raw["cpus"].(int); ok {
		rl.Cpus = fmt.Sprintf("%d", v)
	}
	if v, ok := raw["memory"].(string); ok {
		rl.Memory = v
	}
	return rl
}

func convertRestartPolicy(raw map[string]interface{}) *RestartPolicy {
	rp := &RestartPolicy{}
	if v, ok := raw["condition"].(string); ok {
		rp.Condition = v
	}
	if v, ok := raw["delay"].(string); ok {
		rp.Delay = v
	}
	if v, ok := raw["max_attempts"].(int); ok {
		rp.MaxAttempts = v
	}
	if v, ok := raw["window"].(string); ok {
		rp.Window = v
	}
	return rp
}

func convertLogging(raw map[string]interface{}) *LoggingConfig {
	lc := &LoggingConfig{}
	if v, ok := raw["driver"].(string); ok {
		lc.Driver = v
	}
	if v, ok := raw["options"].(map[string]interface{}); ok {
		lc.Options = convertLabels(v)
	}
	return lc
}

func convertVolume(raw map[string]interface{}) Volume {
	vol := Volume{}
	if v, ok := raw["driver"].(string); ok {
		vol.Driver = v
	}
	if v, ok := raw["external"].(bool); ok {
		vol.External = v
	}
	if v, ok := raw["name"].(string); ok {
		vol.Name = v
	}
	if v, ok := raw["driver_opts"].(map[string]interface{}); ok {
		vol.DriverOpts = convertLabels(v)
	}
	if v, ok := raw["labels"].(map[string]interface{}); ok {
		vol.Labels = convertLabels(v)
	}
	return vol
}

func convertNetwork(raw map[string]interface{}) Network {
	net := Network{}
	if v, ok := raw["driver"].(string); ok {
		net.Driver = v
	}
	if v, ok := raw["external"].(bool); ok {
		net.External = v
	}
	if v, ok := raw["name"].(string); ok {
		net.Name = v
	}
	if v, ok := raw["internal"].(bool); ok {
		net.Internal = v
	}
	if v, ok := raw["attachable"].(bool); ok {
		net.Attachable = v
	}
	if v, ok := raw["driver_opts"].(map[string]interface{}); ok {
		net.DriverOpts = convertLabels(v)
	}
	if v, ok := raw["labels"].(map[string]interface{}); ok {
		net.Labels = convertLabels(v)
	}
	return net
}

func convertSecret(raw map[string]interface{}) Secret {
	sec := Secret{}
	if v, ok := raw["file"].(string); ok {
		sec.File = v
	}
	if v, ok := raw["external"].(bool); ok {
		sec.External = v
	}
	if v, ok := raw["name"].(string); ok {
		sec.Name = v
	}
	return sec
}

func convertConfig(raw map[string]interface{}) Config {
	cfg := Config{}
	if v, ok := raw["file"].(string); ok {
		cfg.File = v
	}
	if v, ok := raw["external"].(bool); ok {
		cfg.External = v
	}
	if v, ok := raw["name"].(string); ok {
		cfg.Name = v
	}
	return cfg
}

func convertLabels(raw map[string]interface{}) map[string]string {
	result := make(map[string]string)
	for k, v := range raw {
		result[k] = fmt.Sprintf("%v", v)
	}
	return result
}

// NormalizeStringList normalizes various YAML list formats to []string
func NormalizeStringList(v interface{}) []string {
	if v == nil {
		return nil
	}
	switch val := v.(type) {
	case []interface{}:
		var result []string
		for _, item := range val {
			result = append(result, fmt.Sprintf("%v", item))
		}
		return result
	case string:
		return []string{val}
	}
	return nil
}

// FindComposeFile searches for docker-compose files in a directory
func FindComposeFile(dir string) (string, error) {
	possibleNames := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
		"docker-compose.override.yml",
		"docker-compose.override.yaml",
	}

	for _, name := range possibleNames {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no docker-compose file found in %s", dir)
}
