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
        ON DELETE CASCADE
);

-- Create players table
CREATE TABLE players (
    id SERIAL,
    tenant_id UUID NOT NULL,
    email VARCHAR(255) NOT NULL,
    first_name VARCHAR(32) NOT NULL,
    last_name VARCHAR(32) NOT NULL,
    hdcp REAL NOT NULL DEFAULT 0,
    photo_path VARCHAR DEFAULT '',
    biography TEXT DEFAULT '',
    tier VARCHAR(32) NOT NULL DEFAULT 'white',
    cups INTEGER NOT NULL DEFAULT 0,
    wins INTEGER NOT NULL DEFAULT 0,
    ties INTEGER NOT NULL DEFAULT 0,
    losses INTEGER NOT NULL DEFAULT 0,
    CONSTRAINT pk__players PRIMARY KEY (id),
    CONSTRAINT uq__players__tenant_id__email UNIQUE (tenant_id, email)
);

-- Create teams table
CREATE TABLE teams (
    id SERIAL,
    tenant_id UUID NOT NULL,
    name VARCHAR(32) NOT NULL,
    CONSTRAINT pk__teams PRIMARY KEY (id),
    CONSTRAINT uq__teams__tenant_id__name UNIQUE (tenant_id, name)
);

-- Create tournaments table
CREATE TABLE tournaments (
    id SERIAL,
    tenant_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    location VARCHAR(255) NOT NULL,
    CONSTRAINT pk__tournaments PRIMARY KEY (id),
    CONSTRAINT uq__tournaments__tenant_id__name_start_date_end_date
        UNIQUE (tenant_id, name, start_date, end_date)
);

-- Create team_members table (composite PK)
CREATE TABLE team_members (
    tournament_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    team_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    is_captain BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT pk__team_members PRIMARY KEY (tournament_id, player_id),
    CONSTRAINT fk__team_members__tournament_id__tournaments
        FOREIGN KEY (tournament_id)
        REFERENCES tournaments(id)
        ON DELETE CASCADE,
    CONSTRAINT fk__team_members__player_id__players
        FOREIGN KEY (player_id)
        REFERENCES players(id)
        ON DELETE CASCADE,
    CONSTRAINT fk__team_members__team_id__teams
        FOREIGN KEY (team_id)
        REFERENCES teams(id)
        ON DELETE CASCADE
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
    CONSTRAINT pk__matches PRIMARY KEY (id),
    CONSTRAINT fk__matches__tournament_id__tournaments
        FOREIGN KEY (tournament_id)
        REFERENCES tournaments(id)
        ON DELETE CASCADE,
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

-- Create match_participants table (composite PK)
CREATE TABLE match_participants (
    tournament_id INTEGER NOT NULL,
    match_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    CONSTRAINT pk__match_participants PRIMARY KEY (match_id, player_id),
    CONSTRAINT fk__match_participants__tournament_id__tournaments
        FOREIGN KEY (tournament_id)
        REFERENCES tournaments(id)
        ON DELETE CASCADE,
    CONSTRAINT fk__match_participants__tournament_id_match_id__matches
        FOREIGN KEY (tournament_id, match_id)
        REFERENCES matches(tournament_id, id)
        ON DELETE CASCADE,
    CONSTRAINT fk__match_participants__player_id__players
        FOREIGN KEY (player_id)
        REFERENCES players(id),
    CONSTRAINT fk__match_participants__tournament_id_player_id__team_members
        FOREIGN KEY (tournament_id, player_id)
        REFERENCES team_members(tournament_id, player_id)
        ON DELETE CASCADE
);

-- Create scores table (composite PK with match_id, player_id, hole_number)
CREATE TABLE scores (
    match_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    course_id INTEGER NOT NULL,
    tee_color_id INTEGER NOT NULL,
    hole_number INTEGER NOT NULL,
    tenant_id UUID NOT NULL,
    strokes INTEGER NOT NULL,
    CONSTRAINT pk__scores PRIMARY KEY (match_id, player_id, hole_number),
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
    CONSTRAINT fk__scores__player_id__players
        FOREIGN KEY (player_id)
        REFERENCES players(id),
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
ALTER TABLE teams ENABLE ROW LEVEL SECURITY;
ALTER TABLE tournaments ENABLE ROW LEVEL SECURITY;
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

CREATE POLICY tenant_isolation_policy ON teams
    FOR ALL TO PUBLIC
    USING (tenant_id = current_setting('app.current_tenant_id', true)::uuid);

CREATE POLICY tenant_isolation_policy ON tournaments
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
CREATE INDEX ix__matches__tenant_id__tournament_id__tee_time ON matches(tenant_id, tournament_id, tee_time);
CREATE INDEX ix__scores__tenant_id__match_id__hole_number ON scores(tenant_id, match_id, hole_number);
CREATE INDEX ix__team_members__tenant_id__tournament_id ON team_members(tenant_id, tournament_id);
CREATE INDEX ix__match_participants__tenant_id__match_id ON match_participants(tenant_id, match_id);
