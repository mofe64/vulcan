package buildmanager

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func DetermineProjectType(dir string) (string, error) {
	isGo, err := isGoProject(dir)
	if err != nil {
		return "", fmt.Errorf("error checking for Go project: %w", err)
	}
	if isGo {
		return "go", nil
	}
	isNode, err := isNodeProject(dir)
	if err != nil {
		return "", fmt.Errorf("error checking for Node.js project: %w", err)
	}
	if isNode {
		return "node", nil
	}
	isJava, projectType, err := DetectJavaProject(dir)
	if err != nil {
		return "", fmt.Errorf("error checking for Java project: %w", err)
	}
	if isJava {
		return projectType, nil
	}
	isPython, err := DetectPythonProject(dir)
	if err != nil {
		return "", fmt.Errorf("error checking for Python project: %w", err)
	}
	if isPython {
		return "python", nil
	}
	// If no known project type is detected, return "unknown".
	return "unknown", nil
}

// isGoProject checks if the provided directory contains files or directories
// that indicate a Go project. It returns true if a Go project marker is found,
// or false if not. An error is returned if any filesystem operation fails.
func isGoProject(dir string) (bool, error) {
	// Convert the provided directory path into an absolute path.
	// This ensures we have a consistent and complete reference to the directory.
	build, err := filepath.Abs(dir)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Define the list of file paths (relative to the build directory) that,
	// if present, indicate a Go project.
	markers := []string{
		"go.mod",                               // Go modules
		"Gopkg.lock",                           // Dep
		filepath.Join("Godeps", "Godeps.json"), // Godeps

	}

	// Iterate over each marker and check if the file exists.
	for _, marker := range markers {
		markerPath := filepath.Join(build, marker)
		if fileExists(markerPath) {
			// If any marker file exists, a Go project is detected.
			return true, nil
		}
	}

	// Check for gb-style projects: a "src" directory containing at least one
	// .go file in a subdirectory (i.e., not directly in "src").
	srcPath := filepath.Join(build, "src")
	if dirExists(srcPath) {
		// foundGoFile will be set to true if a .go file is found.
		foundGoFile := false

		// Walk the "src" directory to search for .go files.
		err := filepath.WalkDir(srcPath, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				// If an error occurs during the walk, propagate it.
				return err
			}
			// We are interested only in files (not directories).
			if !d.IsDir() && filepath.Ext(d.Name()) == ".go" {
				// Calculate the relative path from the "src" directory.
				rel, err := filepath.Rel(srcPath, path)
				if err != nil {
					return err
				}
				// Check if the file is in a subdirectory by ensuring that the
				// relative path contains a path separator.
				if strings.Contains(rel, string(os.PathSeparator)) {
					foundGoFile = true
					// Return io.EOF to signal that we can stop walking the directory.
					return io.EOF
				}
			}
			return nil
		})

		// If we broke out of the walk with io.EOF, that means we found a file.
		if err == io.EOF && foundGoFile {
			return true, nil
		}
		// If an actual error occurred during walking, return it.
		if err != nil && err != io.EOF {
			return false, fmt.Errorf("error while scanning 'src' directory: %w", err)
		}
	}

	// No Go project markers were found in the provided directory.
	return false, nil
}

// isNodeProject checks if the given directory is a Node.js project.
// It does this by verifying that the directory contains a package.json file,
// and by ensuring that package.json is not ignored via a .gitignore file.
// The function returns true if the directory appears to be a valid Node.js project.
func isNodeProject(buildDir string) (bool, error) {
	// Convert the provided buildDir into an absolute path.
	// This ensures that our file operations have a complete path reference.
	absPath, err := filepath.Abs(buildDir)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Construct the full path for package.json.
	pkgJSONPath := filepath.Join(absPath, "package.json")

	// Check if package.json exists in the project root.
	if fileExists(pkgJSONPath) {
		// If the file exists, this is a Node.js project.
		return true, nil
	}

	// If package.json is missing, check if it is explicitly ignored via .gitignore.
	gitignorePath := filepath.Join(absPath, ".gitignore")
	if fileExists(gitignorePath) {
		// Open the .gitignore file.
		file, err := os.Open(gitignorePath)
		if err != nil {
			return false, fmt.Errorf("failed to open .gitignore: %w", err)
		}
		defer file.Close()

		// Use a scanner to read the file line by line.
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			// Check if any line exactly matches "package.json".
			if scanner.Text() == "package.json" {
				return false, fmt.Errorf("vulkan Error: 'package.json' is listed in '.gitignore'. Ensure that your Node.js project file is tracked")
			}
		}
		// Check for any scanning error.
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("error reading .gitignore: %w", err)
		}
	}

	// If no package.json is found and no ignore rule explains it, the directory is not a Node.js project.
	return false, nil
}

// detectJavaProject checks if the specified buildDir contains common Maven or Gradle
// configuration files. It returns a string indicating the type of Java project detected
// ("Java-Maven" or "Java-Gradle") and a nil error if a known configuration file is found.
// If no valid configuration file is detected, an error is returned with details.
func DetectJavaProject(buildDir string) (bool, string, error) {
	// Convert the provided directory into an absolute path.
	absBuildDir, err := filepath.Abs(buildDir)
	if err != nil {
		return false, "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Define slices of file names that indicate Maven or Gradle configurations.
	mavenFiles := []string{
		"pom.xml",    // Standard Maven configuration file.
		"pom.atom",   // Alternative Maven configuration file.
		"pom.clj",    // Maven configuration with Clojure support.
		"pom.groovy", // Maven configuration using Groovy.
		"pom.rb",     // Maven configuration with Ruby support.
		"pom.scala",  // Maven configuration with Scala support.
		"pom.yaml",   // Maven configuration in YAML format.
		"pom.yml",    // Another variant of Maven YAML configuration.
	}
	gradleFiles := []string{
		"build.gradle",     // Standard Gradle build file.
		"build.gradle.kts", // Gradle build file using Kotlin DSL.
	}

	// Helper function to check if any file in the list exists in the directory.
	containsBuildFile := func(dir string, files []string) bool {
		for _, file := range files {
			// Build the full file path.
			path := filepath.Join(dir, file)
			if fileExists(path) {
				return true
			}
		}
		return false
	}

	// Check for Maven configuration files.
	if containsBuildFile(absBuildDir, mavenFiles) {
		return true, "Java-Maven", nil
	}

	// Check for Gradle configuration files.
	if containsBuildFile(absBuildDir, gradleFiles) {
		return true, "Java-Gradle", nil
	}

	// If no known build configuration files were found, build an error message.
	errMsg := fmt.Sprintf("Vulkan Error: No valid Java build configuration file found in '%s' ", absBuildDir)

	return false, "", fmt.Errorf("%sPlease ensure that your Java project contains either Maven or Gradle configuration files", errMsg)
}

// detectPythonProject examines the specified build directory to determine if it contains
// any files or directories that indicate a Python project. It does so by checking for the
// existence of a set of known Python project files.
func DetectPythonProject(buildDir string) (bool, error) {
	// Convert the provided build directory to an absolute path for consistency.
	absBuildDir, err := filepath.Abs(buildDir)
	if err != nil {
		return false, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// List of filenames and directory names that indicate a Python project.
	// This includes common configuration files, entry points, lock files,
	// and even virtual environment directories.
	knownFiles := []string{
		".python-version",
		"__init__.py",
		"app.py",
		"main.py",
		"manage.py",
		"pdm.lock",
		"Pipfile",
		"Pipfile.lock",
		"poetry.lock",
		"pyproject.toml",
		"requirements.txt",
		"runtime.txt",
		"server.py",
		"setup.cfg",
		"setup.py",
		"uv.lock",
		// Common misspellings of requirements.txt.
		"requirement.txt",
		"Requirements.txt",
		"requirements.text",
		"requirements.txt.txt",
		"requirments.txt",
		// Virtual environment directories (the trailing slash is not required).
		".venv",
		"venv",
	}

	// Iterate over each known file or directory name.
	// If any of these exists in the build directory, assume it's a Python project.
	for _, name := range knownFiles {
		// Construct the full path to the file or directory.
		filePath := filepath.Join(absBuildDir, name)
		if fileExists(filePath) {
			// A known Python project file/directory was found.
			return true, nil
		}
	}

	// Construct a detailed error message.
	errMsg := "vulkan Error: Your application is configured to use the Vulkan Python build system, but we couldn't find any supported Python project files"

	return false, fmt.Errorf("%s in '%s'. Please ensure that your Python project contains at least one of the following files: %s", errMsg, absBuildDir, strings.Join(knownFiles, ", "))
}

// fileExists checks if the specified file exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		// File does not exist or some error occurred.
		return false
	}
	return !info.IsDir()
}

// dirExists checks if the specified directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
