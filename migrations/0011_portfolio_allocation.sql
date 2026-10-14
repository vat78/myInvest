CREATE TABLE portfolio_allocation_plans (
 portfolio_id INTEGER PRIMARY KEY REFERENCES portfolios(id),
 generation TEXT NOT NULL UNIQUE CHECK(length(generation)>0),
 revision INTEGER NOT NULL CHECK(revision>0),
 cash_weight_units INTEGER NOT NULL CHECK(cash_weight_units BETWEEN 0 AND 10000)
) STRICT;
CREATE TABLE portfolio_allocation_targets (
 portfolio_id INTEGER NOT NULL REFERENCES portfolio_allocation_plans(portfolio_id) ON DELETE CASCADE,
 asset_id INTEGER NOT NULL REFERENCES assets(id),
 weight_units INTEGER NOT NULL CHECK(weight_units BETWEEN 0 AND 10000),
 PRIMARY KEY(portfolio_id,asset_id)
) STRICT;
