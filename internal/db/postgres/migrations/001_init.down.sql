-- Drop tables in reverse dependency order
DROP TABLE IF EXISTS match_results CASCADE;
DROP TABLE IF EXISTS scores CASCADE;
DROP TABLE IF EXISTS match_participants CASCADE;
DROP TABLE IF EXISTS matches CASCADE;
DROP TABLE IF EXISTS match_formats CASCADE;
DROP TABLE IF EXISTS team_members CASCADE;
DROP TABLE IF EXISTS teams CASCADE;
DROP TABLE IF EXISTS tournaments CASCADE;
DROP TABLE IF EXISTS players CASCADE;
DROP TABLE IF EXISTS holes CASCADE;
DROP TABLE IF EXISTS tee_sets CASCADE;
DROP TABLE IF EXISTS courses CASCADE;
DROP TABLE IF EXISTS tee_colors CASCADE;

-- Drop UUID extension (optional, comment out if shared with other schemas)
-- DROP EXTENSION IF EXISTS "uuid-ossp";
