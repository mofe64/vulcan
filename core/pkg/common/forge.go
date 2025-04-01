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
type Forge struct {
	DB     *sql.DB
	Logger *logrus.Logger
	Config *config.VulkanConfig
}
