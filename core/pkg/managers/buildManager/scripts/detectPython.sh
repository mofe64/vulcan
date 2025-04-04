#!/usr/bin/env bash
# detectPython.sh
#
# Usage: ./detectPython.sh <build-dir>
#
# This script detects whether the provided build directory contains a Python project.
# It does so by checking for the presence of a list of known Python project files.
#
# If a Python project is detected, it prints "Python" and exits with status 0.
# Otherwise, it prints a detailed error message and exits with status 1.
#
# 'set -euo pipefail' ensures robust error handling:
#   - -e: Exit on any error.
#   - -u: Treat unset variables as an error.
#   - -o pipefail: Ensure the entire pipeline fails if any command fails.
set -euo pipefail
shopt -s inherit_errexit

# The first argument is the build directory.
BUILD_DIR="${1}"

# Define an array of filenames and directory names that indicate a Python project.
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
	# Sometimes virtual environments are committed as a fallback.
	.venv/
	venv/
)

# Iterate over each known file. If any one exists in the build directory, assume a Python project.
for filepath in "${KNOWN_PYTHON_PROJECT_FILES[@]}"; do
	if [[ -e "${BUILD_DIR}/${filepath}" ]]; then
		echo "Python"
		exit 0
	fi
done

# If no known Python files are detected, output an error message.
{
	echo "Vulkan Error: Your application is configured to use the Vulkan Python build system,"
	echo "but we couldn't find any supported Python project files."
	echo ""
	echo "A valid Python project must include at least one of the following files in the root"
	echo "directory of its source code:"
	echo ""
	echo "- requirements.txt"
	echo "- Pipfile or Pipfile.lock"
	echo "- poetry.lock"
	echo "- pyproject.toml"
	echo ""
	echo "The current contents of your project's root directory are:"
	ls -1A --indicator-style=slash "${BUILD_DIR}" || true
	echo ""
	echo "Please ensure that your project includes one of these files."
} >&2

exit 1
