#!/usr/bin/env bash
# detectGo.sh
#
# Usage: ./detectGo.sh <build-dir>
#
# This script determines whether the specified directory contains a Golang project.
# It does this by checking for known files that indicate the presence of Go dependency
# management or source organization. If a Go project is detected, it prints "Go" and
# exits with a success status (0). Otherwise, it exits with a failure status (1).
#
# The script supports detection of the following:
#   - Go modules (go.mod)
#   - Dep (Gopkg.lock)
#   - Godeps (Godeps/Godeps.json)
#   - Govendor (vendor/vendor.json)
#   - Glide (glide.yaml)
#   - gb (projects with a 'src' directory that contains .go files in subdirectories)
#
# 'set -e' ensures that the script exits immediately if any command exits with a non-zero status.
set -e

# Convert the provided directory path into an absolute path.
build=$(cd "$1/" && pwd)

# Test for various Go project markers.
if test -f "${build}/go.mod" || \  # Check for Go modules
   test -f "${build}/Gopkg.lock" || \  # Check for Dep
   test -f "${build}/Godeps/Godeps.json" || \  # Check for Godeps
   test -f "${build}/vendor/vendor.json" || \  # Check for Govendor
   test -f "${build}/glide.yaml" || \  # Check for Glide
   (test -d "${build}/src" && test -n "$(find "${build}/src" -mindepth 2 -type f -name '*.go' | sed 1q)")  # Check for gb-style projects
then
  # If any of the tests pass, a Go project is detected.
  echo Go
else
  # No Go project marker was found. Print a Vulkan error message to stderr and exit.
  echo "Vulkan Error: No Golang project files detected in ${build}" >&2
  exit 1
fi
