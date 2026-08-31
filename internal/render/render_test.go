package render

import (
	"strings"
	"testing"

	"github.com/belimm01/platform-blueprint/internal/claim"
)

func renderClaim(owner, name string) claim.ServiceClaim {
	return claim.ServiceClaim{Name: name, Owner: owner, Repository: "https://github.com/example/catalog", Image: "ghcr.io/example/catalog:1.2.3", Port: 8080, Environment: "production", Replicas: 3, CPU: "250m", Memory: "256Mi"}
}

func TestManifestsContainPlatformGuardrailsAndGitOpsApplication(t *testing.T) {
	data, err := Manifests(renderClaim("commerce", "catalog-api"))
	if err != nil {
		t.Fatal(err)
	}
	output := string(data)
	for _, want := range []string{"commerce-production", "name: commerce-catalog-api-production-", "kind: ResourceQuota", "kind: NetworkPolicy", "kind: Application", "ghcr.io/example/catalog", "1.2.3"} {
		if !strings.Contains(output, want) {
			t.Errorf("rendered output does not contain %q", want)
		}
	}
}

func TestManifestsRequireTaggedImage(t *testing.T) {
	c := renderClaim("commerce", "catalog-api")
	c.Image = "ghcr.io/example/catalog"
	if _, err := Manifests(c); err == nil {
		t.Fatal("expected untagged image to be rejected")
	}
}

func TestApplicationNamesAreUniqueAcrossOwnersAndHyphenatedComponents(t *testing.T) {
	commerce, err := Manifests(renderClaim("commerce", "api"))
	if err != nil {
		t.Fatal(err)
	}
	identity, err := Manifests(renderClaim("identity", "api"))
	if err != nil {
		t.Fatal(err)
	}
	ambiguousA := resourceName("a-b", "c", "production")
	ambiguousB := resourceName("a", "b-c", "production")
	if string(commerce) == string(identity) || ambiguousA == ambiguousB {
		t.Fatal("distinct claim tuples produced a colliding application name")
	}
	if !strings.Contains(string(commerce), "name: commerce-api-production-") || !strings.Contains(string(identity), "name: identity-api-production-") {
		t.Fatal("application names do not retain a readable owner prefix")
	}
}

func TestResourceNameIsStableAndDNSLengthSafe(t *testing.T) {
	name := resourceName(strings.Repeat("a", 40), strings.Repeat("b", 40), "development")
	if len(name) > 63 {
		t.Fatalf("resource name has %d characters", len(name))
	}
	if name != resourceName(strings.Repeat("a", 40), strings.Repeat("b", 40), "development") {
		t.Fatal("resource name is not deterministic")
	}
}
