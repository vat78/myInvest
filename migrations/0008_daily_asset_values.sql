-- Exact rational position values in the portfolio currency; NULL means unavailable.
ALTER TABLE daily_asset_balances ADD COLUMN value_numerator TEXT;
ALTER TABLE daily_asset_balances ADD COLUMN value_denominator TEXT
 CHECK ((value_numerator IS NULL AND value_denominator IS NULL) OR
 (value_numerator IS NOT NULL AND value_denominator IS NOT NULL
 AND length(value_denominator)>0 AND value_denominator NOT GLOB '*[^0-9]*'
 AND value_denominator GLOB '*[1-9]*'
 AND length(value_numerator)>0
 AND (value_numerator NOT GLOB '*[^0-9]*' OR
 (substr(value_numerator,1,1)='-' AND length(value_numerator)>1
 AND substr(value_numerator,2) NOT GLOB '*[^0-9]*'))));
ALTER TABLE daily_asset_balances ADD COLUMN price_operation_id INTEGER REFERENCES operations(id) ON DELETE SET NULL;
-- Go fills all values within the migration transaction.
