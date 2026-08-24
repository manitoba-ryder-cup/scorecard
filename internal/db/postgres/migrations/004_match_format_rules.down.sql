ALTER TABLE match_formats
    DROP CONSTRAINT ck__match_formats__players_per_side,
    DROP COLUMN scores_per_player,
    DROP COLUMN players_per_side;
