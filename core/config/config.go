package config

type VulkanConfig struct {
	DBType       string `json:"dbType"` // "sqlite", "postgres", "mysql"
	DBConnection string `json:"dbConnection"`
}
