-- Drop in reverse dependency order

DROP INDEX IF EXISTS idx_proj_clusters_clu;
DROP INDEX IF EXISTS idx_proj_clusters_proj;
DROP INDEX IF EXISTS idx_proj_members_user;
DROP INDEX IF EXISTS idx_org_members_user;

DROP TABLE IF EXISTS project_clusters;
DROP TABLE IF EXISTS clusters;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS orgs;
DROP TABLE IF EXISTS refresh_tokens;
DROP TABLE IF EXISTS users;

DROP TYPE IF EXISTS cluster_status;
DROP TYPE IF EXISTS cluster_type;
DROP TYPE IF EXISTS role_type;
