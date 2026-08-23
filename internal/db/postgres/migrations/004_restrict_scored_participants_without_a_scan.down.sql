-- NOT VALID for the reason the up migration gives.
ALTER TABLE scores
    DROP CONSTRAINT fk__scores__match_id_player_id__match_participants;

ALTER TABLE scores
    ADD CONSTRAINT fk__scores__match_id_player_id__match_participants
        FOREIGN KEY (match_id, player_id)
        REFERENCES match_participants(match_id, player_id)
        ON DELETE CASCADE NOT VALID;
