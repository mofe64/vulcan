package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/google/uuid"
)

// getBaseDir returns an application-specific base directory based on the operating system.
// - Windows: %LOCALAPPDATA%\vulkan
// - macOS: /usr/local/var/vulkan
// - Linux/Unix: /var/vulkan
func getBaseDir() (string, error) {
	var baseDir string
	appFolder := AppName

	switch runtime.GOOS {
	case "windows":
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA environment variable not found")
		}
		baseDir = filepath.Join(localAppData, appFolder)
	case "darwin":
		baseDir = filepath.Join("/usr/local/var", appFolder)
	default: // Linux and other Unix-like OS
		baseDir = filepath.Join("/var", appFolder)
	}

	// Ensure the base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create base directory: %v", err)
	}

	return baseDir, nil
}

// CreateDeploymentId generates a unique deployment ID using a timestamp and UUID.
func CreateDeploymentId() string {
	return fmt.Sprintf("%s-%s", time.Now().Format("20060102-150405"), uuid.New().String())
}

// CreateSourceDir creates and returns a uniquely named temporary source directory
// for cloning a Git repository.
// Directory structure:
// {baseDir}/deployments/{deploymentID}/source
func CreateSourceDir(deploymentID string) (string, error) {
	baseDir, err := getBaseDir()
	if err != nil {
		return "", fmt.Errorf("failed to get base directory: %v", err)
	}

	sourceDir := filepath.Join(baseDir, "deployments", deploymentID, "source")

	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create source directory: %v", err)
	}

	return sourceDir, nil
}

// CreateBuildDir creates and returns a uniquely named temporary build directory
// for storing build artifacts or compiled images.
// Directory structure:
// {baseDir}/deployments/{deploymentID}/build
func CreateBuildDir(deploymentID string) (string, error) {
	baseDir, err := getBaseDir()
	if err != nil {
		return "", fmt.Errorf("failed to get base directory: %v", err)
	}

	buildDir := filepath.Join(baseDir, "deployments", deploymentID, "build")

	if err := os.MkdirAll(buildDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create build directory: %v", err)
	}

	return buildDir, nil
}

// CleanupDeployment cleans up all resources (source, build, and any other directories)
// associated with a given deployment ID.
// It removes the entire directory: {baseDir}/deployments/{deploymentID}
func CleanupDeployment(deploymentID string) error {
	baseDir, err := getBaseDir()
	if err != nil {
		return fmt.Errorf("failed to get base directory: %v", err)
	}

	deploymentRoot := filepath.Join(baseDir, "deployments", deploymentID)

	return os.RemoveAll(deploymentRoot)
}

// checkGitInstalled checks if Git is installed on the system.
func CheckGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
