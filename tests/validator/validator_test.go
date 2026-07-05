package validator_test

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/validator"
)

func TestValidateValidCompose(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
    user: "1000:1000"
    read_only: true
    security_opt:
      - no-new-privileges
    cap_drop:
      - ALL
    environment:
      - NODE_ENV=production
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	if result.ErrorCount > 0 {
		t.Errorf("expected 0 errors, got %d", result.ErrorCount)
		for _, issue := range result.Issues {
			if issue.Severity >= validator.SeverityError {
				t.Logf("  %s: %s", issue.Category, issue.Message)
			}
		}
	}
}

func TestValidateMissingImageAndBuild(t *testing.T) {
	yaml := `
services:
  web:
    ports:
      - "8080:80"
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "missing-field" && issue.Severity == validator.SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about missing image/build")
	}
}

func TestValidatePrivilegedService(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    privileged: true
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "security" && issue.Severity == validator.SeverityCritical {
			if strContains(issue.Message, "privileged") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected critical security issue about privileged mode")
	}
}

func TestValidateRootUser(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    user: root
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "security" && strContains(issue.Message, "root") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about running as root")
	}
}

func TestValidateDangerousCapabilities(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    cap_add:
      - SYS_ADMIN
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "security" && issue.Severity == validator.SeverityCritical {
			if strContains(issue.Message, "SYS_ADMIN") {
				found = true
				break
			}
		}
	}
	if !found {
		t.Error("expected critical security issue about SYS_ADMIN capability")
	}
}

func TestValidateSecretsInEnvironment(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    environment:
      - API_KEY=mysecret
      - DATABASE_PASSWORD=mysecret
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "security" && strContains(issue.Message, "secret") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about secrets in environment")
	}
}

func TestValidateInvalidRestartPolicy(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    restart: invalid-policy
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "invalid-value" && strContains(issue.Message, "restart") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about invalid restart policy")
	}
}

func TestValidateMissingDependency(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    depends_on:
      - nonexistent
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "reference" && strContains(issue.Message, "nonexistent") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about non-existent dependency")
	}
}

func TestValidateSensitiveHostPaths(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "security" && strContains(issue.Message, "docker.sock") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected critical security issue about docker socket mount")
	}
}

func TestValidateNoServices(t *testing.T) {
	yaml := `
version: '3.8'
networks:
  default:
    driver: bridge
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "structure" && issue.Severity == validator.SeverityError {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected error about no services defined")
	}
}

func TestValidateZeroRetriesHealthCheck(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
    healthcheck:
      test: ["CMD", "true"]
      retries: 0
`
	cf, err := parser.ParseString(yaml)
	if err != nil {
		t.Fatalf("ParseString failed: %v", err)
	}
	
	result := validator.Validate(cf)
	
	found := false
	for _, issue := range result.Issues {
		if issue.Category == "healthcheck" && strContains(issue.Message, "retries is 0") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected warning about zero retries")
	}
}

func strContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
