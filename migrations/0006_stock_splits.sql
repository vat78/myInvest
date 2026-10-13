-- Preserve all rows and the AUTOINCREMENT high-water mark inside the migration transaction.
CREATE TEMP TABLE split_migration_operations AS SELECT * FROM operations;
CREATE TEMP TABLE split_migration_cash AS SELECT * FROM cash_flow_details;
CREATE TEMP TABLE split_migration_trades AS SELECT * FROM trade_details;
CREATE TEMP TABLE split_migration_sequence AS SELECT seq FROM sqlite_sequence WHERE name='operations';
DROP TABLE cash_flow_details;
DROP TABLE trade_details;
DROP TABLE operations;
CREATE TABLE operations (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 portfolio_id INTEGER NOT NULL REFERENCES portfolios(id),
 kind TEXT NOT NULL CHECK(kind IN ('deposit','withdrawal','buy','sell','split')),
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
 OR (OLD.kind='split')!=(NEW.kind='split')
BEGIN SELECT RAISE(ABORT,'operation identity or family cannot change'); END;
CREATE TABLE split_details (
 operation_id INTEGER PRIMARY KEY REFERENCES operations(id) ON DELETE CASCADE,
 asset_id INTEGER NOT NULL REFERENCES assets(id),
 old_units TEXT NOT NULL CHECK(length(old_units)>0 AND old_units NOT GLOB '*[^0-9]*' AND substr(old_units,1,1) BETWEEN '1' AND '9'),
 new_units TEXT NOT NULL CHECK(length(new_units)>0 AND new_units NOT GLOB '*[^0-9]*' AND substr(new_units,1,1) BETWEEN '1' AND '9'),
 CHECK(old_units!=new_units)
) STRICT;
CREATE TRIGGER split_details_insert_kind BEFORE INSERT ON split_details
WHEN NOT EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND kind='split')
BEGIN SELECT RAISE(ABORT,'operation detail kind mismatch'); END;
CREATE TRIGGER split_details_update_kind BEFORE UPDATE ON split_details
WHEN NOT EXISTS(SELECT 1 FROM operations WHERE id=NEW.operation_id AND kind='split')
BEGIN SELECT RAISE(ABORT,'operation detail kind mismatch'); END;
INSERT INTO operations SELECT * FROM split_migration_operations;
INSERT INTO cash_flow_details SELECT * FROM split_migration_cash;
INSERT INTO trade_details SELECT * FROM split_migration_trades;
UPDATE sqlite_sequence SET seq=max(seq,coalesce((SELECT seq FROM split_migration_sequence),0)) WHERE name='operations';
INSERT INTO sqlite_sequence(name,seq) SELECT 'operations',seq FROM split_migration_sequence
 WHERE NOT EXISTS(SELECT 1 FROM sqlite_sequence WHERE name='operations');
DROP TABLE split_migration_operations;
DROP TABLE split_migration_cash;
DROP TABLE split_migration_trades;
DROP TABLE split_migration_sequence;
