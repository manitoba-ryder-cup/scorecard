-- Scores used to cascade away with the participant they belonged to, which let two routes
-- destroy a played match without anything recomputing what it left behind: removing a
-- participant, and undrafting a player (that one reaches match_participants by cascade).
-- Either left match_results claiming a finished match whose scores were gone, so the same
-- cup read as finished and never-played at once depending on which endpoint was asked.
--
-- RESTRICT refuses both instead. A match or tournament delete still cascades, because
-- scores reference matches directly and that path clears them in the same statement.
-- IF EXISTS because this runs unattended at startup, and a failed attempt may have left the
-- constraint dropped; the ADD below restores it from whichever of the two states it finds.
ALTER TABLE scores
    DROP CONSTRAINT IF EXISTS fk__scores__match_id_player_id__match_participants;

-- NOT VALID because the CASCADE constraint this replaces already guaranteed these rows, and
-- the scan would run under the FORCE RLS policy, which a migration has no tenant to satisfy.
ALTER TABLE scores
    ADD CONSTRAINT fk__scores__match_id_player_id__match_participants
        FOREIGN KEY (match_id, player_id)
        REFERENCES match_participants(match_id, player_id)
        ON DELETE RESTRICT NOT VALID;
