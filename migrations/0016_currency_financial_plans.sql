CREATE TABLE currency_financial_plans (
 currency_code TEXT PRIMARY KEY REFERENCES currencies(code),
 revision INTEGER NOT NULL CHECK(revision>0),
 start_year INTEGER NOT NULL CHECK(start_year BETWEEN 2 AND 9998),
 years INTEGER NOT NULL CHECK(years BETWEEN 1 AND 200 AND start_year+years<=9999),
 mode TEXT NOT NULL CHECK(mode IN ('goal','contribution')),
 goal TEXT NOT NULL CHECK(length(goal) BETWEEN 1 AND 120 AND goal NOT GLOB '*[^0-9.]*'
  AND substr(goal,1,1)!='.'
  AND substr(goal,-1) NOT IN ('.','-')
  AND length(goal)-length(replace(goal,'.',''))<=1),
 contribution TEXT NOT NULL CHECK(length(contribution) BETWEEN 1 AND 120 AND contribution NOT GLOB '*[^0-9.]*'
  AND substr(contribution,1,1)!='.'
  AND substr(contribution,-1) NOT IN ('.','-')
  AND length(contribution)-length(replace(contribution,'.',''))<=1),
 return_percent TEXT NOT NULL CHECK(length(return_percent) BETWEEN 1 AND 120 AND return_percent NOT GLOB '*[^0-9.-]*'
  AND substr(return_percent,1,1)!='.'
  AND substr(return_percent,-1) NOT IN ('.','-')
  AND length(return_percent)-length(replace(return_percent,'.',''))<=1 AND return_percent!='-'
  AND substr(return_percent,1,2)!='-.'
  AND instr(substr(return_percent,2),'-')=0
  AND (substr(return_percent,1,1)!='-' OR length(ltrim(substr(return_percent,2,instr(return_percent||'.','.')-2),'0'))<=2)),
 inflation_percent TEXT NOT NULL CHECK(length(inflation_percent) BETWEEN 1 AND 120 AND inflation_percent NOT GLOB '*[^0-9.-]*'
  AND substr(inflation_percent,1,1)!='.'
  AND substr(inflation_percent,-1) NOT IN ('.','-')
  AND length(inflation_percent)-length(replace(inflation_percent,'.',''))<=1 AND inflation_percent!='-'
  AND substr(inflation_percent,1,2)!='-.'
  AND instr(substr(inflation_percent,2),'-')=0
  AND (substr(inflation_percent,1,1)!='-' OR length(ltrim(substr(inflation_percent,2,instr(inflation_percent||'.','.')-2),'0'))<=2)),
 benchmark_id INTEGER NOT NULL REFERENCES benchmarks(id),
 allow_estimated INTEGER NOT NULL CHECK(allow_estimated IN (0,1)),
 initial_capital TEXT NOT NULL CHECK(length(initial_capital)>0 AND initial_capital NOT GLOB '*[^0-9.-]*'
  AND substr(initial_capital,1,1)!='.'
  AND substr(initial_capital,-1) NOT IN ('.','-')
  AND length(initial_capital)-length(replace(initial_capital,'.',''))<=1 AND initial_capital!='-'
  AND substr(initial_capital,1,2)!='-.'
  AND instr(substr(initial_capital,2),'-')=0),
 initial_numerator TEXT NOT NULL CHECK(length(initial_numerator)>0 AND initial_numerator NOT GLOB '*[^0-9-]*'
  AND initial_numerator!='-'
  AND instr(substr(initial_numerator,2),'-')=0),
 initial_denominator TEXT NOT NULL CHECK(length(initial_denominator)>0 AND initial_denominator NOT GLOB '*[^0-9]*'

  AND length(ltrim(initial_denominator,'0'))>0),
 original TEXT NOT NULL CHECK(json_valid(original) AND json_type(original)='object')
) STRICT;

-- Copy only currencies with an unambiguous legacy plan. All legacy rows remain intact.
INSERT INTO currency_financial_plans(currency_code,revision,start_year,years,mode,goal,contribution,
 return_percent,inflation_percent,benchmark_id,allow_estimated,initial_capital,
 initial_numerator,initial_denominator,original)
SELECT p.currency_code,f.revision,f.start_year,f.years,f.mode,f.goal,f.contribution,
 f.return_percent,f.inflation_percent,f.benchmark_id,f.allow_estimated,f.initial_capital,
 f.initial_numerator,f.initial_denominator,f.original
FROM financial_plans f JOIN portfolios p ON p.id=f.portfolio_id
WHERE NOT EXISTS (
 SELECT 1 FROM financial_plans other JOIN portfolios op ON op.id=other.portfolio_id
 WHERE op.currency_code=p.currency_code AND other.portfolio_id!=f.portfolio_id
);
