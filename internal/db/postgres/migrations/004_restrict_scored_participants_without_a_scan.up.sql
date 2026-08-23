-- 003 adds this same constraint and fails on a database with scores in it: validating the
-- existing rows scans `scores` under its FORCE RLS policy, which casts an unset tenant to
-- uuid. NOT VALID skips a scan the replaced CASCADE constraint already guaranteed.
ALTER TABLE scores
    DROP CONSTRAINT fk__scores__match_id_player_id__match_participants;

ALTER TABLE scores
    ADD CONSTRAINT fk__scores__match_id_player_id__match_participants
        FOREIGN KEY (match_id, player_id)
        REFERENCES match_participants(match_id, player_id)
        ON DELETE RESTRICT NOT VALID;
