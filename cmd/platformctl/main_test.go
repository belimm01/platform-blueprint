package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCLIOutputBoundaries(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "platformctl")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	run := func(t *testing.T, wantCode int, args ...string) (string, string) {
		t.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, binary, args...)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			var exit *exec.ExitError
			if !errors.As(err, &exit) {
				t.Fatalf("run CLI: %v", err)
			}
			code = exit.ExitCode()
		}
		if code != wantCode {
			t.Fatalf("exit code = %d, want %d; stdout=%q stderr=%q", code, wantCode, stdout.String(), stderr.String())
		}
		return stdout.String(), stderr.String()
	}

	valid := `{"name":"api","owner":"team","repository":"https://github.com/example/api","image":"ghcr.io/example/api:1.2.3","port":8080}`
	writeClaim := func(t *testing.T, document string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "claim.json")
		if err := os.WriteFile(path, []byte(document), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}

	t.Run("validate and render defaults", func(t *testing.T) {
		path := writeClaim(t, valid)
		stdout, stderr := run(t, 0, "validate", "-file", path)
		if stdout != "valid service claim: team/api\n" || stderr != "" {
			t.Fatalf("unexpected validation output: %q, %q", stdout, stderr)
		}
		manifest, stderr := run(t, 0, "render", "-file", path)
		if stderr != "" {
			t.Fatalf("render stderr = %q", stderr)
		}
		for _, want := range []string{"kind: Namespace", "name: team-development", "kind: ResourceQuota", "kind: NetworkPolicy", "kind: Application", "value: \"1.2.3\"", "value: \"100m\"", "value: \"128Mi\"", "name: replicaCount\n          value: \"1\""} {
			if !strings.Contains(manifest, want) {
				t.Errorf("manifest missing %q", want)
			}
		}
		output := filepath.Join(t.TempDir(), "manifests.yaml")
		if err := os.WriteFile(output, []byte("previous manifest\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		stdout, stderr = run(t, 0, "render", "-file", path, "-out", output)
		if stdout != "" || stderr != "" {
			t.Fatalf("file render output: %q, %q", stdout, stderr)
		}
		data, err := os.ReadFile(output)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != manifest {
			t.Fatal("file output differs from stdout output")
		}
	})

	for _, test := range []struct {
		name, document, diagnostic string
	}{
		{"malformed JSON", `{`, "invalid service claim: decode JSON:"},
		{"unknown field", strings.Replace(valid, `"port":8080`, `"port":8080,"replica":2`, 1), "invalid service claim: decode JSON:"},
		{"invalid port", strings.Replace(valid, `8080`, `0`, 1), "invalid service claim: port must"},
		{"untagged image", strings.Replace(valid, `api:1.2.3`, `api`, 1), "render manifests:"},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := writeClaim(t, test.document)
			for _, existing := range []bool{false, true} {
				output := filepath.Join(t.TempDir(), "manifests.yaml")
				original := []byte("previous manifest\n")
				if existing {
					if err := os.WriteFile(output, original, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				stdout, stderr := run(t, 1, "render", "-file", path, "-out", output)
				if stdout != "" || !strings.Contains(stderr, test.diagnostic) {
					t.Fatalf("unexpected failure output: %q, %q", stdout, stderr)
				}
				data, err := os.ReadFile(output)
				if existing {
					if err != nil || !bytes.Equal(data, original) {
						t.Fatalf("failed render changed existing output: %q, %v", data, err)
					}
				} else if !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("failed render created output: %q, %v", data, err)
				}
			}
			stdout, stderr := run(t, 1, "render", "-file", path)
			if stdout != "" || !strings.Contains(stderr, test.diagnostic) {
				t.Fatalf("failed render leaked stdout or lost diagnostic: %q, %q", stdout, stderr)
			}
		})
	}

	t.Run("output write failure", func(t *testing.T) {
		path := writeClaim(t, valid)
		stdout, stderr := run(t, 1, "render", "-file", path, "-out", t.TempDir())
		if stdout != "" || !strings.Contains(stderr, "write manifests:") {
			t.Fatalf("unexpected write failure output: %q, %q", stdout, stderr)
		}
	})
}
