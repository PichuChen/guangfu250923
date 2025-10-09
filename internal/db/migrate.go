package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Simple idempotent migrations.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	stmts := []string{
		`create table if not exists volunteer_organizations (
            id text primary key default gen_random_uuid()::text,
            last_updated timestamptz,
            registration_status text,
            organization_nature text,
            organization_name text,
            coordinator text,
            contact_info text,
            registration_method text,
            service_content text,
            meeting_info text,
            notes text,
            image_url text
        )`,
		`create index if not exists idx_vol_org_updated on volunteer_organizations(last_updated)`,
		`create table if not exists shelters (
            id text primary key default gen_random_uuid()::text,
            name text not null,
            location text not null,
            phone text not null,
            link text,
            status text not null,
            capacity int,
            current_occupancy int,
            available_spaces int,
            facilities text[],
            contact_person text,
            notes text,
            opening_hours text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_shelters_status on shelters(status)`,
		`alter table if exists shelters add column if not exists coordinates jsonb`,
		`create table if not exists medical_stations (
            id text primary key default gen_random_uuid()::text,
            station_type text not null,
            name text not null,
            location text not null,
            detailed_address text,
            phone text,
            contact_person text,
            status text not null,
            services text[],
            equipment text[],
            operating_hours text,
            medical_staff int,
            daily_capacity int,
            affiliated_organization text,
            notes text,
            link text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_medical_stations_status on medical_stations(status)`,
		`create index if not exists idx_medical_stations_station_type on medical_stations(station_type)`,
		`alter table if exists medical_stations add column if not exists coordinates jsonb`,
		`create table if not exists mental_health_resources (
            id text primary key default gen_random_uuid()::text,
            duration_type text not null,
            name text not null,
            service_format text not null,
            service_hours text not null,
            contact_info text not null,
            website_url text,
            target_audience text[],
            specialties text[],
            languages text[],
            is_free boolean not null,
            location text,
            status text not null,
            capacity int,
            waiting_time text,
            notes text,
            emergency_support boolean not null,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_mh_resources_status on mental_health_resources(status)`,
		`create index if not exists idx_mh_resources_duration_type on mental_health_resources(duration_type)`,
		`alter table if exists mental_health_resources add column if not exists coordinates jsonb`,
		`create table if not exists accommodations (
            id text primary key default gen_random_uuid()::text,
            township text not null,
            name text not null,
            has_vacancy text not null,
            available_period text not null,
            restrictions text,
            contact_info text not null,
            room_info text,
            address text not null,
            pricing text not null,
            info_source text,
            notes text,
            capacity int,
            status text not null,
            registration_method text,
            facilities text[],
            distance_to_disaster_area text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_accommodations_status on accommodations(status)`,
		`create index if not exists idx_accommodations_township on accommodations(township)`,
		`create index if not exists idx_accommodations_has_vacancy on accommodations(has_vacancy)`,
		`alter table if exists accommodations add column if not exists coordinates jsonb`,
		`create table if not exists shower_stations (
            id text primary key default gen_random_uuid()::text,
            name text not null,
            address text not null,
            phone text,
            facility_type text not null,
            time_slots text not null,
            gender_schedule jsonb,
            available_period text not null,
            capacity int,
            is_free boolean not null,
            pricing text,
            notes text,
            info_source text,
            status text not null,
            facilities text[],
            distance_to_guangfu text,
            requires_appointment boolean not null,
            contact_method text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_shower_stations_status on shower_stations(status)`,
		`create index if not exists idx_shower_stations_facility_type on shower_stations(facility_type)`,
		`create index if not exists idx_shower_stations_is_free on shower_stations(is_free)`,
		`create index if not exists idx_shower_stations_requires_appointment on shower_stations(requires_appointment)`,
		`alter table if exists shower_stations add column if not exists coordinates jsonb`,
		`create table if not exists water_refill_stations (
            id text primary key default gen_random_uuid()::text,
            name text not null,
            address text not null,
            phone text,
            water_type text not null,
            opening_hours text not null,
            is_free boolean not null,
            container_required text,
            daily_capacity int,
            status text not null,
            water_quality text,
            facilities text[],
            accessibility boolean not null,
            distance_to_disaster_area text,
            notes text,
            info_source text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_water_refill_status on water_refill_stations(status)`,
		`create index if not exists idx_water_refill_water_type on water_refill_stations(water_type)`,
		`create index if not exists idx_water_refill_is_free on water_refill_stations(is_free)`,
		`create index if not exists idx_water_refill_accessibility on water_refill_stations(accessibility)`,
		`alter table if exists water_refill_stations add column if not exists coordinates jsonb`,
		`create table if not exists restrooms (
            id text primary key default gen_random_uuid()::text,
            name text not null,
            address text not null,
            phone text,
            facility_type text not null,
            opening_hours text not null,
            is_free boolean not null,
            male_units int,
            female_units int,
            unisex_units int,
            accessible_units int,
            has_water boolean not null,
            has_lighting boolean not null,
            status text not null,
            cleanliness text,
            last_cleaned timestamptz,
            facilities text[],
            distance_to_disaster_area text,
            notes text,
            info_source text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create table if not exists human_resources (
            id text primary key,
            org text not null,
            address text not null,
            phone text,
            status text not null,
            is_completed boolean not null,
            has_medical boolean,
        pii_date bigint,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now(),
            role_name text not null,
            role_type text not null,
            skills text[],
            certifications text[],
            experience_level text,
            language_requirements text[],
            headcount_need int not null,
            headcount_got int not null,
            headcount_unit text,
            role_status text not null,
            shift_start_ts timestamptz,
            shift_end_ts timestamptz,
            shift_notes text,
            assignment_timestamp timestamptz,
            assignment_count int,
            assignment_notes text,
            total_roles_in_request int,
            completed_roles_in_request int,
            pending_roles_in_request int,
            total_requests int,
            active_requests int,
            completed_requests int,
            cancelled_requests int,
            total_roles int,
            completed_roles int,
            pending_roles int,
            urgent_requests int,
            medical_requests int
        )`,
		// Add valid_pin to human_resources for edit verification (6-digit pin). Keep nullable for backward compatibility; app enforces on create/patch.
		`alter table if exists human_resources add column if not exists valid_pin text`,
		// Relax NOT NULL if previously set
		`do $$ begin
        perform 1 from information_schema.columns where table_name='human_resources' and column_name='phone' and is_nullable='NO';
        if found then
          alter table human_resources alter column phone drop not null;
        end if;
      end $$;`,
		`create index if not exists idx_human_resources_status on human_resources(status)`,
		`create index if not exists idx_human_resources_role_status on human_resources(role_status)`,
		`create index if not exists idx_restrooms_status on restrooms(status)`,
		`create index if not exists idx_restrooms_facility_type on restrooms(facility_type)`,
		`create index if not exists idx_restrooms_is_free on restrooms(is_free)`,
		`create index if not exists idx_restrooms_has_water on restrooms(has_water)`,
		`create index if not exists idx_restrooms_has_lighting on restrooms(has_lighting)`,
		`alter table if exists restrooms add column if not exists coordinates jsonb`,
		`create table if not exists request_logs (
            id uuid primary key default gen_random_uuid(),
            method text not null,
            path text not null,
            query text,
            ip text,
            headers jsonb,
            status_code int,
            error text,
            duration_ms int,
            request_body jsonb,
            original_data jsonb,
            result_data jsonb,
            resource_id text,
            created_at timestamptz not null default now()
        )`,
		// New simplified supplies domain (replaces legacy requests/supply_items usage)
		`create table if not exists supplies (
            id text primary key default gen_random_uuid()::text,
            name text,
            address text,
            phone text,
            notes text,
        pii_date bigint,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`alter table if exists supplies add column if not exists valid_pin text`,
		`create index if not exists idx_supplies_updated_at on supplies(updated_at)`,
		/* Renamed to supply_items (previously 'suppily_items') */
		`create table if not exists supply_items (
            id text primary key default gen_random_uuid()::text,
            supply_id text not null references supplies(id) on delete cascade,
            tag text,
            name text,
            received_count int not null default 0,
            total_number int not null,
            unit text,
            constraint chk_supply_items_received_le_total check (received_count <= total_number)
        )`,
		`create index if not exists idx_supply_items_supply_id on supply_items(supply_id)`,
		// Add new columns if migrating from older version
		`alter table request_logs add column if not exists request_body jsonb`,
		`alter table request_logs add column if not exists original_data jsonb`,
		`alter table request_logs add column if not exists result_data jsonb`,
		`alter table request_logs add column if not exists resource_id text`,
		// If existing column is uuid, attempt to widen to text (safe no-op if already text)
		`do $$ begin
          perform 1 from information_schema.columns where table_name='request_logs' and column_name='resource_id' and data_type='uuid';
          if found then
            alter table request_logs alter column resource_id type text using resource_id::text;
          end if;
        end $$;`,
		`create index if not exists idx_request_logs_created_at on request_logs(created_at)`,
		`create index if not exists idx_request_logs_status_code on request_logs(status_code)`,
		// Reports table
		`create table if not exists reports (
            id text primary key,
            name text not null,
            location_type text not null,
            reason text not null,
            notes text,
            status text not null,
            location_id text not null,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`alter table reports add column if not exists location_id text not null default ''`,
		// Ensure no empty location_id remains (optional: set to placeholder if truly unknown)
		`do $$ begin
          update reports set location_id = 'unknown' where location_id = '';
        end $$;`,
		`create index if not exists idx_reports_status on reports(status)`,
		`create index if not exists idx_reports_updated_at on reports(updated_at)`,
		// IP denylist for middleware (single IP or CIDR patterns)
		`create table if not exists ip_denylist (
            id text primary key default gen_random_uuid()::text,
            pattern text not null,
            reason text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_ip_denylist_pattern on ip_denylist(pattern)`,
		// Spam detection results from LLM validation
		`create table if not exists spam_result (
            id text primary key,
            target_id text not null,
            target_type text not null,
            target_data jsonb not null,
            is_spam boolean not null,
            judgment text not null,
            validated_at bigint not null
        )`,
		`create index if not exists idx_spam_result_target_id on spam_result(target_id)`,
		// Supply item providers
		`create table if not exists supply_providers (
            id text primary key,
            name text not null,
            phone text not null,
            supply_item_id text not null,
            address text not null,
            notes text default '',
            provide_count int not null,
            provide_unit text,
            created_at timestamptz not null default now(),
            updated_at timestamptz not null default now()
        )`,
		`create index if not exists idx_supply_providers_supply_item_id on supply_providers(supply_item_id)`,
	}
	for _, s := range stmts {
		if _, err := pool.Exec(ctx, s); err != nil {
			return err
		}
	}
	return nil
}
