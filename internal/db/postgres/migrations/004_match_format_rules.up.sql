-- Data rather than a rule read off the name, so a lineup that does not fit its format can be
-- refused rather than trusted.
ALTER TABLE match_formats
    ADD COLUMN players_per_side INTEGER NOT NULL DEFAULT 2,
    ADD COLUMN scores_per_player BOOLEAN NOT NULL DEFAULT false,
    ADD CONSTRAINT ck__match_formats__players_per_side CHECK (players_per_side > 0);

UPDATE match_formats SET players_per_side = 1, scores_per_player = true WHERE name = 'Singles';
UPDATE match_formats SET scores_per_player = true WHERE name = 'Fourball';

-- Dropped again so a format added later has to state its rules rather than inherit these.
ALTER TABLE match_formats
    ALTER COLUMN players_per_side DROP DEFAULT,
    ALTER COLUMN scores_per_player DROP DEFAULT;
