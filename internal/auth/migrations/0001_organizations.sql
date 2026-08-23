create table if not exists organizations (
  id text primary key,
  name text not null,
  created_at timestamptz not null default now()
);

create table if not exists teams (
  id text not null,
  organization_id text not null references organizations(id) on delete cascade,
  name text not null,
  primary key (organization_id, id)
);

create table if not exists memberships (
  organization_id text not null references organizations(id) on delete cascade,
  user_subject text not null,
  team_id text,
  role text not null check (role in ('org-admin','project-admin','developer','viewer','auditor')),
  unique (organization_id, user_subject, team_id)
);

create table if not exists project_access (
  organization_id text not null references organizations(id) on delete cascade,
  project text not null,
  team_id text not null,
  role text not null check (role in ('project-admin','developer','viewer','auditor')),
  primary key (organization_id, project, team_id)
);
