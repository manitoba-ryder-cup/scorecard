-- Remove the seeded match formats. RESTRICT-safe: fails if a match references one,
-- which is the correct signal that seeded data is still in use.
DELETE FROM match_formats
WHERE name IN ('Singles', 'Fourball', 'Alt Shot', 'Scramble', 'Scotch');
