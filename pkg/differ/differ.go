package differ

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

// ChangeType represents the type of change
type ChangeType string

const (
	ChangeAdded    ChangeType = "ADDED"
	ChangeRemoved  ChangeType = "REMOVED"
	ChangeModified ChangeType = "MODIFIED"
)

// ServiceDiff represents changes to a single service
type ServiceDiff struct {
	Name    string        `json:"name"`
	Type    ChangeType    `json:"type"`
	Changes []FieldChange `json:"changes,omitempty"`
}

// FieldChange represents a change to a specific field
type FieldChange struct {
	Field    string     `json:"field"`
	OldValue string     `json:"old_value,omitempty"`
	NewValue string     `json:"new_value,omitempty"`
	Type     ChangeType `json:"type"`
}

// ComposeDiff represents the complete diff between two compose files
type ComposeDiff struct {
	ServiceDiffs     []ServiceDiff `json:"service_diffs"`
	AddedServices    []string      `json:"added_services"`
	RemovedServices  []string      `json:"removed_services"`
	ModifiedServices []string      `json:"modified_services"`
	NetworkChanges   []FieldChange `json:"network_changes,omitempty"`
	VolumeChanges    []FieldChange `json:"volume_changes,omitempty"`
	VersionChanged   bool          `json:"version_changed"`
	OldVersion       string        `json:"old_version,omitempty"`
	NewVersion       string        `json:"new_version,omitempty"`
	HasChanges       bool          `json:"has_changes"`
}

// Diff compares two compose files and returns the differences
func Diff(old, new *parser.ComposeFile) *ComposeDiff {
	result := &ComposeDiff{}

	// Check version change
	if old.Version != new.Version {
		result.VersionChanged = true
		result.OldVersion = old.Version
		result.NewVersion = new.Version
	}

	// Build maps for comparison
	oldServices := make(map[string]parser.Service)
	newServices := make(map[string]parser.Service)

	for name, svc := range old.Services {
		oldServices[name] = svc
	}
	for name, svc := range new.Services {
		newServices[name] = svc
	}

	// Find added services
	for name := range newServices {
		if _, exists := oldServices[name]; !exists {
			result.AddedServices = append(result.AddedServices, name)
			result.ServiceDiffs = append(result.ServiceDiffs, ServiceDiff{
				Name: name,
				Type: ChangeAdded,
			})
		}
	}

	// Find removed services
	for name := range oldServices {
		if _, exists := newServices[name]; !exists {
			result.RemovedServices = append(result.RemovedServices, name)
			result.ServiceDiffs = append(result.ServiceDiffs, ServiceDiff{
				Name: name,
				Type: ChangeRemoved,
			})
		}
	}

	// Find modified services
	for name := range oldServices {
		if newSvc, exists := newServices[name]; exists {
			oldSvc := oldServices[name]
			changes := diffService(name, oldSvc, newSvc)
			if len(changes) > 0 {
				result.ModifiedServices = append(result.ModifiedServices, name)
				result.ServiceDiffs = append(result.ServiceDiffs, ServiceDiff{
					Name:    name,
					Type:    ChangeModified,
					Changes: changes,
				})
			}
		}
	}

	// Compare networks
	result.NetworkChanges = diffNetworks(old.Networks, new.Networks)

	// Compare volumes
	result.VolumeChanges = diffVolumes(old.Volumes, new.Volumes)

	result.HasChanges = len(result.AddedServices) > 0 ||
		len(result.RemovedServices) > 0 ||
		len(result.ModifiedServices) > 0 ||
		result.VersionChanged ||
		len(result.NetworkChanges) > 0 ||
		len(result.VolumeChanges) > 0

	return result
}

func diffService(name string, old, new parser.Service) []FieldChange {
	var changes []FieldChange

	// Simple string fields
	changes = append(changes, diffStringField("image", old.Image, new.Image)...)
	changes = append(changes, diffStringField("hostname", old.HostName, new.HostName)...)
	changes = append(changes, diffStringField("container_name", old.ContainerName, new.ContainerName)...)
	changes = append(changes, diffStringField("restart", old.Restart, new.Restart)...)
	changes = append(changes, diffStringField("user", old.User, new.User)...)
	changes = append(changes, diffStringField("working_dir", old.WorkingDir, new.WorkingDir)...)
	changes = append(changes, diffStringField("stop_signal", old.StopSignal, new.StopSignal)...)
	changes = append(changes, diffStringField("stop_grace_period", old.StopGracePeriod, new.StopGracePeriod)...)

	// Boolean fields
	changes = append(changes, diffBoolField("privileged", old.Privileged, new.Privileged)...)
	changes = append(changes, diffBoolField("stdin_open", old.StdinOpen, new.StdinOpen)...)
	changes = append(changes, diffBoolField("tty", old.Tty, new.Tty)...)
	changes = append(changes, diffBoolField("read_only", old.ReadOnly, new.ReadOnly)...)

	// Slice fields
	changes = append(changes, diffStringSliceField("ports", old.Ports, new.Ports)...)
	changes = append(changes, diffStringSliceField("expose", old.Expose, new.Expose)...)
	changes = append(changes, diffStringSliceField("volumes", old.Volumes, new.Volumes)...)
	changes = append(changes, diffStringSliceField("cap_add", old.CapAdd, new.CapAdd)...)
	changes = append(changes, diffStringSliceField("cap_drop", old.CapDrop, new.CapDrop)...)
	changes = append(changes, diffStringSliceField("security_opt", old.SecurityOpt, new.SecurityOpt)...)
	changes = append(changes, diffStringSliceField("extra_hosts", old.ExtraHosts, new.ExtraHosts)...)

	// Map fields
	changes = append(changes, diffMapField("labels", old.Labels, new.Labels)...)

	// Environment
	changes = append(changes, diffEnvironment(old.Environment, new.Environment)...)

	// Depends_on
	changes = append(changes, diffDependsOn(old.DependsOn, new.DependsOn)...)

	// Build
	changes = append(changes, diffBuild(old.Build, new.Build)...)

	// Healthcheck
	changes = append(changes, diffHealthCheck(old.HealthCheck, new.HealthCheck)...)

	// Filter out empty changes
	var result []FieldChange
	for _, c := range changes {
		if c.OldValue != c.NewValue || c.Type == ChangeAdded || c.Type == ChangeRemoved {
			result = append(result, c)
		}
	}

	return result
}

func diffStringField(field, old, new string) []FieldChange {
	if old == new {
		return nil
	}
	changes := []FieldChange{{
		Field:    field,
		OldValue: old,
		NewValue: new,
		Type:     ChangeModified,
	}}
	return changes
}

func diffBoolField(field string, old, new bool) []FieldChange {
	if old == new {
		return nil
	}
	return []FieldChange{{
		Field:    field,
		OldValue: fmt.Sprintf("%v", old),
		NewValue: fmt.Sprintf("%v", new),
		Type:     ChangeModified,
	}}
}

func diffStringSliceField(field string, old, new []string) []FieldChange {
	oldSorted := make([]string, len(old))
	newSorted := make([]string, len(new))
	copy(oldSorted, old)
	copy(newSorted, new)
	sort.Strings(oldSorted)
	sort.Strings(newSorted)

	oldStr := strings.Join(oldSorted, ",")
	newStr := strings.Join(newSorted, ",")

	if oldStr == newStr {
		return nil
	}

	return []FieldChange{{
		Field:    field,
		OldValue: strings.Join(old, " "),
		NewValue: strings.Join(new, " "),
		Type:     ChangeModified,
	}}
}

func diffMapField(field string, old, new map[string]string) []FieldChange {
	if len(old) == 0 && len(new) == 0 {
		return nil
	}

	oldStr := formatMap(old)
	newStr := formatMap(new)

	if oldStr == newStr {
		return nil
	}

	return []FieldChange{{
		Field:    field,
		OldValue: oldStr,
		NewValue: newStr,
		Type:     ChangeModified,
	}}
}

func formatMap(m map[string]string) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", k, m[k]))
	}
	return strings.Join(parts, ", ")
}

func diffEnvironment(old, new interface{}) []FieldChange {
	oldEnv := parser.NormalizeEnvironment(old)
	newEnv := parser.NormalizeEnvironment(new)

	if len(oldEnv) == 0 && len(newEnv) == 0 {
		return nil
	}

	var changes []FieldChange

	// Find added/modified
	for key, newVal := range newEnv {
		oldVal, exists := oldEnv[key]
		if !exists {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("environment.%s", key),
				NewValue: newVal,
				Type:     ChangeAdded,
			})
		} else if oldVal != newVal {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("environment.%s", key),
				OldValue: oldVal,
				NewValue: newVal,
				Type:     ChangeModified,
			})
		}
	}

	// Find removed
	for key, oldVal := range oldEnv {
		if _, exists := newEnv[key]; !exists {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("environment.%s", key),
				OldValue: oldVal,
				Type:     ChangeRemoved,
			})
		}
	}

	return changes
}

func diffDependsOn(old, new interface{}) []FieldChange {
	oldDeps := parser.NormalizeDependsOn(old)
	newDeps := parser.NormalizeDependsOn(new)

	oldStr := strings.Join(oldDeps, ",")
	newStr := strings.Join(newDeps, ",")

	if oldStr == newStr {
		return nil
	}

	return []FieldChange{{
		Field:    "depends_on",
		OldValue: strings.Join(oldDeps, " "),
		NewValue: strings.Join(newDeps, " "),
		Type:     ChangeModified,
	}}
}

func diffBuild(old, new *parser.BuildConfig) []FieldChange {
	if old == nil && new == nil {
		return nil
	}
	if old == nil && new != nil {
		return []FieldChange{{
			Field:    "build",
			NewValue: fmt.Sprintf("context=%s dockerfile=%s", new.Context, new.Dockerfile),
			Type:     ChangeAdded,
		}}
	}
	if old != nil && new == nil {
		return []FieldChange{{
			Field:    "build",
			OldValue: fmt.Sprintf("context=%s dockerfile=%s", old.Context, old.Dockerfile),
			Type:     ChangeRemoved,
		}}
	}

	var changes []FieldChange
	changes = append(changes, diffStringField("build.context", old.Context, new.Context)...)
	changes = append(changes, diffStringField("build.dockerfile", old.Dockerfile, new.Dockerfile)...)
	changes = append(changes, diffStringField("build.target", old.Target, new.Target)...)
	return changes
}

func diffHealthCheck(old, new *parser.HealthCheck) []FieldChange {
	if old == nil && new == nil {
		return nil
	}
	if old == nil && new != nil {
		return []FieldChange{{
			Field:    "healthcheck",
			NewValue: "added",
			Type:     ChangeAdded,
		}}
	}
	if old != nil && new == nil {
		return []FieldChange{{
			Field:    "healthcheck",
			OldValue: "removed",
			Type:     ChangeRemoved,
		}}
	}

	var changes []FieldChange
	changes = append(changes, diffStringField("healthcheck.interval", old.Interval, new.Interval)...)
	changes = append(changes, diffStringField("healthcheck.timeout", old.Timeout, new.Timeout)...)
	if old.Retries != new.Retries {
		changes = append(changes, FieldChange{
			Field:    "healthcheck.retries",
			OldValue: fmt.Sprintf("%d", old.Retries),
			NewValue: fmt.Sprintf("%d", new.Retries),
			Type:     ChangeModified,
		})
	}
	return changes
}

func diffNetworks(old, new map[string]parser.Network) []FieldChange {
	var changes []FieldChange

	// Added/modified
	for name, newNet := range new {
		oldNet, exists := old[name]
		if !exists {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("network.%s", name),
				NewValue: fmt.Sprintf("driver=%s", newNet.Driver),
				Type:     ChangeAdded,
			})
		} else {
			if oldNet.Driver != newNet.Driver {
				changes = append(changes, FieldChange{
					Field:    fmt.Sprintf("network.%s.driver", name),
					OldValue: oldNet.Driver,
					NewValue: newNet.Driver,
					Type:     ChangeModified,
				})
			}
		}
	}

	// Removed
	for name := range old {
		if _, exists := new[name]; !exists {
			changes = append(changes, FieldChange{
				Field: fmt.Sprintf("network.%s", name),
				Type:  ChangeRemoved,
			})
		}
	}

	return changes
}

func diffVolumes(old, new map[string]parser.Volume) []FieldChange {
	var changes []FieldChange

	for name, newVol := range new {
		oldVol, exists := old[name]
		if !exists {
			changes = append(changes, FieldChange{
				Field:    fmt.Sprintf("volume.%s", name),
				NewValue: fmt.Sprintf("driver=%s", newVol.Driver),
				Type:     ChangeAdded,
			})
		} else {
			if oldVol.Driver != newVol.Driver {
				changes = append(changes, FieldChange{
					Field:    fmt.Sprintf("volume.%s.driver", name),
					OldValue: oldVol.Driver,
					NewValue: newVol.Driver,
					Type:     ChangeModified,
				})
			}
			if oldVol.External != newVol.External {
				changes = append(changes, FieldChange{
					Field:    fmt.Sprintf("volume.%s.external", name),
					OldValue: fmt.Sprintf("%v", oldVol.External),
					NewValue: fmt.Sprintf("%v", newVol.External),
					Type:     ChangeModified,
				})
			}
		}
	}

	for name := range old {
		if _, exists := new[name]; !exists {
			changes = append(changes, FieldChange{
				Field: fmt.Sprintf("volume.%s", name),
				Type:  ChangeRemoved,
			})
		}
	}

	return changes
}

// FormatDiff returns a human-readable diff output
func FormatDiff(diff *ComposeDiff) string {
	if !diff.HasChanges {
		return "No changes detected."
	}

	var sb strings.Builder

	if diff.VersionChanged {
		sb.WriteString(fmt.Sprintf("Version: %s → %s\n\n", diff.OldVersion, diff.NewVersion))
	}

	if len(diff.AddedServices) > 0 {
		sb.WriteString("Added services:\n")
		for _, svc := range diff.AddedServices {
			sb.WriteString(fmt.Sprintf("  + %s\n", svc))
		}
		sb.WriteString("\n")
	}

	if len(diff.RemovedServices) > 0 {
		sb.WriteString("Removed services:\n")
		for _, svc := range diff.RemovedServices {
			sb.WriteString(fmt.Sprintf("  - %s\n", svc))
		}
		sb.WriteString("\n")
	}

	for _, svcDiff := range diff.ServiceDiffs {
		if svcDiff.Type == ChangeModified && len(svcDiff.Changes) > 0 {
			sb.WriteString(fmt.Sprintf("Modified: %s\n", svcDiff.Name))
			for _, change := range svcDiff.Changes {
				switch change.Type {
				case ChangeAdded:
					sb.WriteString(fmt.Sprintf("  + %s: %s\n", change.Field, change.NewValue))
				case ChangeRemoved:
					sb.WriteString(fmt.Sprintf("  - %s: %s\n", change.Field, change.OldValue))
				case ChangeModified:
					sb.WriteString(fmt.Sprintf("  ~ %s: %s → %s\n", change.Field, change.OldValue, change.NewValue))
				}
			}
			sb.WriteString("\n")
		}
	}

	return strings.TrimSpace(sb.String())
}
