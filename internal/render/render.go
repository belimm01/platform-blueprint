package render

import (
	"bytes"
	"fmt"
	"sort"
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
  name: {{ .Name }}-{{ .Environment }}
  namespace: argocd
spec:
  project: default
  source:
    repoURL: {{ .Repository }}
    targetRevision: main
    path: deploy/chart
    helm:
      parameters:
        - name: image.repository
          value: {{ .ImageRepository }}
        - name: image.tag
          value: {{ .ImageTag }}
        - name: replicaCount
          value: "{{ .Replicas }}"
        - name: service.port
          value: "{{ .Port }}"
        - name: resources.requests.cpu
          value: {{ .CPU }}
        - name: resources.requests.memory
          value: {{ .Memory }}
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
	Namespace       string
	ImageRepository string
	ImageTag        string
}

func Manifests(c claim.ServiceClaim) ([]byte, error) {
	repo, tag, err := splitImage(c.Image)
	if err != nil {
		return nil, err
	}
	// Stable output makes generated GitOps changes reviewable and reproducible.
	sortedLabels := make(map[string]string, len(c.Labels))
	keys := make([]string, 0, len(c.Labels))
	for k := range c.Labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sortedLabels[k] = c.Labels[k]
	}
	c.Labels = sortedLabels

	var out bytes.Buffer
	err = manifestTemplate.Execute(&out, view{ServiceClaim: c, Namespace: c.Owner + "-" + c.Environment, ImageRepository: repo, ImageTag: tag})
	return out.Bytes(), err
}

func splitImage(image string) (string, string, error) {
	lastSlash, lastColon := -1, -1
	for i, r := range image {
		if r == '/' {
			lastSlash = i
		}
		if r == ':' {
			lastColon = i
		}
	}
	if lastColon <= lastSlash || lastColon == len(image)-1 {
		return "", "", fmt.Errorf("image %q must include an immutable tag", image)
	}
	return image[:lastColon], image[lastColon+1:], nil
}
