package auth

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type GroupMapping struct {
	Group   string `yaml:"group" json:"group"`
	Role    Role   `yaml:"role" json:"role"`
	Scope   Scope  `yaml:"scope" json:"scope"`
	Project string `yaml:"project,omitempty" json:"project,omitempty"`
}

type GroupMappingConfig struct {
	Mappings []GroupMapping `yaml:"mappings" json:"mappings"`
}

func LoadGroupMappingFile(path string) (GroupMappingConfig, error) {
	if path == "" {
		return GroupMappingConfig{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return GroupMappingConfig{}, fmt.Errorf("read auth group mapping file: %w", err)
	}
	var config GroupMappingConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return GroupMappingConfig{}, fmt.Errorf("parse auth group mapping file: %w", err)
	}
	return config, nil
}

func RolesForGroups(groups []string, config GroupMappingConfig) []Role {
	seenGroups := map[string]bool{}
	for _, group := range groups {
		seenGroups[group] = true
	}
	seenRoles := map[Role]bool{}
	roles := []Role{}
	for _, mapping := range config.Mappings {
		if mapping.Group == "" || !seenGroups[mapping.Group] || mapping.Role == "" || seenRoles[mapping.Role] {
			continue
		}
		seenRoles[mapping.Role] = true
		roles = append(roles, mapping.Role)
	}
	if len(roles) == 0 {
		roles = append(roles, RoleViewer)
	}
	return roles
}
