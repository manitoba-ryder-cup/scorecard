-- Undrafting a player used to cascade her out of every lineup she held a place in, leaving the
-- match a side short — a lineup nothing can write and nobody can play. Where the match had
-- been played it also stranded the scores, and only the per-player ones were guarded.
--
-- RESTRICT refuses the undraft instead, and unlike the constraint in 003 it covers every
-- format: a lineup names its players whatever the scoring grain, so there is no null column
-- here for MATCH SIMPLE to skip. Substituting the player out is the way through.
-- IF EXISTS because this runs unattended at startup, and a failed attempt may have left the
-- constraint dropped; the ADD below restores it from whichever of the two states it finds.
ALTER TABLE match_participants
    DROP CONSTRAINT IF EXISTS fk__match_participants__team_id_player_id__team_members;

-- NOT VALID because the CASCADE constraint this replaces already guaranteed these rows, and
-- the scan would run under the FORCE RLS policy, which a migration has no tenant to satisfy.
ALTER TABLE match_participants
    ADD CONSTRAINT fk__match_participants__team_id_player_id__team_members
        FOREIGN KEY (team_id, player_id)
        REFERENCES team_members(team_id, player_id)
        ON DELETE RESTRICT NOT VALID;
