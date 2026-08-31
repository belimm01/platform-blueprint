package render

import (
	"strings"
	"testing"

	"github.com/belimm01/platform-blueprint/internal/claim"
)

func TestManifestsContainPlatformGuardrailsAndGitOpsApplication(t *testing.T) {
	c := claim.ServiceClaim{Name: "catalog-api", Owner: "commerce", Repository: "https://github.com/example/catalog", Image: "ghcr.io/example/catalog:1.2.3", Port: 8080, Environment: "production", Replicas: 3, CPU: "250m", Memory: "256Mi"}
	data, err := Manifests(c)
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	for _, want := range []string{"commerce-production", "kind: ResourceQuota", "kind: NetworkPolicy", "kind: Application", "ghcr.io/example/catalog", "1.2.3"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered output does not contain %q", want)
		}
	}
}

func TestManifestsRequireTaggedImage(t *testing.T) {
	c := claim.ServiceClaim{Image: "ghcr.io/example/catalog"}
	if _, err := Manifests(c); err == nil {
		t.Fatal("expected untagged image to be rejected")
	}
}
