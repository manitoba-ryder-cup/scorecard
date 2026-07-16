-- Seed the code-defined match formats. Each name corresponds to scoring rules
-- implemented in the application, so formats are seeded here rather than created by
-- users. Singles/Fourball score per player; Alternate Shot/Scramble/Modified Scotch
-- are one-ball (per team). Add a new format here when its scoring rule is coded.
INSERT INTO match_formats (name) VALUES
    ('Singles'),
    ('Fourball'),
    ('Alternate Shot'),
    ('Scramble'),
    ('Modified Scotch')
ON CONFLICT (name) DO NOTHING;
