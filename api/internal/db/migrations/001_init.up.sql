-- enums
CREATE TYPE role_type      AS ENUM ('admin', 'maintainer', 'viewer');
CREATE TYPE cluster_type   AS ENUM ('attached', 'remote_public', 'remote_private', 'remote_eks');
CREATE TYPE cluster_status AS ENUM ('provisioning', 'ready', 'error');

-- users table (OIDC identities)
CREATE TABLE users (
    id         UUID PRIMARY KEY,
    oidc_sub   TEXT UNIQUE NOT NULL,          -- OIDC subject
    email      TEXT UNIQUE,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- long-lived refresh tokens
CREATE TABLE refresh_tokens (
    token_id   UUID PRIMARY KEY,
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- organisations
CREATE TABLE orgs (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    domain     TEXT UNIQUE NOT NULL,
    owner_user UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT now()
);

-- org membership (RBAC)
CREATE TABLE org_members (
    user_id  UUID REFERENCES users(id) ON DELETE CASCADE,
    org_id   UUID REFERENCES orgs(id)  ON DELETE CASCADE,
    role     role_type NOT NULL,
    added_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (user_id, org_id)
);

-- projects  (one Org  → many Projects)
CREATE TABLE projects (
    id         UUID PRIMARY KEY,
    org_id     UUID REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (org_id, name)            -- no duplicate names inside an org
);

-- project-level RBAC
CREATE TABLE project_members (
    user_id    UUID REFERENCES users(id)    ON DELETE CASCADE,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    role       role_type NOT NULL,
    PRIMARY KEY (user_id, project_id)
);

-- applications  (many apps per project)
CREATE TABLE apps (
    id         UUID PRIMARY KEY,
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    repo_url   TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (project_id, name)
);

-- clusters  (one Org  → many clusters)
CREATE TABLE clusters (
    id         UUID PRIMARY KEY,
    org_id     UUID REFERENCES orgs(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    type       cluster_type   NOT NULL,
    region     TEXT,
    status     cluster_status NOT NULL DEFAULT 'provisioning',
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE (org_id, name)               -- friendly name unique per org
);



-- link table  Project  ⇄  Cluster   (many-to-many)
CREATE TABLE project_clusters (
    project_id UUID REFERENCES projects(id)  ON DELETE CASCADE,
    cluster_id UUID REFERENCES clusters(id)  ON DELETE CASCADE,
    created_at TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (project_id, cluster_id)
);

-- indexes
CREATE INDEX idx_org_members_user   ON org_members(user_id);
CREATE INDEX idx_proj_members_user  ON project_members(user_id);
CREATE INDEX idx_proj_clusters_proj ON project_clusters(project_id);
CREATE INDEX idx_proj_clusters_clu  ON project_clusters(cluster_id);
