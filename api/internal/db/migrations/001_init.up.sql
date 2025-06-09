-- 1) Basic enums
CREATE TYPE role_type        AS ENUM ('admin', 'maintainer', 'viewer');
CREATE TYPE cluster_type     AS ENUM ('attached', 'eks');
CREATE TYPE cluster_status   AS ENUM ('provisioning', 'ready', 'error');

-- 2) Users table (OIDC identities)
CREATE TABLE users (
    id            UUID PRIMARY KEY,
    oidc_sub      TEXT UNIQUE NOT NULL,    -- subject claim
    email         TEXT UNIQUE,
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- 3) Long-lived refresh tokens
CREATE TABLE refresh_tokens (
    token_id      UUID PRIMARY KEY,
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    expires_at    TIMESTAMPTZ NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- 4) Organisations
CREATE TABLE orgs (
    id            UUID PRIMARY KEY,
    name          TEXT NOT NULL,
    owner_user    UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at    TIMESTAMPTZ DEFAULT now()
);

-- 5) Org membership & RBAC
CREATE TABLE org_members (
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    org_id        UUID REFERENCES orgs(id)  ON DELETE CASCADE,
    role          role_type NOT NULL,
    added_at      TIMESTAMPTZ DEFAULT now(),
    PRIMARY KEY (user_id, org_id)
);

-- 6) Projects
CREATE TABLE projects (
    id            UUID PRIMARY KEY,
    org_id        UUID REFERENCES orgs(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (org_id, name)                   -- no dup names within an org
);

-- 7) Optional project-level RBAC
CREATE TABLE project_members (
    user_id       UUID REFERENCES users(id)    ON DELETE CASCADE,
    project_id    UUID REFERENCES projects(id) ON DELETE CASCADE,
    role          role_type NOT NULL,
    PRIMARY KEY (user_id, project_id)
);

-- 8) Applications
CREATE TABLE apps (
    id            UUID PRIMARY KEY,
    project_id    UUID REFERENCES projects(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    repo_url      TEXT NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (project_id, name)
);

-- 9) Clusters
CREATE TABLE clusters (
    id            UUID PRIMARY KEY,
    project_id    UUID REFERENCES projects(id) ON DELETE CASCADE,
    name          TEXT NOT NULL,
    type          cluster_type   NOT NULL,
    region        TEXT,
    status        cluster_status NOT NULL DEFAULT 'provisioning',
    created_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE (project_id, name)
);

-- 10) Helpful index for quick look-ups
CREATE INDEX idx_org_members_user  ON org_members(user_id);
CREATE INDEX idx_proj_members_user ON project_members(user_id);
