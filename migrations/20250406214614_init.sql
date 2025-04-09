-- +goose Up
CREATE TABLE applications (
    id UUID NOT NULL default gen_random_uuid() PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TABLE application_profiles (
    id UUID NOT NULL default gen_random_uuid() PRIMARY KEY,
    application_id UUID NOT NULL REFERENCES applications(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    graph_id UUID NOT NULL UNIQUE,
    UNIQUE (application_id, version)
);

CREATE INDEX idx_application_profiles_application_id ON application_profiles(application_id);

CREATE TABLE application_profile_versions (
    application_id UUID PRIMARY KEY REFERENCES applications(id) ON DELETE CASCADE,
    last_version INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS application_profiles;
DROP TABLE IF EXISTS applications;
DROP TABLE IF EXISTS application_profile_versions;
