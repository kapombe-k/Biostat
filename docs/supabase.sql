-- Run once in Supabase SQL Editor for the optional persistent BioScope library.
-- The application uses the service-role key only on its Go server; do not expose
-- that key in browser JavaScript or commit it to source control.
create table if not exists public.bioscope_records (
    owner_id uuid not null,
    kind text not null check (kind in ('search', 'workspace')),
    record_key uuid not null,
    payload jsonb not null check (jsonb_typeof(payload) = 'object'),
    updated_at timestamptz not null default now(),
    primary key (owner_id, kind, record_key)
);

create index if not exists bioscope_records_owner_updated_idx
    on public.bioscope_records (owner_id, updated_at desc);

-- No anon/authenticated policies are intentionally defined. Only the backend's
-- service role should access this table. Supabase service-role keys bypass RLS.
alter table public.bioscope_records enable row level security;
