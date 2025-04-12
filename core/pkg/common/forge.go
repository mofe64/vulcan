package common

import (
	"database/sql"

	"github.com/mofe64/vulcan/config"
	"github.com/sirupsen/logrus"
)

/*
A Forge contains all core dependencies, resources, and configurations required by the application.
It encapsulates shared services like database connections, logging utilities, and configuration objects, making them readily available to all components

Fields:
  - DB: (*sql.DB) Central database connection shared across application.
  - Logger: (*logrus.Logger) Application-wide logger for consistent event tracking.
  - Config: (*config.VulkanConfig) Loaded application configurations.
*/
type forge struct {
	DB     *sql.DB
	Logger *logrus.Logger
	Config *config.VulkanConfig
}

type Forge interface {
	GetDB() *sql.DB
	GetLogger() *logrus.Logger
	GetConfig() *config.VulkanConfig
}

func NewForge(db *sql.DB, logger *logrus.Logger, config *config.VulkanConfig) Forge {
	return &forge{
		DB:     db,
		Logger: logger,
		Config: config,
	}
}

func (f *forge) GetDB() *sql.DB {
	return f.DB
}

func (f *forge) GetLogger() *logrus.Logger {
	return f.Logger
}

func (f *forge) GetConfig() *config.VulkanConfig {
	return f.Config
}
