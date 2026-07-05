package merger

import (
	"fmt"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

// MergeStrategy defines how to merge conflicting values
type MergeStrategy int

const (
	// MergeStrategyOverride overwrites base values with override values
	MergeStrategyOverride MergeStrategy = iota
	// MergeStrategyMerge merges maps and slices
	MergeStrategyMerge
)

// MergeOptions configures the merge behavior
type MergeOptions struct {
	Strategy MergeStrategy
}

// Merge merges multiple compose files, with later files overriding earlier ones
func Merge(files ...*parser.ComposeFile) (*parser.ComposeFile, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("no files to merge")
	}
	
	result := deepCopy(files[0])
	
	for i := 1; i < len(files); i++ {
		mergeComposeFile(result, files[i])
	}
	
	return result, nil
}

func mergeComposeFile(base, override *parser.ComposeFile) {
	// Version: use override if present
	if override.Version != "" {
		base.Version = override.Version
	}
	
	// Name: use override if present
	if override.Name != "" {
		base.Name = override.Name
	}
	
	// Services: merge each service
	for name, overrideSvc := range override.Services {
		if baseSvc, exists := base.Services[name]; exists {
			mergeService(&baseSvc, &overrideSvc)
			base.Services[name] = baseSvc
		} else {
			base.Services[name] = overrideSvc
		}
	}
	
	// Networks: merge
	for name, overrideNet := range override.Networks {
		if _, exists := base.Networks[name]; !exists {
			base.Networks[name] = overrideNet
		}
		// Note: deep merge of network config could be added here
	}
	
	// Volumes: merge
	for name, overrideVol := range override.Volumes {
		if _, exists := base.Volumes[name]; !exists {
			base.Volumes[name] = overrideVol
		}
	}
	
	// Secrets: merge
	for name, overrideSecret := range override.Secrets {
		if _, exists := base.Secrets[name]; !exists {
			base.Secrets[name] = overrideSecret
		}
	}
	
	// Configs: merge
	for name, overrideConfig := range override.Configs {
		if _, exists := base.Configs[name]; !exists {
			base.Configs[name] = overrideConfig
		}
	}
}

func mergeService(base, override *parser.Service) {
	// String fields: override if non-empty
	if override.Image != "" {
		base.Image = override.Image
	}
	if override.HostName != "" {
		base.HostName = override.HostName
	}
	if override.ContainerName != "" {
		base.ContainerName = override.ContainerName
	}
	if override.Restart != "" {
		base.Restart = override.Restart
	}
	if override.User != "" {
		base.User = override.User
	}
	if override.WorkingDir != "" {
		base.WorkingDir = override.WorkingDir
	}
	if override.StopSignal != "" {
		base.StopSignal = override.StopSignal
	}
	if override.StopGracePeriod != "" {
		base.StopGracePeriod = override.StopGracePeriod
	}
	
	// Override non-zero fields
	if override.Privileged {
		base.Privileged = override.Privileged
	}
	if override.Tty {
		base.Tty = override.Tty
	}
	if override.StdinOpen {
		base.StdinOpen = override.StdinOpen
	}
	if override.ReadOnly {
		base.ReadOnly = override.ReadOnly
	}
	
	// Override fields that can be explicitly set to zero value
	// These use pointer semantics - check if explicitly set in override
	// For simplicity, we always override these
	base.Privileged = override.Privileged
	base.Tty = override.Tty
	base.StdinOpen = override.StdinOpen
	base.ReadOnly = override.ReadOnly
	
	// Command/Entrypoint: override if non-nil
	if override.Command != nil {
		base.Command = override.Command
	}
	if override.Entrypoint != nil {
		base.Entrypoint = override.Entrypoint
	}
	
	// Environment: merge (override values take precedence)
	if override.Environment != nil {
		baseEnv := parser.NormalizeEnvironment(base.Environment)
		overrideEnv := parser.NormalizeEnvironment(override.Environment)
		merged := make(map[string]interface{})
		for k, v := range baseEnv {
			merged[k] = v
		}
		for k, v := range overrideEnv {
			merged[k] = v
		}
		base.Environment = merged
	}
	
	// Volumes: append override volumes
	if len(override.Volumes) > 0 {
		// Check for duplicates by mount path
		existingMounts := make(map[string]bool)
		for _, vol := range base.Volumes {
			mountPath := vol
			// Extract the mount path (after the last colon, excluding options)
			for i := len(vol) - 1; i >= 0; i-- {
				if vol[i] == ':' {
					mountPath = vol[i+1:]
					// Remove :ro, :rw, etc.
					for j := len(mountPath) - 1; j >= 0; j-- {
						if mountPath[j] == ':' {
							mountPath = mountPath[j+1:]
							break
						}
					}
					break
				}
			}
			existingMounts[mountPath] = true
		}
		
		for _, vol := range override.Volumes {
			mountPath := vol
			for i := len(vol) - 1; i >= 0; i-- {
				if vol[i] == ':' {
					mountPath = vol[i+1:]
					for j := len(mountPath) - 1; j >= 0; j-- {
						if mountPath[j] == ':' {
							mountPath = mountPath[j+1:]
							break
						}
					}
					break
				}
			}
			if !existingMounts[mountPath] {
				base.Volumes = append(base.Volumes, vol)
			}
		}
	}
	
	// Ports: override (replace all)
	if len(override.Ports) > 0 {
		base.Ports = override.Ports
	}
	
	// Depends_on: override
	if override.DependsOn != nil {
		base.DependsOn = override.DependsOn
	}
	
	// Networks: override
	if override.Networks != nil {
		base.Networks = override.Networks
	}
	
	// Labels: merge
	if len(override.Labels) > 0 {
		if base.Labels == nil {
			base.Labels = make(map[string]string)
		}
		for k, v := range override.Labels {
			base.Labels[k] = v
		}
	}
	
	// Build: override if present
	if override.Build != nil {
		base.Build = override.Build
	}
	
	// Healthcheck: override if present
	if override.HealthCheck != nil {
		base.HealthCheck = override.HealthCheck
	}
	
	// Deploy: override if present
	if override.Deploy != nil {
		base.Deploy = override.Deploy
	}
	
	// Logging: override if present
	if override.Logging != nil {
		base.Logging = override.Logging
	}
	
	// Env_file: override if present
	if override.EnvFile != nil {
		base.EnvFile = override.EnvFile
	}
	
	// Security: override
	if len(override.CapAdd) > 0 {
		base.CapAdd = override.CapAdd
	}
	if len(override.CapDrop) > 0 {
		base.CapDrop = override.CapDrop
	}
	if len(override.SecurityOpt) > 0 {
		base.SecurityOpt = override.SecurityOpt
	}
	if len(override.ExtraHosts) > 0 {
		base.ExtraHosts = override.ExtraHosts
	}
	
	// DNS: override if present
	if override.DNS != nil {
		base.DNS = override.DNS
	}
	
	// Sysctls: override if present
	if override.Sysctls != nil {
		base.Sysctls = override.Sysctls
	}
	
	// Ulimits: override if present
	if override.Ulimits != nil {
		base.Ulimits = override.Ulimits
	}
	
	// Tmpfs: override if present
	if override.Tmpfs != nil {
		base.Tmpfs = override.Tmpfs
	}
	
	// Pids_limit: override
	base.PidsLimit = override.PidsLimit
	
	// Shm_size: override
	base.ShmSize = override.ShmSize
}

func deepCopy(cf *parser.ComposeFile) *parser.ComposeFile {
	result := &parser.ComposeFile{
		Version:  cf.Version,
		Name:     cf.Name,
		Services: make(map[string]parser.Service),
		Volumes:  make(map[string]parser.Volume),
		Networks: make(map[string]parser.Network),
		Secrets:  make(map[string]parser.Secret),
		Configs:  make(map[string]parser.Config),
	}
	
	for k, v := range cf.Services {
		result.Services[k] = v
	}
	for k, v := range cf.Volumes {
		result.Volumes[k] = v
	}
	for k, v := range cf.Networks {
		result.Networks[k] = v
	}
	for k, v := range cf.Secrets {
		result.Secrets[k] = v
	}
	for k, v := range cf.Configs {
		result.Configs[k] = v
	}
	
	return result
}
