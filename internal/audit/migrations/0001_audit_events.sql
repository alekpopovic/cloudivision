create table if not exists audit_events (
    id text primary key,
    type text not null,
    actor text,
    project text,
    repository text,
    build_run text,
    release text,
    message text,
    metadata jsonb not null default '{}'::jsonb,
    created_at timestamptz not null default now()
);

create index if not exists audit_events_project_created_at_idx
    on audit_events (project, created_at desc);

create index if not exists audit_events_build_run_created_at_idx
    on audit_events (build_run, created_at desc);

create index if not exists audit_events_release_created_at_idx
    on audit_events (release, created_at desc);

create index if not exists audit_events_type_created_at_idx
    on audit_events (type, created_at desc);
