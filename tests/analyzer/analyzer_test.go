package analyzer_test

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/analyzer"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

func TestAnalyzeBasic(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    image: nginx
    ports:
      - "8080:80"
    depends_on:
      - api
  api:
    image: myapi:latest
    depends_on:
      - db
    healthcheck:
      test: ["CMD", "true"]
      interval: 30s
  db:
    image: postgres:15
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	analysis := analyzer.Analyze(cf)

	if len(analysis.Services) != 3 {
		t.Errorf("expected 3 services, got %d", len(analysis.Services))
	}

	if analysis.TotalPorts != 1 {
		t.Errorf("expected 1 port, got %d", analysis.TotalPorts)
	}

	if analysis.TotalNetworks != 2 {
		t.Errorf("expected 2 networks, got %d", analysis.TotalNetworks)
	}

	// Check build order
	if len(analysis.BuildOrder) != 3 {
		t.Errorf("expected 3 items in build order, got %d", len(analysis.BuildOrder))
	}

	// db should come before api, api before web
	dbIdx := -1
	apiIdx := -1
	webIdx := -1
	for i, name := range analysis.BuildOrder {
		switch name {
		case "db":
			dbIdx = i
		case "api":
			apiIdx = i
		case "web":
			webIdx = i
		}
	}

	if dbIdx >= apiIdx || apiIdx >= webIdx {
		t.Errorf("build order incorrect: %v (db=%d, api=%d, web=%d)",
			analysis.BuildOrder, dbIdx, apiIdx, webIdx)
	}
}

func TestAnalyzeCircularDependency(t *testing.T) {
	yaml := `
services:
  a:
    image: alpine
    depends_on:
      - b
  b:
    image: alpine
    depends_on:
      - a
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	analysis := analyzer.Analyze(cf)

	found := false
	for _, w := range analysis.Warnings {
		if contains(w, "circular") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected circular dependency warning")
	}
}

func TestAnalyzeServiceInfo(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    privileged: true
    restart: always
    healthcheck:
      test: ["CMD", "true"]
    build:
      context: .
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	analysis := analyzer.Analyze(cf)

	webInfo := findService(analysis.Services, "web")
	if webInfo == nil {
		t.Fatal("web service not found")
	}

	if !webInfo.IsPrivileged {
		t.Error("expected web to be privileged")
	}
	if webInfo.RestartPolicy != "always" {
		t.Errorf("expected restart policy 'always', got '%s'", webInfo.RestartPolicy)
	}
	if !webInfo.HasHealthCheck {
		t.Error("expected web to have healthcheck")
	}
	if !webInfo.HasBuild {
		t.Error("expected web to have build")
	}
}

func TestAnalyzeDepthCalculation(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    depends_on:
      - api
  api:
    image: myapi
    depends_on:
      - db
  db:
    image: postgres
  worker:
    image: worker
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	analysis := analyzer.Analyze(cf)

	dbInfo := findService(analysis.Services, "db")
	apiInfo := findService(analysis.Services, "api")
	webInfo := findService(analysis.Services, "web")
	workerInfo := findService(analysis.Services, "worker")

	if dbInfo == nil || apiInfo == nil || webInfo == nil || workerInfo == nil {
		t.Fatal("missing services")
	}

	if dbInfo.Depth != 0 {
		t.Errorf("expected db depth 0, got %d", dbInfo.Depth)
	}
	if workerInfo.Depth != 0 {
		t.Errorf("expected worker depth 0, got %d", workerInfo.Depth)
	}
	if apiInfo.Depth != 1 {
		t.Errorf("expected api depth 1, got %d", apiInfo.Depth)
	}
	if webInfo.Depth != 2 {
		t.Errorf("expected web depth 2, got %d", webInfo.Depth)
	}
}

func TestFormatDependencyTree(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    depends_on:
      - api
  api:
    image: myapi
    depends_on:
      - db
  db:
    image: postgres
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	analysis := analyzer.Analyze(cf)
	tree := analyzer.FormatDependencyTree(analysis)

	if tree == "" {
		t.Error("expected non-empty dependency tree")
	}
}

func TestSummary(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
  db:
    image: postgres
volumes:
  data:
networks:
  default:
    driver: bridge
secrets:
  db_pass:
    file: ./pass.txt
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}

	analysis := analyzer.Analyze(cf)
	summary := analysis.Summary()

	if !contains(summary, "Services: 2") {
		t.Error("expected 'Services: 2' in summary")
	}
	if !contains(summary, "Volumes: 1") {
		t.Error("expected 'Volumes: 1' in summary")
	}
	if !contains(summary, "Networks: 1") {
		t.Error("expected 'Networks: 1' in summary")
	}
	if !contains(summary, "Secrets: 1") {
		t.Error("expected 'Secrets: 1' in summary")
	}
}

func findService(services []analyzer.ServiceInfo, name string) *analyzer.ServiceInfo {
	for i := range services {
		if services[i].Name == name {
			return &services[i]
		}
	}
	return nil
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
