#!/usr/bin/env bash
# detectJs.sh
#
# Usage: ./detectJs.sh <build-dir>
#
# This script detects whether the specified project directory is a Node.js project.
# It checks for the presence of a 'package.json' file.
#
# If the file is present, it outputs "Node.js" and exits with status 0.
# If the file is missing or if it is explicitly ignored by .slugignore or .gitignore,
# the script outputs a Vulkan-specific error message and exits with status 1.
#
# 'set -euo pipefail' ensures the script stops on error, treats unset variables as errors,
# and propagates errors through pipes.
set -euo pipefail
shopt -s inherit_errexit

# The first argument is the build directory containing the project source.
BUILD_DIR="$1"

# Check if package.json exists in the project root.
if [ -f "${BUILD_DIR}/package.json" ]; then
    echo "Node.js"
    exit 0
fi


# Check if package.json is being excluded by a .gitignore file.
if [[ -f "${BUILD_DIR}/.gitignore" ]] && grep -Fxq "package.json" "${BUILD_DIR}/.gitignore"; then
    echo "Vulkan Error: 'package.json' is listed in '.gitignore'. Ensure that your Node.js project file is tracked." >&2
    exit 1
fi

# If no package.json is found and there is no explanation in ignore files, output an error.
echo "Vulkan Error: Could not detect a Node.js codebase. The project does not include a 'package.json' file at the root." >&2
echo "Directory listing of '${BUILD_DIR}':" >&2
ls -1A "${BUILD_DIR}" >&2
exit 1
