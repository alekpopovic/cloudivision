alter table audit_events
  add column if not exists organization text;

create index if not exists audit_events_organization_created_at_idx
  on audit_events (organization, created_at desc);
