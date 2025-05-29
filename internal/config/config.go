package config

import "github.com/caarlos0/env"

type VulkanConfig struct {
	DBURL            string `env:"VULKAN_DATABASE_URL,required"`
	OIDCJWKSURL      string `env:"OIDC_JWKS_URL,required"`
	VulkanServerPort string `env:"VULKAN_PORT"           default:"9021"`
	InCluster        bool   `env:"K8S_IN_CLUSTER" default:"false"`
	LOG_LEVEL        string `env:"LOG_LEVEL" default:"info"`
}

// Load parses the environment into a Config struct and returns it.
// Call this once at startup—preferably right at the top of main().
func Load() (*VulkanConfig, error) {
	var cfg VulkanConfig
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
