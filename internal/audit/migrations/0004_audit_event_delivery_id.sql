alter table audit_events
    add column if not exists event_id text;

create index if not exists audit_events_event_id_idx
    on audit_events (event_id);
