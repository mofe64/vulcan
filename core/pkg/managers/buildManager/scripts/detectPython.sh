#!/usr/bin/env bash
# detectPython.sh
#
# Usage: ./detectPython.sh <build-dir>
#
# This script detects whether the provided build directory contains a Python project.
# It does so by checking for the presence of a list of known Python project files,
# such as package manager files, common entry point files, and configuration files.
#
# If a Python project is detected, it prints "Python" and exits with status 0.
# If none of the known Python files are found, it prints a detailed error message
# and exits with status 1.
#
# 'set -euo pipefail' ensures robust error handling:
#   - -e: Exit on any error.
#   - -u: Treat unset variables as an error.
#   - -o pipefail: Ensure the entire pipeline fails if any command fails.
set -euo pipefail
shopt -s inherit_errexit

# Set the build directory from the first argument.
BUILD_DIR="${1}"

# Determine the absolute path of the Vulkan build system.
# (Assumes that this script is located inside the Vulkan tool's bin directory.)
BUILDPACK_DIR=$(cd "$(dirname "$(dirname "${BASH_SOURCE[0]}")")" && pwd)

# Load the output helper functions (if available) to print errors.
# In this example, we assume a script exists at lib/output.sh that defines output::error.
# If not available, you can replace output::error with a custom error function.
source "${BUILDPACK_DIR}/lib/output.sh"

# Define an array of filenames and directory names that, if present, indicate a Python project.
KNOWN_PYTHON_PROJECT_FILES=(
	.python-version
	__init__.py
	app.py
	main.py
	manage.py
	pdm.lock
	Pipfile
	Pipfile.lock
	poetry.lock
	pyproject.toml
	requirements.txt
	runtime.txt
	server.py
	setup.cfg
	setup.py
	uv.lock
	# Common misspellings of requirements.txt.
	requirement.txt
	Requirements.txt
	requirements.text
	requirements.txt.txt
	requirments.txt
	# Sometimes virtual environments are committed use this a fallback.
	.venv/
	venv/
)

# Iterate over each known file. If any one exists in the build directory,
# we assume this is a Python project and output "Python".
for filepath in "${KNOWN_PYTHON_PROJECT_FILES[@]}"; do
	if [[ -e "${BUILD_DIR}/${filepath}" ]]; then
		echo "Python"
		exit 0
	fi
done

# If no known Python files are detected, output a detailed error message and exit with status 1.
output::error <<EOF
Vulkan Error: Your application is configured to use the Vulkan Python build system,
but we couldn't find any supported Python project files.

A valid Python project must include at least one of the following files in the root
directory of its source code:

- requirements.txt
- Pipfile or Pipfile.lock
- poetry.lock
- pyproject.toml

The current contents of your project's root directory are:

$(ls -1A --indicator-style=slash "${BUILD_DIR}" || true)

Please ensure that your project includes one of these files and that:
1. It is located in the top-level directory.
2. It is correctly spelled (filenames are case-sensitive).
3. It is not excluded by .gitignore or similar mechanisms.

For additional help, please refer to your organization's Vulkan documentation for
building Python projects.
EOF

exit 1
