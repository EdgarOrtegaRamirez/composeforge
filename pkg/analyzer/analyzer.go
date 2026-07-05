package analyzer

import (
	"fmt"
	"sort"
	"strings"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

// ServiceInfo contains analysis information about a service
type ServiceInfo struct {
	Name           string   `json:"name"`
	Image          string   `json:"image"`
	Dependencies   []string `json:"dependencies"`
	Dependents     []string `json:"dependents"`
	Ports          []string `json:"ports"`
	Volumes        []string `json:"volumes"`
	Networks       []string `json:"networks"`
	Environment    map[string]string `json:"environment"`
	HasHealthCheck bool     `json:"has_health_check"`
	HasBuild       bool     `json:"has_build"`
	IsPrivileged   bool     `json:"is_privileged"`
	RestartPolicy  string   `json:"restart_policy"`
	Depth          int      `json:"depth"` // dependency depth
}

// ComposeAnalysis contains the full analysis of a compose file
type ComposeAnalysis struct {
	Services      []ServiceInfo      `json:"services"`
	DependencyGraph map[string][]string `json:"dependency_graph"`
	BuildOrder    []string           `json:"build_order"`
	NetworkGraph  map[string][]string `json:"network_graph"`
	TotalPorts    int                `json:"total_ports"`
	TotalVolumes  int                `json:"total_volumes"`
	TotalSecrets  int                `json:"total_secrets"`
	TotalNetworks int                `json:"total_networks"`
	Warnings      []string           `json:"warnings"`
}

// Analyze performs comprehensive analysis on a compose file
func Analyze(cf *parser.ComposeFile) *ComposeAnalysis {
	analysis := &ComposeAnalysis{
		DependencyGraph: make(map[string][]string),
		NetworkGraph:    make(map[string][]string),
	}
	
	// Build service info
	for name, svc := range cf.Services {
		info := buildServiceInfo(name, svc, cf)
		analysis.Services = append(analysis.Services, info)
		analysis.TotalPorts += len(svc.Ports)
	}
	
	// Count volumes
	for _, vol := range cf.Services {
		analysis.TotalVolumes += len(vol.Volumes)
	}
	// Also count named volumes defined in the volumes section
	analysis.TotalVolumes += len(cf.Volumes)
	
	analysis.TotalSecrets = len(cf.Secrets)
	analysis.TotalNetworks = len(cf.Networks)
	
	// Build dependency graph
	analysis.DependencyGraph = buildDependencyGraph(cf)
	
	// Calculate build order using topological sort
	analysis.BuildOrder = topologicalSort(analysis.DependencyGraph)
	
	// Calculate dependency depths
	analysis.calculateDepths()
	
	// Build network graph
	analysis.NetworkGraph = buildNetworkGraph(cf)
	
	// Generate warnings
	analysis.generateWarnings(cf)
	
	return analysis
}

func buildServiceInfo(name string, svc parser.Service, cf *parser.ComposeFile) ServiceInfo {
	info := ServiceInfo{
		Name:           name,
		Image:          svc.Image,
		Dependencies:   parser.NormalizeDependsOn(svc.DependsOn),
		Ports:          svc.Ports,
		Volumes:        svc.Volumes,
		Environment:    parser.NormalizeEnvironment(svc.Environment),
		HasHealthCheck: svc.HealthCheck != nil,
		HasBuild:       svc.Build != nil,
		IsPrivileged:   svc.Privileged,
		RestartPolicy:  svc.Restart,
	}
	
	// Determine networks
	if svc.Networks != nil {
		switch nets := svc.Networks.(type) {
		case []interface{}:
			for _, n := range nets {
				if s, ok := n.(string); ok {
					info.Networks = append(info.Networks, s)
				}
			}
		case map[string]interface{}:
			for netName := range nets {
				info.Networks = append(info.Networks, netName)
			}
		}
	}
	if len(info.Networks) == 0 {
		info.Networks = []string{"default"}
	}
	
	return info
}

func buildDependencyGraph(cf *parser.ComposeFile) map[string][]string {
	graph := make(map[string][]string)
	for name, svc := range cf.Services {
		graph[name] = parser.NormalizeDependsOn(svc.DependsOn)
	}
	return graph
}

// topologicalSort returns services in dependency order using Kahn's algorithm
func topologicalSort(graph map[string][]string) []string {
	// Build reverse adjacency list (dependents)
	reverseGraph := make(map[string][]string)
	inDegree := make(map[string]int)
	allNodes := make(map[string]bool)
	
	for node := range graph {
		allNodes[node] = true
		inDegree[node] = 0
	}
	
	for node, deps := range graph {
		for _, dep := range deps {
			allNodes[dep] = true
			reverseGraph[dep] = append(reverseGraph[dep], node)
			inDegree[node]++
		}
	}
	
	// Ensure all nodes have entries
	for node := range allNodes {
		if _, ok := inDegree[node]; !ok {
			inDegree[node] = 0
		}
	}
	
	// Find nodes with no incoming edges
	var queue []string
	for node := range allNodes {
		if inDegree[node] == 0 {
			queue = append(queue, node)
		}
	}
	sort.Strings(queue) // deterministic order
	
	var result []string
	for len(queue) > 0 {
		sort.Strings(queue)
		node := queue[0]
		queue = queue[1:]
		result = append(result, node)
		
		// Reduce in-degree for dependents
		for _, dependent := range reverseGraph[node] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}
	
	// If we couldn't sort all nodes (cycles), add remaining
	if len(result) < len(allNodes) {
		for node := range allNodes {
			found := false
			for _, r := range result {
				if r == node {
					found = true
					break
				}
			}
			if !found {
				result = append(result, node)
			}
		}
	}
	
	return result
}

func (a *ComposeAnalysis) calculateDepths() {
	depthMap := make(map[string]int)
	
	// BFS from root nodes (nodes with no dependencies)
	var queue []string
	for _, info := range a.Services {
		if len(info.Dependencies) == 0 {
			depthMap[info.Name] = 0
			queue = append(queue, info.Name)
		}
	}
	
	for len(queue) > 0 {
		node := queue[0]
		queue = queue[1:]
		
		// Find all services that depend on this node
		for _, info := range a.Services {
			for _, dep := range info.Dependencies {
				if dep == node {
					newDepth := depthMap[node] + 1
					if current, exists := depthMap[info.Name]; !exists || newDepth > current {
						depthMap[info.Name] = newDepth
						queue = append(queue, info.Name)
					}
				}
			}
		}
	}
	
	for i := range a.Services {
		a.Services[i].Depth = depthMap[a.Services[i].Name]
	}
}

func buildNetworkGraph(cf *parser.ComposeFile) map[string][]string {
	graph := make(map[string][]string)
	
	// Initialize with defined networks
	for name := range cf.Networks {
		graph[name] = nil
	}
	if _, exists := graph["default"]; !exists {
		graph["default"] = nil
	}
	
	// Map services to networks
	for svcName, svc := range cf.Services {
		networks := []string{"default"}
		
		if svc.Networks != nil {
			switch nets := svc.Networks.(type) {
			case []interface{}:
				networks = nil
				for _, n := range nets {
					if s, ok := n.(string); ok {
						networks = append(networks, s)
					}
				}
			case map[string]interface{}:
				networks = nil
				for netName := range nets {
					networks = append(networks, netName)
				}
			}
		}
		
		for _, net := range networks {
			graph[net] = append(graph[net], svcName)
		}
	}
	
	return graph
}

func (a *ComposeAnalysis) generateWarnings(cf *parser.ComposeFile) {
	// Check for services without health checks
	for _, info := range a.Services {
		if !info.HasHealthCheck && info.RestartPolicy == "always" {
			a.Warnings = append(a.Warnings,
				fmt.Sprintf("service '%s' has restart:always but no healthcheck", info.Name))
		}
	}
	
	// Check for circular dependencies
	if hasCycle(a.DependencyGraph) {
		a.Warnings = append(a.Warnings, "circular dependency detected between services")
	}
	
	// Check for services that depend on non-existent services
	for _, info := range a.Services {
		for _, dep := range info.Dependencies {
			found := false
			for _, svc := range a.Services {
				if svc.Name == dep {
					found = true
					break
				}
			}
			if !found {
				a.Warnings = append(a.Warnings,
					fmt.Sprintf("service '%s' depends on non-existent service '%s'", info.Name, dep))
			}
		}
	}
}

// hasCycle detects cycles in a dependency graph using DFS
func hasCycle(graph map[string][]string) bool {
	const (
		white = 0 // unvisited
		gray  = 1 // in progress
		black = 2 // done
	)
	
	color := make(map[string]int)
	for node := range graph {
		color[node] = white
	}
	
	var dfs func(string) bool
	dfs = func(node string) bool {
		color[node] = gray
		for _, dep := range graph[node] {
			if color[dep] == gray {
				return true
			}
			if color[dep] == white {
				if dfs(dep) {
					return true
				}
			}
		}
		color[node] = black
		return false
	}
	
	for node := range graph {
		if color[node] == white {
			if dfs(node) {
				return true
			}
		}
	}
	return false
}

// FormatDependencyTree returns a tree-formatted string of service dependencies
func FormatDependencyTree(analysis *ComposeAnalysis) string {
	var sb strings.Builder
	visited := make(map[string]bool)
	
	var printNode func(name string, indent int)
	printNode = func(name string, indent int) {
		prefix := strings.Repeat("  ", indent)
		if indent > 0 {
			prefix += "└── "
		}
		
		if visited[name] {
			sb.WriteString(fmt.Sprintf("%s%s (circular)\n", prefix, name))
			return
		}
		visited[name] = true
		
		sb.WriteString(fmt.Sprintf("%s%s\n", prefix, name))
		
		for _, info := range analysis.Services {
			if info.Name == name {
				for _, dep := range info.Dependencies {
					printNode(dep, indent+1)
				}
				break
			}
		}
	}
	
	// Find root nodes (no dependencies)
	for _, info := range analysis.Services {
		if len(info.Dependencies) == 0 {
			printNode(info.Name, 0)
		}
	}
	
	// Handle remaining nodes (part of cycles or isolated)
	for _, info := range analysis.Services {
		if !visited[info.Name] {
			printNode(info.Name, 0)
		}
	}
	
	return sb.String()
}

// FormatNetworkGraph returns a formatted string showing network membership
func FormatNetworkGraph(analysis *ComposeAnalysis) string {
	var sb strings.Builder
	
	// Sort networks for consistent output
	var netNames []string
	for net := range analysis.NetworkGraph {
		netNames = append(netNames, net)
	}
	sort.Strings(netNames)
	
	for _, net := range netNames {
		services := analysis.NetworkGraph[net]
		sb.WriteString(fmt.Sprintf("Network: %s\n", net))
		if len(services) == 0 {
			sb.WriteString("  (no services)\n")
		} else {
			sort.Strings(services)
			for _, svc := range services {
				sb.WriteString(fmt.Sprintf("  ├── %s\n", svc))
			}
		}
	}
	
	return sb.String()
}

// Summary returns a brief summary of the analysis
func (a *ComposeAnalysis) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Services: %d\n", len(a.Services)))
	sb.WriteString(fmt.Sprintf("Networks: %d\n", a.TotalNetworks))
	sb.WriteString(fmt.Sprintf("Volumes: %d\n", a.TotalVolumes))
	sb.WriteString(fmt.Sprintf("Secrets: %d\n", a.TotalSecrets))
	sb.WriteString(fmt.Sprintf("Exposed Ports: %d\n", a.TotalPorts))
	
	// Count build vs image services
	buildCount := 0
	for _, svc := range a.Services {
		if svc.HasBuild {
			buildCount++
		}
	}
	if buildCount > 0 {
		sb.WriteString(fmt.Sprintf("Build services: %d\n", buildCount))
	}
	
	// Count privileged
	privCount := 0
	for _, svc := range a.Services {
		if svc.IsPrivileged {
			privCount++
		}
	}
	if privCount > 0 {
		sb.WriteString(fmt.Sprintf("Privileged services: %d ⚠️\n", privCount))
	}
	
	if len(a.Warnings) > 0 {
		sb.WriteString(fmt.Sprintf("Warnings: %d\n", len(a.Warnings)))
	}
	
	return sb.String()
}
