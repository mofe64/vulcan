package config

import (
	"time"

	"github.com/caarlos0/env"
)

// OPA_URL has been set in the environment variable OPA_URL
// this env var is loaded into container by the helm deployment.yaml file for the api server
type VulkanConfig struct {
	DBURL              string        `env:"VULKAN_DATABASE_URL,required"`
	OIDCJWKSURL        string        `env:"OIDC_JWKS_URL,required"`
	VulkanServerPort   string        `env:"VULKAN_PORT"           default:"9021"`
	InCluster          bool          `env:"K8S_IN_CLUSTER" default:"false"`
	LOG_LEVEL          string        `env:"LOG_LEVEL" default:"info"`
	OpaUrl             string        `env:"OPA_URL"`
	OpaPolicy_Path     string        `env:"OPA_POLICY_PATH"`
	OpaReqTIMEOUT      time.Duration `env:"OPA_REQ_TIMEOUT" default:"300"`
	NATS_URL           string        `env:"NATS_URL"`
	DEX_URL            string        `env:"DEX_URL,required"`
	OIDC_CLIENT_ID     string        `env:"VULKAN_OIDC_CLIENT_ID,required"`
	OIDC_CLIENT_SECRET string        `env:"VULKAN_OIDC_CLIENT_SECRET,required"`
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
