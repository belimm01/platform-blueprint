package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/belimm01/platform-blueprint/internal/claim"
)

var manifestTemplate = template.Must(template.New("manifests").Parse(`apiVersion: v1
kind: Namespace
metadata:
  name: {{ .Namespace }}
  labels:
    platform.belimm.dev/owner: {{ .Owner }}
    platform.belimm.dev/environment: {{ .Environment }}
---
apiVersion: v1
kind: ResourceQuota
metadata:
  name: workload-budget
  namespace: {{ .Namespace }}
spec:
  hard:
    requests.cpu: "4"
    requests.memory: 8Gi
    limits.cpu: "8"
    limits.memory: 16Gi
---
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-ingress
  namespace: {{ .Namespace }}
spec:
  podSelector: {}
  policyTypes: [Ingress]
---
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{ .ApplicationName }}
  namespace: argocd
spec:
  project: default
  source:
    repoURL: {{ .RepositoryYAML }}
    targetRevision: main
    path: deploy/chart
    helm:
      parameters:
        - name: image.repository
          value: {{ .ImageRepositoryYAML }}
        - name: image.tag
          value: {{ .ImageTagYAML }}
        - name: replicaCount
          value: "{{ .Replicas }}"
        - name: service.port
          value: "{{ .Port }}"
        - name: resources.requests.cpu
          value: {{ .CPUYAML }}
        - name: resources.requests.memory
          value: {{ .MemoryYAML }}
  destination:
    server: https://kubernetes.default.svc
    namespace: {{ .Namespace }}
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=false
`))

type view struct {
	claim.ServiceClaim
	Namespace           string
	ApplicationName     string
	RepositoryYAML      string
	ImageRepositoryYAML string
	ImageTagYAML        string
	CPUYAML             string
	MemoryYAML          string
}

func Manifests(c claim.ServiceClaim) ([]byte, error) {
	repository, tag, err := splitImage(c.Image)
	if err != nil {
		return nil, err
	}
	// Stable label traversal preserves byte-identical output if labels are rendered later.
	sortedLabels := make(map[string]string, len(c.Labels))
	keys := make([]string, 0, len(c.Labels))
	for key := range c.Labels {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		sortedLabels[key] = c.Labels[key]
	}
	c.Labels = sortedLabels

	data := view{
		ServiceClaim:        c,
		Namespace:           c.Owner + "-" + c.Environment,
		ApplicationName:     resourceName(c.Owner, c.Name, c.Environment),
		RepositoryYAML:      strconv.Quote(c.Repository),
		ImageRepositoryYAML: strconv.Quote(repository),
		ImageTagYAML:        strconv.Quote(tag),
		CPUYAML:             strconv.Quote(c.CPU),
		MemoryYAML:          strconv.Quote(c.Memory),
	}
	var output bytes.Buffer
	err = manifestTemplate.Execute(&output, data)
	return output.Bytes(), err
}

func splitImage(image string) (string, string, error) {
	lastSlash, lastColon := -1, -1
	for index, character := range image {
		if character == '/' {
			lastSlash = index
		}
		if character == ':' {
			lastColon = index
		}
	}
	if lastColon <= lastSlash || lastColon == len(image)-1 {
		return "", "", fmt.Errorf("image %q must include an immutable tag", image)
	}
	return image[:lastColon], image[lastColon+1:], nil
}

func resourceName(parts ...string) string {
	name := strings.Join(parts, "-")
	if len(name) <= 63 {
		return name
	}
	digest := sha256.Sum256([]byte(name))
	suffix := hex.EncodeToString(digest[:4])
	prefix := strings.TrimRight(name[:63-len(suffix)-1], "-")
	return prefix + "-" + suffix
}
