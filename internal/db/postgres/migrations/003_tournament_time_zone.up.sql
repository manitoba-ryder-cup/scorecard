-- A tournament's start_date/end_date are calendar dates, and a calendar date only means
-- something somewhere: "the day of" has to be read where the golf is played. Its tee
-- times are absolute instants that need the same zone to render as a wall clock.
--
-- An IANA name rather than an offset, so DST is the zone's problem and not ours. Existing
-- rows are the Manitoba Ryder Cup, which is where the default comes from.
ALTER TABLE tournaments
    ADD COLUMN time_zone VARCHAR(64) NOT NULL DEFAULT 'America/Winnipeg';
