ALTER TABLE match_participants
    DROP CONSTRAINT IF EXISTS fk__match_participants__team_id_player_id__team_members;

-- NOT VALID for the reason the up migration gives.
ALTER TABLE match_participants
    ADD CONSTRAINT fk__match_participants__team_id_player_id__team_members
        FOREIGN KEY (team_id, player_id)
        REFERENCES team_members(team_id, player_id)
        ON DELETE CASCADE NOT VALID;
