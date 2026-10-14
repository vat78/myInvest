-- Cumulative deposits from stored RUB amounts, never from directory rates.
-- NULL means at least one deposit has no stored RUB amount.
ALTER TABLE daily_cash_balances ADD COLUMN deposits_rub TEXT
 CHECK (deposits_rub IS NULL OR (length(deposits_rub)>0
 AND deposits_rub NOT GLOB '*[^0-9.]*'
 AND deposits_rub NOT GLOB '*.*.*'
 AND deposits_rub NOT LIKE '.%' AND deposits_rub NOT LIKE '%.'));
-- Go backfills exact cumulative values in the migration transaction.
