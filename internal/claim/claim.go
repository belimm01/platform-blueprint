package claim

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var dnsLabel = regexp.MustCompile(`^[a-z]([-a-z0-9]*[a-z0-9])?$`)

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

func Load(path string) (ServiceClaim, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ServiceClaim{}, fmt.Errorf("read %s: %w", path, err)
	}
	var c ServiceClaim
	if err := json.Unmarshal(data, &c); err != nil {
		return ServiceClaim{}, fmt.Errorf("decode JSON: %w", err)
	}
	c.applyDefaults()
	return c, nil
}

func (c *ServiceClaim) applyDefaults() {
	if c.Environment == "" {
		c.Environment = "development"
	}
	if c.Replicas == 0 {
		c.Replicas = 1
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
	if !strings.HasPrefix(c.Repository, "https://github.com/") {
		problems = append(problems, "repository must be an https://github.com URL")
	}
	if c.Image == "" || strings.ContainsAny(c.Image, " \t\n") {
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
	if len(problems) > 0 {
		return errors.New(strings.Join(problems, "; "))
	}
	return nil
}
