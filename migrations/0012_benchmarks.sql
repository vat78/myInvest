CREATE TABLE benchmarks (
 id INTEGER PRIMARY KEY AUTOINCREMENT,
 name TEXT NOT NULL CHECK(length(trim(name)) BETWEEN 1 AND 120),
 name_key TEXT NOT NULL UNIQUE CHECK(length(name_key)>0),
 currency_code TEXT REFERENCES currencies(code)
) STRICT;
CREATE TABLE benchmark_points (
 benchmark_id INTEGER NOT NULL REFERENCES benchmarks(id),
 date TEXT NOT NULL CHECK (
  date GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
  AND substr(date,1,4)!='0000'
  AND date(date,'+0 days') IS NOT NULL AND date(date,'+0 days')=date
 ),
 mode TEXT NOT NULL CHECK(mode IN ('value','growth')),
 input TEXT NOT NULL CHECK(length(input)>0 AND input NOT GLOB '*[^0-9.-]*'
  AND substr(input,1,1)!='.' AND substr(input,1,2)!='-.' AND substr(input,-1) NOT IN ('.','-')
  AND instr(substr(input,2),'-')=0
  AND length(input)-length(replace(input,'.',''))<=1),
 value TEXT NOT NULL CHECK (
  length(value)>0 AND value NOT GLOB '*[^0-9.]*'
  AND substr(value,1,1)!='.' AND substr(value,-1)!='.'
  AND length(value)-length(replace(value,'.',''))<=1
  AND length(replace(replace(value,'0',''),'.',''))>0
 ),
 growth TEXT CHECK(growth IS NULL OR (length(growth)>0 AND growth NOT GLOB '*[^0-9.-]*'
  AND substr(growth,1,1)!='.' AND substr(growth,1,2)!='-.' AND substr(growth,-1) NOT IN ('.','-')
  AND instr(substr(growth,2),'-')=0
  AND length(growth)-length(replace(growth,'.',''))<=1)),
 CHECK(mode!='value' OR input=value),
 CHECK(mode!='growth' OR (growth IS NOT NULL AND growth=input
  AND (substr(input,1,1)!='-' OR
   length(ltrim(substr(input,2,instr(input||'.','.')-2),'0'))<=2))),
 PRIMARY KEY(benchmark_id,date)
) STRICT;
