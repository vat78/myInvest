CREATE TABLE asset_quotes (
 asset_id INTEGER NOT NULL REFERENCES assets(id),
 date TEXT NOT NULL CHECK (
  date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
  AND substr(date,1,4)!='0000'
  AND date(date,'+0 days') IS NOT NULL AND date(date,'+0 days')=date
 ),
 currency_code TEXT NOT NULL REFERENCES currencies(code),
 price TEXT NOT NULL CHECK (
  length(price)>0 AND price NOT GLOB '*[^0-9.]*'
  AND substr(price,1,1)!='.' AND substr(price,-1)!='.'
  AND length(price)-length(replace(price,'.',''))<=1
  AND length(replace(replace(price,'0',''),'.',''))>0
 ),
 PRIMARY KEY(asset_id,date,currency_code)
) STRICT;
CREATE INDEX asset_quotes_date ON asset_quotes(date DESC,asset_id,currency_code);
