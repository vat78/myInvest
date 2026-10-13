CREATE TABLE portfolios (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    name_key TEXT NOT NULL UNIQUE CHECK (length(trim(name_key)) > 0),
    opened_on TEXT NOT NULL CHECK (
        opened_on GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
        AND substr(opened_on, 1, 4) != '0000'
        AND date(opened_on, '+0 days') IS NOT NULL AND date(opened_on, '+0 days') = opened_on
    ),
    currency_code TEXT NOT NULL REFERENCES currencies(code)
) STRICT;
