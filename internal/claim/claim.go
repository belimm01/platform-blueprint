package claim

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var dnsLabel = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)
var cpuQuantity = regexp.MustCompile(`^(?:[1-9][0-9]*|[1-9][0-9]*m|0\.[0-9]+)$`)
var memoryQuantity = regexp.MustCompile(`^[1-9][0-9]*(?:Ki|Mi|Gi|Ti)$`)

var allowedEnvironments = map[string]bool{
	"development": true,
	"staging":     true,
	"production":  true,
}

// ServiceClaim is the small, stable platform API presented to application teams.
type ServiceClaim struct {
	Name        string            `json:"name"`
	Owner       string            `json:"owner"`
	Repository  string            `json:"repository"`
	Image       string            `json:"image"`
	Port        int               `json:"port"`
	Environment string            `json:"environment"`
	Replicas    int               `json:"replicas"`
	CPU         string            `json:"cpu"`
	Memory      string            `json:"memory"`
	Public      bool              `json:"public"`
	Labels      map[string]string `json:"labels"`
}

type serviceClaimDocument struct {
	Name        string            `json:"name"`
	Owner       string            `json:"owner"`
	Repository  string            `json:"repository"`
	Image       string            `json:"image"`
	Port        int               `json:"port"`
	Environment string            `json:"environment"`
	Replicas    *int              `json:"replicas"`
	CPU         string            `json:"cpu"`
	Memory      string            `json:"memory"`
	Public      bool              `json:"public"`
	Labels      map[string]string `json:"labels"`
}

func Load(path string) (ServiceClaim, error) {
	file, err := os.Open(path)
	if err != nil {
		return ServiceClaim{}, fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var document serviceClaimDocument
	if err := decoder.Decode(&document); err != nil {
		return ServiceClaim{}, fmt.Errorf("decode JSON: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return ServiceClaim{}, errors.New("decode JSON: multiple values are not allowed")
		}
		return ServiceClaim{}, fmt.Errorf("decode JSON: %w", err)
	}

	replicas := 1
	if document.Replicas != nil {
		replicas = *document.Replicas
	}
	claim := ServiceClaim{
		Name: document.Name, Owner: document.Owner, Repository: document.Repository,
		Image: document.Image, Port: document.Port, Environment: document.Environment,
		Replicas: replicas, CPU: document.CPU, Memory: document.Memory,
		Public: document.Public, Labels: document.Labels,
	}
	claim.applyDefaults()
	return claim, nil
}

func (c *ServiceClaim) applyDefaults() {
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.CPU == "" {
		c.CPU = "100m"
	}
	if c.Memory == "" {
		c.Memory = "128Mi"
	}
}

func (c ServiceClaim) Validate() error {
	var problems []string
	if len(c.Name) > 40 || !dnsLabel.MatchString(c.Name) {
		problems = append(problems, "name must be a DNS label of at most 40 characters")
	}
	if len(c.Owner) > 40 || !dnsLabel.MatchString(c.Owner) {
		problems = append(problems, "owner must be a DNS label of at most 40 characters")
	}
	if !validRepository(c.Repository) {
		problems = append(problems, "repository must be an https://github.com/<owner>/<repository> URL without credentials or control characters")
	}
	if c.Image == "" || strings.ContainsAny(c.Image, " \t\r\n") {
		problems = append(problems, "image must be a non-empty container reference without whitespace")
	}
	if c.Port < 1 || c.Port > 65535 {
		problems = append(problems, "port must be between 1 and 65535")
	}
	if !allowedEnvironments[c.Environment] {
		problems = append(problems, "environment must be development, staging, or production")
	}
	if c.Replicas < 1 || c.Replicas > 20 {
		problems = append(problems, "replicas must be between 1 and 20")
	}
	if !cpuQuantity.MatchString(c.CPU) {
		problems = append(problems, "cpu must be a positive CPU quantity such as 250m, 1, or 0.5")
	}
	if !memoryQuantity.MatchString(c.Memory) {
		problems = append(problems, "memory must be a positive binary quantity such as 128Mi or 2Gi")
	}
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}

func validRepository(value string) bool {
	if strings.ContainsAny(value, "\r\n\t") {
		return false
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host != "github.com" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}
