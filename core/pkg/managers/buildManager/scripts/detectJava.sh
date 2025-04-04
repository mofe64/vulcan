#!/usr/bin/env bash
# detectJava.sh
#
# Usage: ./detectJava.sh <build-dir>
#
# This script detects whether the specified project directory contains
# a Java project by looking for common Maven or Gradle configuration files.

# Ensure that the build directory is provided as the first argument.
BUILD_DIR="${1:?Usage: $0 <build-dir>}"

# Define arrays of Maven and Gradle build files
MAVEN_FILES=(pom.xml pom.atom pom.clj pom.groovy pom.rb pom.scala pom.yaml pom.yml)
GRADLE_FILES=(build.gradle build.gradle.kts)

# Function to check for the existence of any file in a list
function contains_build_file {
    local dir="$1"
    shift
    local files=("$@")

    for file in "${files[@]}"; do
        if [ -f "$dir/$file" ]; then
            return 0
        fi
    done
    return 1
}

# Check for Maven build files
if contains_build_file "$BUILD_DIR" "${MAVEN_FILES[@]}"; then
    echo "Java-Maven"
    exit 0
fi

# Check for Gradle build files
if contains_build_file "$BUILD_DIR" "${GRADLE_FILES[@]}"; then
    echo "Java-Gradle"
    exit 0
fi

# If no known Java build files were found, print an error
{
    echo "Vulkan Error: No valid Java build configuration file found in '${BUILD_DIR}'. Ensure one of the following exists:"
    for file in "${MAVEN_FILES[@]}"; do echo "  - $file"; done
    for file in "${GRADLE_FILES[@]}"; do echo "  - $file"; done
} >&2

exit 1
