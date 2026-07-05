package differ_test

import (
	"testing"

	"github.com/EdgarOrtegaRamirez/composeforge/pkg/differ"
	"github.com/EdgarOrtegaRamirez/composeforge/pkg/parser"
)

func TestDiffNoChanges(t *testing.T) {
	yaml := `
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`
	cf1, _ := parser.ParseString(yaml)
	cf2, _ := parser.ParseString(yaml)
	
	diff := differ.Diff(cf1, cf2)
	
	if diff.HasChanges {
		t.Error("expected no changes")
	}
}

func TestDiffAddedService(t *testing.T) {
	yaml1 := `
version: '3.8'
services:
  web:
    image: nginx
`
	yaml2 := `
version: '3.8'
services:
  web:
    image: nginx
  api:
    image: myapi
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	
	if !diff.HasChanges {
		t.Error("expected changes")
	}
	if len(diff.AddedServices) != 1 || diff.AddedServices[0] != "api" {
		t.Errorf("expected added service 'api', got %v", diff.AddedServices)
	}
}

func TestDiffRemovedService(t *testing.T) {
	yaml1 := `
version: '3.8'
services:
  web:
    image: nginx
  api:
    image: myapi
`
	yaml2 := `
version: '3.8'
services:
  web:
    image: nginx
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	
	if !diff.HasChanges {
		t.Error("expected changes")
	}
	if len(diff.RemovedServices) != 1 || diff.RemovedServices[0] != "api" {
		t.Errorf("expected removed service 'api', got %v", diff.RemovedServices)
	}
}

func TestDiffModifiedImage(t *testing.T) {
	yaml1 := `
services:
  web:
    image: nginx:1.0
`
	yaml2 := `
services:
  web:
    image: nginx:2.0
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	
	if !diff.HasChanges {
		t.Error("expected changes")
	}
	if len(diff.ModifiedServices) != 1 || diff.ModifiedServices[0] != "web" {
		t.Errorf("expected modified service 'web', got %v", diff.ModifiedServices)
	}
}

func TestDiffAddedPort(t *testing.T) {
	yaml1 := `
services:
  web:
    image: nginx
`
	yaml2 := `
services:
  web:
    image: nginx
    ports:
      - "8080:80"
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	
	if !diff.HasChanges {
		t.Error("expected changes")
	}
	
	webDiff := findServiceDiff(diff.ServiceDiffs, "web")
	if webDiff == nil {
		t.Fatal("expected web service diff")
	}
	
	found := false
	for _, change := range webDiff.Changes {
		if change.Field == "ports" && change.Type == differ.ChangeModified {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected port change")
	}
}

func TestDiffVersionChanged(t *testing.T) {
	yaml1 := `version: '3.7'
services:
  web:
    image: nginx`
	yaml2 := `version: '3.8'
services:
  web:
    image: nginx`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	
	if !diff.VersionChanged {
		t.Error("expected version change")
	}
	if diff.OldVersion != "3.7" || diff.NewVersion != "3.8" {
		t.Errorf("expected version change 3.7 -> 3.8, got %s -> %s", diff.OldVersion, diff.NewVersion)
	}
}

func TestDiffEnvironmentChanges(t *testing.T) {
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
      FOO: bar
      BAZ: new
      NEW_VAR: value
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	
	if !diff.HasChanges {
		t.Error("expected changes")
	}
	
	webDiff := findServiceDiff(diff.ServiceDiffs, "web")
	if webDiff == nil {
		t.Fatal("expected web service diff")
	}
	
	modifiedFound := false
	addedFound := false
	for _, change := range webDiff.Changes {
		if change.Field == "environment.BAZ" && change.Type == differ.ChangeModified {
			modifiedFound = true
		}
		if change.Field == "environment.NEW_VAR" && change.Type == differ.ChangeAdded {
			addedFound = true
		}
	}
	
	if !modifiedFound {
		t.Error("expected BAZ modification")
	}
	if !addedFound {
		t.Error("expected NEW_VAR addition")
	}
}

func TestFormatDiff(t *testing.T) {
	yaml1 := `
services:
  web:
    image: nginx:1.0
`
	yaml2 := `
services:
  web:
    image: nginx:2.0
  api:
    image: myapi
`
	cf1, _ := parser.ParseString(yaml1)
	cf2, _ := parser.ParseString(yaml2)
	
	diff := differ.Diff(cf1, cf2)
	output := differ.FormatDiff(diff)
	
	if output == "" {
		t.Error("expected non-empty diff output")
	}
	if !contains(output, "Added services") {
		t.Error("expected 'Added services' in output")
	}
	if !contains(output, "Modified: web") {
		t.Error("expected 'Modified: web' in output")
	}
}

func TestFormatDiffNoChanges(t *testing.T) {
	yaml := `
services:
  web:
    image: nginx
`
	cf1, _ := parser.ParseString(yaml)
	cf2, _ := parser.ParseString(yaml)
	
	diff := differ.Diff(cf1, cf2)
	output := differ.FormatDiff(diff)
	
	if output != "No changes detected." {
		t.Errorf("expected 'No changes detected.', got '%s'", output)
	}
}

func findServiceDiff(diffs []differ.ServiceDiff, name string) *differ.ServiceDiff {
	for i := range diffs {
		if diffs[i].Name == name {
			return &diffs[i]
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
