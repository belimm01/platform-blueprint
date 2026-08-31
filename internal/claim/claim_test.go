package claim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validClaim() ServiceClaim {
	return ServiceClaim{Name: "catalog-api", Owner: "commerce", Repository: "https://github.com/example/catalog", Image: "ghcr.io/example/catalog:1.2.3", Port: 8080, Environment: "production", Replicas: 3, CPU: "250m", Memory: "256Mi"}
}

func TestValidateAcceptsProductionClaim(t *testing.T) {
	if err := validClaim().Validate(); err != nil {
		t.Fatalf("expected valid claim, got %v", err)
	}
}

func TestValidateReportsAllContractViolations(t *testing.T) {
	c := validClaim()
	c.Name = "Bad_Name"
	c.Repository = "git@example/catalog"
	c.Port = 70000
	c.Environment = "qa"
	c.Replicas = 100

	err := c.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	for _, want := range []string{"name", "repository", "port", "environment", "replicas"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestValidateRejectsControlCharactersAndInvalidQuantities(t *testing.T) {
	for name, mutate := range map[string]func(*ServiceClaim){
		"repository newline": func(c *ServiceClaim) { c.Repository += "\nchart: injected" },
		"cpu newline":        func(c *ServiceClaim) { c.CPU += "\nvalue: injected" },
		"memory newline":     func(c *ServiceClaim) { c.Memory += "\nvalue: injected" },
	} {
		t.Run(name, func(t *testing.T) {
			c := validClaim()
			mutate(&c)
			if err := c.Validate(); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestValidateRejectsZeroCPUQuantities(t *testing.T) {
	for _, value := range []string{"0", "0m", "0.0", "0.00"} {
		c := validClaim()
		c.CPU = value
		if err := c.Validate(); err == nil {
			t.Errorf("expected CPU quantity %q to be rejected", value)
		}
	}
	for _, value := range []string{"1m", "0.001", "0.5", "1"} {
		c := validClaim()
		c.CPU = value
		if err := c.Validate(); err != nil {
			t.Errorf("expected CPU quantity %q to be accepted: %v", value, err)
		}
	}
}

func TestLoadRejectsUnknownFieldsAndExplicitZeroReplicas(t *testing.T) {
	for name, document := range map[string]string{
		"unknown field": `{"name":"api","owner":"team","repository":"https://github.com/example/api","image":"ghcr.io/example/api:1","port":8080,"replica":7}`,
		"zero replicas": `{"name":"api","owner":"team","repository":"https://github.com/example/api","image":"ghcr.io/example/api:1","port":8080,"replicas":0}`,
	} {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "claim.json")
			if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
				t.Fatal(err)
			}
			c, err := Load(path)
			if err == nil {
				err = c.Validate()
			}
			if err == nil {
				t.Fatal("expected claim to be rejected")
			}
		})
	}
}

func TestLoadDefaultsMissingReplicas(t *testing.T) {
	path := filepath.Join(t.TempDir(), "claim.json")
	document := `{"name":"api","owner":"team","repository":"https://github.com/example/api","image":"ghcr.io/example/api:1","port":8080}`
	if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if c.Replicas != 1 {
		t.Fatalf("expected default replicas 1, got %d", c.Replicas)
	}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
}
