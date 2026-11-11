package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
{{- range .ClaimLabels }}
    {{ .Key }}: {{ .ValueYAML }}
{{- end }}
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

type labelLine struct {
	Key       string
	ValueYAML string
}

type view struct {
	claim.ServiceClaim
	Namespace           string
	ApplicationName     string
	RepositoryYAML      string
	ImageRepositoryYAML string
	ImageTagYAML        string
	CPUYAML             string
	MemoryYAML          string
	ClaimLabels         []labelLine
}

func Manifests(c claim.ServiceClaim) ([]byte, error) {
	repository, tag, err := claim.SplitImage(c.Image)
	if err != nil {
		return nil, err
	}
	claimLabels := make([]labelLine, 0, len(c.Labels))
	for _, key := range c.OrderedLabelKeys() {
		claimLabels = append(claimLabels, labelLine{Key: key, ValueYAML: strconv.Quote(c.Labels[key])})
	}

	data := view{
		ServiceClaim:        c,
		Namespace:           c.Owner + "-" + c.Environment,
		ApplicationName:     resourceName(c.Owner, c.Name, c.Environment),
		RepositoryYAML:      strconv.Quote(c.Repository),
		ImageRepositoryYAML: strconv.Quote(repository),
		ImageTagYAML:        strconv.Quote(tag),
		CPUYAML:             strconv.Quote(c.CPU),
		MemoryYAML:          strconv.Quote(c.Memory),
		ClaimLabels:         claimLabels,
	}
	var output bytes.Buffer
	err = manifestTemplate.Execute(&output, data)
	return output.Bytes(), err
}

func resourceName(parts ...string) string {
	readable := strings.Join(parts, "-")
	hashInput := strings.Join(parts, "\x00")
	digest := sha256.Sum256([]byte(hashInput))
	suffix := hex.EncodeToString(digest[:16])
	maxPrefixLength := 63 - len(suffix) - 1
	if len(readable) > maxPrefixLength {
		readable = readable[:maxPrefixLength]
	}
	prefix := strings.TrimRight(readable, "-")
	return prefix + "-" + suffix
}
