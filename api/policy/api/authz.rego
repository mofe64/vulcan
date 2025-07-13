################################################################################
# policy/api/authz.rego
################################################################################
package api.authz

default allow := false # deny-by-default

allow if {
	vulkan_admin
}

# ────────────────────────────────────────────
#  Org-level permissions
# ────────────────────────────────────────────
allow if {
	resource_in({"org", "project", "application"})
	input.action in {"read", "write"}
	has_org_role("org-admin")
}

allow if {
	resource_in({"org", "project", "application"})
	input.action == "read"
	has_org_role("org-read")
}

# ────────────────────────────────────────────
#  Project-level permissions
# ────────────────────────────────────────────
allow if {
	resource_in({"project", "application"})
	input.action in {"read", "write"}
	has_project_role("project-admin")
}

allow if {
	resource_in({"project", "application"})
	input.action == "read"
	has_project_role("project-read")
}

# ────────────────────────────────────────────
#  Application-level permissions
# ────────────────────────────────────────────
allow if {
	input.resource.kind == "application"
	input.action in {"read", "write"}
	has_app_role("app-admin")
}

allow if {
	input.resource.kind == "application"
	input.action == "read"
	has_app_role("app-read")
}

###########
# HELPERS #
###########

vulkan_admin if {
	"vulkan-admin" in input.subject.roles[_]
}

resource_in(set) if {
	set[input.resource.kind]
}

# Return true if the caller holds ROLE on the target Org
has_org_role(role) if {
	some item
	item := input.subject.scoped_roles[_]
	item.role == role
	item.org_id == input.resource.org_id
}

# Return true if the caller holds ROLE on the target Project
has_project_role(role) if {
	some item
	item := input.subject.scoped_roles[_]
	item.role == role
	item.project_id == input.resource.project_id
}

# Return true if the caller holds ROLE on the target Application
has_app_role(role) if {
	some item
	item := input.subject.scoped_roles[_]
	item.role == role
	item.app_id == input.resource.app_id
}
