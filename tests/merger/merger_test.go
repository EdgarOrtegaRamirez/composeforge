package merger_test

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/merger"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

func TestMergeBasic(t *testing.T) {
	yaml1 := `
version: '3.7'
services:
  web:
    image: nginx:1.0
    ports:
      - "8080:80"
`
	yaml2 := `
version: '3.8'
services:
  web:
    image: nginx:2.0
  api:
    image: myapi
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)

	merged, err := merger.Merge(cf1, cf2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Version should be overridden
	if merged.Version != "3.8" {
		t.Errorf("expected version '3.8', got '%s'", merged.Version)
	}

	// Should have both services
	if len(merged.Services) != 2 {
		t.Errorf("expected 2 services, got %d", len(merged.Services))
	}

	// web image should be overridden
	if merged.Services["web"].Image != "nginx:2.0" {
		t.Errorf("expected web image 'nginx:2.0', got '%s'", merged.Services["web"].Image)
	}

	// api should be added
	if _, ok := merged.Services["api"]; !ok {
		t.Error("expected api service to be added")
	}
}

func TestMergePreservesBasePorts(t *testing.T) {
	yaml1 := `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`
	yaml2 := `
services:
  web:
    image: nginx:2.0
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)

	merged, err := merger.Merge(cf1, cf2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	// Ports should be preserved from base
	if len(merged.Services["web"].Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(merged.Services["web"].Ports))
	}
}

func TestMergeEnvironmentMerging(t *testing.T) {
	yaml1 := `
services:
  web:
    image: nginx
    environment:
      FOO: bar
      BAZ: old
`
	yaml2 := `
services:
  web:
    image: nginx
    environment:
      BAZ: new
      QUX: value
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)

	merged, err := merger.Merge(cf1, cf2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	env := parser.NormalizeEnvironment(merged.Services["web"].Environment)

	// FOO should be preserved from base
	if env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got FOO=%s", env["FOO"])
	}
	// BAZ should be overridden
	if env["BAZ"] != "new" {
		t.Errorf("expected BAZ=new, got BAZ=%s", env["BAZ"])
	}
	// QUX should be added
	if env["QUX"] != "value" {
		t.Errorf("expected QUX=value, got QUX=%s", env["QUX"])
	}
}

func TestMergeNetworks(t *testing.T) {
	yaml1 := `
services:
  web:
    image: nginx
networks:
  frontend:
    driver: bridge
`
	yaml2 := `
services:
  web:
    image: nginx
networks:
  backend:
    driver: bridge
    internal: true
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)

	merged, err := merger.Merge(cf1, cf2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if len(merged.Networks) != 2 {
		t.Errorf("expected 2 networks, got %d", len(merged.Networks))
	}

	if _, ok := merged.Networks["frontend"]; !ok {
		t.Error("expected frontend network")
	}
	if _, ok := merged.Networks["backend"]; !ok {
		t.Error("expected backend network")
	}
}

func TestMergeVolumes(t *testing.T) {
	yaml1 := `
services:
  db:
    image: postgres
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
`
	yaml2 := `
services:
  db:
    image: postgres
volumes:
  backup:
    driver: local
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)

	merged, err := merger.Merge(cf1, cf2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if len(merged.Volumes) != 2 {
		t.Errorf("expected 2 volumes, got %d", len(merged.Volumes))
	}
}

func TestMergeEmptyFiles(t *testing.T) {
	_, err := merger.Merge()
	if err == nil {
		t.Error("expected error when merging no files")
	}
}

func TestMergeSingleFile(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    image: nginx
`
	cf, _ := parser.ParseString(yaml)

	merged, err := merger.Merge(cf)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if merged.Version != "3.8" {
		t.Errorf("expected version '3.8', got '%s'", merged.Version)
	}
}

func TestMergeMultipleFiles(t *testing.T) {
	yaml1 := `
version: '3.7'
services:
  web:
    image: nginx:1.0
`
	yaml2 := `
services:
  web:
    image: nginx:2.0
    ports:
      - "8080:80"
`
	yaml3 := `
services:
  web:
    environment:
      NODE_ENV: production
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	cf3, _ := parser.ParseString(yaml3)

	merged, err := merger.Merge(cf1, cf2, cf3)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	web := merged.Services["web"]

	// Image from second file
	if web.Image != "nginx:2.0" {
		t.Errorf("expected image 'nginx:2.0', got '%s'", web.Image)
	}

	// Ports from second file
	if len(web.Ports) != 1 {
		t.Errorf("expected 1 port, got %d", len(web.Ports))
	}

	// Environment from third file
	env := parser.NormalizeEnvironment(web.Environment)
	if env["NODE_ENV"] != "production" {
		t.Errorf("expected NODE_ENV=production, got %s", env["NODE_ENV"])
	}
}

func TestMergeName(t *testing.T) {
	yaml1 := `
name: myapp
services:
  web:
    image: nginx
`
	yaml2 := `
name: myapp-prod
services:
  web:
    image: nginx
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)

	merged, err := merger.Merge(cf1, cf2)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}

	if merged.Name != "myapp-prod" {
		t.Errorf("expected name 'myapp-prod', got '%s'", merged.Name)
	}
}
