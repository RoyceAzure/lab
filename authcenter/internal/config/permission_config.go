package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Permission struct {
	ID       int32  `yaml:"id"`
	Name     string `yaml:"name"`
	Resource string `yaml:"resource"`
	Actions  string `yaml:"actions"`
}

type RolePermission struct {
	Name        string  `yaml:"name"`
	RoleID      int32   `yaml:"role_id"`
	Permissions []int32 `yaml:"permissions"`
}

type PermissionConfig struct {
	Permissions    []Permission     `yaml:"permissions"`
	RolePermission []RolePermission `yaml:"role_permissions"`
}

// yaml path : github.com/RoyceAzure/sexy_stock/permission.yaml
func LoadPermissionConfig(path string) (*PermissionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	config := &PermissionConfig{}
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return nil, err
	}

	return config, nil
}
