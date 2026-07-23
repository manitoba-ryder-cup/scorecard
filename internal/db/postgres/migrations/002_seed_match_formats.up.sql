-- Seed the code-defined match formats. Each name corresponds to scoring rules
-- implemented in the application, so formats are seeded here rather than created by
-- users. Singles/Fourball score per player; Alt Shot/Scramble/Scotch
-- are one-ball (per team). Add a new format here when its scoring rule is coded.
INSERT INTO match_formats (name) VALUES
    ('Singles'),
    ('Fourball'),
    ('Alt Shot'),
    ('Scramble'),
    ('Scotch')
ON CONFLICT (name) DO NOTHING;
