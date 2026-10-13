CREATE TABLE currencies (
    code TEXT PRIMARY KEY NOT NULL CHECK (length(code) = 3 AND code NOT GLOB '*[^A-Z]*')
) STRICT;

INSERT INTO currencies (code) VALUES ('USD'), ('RUB');

CREATE TABLE currency_rates (
    date TEXT NOT NULL CHECK (
        date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
        AND substr(date, 1, 4) != '0000'
        AND date(date, '+0 days') IS NOT NULL AND date(date, '+0 days') = date
    ),
    base TEXT NOT NULL REFERENCES currencies(code),
    quote TEXT NOT NULL REFERENCES currencies(code),
    value TEXT NOT NULL CHECK (
        length(value) > 0 AND value NOT GLOB '*[^0-9.]*'
        AND substr(value, 1, 1) != '.' AND substr(value, -1) != '.'
        AND length(value) - length(replace(value, '.', '')) <= 1
        AND length(replace(replace(value, '0', ''), '.', '')) > 0
    ),
    PRIMARY KEY (date, base, quote),
    CHECK (base != quote)
) STRICT;
