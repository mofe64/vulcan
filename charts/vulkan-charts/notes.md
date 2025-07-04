**Some Helm Template Concepts Encountered**

1.  **`include` function:**

    - **Purpose:** Embeds the content of a named template (e.g., `vulkan.fullname`) at the current location during rendering.
    - **Usage:** `{{ include "templateName" context }}`.
    - **Benefit:** Promotes reusability of logic and consistent naming patterns across your Kubernetes manifests.

2.  **The `.` (Dot) Symbol:**

    - **Meaning:** Represents the **current context or scope** of data available to the template engine at a given point.
    - **Contents:** Typically holds the entire Helm context, including:
      - `.Values`: User-defined values from `values.yaml`.
      - `.Release`: Information about the current Helm release (name, namespace, etc.).
      - `.Chart`: Metadata from `Chart.yaml` (name, version, etc.).
      - `.Capabilities`: Information about the Kubernetes cluster's capabilities.
    - **Usage:** Used to access fields within the current context (e.g., `.Release.Name`) or to pass the entire context to another template (`{{ include "template" . }}`).

3.  **User-Defined Variables (`$variable`):**

    - **Definition:** Variables can be created within a template using `{{- $variableName := value -}}`.
    - **Purpose:** To store and reuse values or sub-contexts within a template, often improving readability and logic flow.

4.  **Passing Context and `$context := .context | default .` Pattern:**
    - When calling an `include` function, you pass a context to the included template. This can be the full global context (`.`) or a specific dictionary.
    - The `{{- $context := .context | default . -}}` pattern is a robust way to ensure a variable (like `$context`) **always holds the full Helm global context**, regardless of how the helper template was called:
      - If the global context was passed wrapped in a dictionary (e.g., `(dict "context" .)`), it extracts it from `.context`.
      - If the global context was passed directly (`.`), it uses that as the default.
    - This makes helpers flexible, usable in various scenarios while consistently providing access to `.Values`, `.Release`, `.Chart`, etc.
