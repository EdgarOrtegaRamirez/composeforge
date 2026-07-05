package parser_test

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

func TestParseBasicCompose(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    environment:
      - NODE_ENV=production
    volumes:
      - ./html:/usr/share/nginx/html
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
networks:
  default:
    driver: bridge
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	if cf.Version != "3.8" {
		t.Errorf("expected version '3.8', got '%s'", cf.Version)
	}
	
	if len(cf.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(cf.Services))
	}
	
	web, ok := cf.Services["web"]
	if !ok {
		t.Fatal("service 'web' not found")
	}
	if web.Image != "nginx:latest" {
		t.Errorf("expected image 'nginx:latest', got '%s'", web.Image)
	}
	if len(web.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(web.Ports))
	}
	if web.Ports[0] != "8080:80" {
		t.Errorf("expected port '8080:80', got '%s'", web.Ports[0])
	}
	
	db, ok := cf.Services["db"]
	if !ok {
		t.Fatal("service 'db' not found")
	}
	if db.Image != "postgres:15" {
		t.Errorf("expected image 'postgres:15', got '%s'", db.Image)
	}
	
	if len(cf.Volumes) != 1 {
		t.Errorf("expected 1 volume, got %d", len(cf.Volumes))
	}
	if _, ok := cf.Volumes["pgdata"]; !ok {
		t.Error("volume 'pgdata' not found")
	}
}

func TestParseWithBuild(t *testing.T) {
	yaml := `
services:
  app:
    build:
      context: .
      dockerfile: Dockerfile.dev
      target: development
      args:
        NODE_ENV: development
    ports:
      - "3000:3000"
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	app := cf.Services["app"]
	if app.Build == nil {
		t.Fatal("expected build config for 'app'")
	}
	if app.Build.Context != "." {
		t.Errorf("expected context '.', got '%s'", app.Build.Context)
	}
	if app.Build.Dockerfile != "Dockerfile.dev" {
		t.Errorf("expected dockerfile 'Dockerfile.dev', got '%s'", app.Build.Dockerfile)
	}
	if app.Build.Target != "development" {
		t.Errorf("expected target 'development', got '%s'", app.Build.Target)
	}
}

func TestParseWithHealthCheck(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 5s
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	web := cf.Services["web"]
	if web.HealthCheck == nil {
		t.Fatal("expected healthcheck for 'web'")
	}
	if web.HealthCheck.Interval != "30s" {
		t.Errorf("expected interval '30s', got '%s'", web.HealthCheck.Interval)
	}
	if web.HealthCheck.Timeout != "10s" {
		t.Errorf("expected timeout '10s', got '%s'", web.HealthCheck.Timeout)
	}
	if web.HealthCheck.Retries != 3 {
		t.Errorf("expected retries 3, got %d", web.HealthCheck.Retries)
	}
}

func TestParseWithDeploy(t *testing.T) {
	yaml := `
services:
  worker:
    image: myapp:latest
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '0.5'
          memory: 512M
        reservations:
          cpus: '0.25'
          memory: 256M
      restart_policy:
        condition: on-failure
        max_attempts: 5
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	worker := cf.Services["worker"]
	if worker.Deploy == nil {
		t.Fatal("expected deploy config for 'worker'")
	}
	if worker.Deploy.Replicas != 3 {
		t.Errorf("expected replicas 3, got %d", worker.Deploy.Replicas)
	}
	if worker.Deploy.Resources == nil || worker.Deploy.Resources.Limits == nil {
		t.Fatal("expected resource limits")
	}
	if worker.Deploy.Resources.Limits.Memory != "512M" {
		t.Errorf("expected memory limit '512M', got '%s'", worker.Deploy.Resources.Limits.Memory)
	}
}

func TestParseWithDependsOn(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    depends_on:
      - api
  api:
    image: myapi:latest
    depends_on:
      - db
  db:
    image: postgres:15
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	web := cf.Services["web"]
	deps := parser.NormalizeDependsOn(web.DependsOn)
	if len(deps) != 1 || deps[0] != "api" {
		t.Errorf("expected web to depend on [api], got %v", deps)
	}
	
	api := cf.Services["api"]
	deps = parser.NormalizeDependsOn(api.DependsOn)
	if len(deps) != 1 || deps[0] != "db" {
		t.Errorf("expected api to depend on [db], got %v", deps)
	}
}

func TestParseWithMapDependsOn(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    depends_on:
      api:
        condition: service_healthy
      db:
        condition: service_started
  api:
    image: myapi:latest
  db:
    image: postgres:15
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	web := cf.Services["web"]
	deps := parser.NormalizeDependsOn(web.DependsOn)
	if len(deps) != 2 {
		t.Errorf("expected 2 dependencies, got %d", len(deps))
	}
}

func TestNormalizeEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]string
	}{
		{
			name:     "nil",
			input:    nil,
			expected: map[string]string{},
		},
		{
			name: "map",
			input: map[string]interface{}{
				"FOO": "bar",
				"BAZ": "123",
			},
			expected: map[string]string{"FOO": "bar", "BAZ": "123"},
		},
		{
			name:  "list",
			input: []interface{}{"FOO=bar", "BAZ=123"},
			expected: map[string]string{"FOO": "bar", "BAZ": "123"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parser.NormalizeEnvironment(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected %d env vars, got %d", len(tt.expected), len(result))
			}
			for k, v := range tt.expected {
				if result[k] != v {
					t.Errorf("expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestParseMemorySize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"512M", 512 * 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"256k", 256 * 1024, false},
		{"1024", 1024, false},
		{"1.5G", int64(1.5 * 1024 * 1024 * 1024), false},
		{"", 0, false},
		{"invalid", 0, true},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parser.ParseMemorySize(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMemorySize(%s) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseMemorySize(%s) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatMemorySize(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}
	
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := parser.FormatMemorySize(tt.input)
			if result != tt.expected {
				t.Errorf("FormatMemorySize(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseEmpty(t *testing.T) {
	yaml := ``
	_, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed on empty input: %v", err)
	}
}

func TestParseWithSecretsAndConfigs(t *testing.T) {
	yaml := `
services:
  db:
    image: postgres
    secrets:
      - db_password
    configs:
      - db_config
secrets:
  db_password:
    file: ./secrets/db_password.txt
configs:
  db_config:
    file: ./configs/db.conf
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	if len(cf.Secrets) != 1 {
		t.Errorf("expected 1 secret, got %d", len(cf.Secrets))
	}
	if _, ok := cf.Secrets["db_password"]; !ok {
		t.Error("secret 'db_password' not found")
	}
	
	if len(cf.Configs) != 1 {
		t.Errorf("expected 1 config, got %d", len(cf.Configs))
	}
	if _, ok := cf.Configs["db_config"]; !ok {
		t.Error("config 'db_config' not found")
	}
}

func TestParseWithNetworks(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    networks:
      - frontend
      - backend
  api:
    image: myapi
    networks:
      - backend
networks:
  frontend:
    driver: bridge
  backend:
    driver: bridge
    internal: true
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	if len(cf.Networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(cf.Networks))
	}
	
	backend, ok := cf.Networks["backend"]
	if !ok {
		t.Fatal("network 'backend' not found")
	}
	if !backend.Internal {
		t.Error("expected backend network to be internal")
	}
}
