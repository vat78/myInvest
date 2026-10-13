CREATE TABLE assets (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 ticker TEXT NOT NULL UNIQUE CHECK(length(ticker)>0 AND trim(ticker)=ticker)
) STRICT;
CREATE TABLE operations (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 portfolio_id INTEGER NOT NULL REFERENCES portfolios(id),
 kind TEXT NOT NULL CHECK(kind IN ('deposit','withdrawal','buy','sell')),
 date TEXT NOT NULL CHECK (
  date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
  AND substr(date,1,4)!='0000'
  AND date(date,'+0 days') IS NOT NULL AND date(date,'+0 days')=date
 ),
 comment TEXT NOT NULL
) STRICT;
CREATE INDEX operations_portfolio_date ON operations(portfolio_id,date DESC,id DESC);
CREATE TABLE cash_flow_details (
 operation_id INTEGER PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE,
 currency_code TEXT NOT NULL REFERENCES currencies(code),
amount TEXT NOT NULL CHECK (
  length(amount)>0 AND amount NOT GLOB '*[^0-9.]*'
  AND substr(amount,1,1)!='.' AND substr(amount,-1)!='.'
  AND length(amount)-length(replace(amount,'.',''))<=1
  AND length(replace(replace(amount,'0',''),'.',''))>0
 ),
portfolio_amount TEXT NOT NULL CHECK (
  length(portfolio_amount)>0 AND portfolio_amount NOT GLOB '*[^0-9.]*'
  AND substr(portfolio_amount,1,1)!='.' AND substr(portfolio_amount,-1)!='.'
  AND length(portfolio_amount)-length(replace(portfolio_amount,'.',''))<=1
  AND length(replace(replace(portfolio_amount,'0',''),'.',''))>0
 )
) STRICT;
CREATE TABLE trade_details (
 operation_id INTEGER PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE,
 asset_id INTEGER NOT NULL REFERENCES assets(id),
quantity TEXT NOT NULL CHECK (
  length(quantity)>0 AND quantity NOT GLOB '*[^0-9.]*'
  AND substr(quantity,1,1)!='.' AND substr(quantity,-1)!='.'
  AND length(quantity)-length(replace(quantity,'.',''))<=1
  AND length(replace(replace(quantity,'0',''),'.',''))>0
 ),
amount TEXT NOT NULL CHECK (
  length(amount)>0 AND amount NOT GLOB '*[^0-9.]*'
  AND substr(amount,1,1)!='.' AND substr(amount,-1)!='.'
  AND length(amount)-length(replace(amount,'.',''))<=1
  AND length(replace(replace(amount,'0',''),'.',''))>0
 ),
expenses TEXT NOT NULL CHECK (
  length(expenses)>0 AND expenses NOT GLOB '*[^0-9.]*'
  AND substr(expenses,1,1)!='.' AND substr(expenses,-1)!='.'
  AND length(expenses)-length(replace(expenses,'.',''))<=1
 ),
average_price TEXT NOT NULL CHECK (
  length(average_price)>0 AND average_price NOT GLOB '*[^0-9.]*'
  AND substr(average_price,1,1)!='.' AND substr(average_price,-1)!='.'
  AND length(average_price)-length(replace(average_price,'.',''))<=1
 )
) STRICT;
CREATE TRIGGER cash_flow_details_insert_kind BEFORE INSERT ON cash_flow_details
WHEN NOT EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND kind IN ('deposit','withdrawal'))
BEGIN SELECT RAISE(ABORT,'operation detail kind mismatch'); END;
CREATE TRIGGER cash_flow_details_update_kind BEFORE UPDATE ON cash_flow_details
WHEN NOT EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND kind IN ('deposit','withdrawal'))
BEGIN SELECT RAISE(ABORT,'operation detail kind mismatch'); END;
CREATE TRIGGER trade_details_insert_kind BEFORE INSERT ON trade_details
WHEN NOT EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND kind IN ('buy','sell'))
BEGIN SELECT RAISE(ABORT,'operation detail kind mismatch'); END;
CREATE TRIGGER trade_details_update_kind BEFORE UPDATE ON trade_details
WHEN NOT EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND kind IN ('buy','sell'))
BEGIN SELECT RAISE(ABORT,'operation detail kind mismatch'); END;
CREATE TRIGGER operations_keep_family BEFORE UPDATE ON operations
WHEN NEW.id!=OLD.id OR NEW.portfolio_id!=OLD.portfolio_id
 OR (OLD.kind IN ('buy','sell'))!=(NEW.kind IN ('buy','sell'))
BEGIN SELECT RAISE(ABORT,'operation identity or family cannot change'); END;
CREATE TABLE daily_cash_balances (
 portfolio_id INTEGER NOT NULL REFERENCES portfolios(id),
 date TEXT NOT NULL,
 balance TEXT NOT NULL,
 PRIMARY KEY(portfolio_id,date)
) STRICT;
CREATE TABLE daily_asset_balances (
 portfolio_id INTEGER NOT NULL,
 date TEXT NOT NULL,
 asset_id INTEGER NOT NULL REFERENCES assets(id),
 quantity TEXT NOT NULL,
 PRIMARY KEY(portfolio_id,date,asset_id),
 FOREIGN KEY(portfolio_id,date) REFERENCES daily_cash_balances(portfolio_id,date) ON DELETE CASCADE
) STRICT;
INSERT INTO operations(id,portfolio_id,kind,date,comment)
 SELECT id,portfolio_id,kind,date,comment FROM cash_flows ORDER BY id;
INSERT INTO cash_flow_details(operation_id,currency_code,amount,portfolio_amount)
 SELECT id,currency_code,amount,portfolio_amount FROM cash_flows;
-- Preserve the AUTOINCREMENT high-water mark, including deleted legacy IDs.
UPDATE sqlite_sequence SET seq=max(seq,coalesce((SELECT seq FROM sqlite_sequence WHERE name='cash_flows'),0)) WHERE name='operations';
INSERT INTO sqlite_sequence(name,seq)
 SELECT 'operations',seq FROM sqlite_sequence WHERE name='cash_flows'
 AND NOT EXISTS(SELECT 1 FROM sqlite_sequence WHERE name='operations');
DROP TABLE cash_flows;
-- Exact initial daily totals are built in Go within this migration transaction.
