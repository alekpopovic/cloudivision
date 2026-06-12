create table if not exists webhook_events (
    provider text not null,
    repository text not null,
    event_id text not null,
    project text,
    build_run text not null,
    created_at timestamptz not null default now(),
    primary key (provider, repository, event_id)
);

create index if not exists webhook_events_build_run_idx
    on webhook_events (build_run);

create index if not exists webhook_events_created_at_idx
    on webhook_events (created_at desc);
