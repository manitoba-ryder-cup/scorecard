-- Enable UUID extension for tenant IDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create tee_colors table
CREATE TABLE tee_colors (
    id SERIAL,
    tenant_id UUID NOT NULL,
    color VARCHAR(32) NOT NULL,
    CONSTRAINT pk__tee_colors PRIMARY KEY (id),
    CONSTRAINT uq__tee_colors__tenant_id__color UNIQUE (tenant_id, color)
);

-- Create courses table
CREATE TABLE courses (
    id SERIAL,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    CONSTRAINT pk__courses PRIMARY KEY (id),
    CONSTRAINT uq__courses__tenant_id__name UNIQUE (tenant_id, name)
);

-- Create tee_sets table (composite PK with course_id and tee_color_id)
CREATE TABLE tee_sets (
    course_id INTEGER NOT NULL,
    tee_color_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    slope INTEGER NOT NULL,
    rating NUMERIC(4, 1) NOT NULL,
    CONSTRAINT pk__tee_sets PRIMARY KEY (course_id, tee_color_id),
    CONSTRAINT fk__tee_sets__course_id__courses FOREIGN KEY (course_id) REFERENCES courses(id) ON DELETE CASCADE,
    CONSTRAINT fk__tee_sets__tee_color_id__tee_colors FOREIGN KEY (tee_color_id) REFERENCES tee_colors(id),
    CONSTRAINT ck__tee_sets__slope CHECK (slope >= 55 AND slope <= 155)
);

-- Create holes table (composite PK with course_id, tee_color_id, number)
CREATE TABLE holes (
    course_id INTEGER NOT NULL,
    tee_color_id INTEGER NOT NULL,
    number INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    par INTEGER NOT NULL,
    hdcp INTEGER NOT NULL,
    yards INTEGER NOT NULL,
    CONSTRAINT pk__holes PRIMARY KEY (course_id, tee_color_id, number),
    CONSTRAINT fk__holes__course_id_tee_color_id__tee_sets
        FOREIGN KEY (course_id, tee_color_id)
        REFERENCES tee_sets(course_id, tee_color_id)
        ON DELETE CASCADE,
    CONSTRAINT ck__holes__number CHECK (number >= 1 AND number <= 18),
    CONSTRAINT ck__holes__par CHECK (par >= 3 AND par <= 6),
    CONSTRAINT ck__holes__hdcp CHECK (hdcp >= 1 AND hdcp <= 18),
    -- Stroke indexes 1-18 must be unique per tee set, or handicap allocation breaks.
    CONSTRAINT uq__holes__course_id__tee_color_id__hdcp UNIQUE (course_id, tee_color_id, hdcp)
);

-- Players hold stable identity and career totals only. Per-tournament attributes
-- (tier, biography, handicap) live on team_members, since they change year to year.
-- user_id links to a heimdall account when one exists; it is a distinct identifier
-- from players.id on purpose — the two services must not share an ID space, and
-- roster-only players (no login) simply leave it NULL.
CREATE TABLE players (
    id SERIAL,
    tenant_id UUID NOT NULL,
    user_id UUID,                        -- heimdall user; NULL for roster-only players
    email VARCHAR(255),                  -- optional contact; identity lives in heimdall
    first_name VARCHAR(32) NOT NULL,
    last_name VARCHAR(32) NOT NULL,
    photo_path VARCHAR NOT NULL DEFAULT '',
    -- Career totals, materialized from match results (not recomputed on read).
    cups INTEGER NOT NULL DEFAULT 0,
    wins INTEGER NOT NULL DEFAULT 0,
    ties INTEGER NOT NULL DEFAULT 0,
    losses INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk__players PRIMARY KEY (id),
    -- Nullable uniqueness: Postgres treats NULLs as distinct, so roster-only
    -- players (NULL email/user_id) don't collide.
    CONSTRAINT uq__players__tenant_id__email UNIQUE (tenant_id, email),
    CONSTRAINT uq__players__tenant_id__user_id UNIQUE (tenant_id, user_id)
);

-- Create tournaments table (before teams, which reference it)
CREATE TABLE tournaments (
    id SERIAL,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    location VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk__tournaments PRIMARY KEY (id),
    CONSTRAINT uq__tournaments__tenant_id__name_start_date_end_date
        UNIQUE (tenant_id, name, start_date, end_date)
);

-- Teams are per-tournament first-class entities. A Ryder Cup has exactly two
-- sides: the color CHECK + UNIQUE(tournament_id, color) caps a tournament at one
-- Red and one Blue. captain_id is a first-class nullable FK to a player (the app
-- additionally requires the captain to be a member of the team).
CREATE TABLE teams (
    id SERIAL,
    tenant_id UUID NOT NULL,
    tournament_id INTEGER NOT NULL,
    color VARCHAR(16) NOT NULL,
    captain_id INTEGER,
    CONSTRAINT pk__teams PRIMARY KEY (id),
    CONSTRAINT fk__teams__tournament_id__tournaments
        FOREIGN KEY (tournament_id) REFERENCES tournaments(id) ON DELETE CASCADE,
    CONSTRAINT fk__teams__captain_id__players
        FOREIGN KEY (captain_id) REFERENCES players(id) ON DELETE SET NULL,
    CONSTRAINT ck__teams__color CHECK (color IN ('Red', 'Blue')),
    CONSTRAINT uq__teams__tournament_id__color UNIQUE (tournament_id, color),
    -- Lets team_members and match_participants prove a team belongs to a tournament.
    CONSTRAINT uq__teams__id__tournament_id UNIQUE (id, tournament_id)
);

-- Team membership for a tournament, plus the player's per-tournament attributes.
-- The composite FK (team_id, tournament_id) -> teams keeps tournament_id honest
-- (it must match the team's tournament); UNIQUE(tournament_id, player_id) enforces
-- one side per player per tournament.
CREATE TABLE team_members (
    team_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    tournament_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    tier VARCHAR(32) NOT NULL DEFAULT 'white',
    biography TEXT NOT NULL DEFAULT '',
    hdcp REAL NOT NULL DEFAULT 0,
    CONSTRAINT pk__team_members PRIMARY KEY (team_id, player_id),
    CONSTRAINT fk__team_members__team_id_tournament_id__teams
        FOREIGN KEY (team_id, tournament_id)
        REFERENCES teams(id, tournament_id)
        ON DELETE CASCADE,
    CONSTRAINT fk__team_members__player_id__players
        FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE,
    CONSTRAINT uq__team_members__tournament_id__player_id UNIQUE (tournament_id, player_id)
);

-- Create match_formats table
CREATE TABLE match_formats (
    id SERIAL,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    CONSTRAINT pk__match_formats PRIMARY KEY (id),
    CONSTRAINT uq__match_formats__tenant_id__name UNIQUE (tenant_id, name)
);

-- Create matches table (with composite unique constraints for foreign keys)
CREATE TABLE matches (
    id SERIAL,
    tournament_id INTEGER NOT NULL,
    course_id INTEGER NOT NULL,
    tee_color_id INTEGER NOT NULL,
    match_format_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    tee_time TIMESTAMP,
    handicapped BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk__matches PRIMARY KEY (id),
    -- RESTRICT: a tournament that has matches cannot be deleted (protects history).
    CONSTRAINT fk__matches__tournament_id__tournaments
        FOREIGN KEY (tournament_id)
        REFERENCES tournaments(id)
        ON DELETE RESTRICT,
    CONSTRAINT fk__matches__course_id__courses
        FOREIGN KEY (course_id)
        REFERENCES courses(id),
    CONSTRAINT fk__matches__tee_color_id__tee_colors
        FOREIGN KEY (tee_color_id)
        REFERENCES tee_colors(id),
    CONSTRAINT fk__matches__course_id_tee_color_id__tee_sets
        FOREIGN KEY (course_id, tee_color_id)
        REFERENCES tee_sets(course_id, tee_color_id),
    CONSTRAINT fk__matches__match_format_id__match_formats
        FOREIGN KEY (match_format_id)
        REFERENCES match_formats(id),
    CONSTRAINT uq__matches__id__tournament_id UNIQUE (id, tournament_id),
    CONSTRAINT uq__matches__id__course_id__tee_color_id UNIQUE (id, course_id, tee_color_id)
);

-- Match participants carry their team_id, so a match's two sides are explicit
-- rather than derived. The composite FKs enforce integrity: the player is a member
-- of the team (team_id, player_id -> team_members), and the team belongs to the
-- match's tournament (team_id, tournament_id -> teams). The direct player FK is
-- RESTRICT so a player who has played in a match cannot be deleted.
CREATE TABLE match_participants (
    tournament_id INTEGER NOT NULL,
    match_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    team_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    CONSTRAINT pk__match_participants PRIMARY KEY (match_id, player_id),
    CONSTRAINT fk__match_participants__tournament_id_match_id__matches
        FOREIGN KEY (tournament_id, match_id)
        REFERENCES matches(tournament_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk__match_participants__team_id_player_id__team_members
        FOREIGN KEY (team_id, player_id)
        REFERENCES team_members(team_id, player_id)
        ON DELETE CASCADE,
    CONSTRAINT fk__match_participants__team_id_tournament_id__teams
        FOREIGN KEY (team_id, tournament_id)
        REFERENCES teams(id, tournament_id),
    CONSTRAINT fk__match_participants__player_id__players
        FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE RESTRICT
);

-- Scores are attributed to a side (team_id) and, for per-player formats, to a
-- player. Singles/Fourball record a row per player (player_id set) — individual
-- gross scores that also feed player stats. One-ball formats (alternate shot,
-- scramble, mod-scotch) record one row per team per hole (player_id NULL), which
-- keeps them out of individual stats. Partial unique indexes enforce the two grains.
CREATE TABLE scores (
    id SERIAL,
    match_id INTEGER NOT NULL,
    team_id INTEGER NOT NULL,
    player_id INTEGER,                   -- NULL for one-ball team scores
    course_id INTEGER NOT NULL,
    tee_color_id INTEGER NOT NULL,
    hole_number INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    strokes INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk__scores PRIMARY KEY (id),
    CONSTRAINT fk__scores__team_id__teams
        FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    -- Enforced only for per-player scores; NULL player_id skips it (MATCH SIMPLE).
    CONSTRAINT fk__scores__match_id_player_id__match_participants
        FOREIGN KEY (match_id, player_id)
        REFERENCES match_participants(match_id, player_id)
        ON DELETE CASCADE,
    CONSTRAINT fk__scores__course_id_tee_color_id_match_id__matches
        FOREIGN KEY (course_id, tee_color_id, match_id)
        REFERENCES matches(course_id, tee_color_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk__scores__course_id_tee_color_id_hole_number__holes
        FOREIGN KEY (course_id, tee_color_id, hole_number)
        REFERENCES holes(course_id, tee_color_id, number),
    CONSTRAINT fk__scores__match_id__matches
        FOREIGN KEY (match_id)
        REFERENCES matches(id)
        ON DELETE CASCADE,
    CONSTRAINT fk__scores__course_id__courses
        FOREIGN KEY (course_id)
        REFERENCES courses(id),
    CONSTRAINT fk__scores__tee_color_id__tee_colors
        FOREIGN KEY (tee_color_id)
        REFERENCES tee_colors(id),
    CONSTRAINT ck__scores__strokes CHECK (strokes > 0)
);

-- Enable Row Level Security on all tables
ALTER TABLE tee_colors ENABLE ROW LEVEL SECURITY;
ALTER TABLE courses ENABLE ROW LEVEL SECURITY;
ALTER TABLE tee_sets ENABLE ROW LEVEL SECURITY;
ALTER TABLE holes ENABLE ROW LEVEL SECURITY;
ALTER TABLE players ENABLE ROW LEVEL SECURITY;
ALTER TABLE tournaments ENABLE ROW LEVEL SECURITY;
ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE team_members ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_formats ENABLE ROW LEVEL SECURITY;
ALTER TABLE matches ENABLE ROW LEVEL SECURITY;
ALTER TABLE match_participants ENABLE ROW LEVEL SECURITY;
ALTER TABLE scores ENABLE ROW LEVEL SECURITY;

-- Create RLS policies for tenant isolation
CREATE POLICY tenant_isolation_policy ON tee_colors
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON courses
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON tee_sets
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON holes
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON players
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON tournaments
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON teams
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON team_members
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON match_formats
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON matches
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON match_participants
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON scores
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

-- Create indexes for common query patterns
CREATE INDEX ix__players__tenant_id__email ON players(tenant_id, email);
CREATE INDEX ix__tournaments__tenant_id__start_date__end_date ON tournaments(tenant_id, start_date, end_date);
CREATE INDEX ix__teams__tenant_id__tournament_id ON teams(tenant_id, tournament_id);
CREATE INDEX ix__matches__tenant_id__tournament_id__tee_time ON matches(tenant_id, tournament_id, tee_time);
CREATE INDEX ix__scores__tenant_id__match_id__hole_number ON scores(tenant_id, match_id, hole_number);
CREATE INDEX ix__team_members__tenant_id__tournament_id ON team_members(tenant_id, tournament_id);
CREATE INDEX ix__team_members__tenant_id__team_id ON team_members(tenant_id, team_id);
CREATE INDEX ix__match_participants__tenant_id__match_id ON match_participants(tenant_id, match_id);

-- Score uniqueness has two grains: one row per player per hole (per-player formats)
-- and one row per team per hole (one-ball formats, player_id NULL).
CREATE UNIQUE INDEX uq__scores__match_id__hole_number__player_id
    ON scores(match_id, hole_number, player_id) WHERE player_id IS NOT NULL;
CREATE UNIQUE INDEX uq__scores__match_id__hole_number__team_id
    ON scores(match_id, hole_number, team_id) WHERE player_id IS NULL;
