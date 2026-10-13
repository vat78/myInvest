ALTER TABLE assets ADD COLUMN type_code TEXT CHECK (
    type_code IS NULL OR type_code IN (
        'commodities', 'bonds', 'money_market_funds', 'stocks',
        'real_estate', 'cryptocurrencies', 'altcoins'
    )
);
