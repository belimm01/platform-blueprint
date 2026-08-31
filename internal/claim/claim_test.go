package claim

import (
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
