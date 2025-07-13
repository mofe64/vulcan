################################################################################
# policy/api/authz_test.rego
################################################################################
package api.authz

import rego.v1

# Test data helpers
vulkan_admin_subject := {
	"roles": ["vulkan-admin"],
	"scoped_roles": []
}

org_admin_subject := {
	"roles": [],
	"scoped_roles": [
		{
			"role": "org-admin",
			"org_id": "org-123"
		}
	]
}

org_read_subject := {
	"roles": [],
	"scoped_roles": [
		{
			"role": "org-read",
			"org_id": "org-123"
		}
	]
}

project_admin_subject := {
	"roles": [],
	"scoped_roles": [
		{
			"role": "project-admin",
			"project_id": "proj-456",
			"org_id": "org-123"
		}
	]
}

project_read_subject := {
	"roles": [],
	"scoped_roles": [
		{
			"role": "project-read",
			"project_id": "proj-456",
			"org_id": "org-123"
		}
	]
}

app_admin_subject := {
	"roles": [],
	"scoped_roles": [
		{
			"role": "app-admin",
			"app_id": "app-789",
			"project_id": "proj-456",
			"org_id": "org-123"
		}
	]
}

app_read_subject := {
	"roles": [],
	"scoped_roles": [
		{
			"role": "app-read",
			"app_id": "app-789",
			"project_id": "proj-456",
			"org_id": "org-123"
		}
	]
}

no_permissions_subject := {
	"roles": [],
	"scoped_roles": []
}

# ────────────────────────────────────────────
#  Global Admin Tests
# ────────────────────────────────────────────