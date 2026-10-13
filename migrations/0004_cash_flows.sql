CREATE TABLE cash_flows (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 portfolio_id INTEGER NOT NULL REFERENCES portfolios(id),
 kind TEXT NOT NULL CHECK (kind IN ('deposit','withdrawal')),
 date TEXT NOT NULL CHECK (
  date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
  AND substr(date,1,4) != '0000'
  AND date(date,'+0 days') IS NOT NULL AND date(date,'+0 days') = date
 ),
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
 ),
 comment TEXT NOT NULL
) STRICT;
CREATE INDEX cash_flows_portfolio_date ON cash_flows(portfolio_id,date DESC,id DESC);
