package verification

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("PLATFORM_MANIFEST_TEST_HELPER") != "1" {
		os.Exit(m.Run())
	}
	if len(os.Args) < 3 || os.Args[1] != "run" {
		os.Exit(90)
	}
	calls, err := os.OpenFile(os.Getenv("PLATFORM_MANIFEST_TEST_CALLS"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		os.Exit(91)
	}
	_, err = fmt.Fprintln(calls, os.Args[2])
	closeErr := calls.Close()
	if err != nil || closeErr != nil {
		os.Exit(92)
	}
	if os.Args[2] == "./cmd/platformctl" {
		if len(os.Args) != 8 || os.Args[3] != "render" || os.Args[6] != "-out" {
			os.Exit(93)
		}
		if err := os.WriteFile(os.Args[7], []byte("partial render\n"), 0o600); err != nil {
			os.Exit(94)
		}
		if os.Getenv("PLATFORM_MANIFEST_TEST_FAILURE") == "render" {
			os.Exit(17)
		}
	} else if strings.HasPrefix(os.Args[2], "github.com/yannh/kubeconform/cmd/kubeconform@") {
		if os.Getenv("PLATFORM_MANIFEST_TEST_FAILURE") == "schema" {
			os.Exit(19)
		}
	} else {
		os.Exit(95)
	}
	os.Exit(0)
}

func TestManifestVerificationFailureBoundaries(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	makePath, err := exec.LookPath("make")
	if err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	helper, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	for _, failure := range []string{"render", "schema", "none"} {
		t.Run(failure, func(t *testing.T) {
			dir := t.TempDir()
			bin := filepath.Join(dir, "bin")
			tmp := filepath.Join(dir, "tmp")
			for _, path := range []string{bin, tmp} {
				if err := os.Mkdir(path, 0o700); err != nil {
					t.Fatal(err)
				}
			}
			if err := os.WriteFile(filepath.Join(bin, "go"), helper, 0o700); err != nil {
				t.Fatal(err)
			}
			calls := filepath.Join(dir, "calls")
			command := exec.Command(makePath, "verify-manifests")
			command.Dir = root
			command.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"TMPDIR="+tmp,
				"PLATFORM_MANIFEST_TEST_HELPER=1",
				"PLATFORM_MANIFEST_TEST_CALLS="+calls,
				"PLATFORM_MANIFEST_TEST_FAILURE="+failure,
			)
			output, err := command.CombinedOutput()
			if (err != nil) != (failure != "none") {
				t.Errorf("make error = %v, failure = %s, output = %s", err, failure, output)
			}
			data, err := os.ReadFile(calls)
			if err != nil {
				t.Fatal(err)
			}
			invocations := strings.Fields(string(data))
			wantCalls := 2
			if failure == "render" {
				wantCalls = 1
			}
			if len(invocations) != wantCalls || invocations[0] != "./cmd/platformctl" {
				t.Errorf("unexpected tool calls: %q; want %d starting with renderer", invocations, wantCalls)
			}
			entries, err := os.ReadDir(tmp)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 0 {
				t.Errorf("temporary manifests were not removed: %v", entries)
			}
		})
	}
}
