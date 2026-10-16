export namespace benchmarks {

	export class Benchmark {
	    sources: string;
	    id: string;
	    name: string;
	    currencyCode: string;

	    static createFrom(source: any = {}) {
	        return new Benchmark(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sources = source["sources"];
	        this.id = source["id"];
	        this.name = source["name"];
	        this.currencyCode = source["currencyCode"];
	    }
	}
	export class Point {
	    benchmarkID: string;
	    date: string;
	    mode: string;
	    input: string;
	    value: string;
	    growth: string;

	    static createFrom(source: any = {}) {
	        return new Point(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.benchmarkID = source["benchmarkID"];
	        this.date = source["date"];
	        this.mode = source["mode"];
	        this.input = source["input"];
	        this.value = source["value"];
	        this.growth = source["growth"];
	    }
	}

}

export namespace desktop {

	export class AllocationResult {
	    errors: Record<string, string>;
	    message: string;
	    conflict: boolean;

	    static createFrom(source: any = {}) {
	        return new AllocationResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.errors = source["errors"];
	        this.message = source["message"];
	        this.conflict = source["conflict"];
	    }
	}
	export class AssetDTO {
	    id: string;
	    ticker: string;
	    typeCode: string;

	    static createFrom(source: any = {}) {
	        return new AssetDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ticker = source["ticker"];
	        this.typeCode = source["typeCode"];
	    }
	}
	export class AssetInputDTO {
	    ticker: string;
	    typeCode: string;

	    static createFrom(source: any = {}) {
	        return new AssetInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticker = source["ticker"];
	        this.typeCode = source["typeCode"];
	    }
	}
	export class AssetResultDTO {
	    asset: AssetDTO;
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new AssetResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.asset = this.convertValues(source["asset"], AssetDTO);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class AssetTypeDTO {
	    code: string;
	    name: string;

	    static createFrom(source: any = {}) {
	        return new AssetTypeDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	    }
	}
	export class AssetTypeInputDTO {
	    id: string;
	    typeCode: string;

	    static createFrom(source: any = {}) {
	        return new AssetTypeInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.typeCode = source["typeCode"];
	    }
	}
	export class BenchmarkPointResult {
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new BenchmarkPointResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }
	}
	export class BenchmarkPreviewResult {
	    point: benchmarks.Point;
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new BenchmarkPreviewResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.point = this.convertValues(source["point"], benchmarks.Point);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class BenchmarkResult {
	    benchmark: benchmarks.Benchmark;
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new BenchmarkResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.benchmark = this.convertValues(source["benchmark"], benchmarks.Benchmark);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class CashFlowDTO {
	    id: string;
	    portfolioID: string;
	    kind: string;
	    date: string;
	    currencyCode: string;
	    amount: string;
	    portfolioAmount: string;
	    comment: string;
	    rate: string;
	    assetID: string;
	    ticker: string;
	    quantity: string;
	    expenses: string;
	    averagePrice: string;
	    rounded: boolean;
	    moneyChange: string;
	    oldUnits: string;
	    newUnits: string;
	    quantityBefore: string;
	    quantityAfter: string;
	    reverseSplit: boolean;

	    static createFrom(source: any = {}) {
	        return new CashFlowDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.portfolioID = source["portfolioID"];
	        this.kind = source["kind"];
	        this.date = source["date"];
	        this.currencyCode = source["currencyCode"];
	        this.amount = source["amount"];
	        this.portfolioAmount = source["portfolioAmount"];
	        this.comment = source["comment"];
	        this.rate = source["rate"];
	        this.assetID = source["assetID"];
	        this.ticker = source["ticker"];
	        this.quantity = source["quantity"];
	        this.expenses = source["expenses"];
	        this.averagePrice = source["averagePrice"];
	        this.rounded = source["rounded"];
	        this.moneyChange = source["moneyChange"];
	        this.oldUnits = source["oldUnits"];
	        this.newUnits = source["newUnits"];
	        this.quantityBefore = source["quantityBefore"];
	        this.quantityAfter = source["quantityAfter"];
	        this.reverseSplit = source["reverseSplit"];
	    }
	}
	export class CashFlowInputDTO {
	    kind: string;
	    date: string;
	    currencyCode: string;
	    amount: string;
	    portfolioAmount: string;
	    comment: string;

	    static createFrom(source: any = {}) {
	        return new CashFlowInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.date = source["date"];
	        this.currencyCode = source["currencyCode"];
	        this.amount = source["amount"];
	        this.portfolioAmount = source["portfolioAmount"];
	        this.comment = source["comment"];
	    }
	}
	export class CashRateDTO {
	    rate: string;
	    errors: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new CashRateDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.rate = source["rate"];
	        this.errors = source["errors"];
	    }
	}
	export class CashResultDTO {
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new CashResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }
	}
	export class CashWarningDTO {
	    date: string;
	    balance: string;

	    static createFrom(source: any = {}) {
	        return new CashWarningDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.balance = source["balance"];
	    }
	}
	export class WarningDTO {
	    operationID: string;
	    date: string;
	    ticker: string;
	    available: string;
	    quantity: string;

	    static createFrom(source: any = {}) {
	        return new WarningDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.operationID = source["operationID"];
	        this.date = source["date"];
	        this.ticker = source["ticker"];
	        this.available = source["available"];
	        this.quantity = source["quantity"];
	    }
	}
	export class CashSnapshotDTO {
	    currencyCode: string;
	    balance: string;
	    negative: boolean;
	    operations: CashFlowDTO[];
	    warnings: WarningDTO[];
	    cashWarnings: CashWarningDTO[];

	    static createFrom(source: any = {}) {
	        return new CashSnapshotDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currencyCode = source["currencyCode"];
	        this.balance = source["balance"];
	        this.negative = source["negative"];
	        this.operations = this.convertValues(source["operations"], CashFlowDTO);
	        this.warnings = this.convertValues(source["warnings"], WarningDTO);
	        this.cashWarnings = this.convertValues(source["cashWarnings"], CashWarningDTO);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class CurrencyPairDTO {
	    date: string;
	    base: string;
	    quote: string;

	    static createFrom(source: any = {}) {
	        return new CurrencyPairDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.base = source["base"];
	        this.quote = source["quote"];
	    }
	}
	export class CurrencyRateDTO {
	    displayValue: string;
	    value: string;
	    found: boolean;

	    static createFrom(source: any = {}) {
	        return new CurrencyRateDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.displayValue = source["displayValue"];
	        this.value = source["value"];
	        this.found = source["found"];
	    }
	}
	export class DividendInputDTO {
	    date: string;
	    assetID: string;
	    amount: string;
	    comment: string;

	    static createFrom(source: any = {}) {
	        return new DividendInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.assetID = source["assetID"];
	        this.amount = source["amount"];
	        this.comment = source["comment"];
	    }
	}
	export class PlanningResult {
	    plan: planning.Plan;
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new PlanningResult(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.plan = this.convertValues(source["plan"], planning.Plan);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class PortfolioDTO {
	    id: string;
	    name: string;
	    openedOn: string;
	    currencyCode: string;

	    static createFrom(source: any = {}) {
	        return new PortfolioDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.openedOn = source["openedOn"];
	        this.currencyCode = source["currencyCode"];
	    }
	}
	export class PortfolioInputDTO {
	    name: string;
	    openedOn: string;
	    currencyCode: string;

	    static createFrom(source: any = {}) {
	        return new PortfolioInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.openedOn = source["openedOn"];
	        this.currencyCode = source["currencyCode"];
	    }
	}
	export class QuoteDTO {
	    assetID: string;
	    date: string;
	    currencyCode: string;
	    price: string;
	    ticker: string;

	    static createFrom(source: any = {}) {
	        return new QuoteDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetID = source["assetID"];
	        this.date = source["date"];
	        this.currencyCode = source["currencyCode"];
	        this.price = source["price"];
	        this.ticker = source["ticker"];
	    }
	}
	export class QuoteResultDTO {
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new QuoteResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }
	}
	export class SaveCurrencyRateDTO {
	    date: string;
	    base: string;
	    quote: string;
	    value: string;

	    static createFrom(source: any = {}) {
	        return new SaveCurrencyRateDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.base = source["base"];
	        this.quote = source["quote"];
	        this.value = source["value"];
	    }
	}
	export class SavePortfolioResultDTO {
	    portfolio: PortfolioDTO;
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new SavePortfolioResultDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.portfolio = this.convertValues(source["portfolio"], PortfolioDTO);
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class SplitInputDTO {
	    date: string;
	    assetID: string;
	    oldUnits: string;
	    newUnits: string;
	    comment: string;

	    static createFrom(source: any = {}) {
	        return new SplitInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.assetID = source["assetID"];
	        this.oldUnits = source["oldUnits"];
	        this.newUnits = source["newUnits"];
	        this.comment = source["comment"];
	    }
	}
	export class SplitPreviewDTO {
	    quantityBefore: string;
	    quantityAfter: string;
	    errors: Record<string, string>;
	    message: string;

	    static createFrom(source: any = {}) {
	        return new SplitPreviewDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.quantityBefore = source["quantityBefore"];
	        this.quantityAfter = source["quantityAfter"];
	        this.errors = source["errors"];
	        this.message = source["message"];
	    }
	}
	export class TradeInputDTO {
	    kind: string;
	    date: string;
	    assetID: string;
	    quantity: string;
	    amount: string;
	    expenses: string;
	    comment: string;

	    static createFrom(source: any = {}) {
	        return new TradeInputDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.date = source["date"];
	        this.assetID = source["assetID"];
	        this.quantity = source["quantity"];
	        this.amount = source["amount"];
	        this.expenses = source["expenses"];
	        this.comment = source["comment"];
	    }
	}
	export class TradePreviewDTO {
	    averagePrice: string;
	    rounded: boolean;
	    moneyChange: string;
	    errors: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new TradePreviewDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.averagePrice = source["averagePrice"];
	        this.rounded = source["rounded"];
	        this.moneyChange = source["moneyChange"];
	        this.errors = source["errors"];
	    }
	}

}

export namespace main {

	export class AppInfo {
	    name: string;

	    static createFrom(source: any = {}) {
	        return new AppInfo(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	    }
	}

}

export namespace operations {

	export class Asset {
	    ID: string;
	    Ticker: string;
	    TypeCode: string;

	    static createFrom(source: any = {}) {
	        return new Asset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ID = source["ID"];
	        this.Ticker = source["Ticker"];
	        this.TypeCode = source["TypeCode"];
	    }
	}

}

export namespace planning {

	export class Factor {
	    year: string;
	    value: string;
	    source: string;

	    static createFrom(source: any = {}) {
	        return new Factor(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.value = source["value"];
	        this.source = source["source"];
	    }
	}
	export class Parameters {
	    currencyCode: string;
	    revision: string;
	    startYear: string;
	    years: string;
	    mode: string;
	    goal: string;
	    contribution: string;
	    returnPercent: string;
	    inflationPercent: string;
	    benchmarkID: string;
	    allowEstimated: boolean;
	    initialCapital: string;

	    static createFrom(source: any = {}) {
	        return new Parameters(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.currencyCode = source["currencyCode"];
	        this.revision = source["revision"];
	        this.startYear = source["startYear"];
	        this.years = source["years"];
	        this.mode = source["mode"];
	        this.goal = source["goal"];
	        this.contribution = source["contribution"];
	        this.returnPercent = source["returnPercent"];
	        this.inflationPercent = source["inflationPercent"];
	        this.benchmarkID = source["benchmarkID"];
	        this.allowEstimated = source["allowEstimated"];
	        this.initialCapital = source["initialCapital"];
	    }
	}
	export class Point {
	    date: string;
	    nominal: string;
	    real: string;

	    static createFrom(source: any = {}) {
	        return new Point(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.nominal = source["nominal"];
	        this.real = source["real"];
	    }
	}
	export class Row {
	    year: string;
	    before: string;
	    contribution: string;
	    inflation: string;
	    source: string;
	    income: string;
	    capital: string;
	    realCapital: string;

	    static createFrom(source: any = {}) {
	        return new Row(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.year = source["year"];
	        this.before = source["before"];
	        this.contribution = source["contribution"];
	        this.inflation = source["inflation"];
	        this.source = source["source"];
	        this.income = source["income"];
	        this.capital = source["capital"];
	        this.realCapital = source["realCapital"];
	    }
	}
	export class Scenario {
	    complete: boolean;
	    issues: string[];
	    firstContribution: string;
	    finalCapital: string;
	    finalRealCapital: string;
	    excess: string;
	    goalReached: boolean;
	    rows: Row[];
	    points: Point[];
	    factors: Factor[];

	    static createFrom(source: any = {}) {
	        return new Scenario(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.complete = source["complete"];
	        this.issues = source["issues"];
	        this.firstContribution = source["firstContribution"];
	        this.finalCapital = source["finalCapital"];
	        this.finalRealCapital = source["finalRealCapital"];
	        this.excess = source["excess"];
	        this.goalReached = source["goalReached"];
	        this.rows = this.convertValues(source["rows"], Row);
	        this.points = this.convertValues(source["points"], Point);
	        this.factors = this.convertValues(source["factors"], Factor);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Plan {
	    found: boolean;
	    parameters: Parameters;
	    currency: string;
	    original: Scenario;
	    current: Scenario;

	    static createFrom(source: any = {}) {
	        return new Plan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.parameters = this.convertValues(source["parameters"], Parameters);
	        this.currency = source["currency"];
	        this.original = this.convertValues(source["original"], Scenario);
	        this.current = this.convertValues(source["current"], Scenario);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}



}

export namespace valuation {

	export class AllocationRow {
	    id: string;
	    ticker: string;
	    cash: boolean;
	    quantity: string;
	    currentValue: string;
	    percent: string;
	    targetValue: string;
	    targetQuantity: string;
	    buyQuantity: string;
	    buyValue: string;
	    rebalanceQuantity: string;
	    rebalanceValue: string;
	    priceSource: string;
	    priceDate: string;
	    priceCurrency: string;
	    priceRate: string;
	    priceRateDate: string;
	    issue: string;

	    static createFrom(source: any = {}) {
	        return new AllocationRow(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ticker = source["ticker"];
	        this.cash = source["cash"];
	        this.quantity = source["quantity"];
	        this.currentValue = source["currentValue"];
	        this.percent = source["percent"];
	        this.targetValue = source["targetValue"];
	        this.targetQuantity = source["targetQuantity"];
	        this.buyQuantity = source["buyQuantity"];
	        this.buyValue = source["buyValue"];
	        this.rebalanceQuantity = source["rebalanceQuantity"];
	        this.rebalanceValue = source["rebalanceValue"];
	        this.priceSource = source["priceSource"];
	        this.priceDate = source["priceDate"];
	        this.priceCurrency = source["priceCurrency"];
	        this.priceRate = source["priceRate"];
	        this.priceRateDate = source["priceRateDate"];
	        this.issue = source["issue"];
	    }
	}
	export class Target {
	    assetID: string;
	    percent: string;

	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.assetID = source["assetID"];
	        this.percent = source["percent"];
	    }
	}
	export class AllocationPlan {
	    generation: string;
	    revision: string;
	    cashPercent: string;
	    targets: Target[];

	    static createFrom(source: any = {}) {
	        return new AllocationPlan(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.generation = source["generation"];
	        this.revision = source["revision"];
	        this.cashPercent = source["cashPercent"];
	        this.targets = this.convertValues(source["targets"], Target);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Allocation {
	    portfolioID: string;
	    date: string;
	    currency: string;
	    plan: AllocationPlan;
	    assets: operations.Asset[];
	    rows: AllocationRow[];
	    total: string;
	    cash: string;
	    reserve: string;
	    cashAfterBuys: string;
	    reserveShortfall: string;
	    complete: boolean;
	    canCalculate: boolean;
	    issues: string[];

	    static createFrom(source: any = {}) {
	        return new Allocation(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.portfolioID = source["portfolioID"];
	        this.date = source["date"];
	        this.currency = source["currency"];
	        this.plan = this.convertValues(source["plan"], AllocationPlan);
	        this.assets = this.convertValues(source["assets"], operations.Asset);
	        this.rows = this.convertValues(source["rows"], AllocationRow);
	        this.total = source["total"];
	        this.cash = source["cash"];
	        this.reserve = source["reserve"];
	        this.cashAfterBuys = source["cashAfterBuys"];
	        this.reserveShortfall = source["reserveShortfall"];
	        this.complete = source["complete"];
	        this.canCalculate = source["canCalculate"];
	        this.issues = source["issues"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class AllocationPreview {
	    total: string;
	    difference: string;
	    errors: Record<string, string>;

	    static createFrom(source: any = {}) {
	        return new AllocationPreview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.difference = source["difference"];
	        this.errors = source["errors"];
	    }
	}

	export class AnalysisAsset {
	    priceSource: string;
	    priceDate: string;
	    priceCurrency: string;
	    priceRate: string;
	    priceRateDate: string;
	    id: string;
	    ticker: string;
	    quantity: string;
	    currentValue: string;
	    percent: string;
	    averagePrice: string;
	    averageValue: string;
	    chartValue: string;

	    static createFrom(source: any = {}) {
	        return new AnalysisAsset(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.priceSource = source["priceSource"];
	        this.priceDate = source["priceDate"];
	        this.priceCurrency = source["priceCurrency"];
	        this.priceRate = source["priceRate"];
	        this.priceRateDate = source["priceRateDate"];
	        this.id = source["id"];
	        this.ticker = source["ticker"];
	        this.quantity = source["quantity"];
	        this.currentValue = source["currentValue"];
	        this.percent = source["percent"];
	        this.averagePrice = source["averagePrice"];
	        this.averageValue = source["averageValue"];
	        this.chartValue = source["chartValue"];
	    }
	}
	export class Analysis {
	    portfolioID: string;
	    currency: string;
	    date: string;
	    total: string;
	    complete: boolean;
	    negative: boolean;
	    assets: AnalysisAsset[];
	    issues: string[];

	    static createFrom(source: any = {}) {
	        return new Analysis(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.portfolioID = source["portfolioID"];
	        this.currency = source["currency"];
	        this.date = source["date"];
	        this.total = source["total"];
	        this.complete = source["complete"];
	        this.negative = source["negative"];
	        this.assets = this.convertValues(source["assets"], AnalysisAsset);
	        this.issues = source["issues"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class CapitalPoint {
	    date: string;
	    value: string;
	    complete: boolean;
	    issues: string[];

	    static createFrom(source: any = {}) {
	        return new CapitalPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.value = source["value"];
	        this.complete = source["complete"];
	        this.issues = source["issues"];
	    }
	}
	export class Group {
	    code: string;
	    name: string;
	    value: string;
	    percent: string;

	    static createFrom(source: any = {}) {
	        return new Group(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.name = source["name"];
	        this.value = source["value"];
	        this.percent = source["percent"];
	    }
	}
	export class Summary {
	    value: string;
	    complete: boolean;
	    negative: boolean;
	    groups: Group[];
	    issues: string[];

	    static createFrom(source: any = {}) {
	        return new Summary(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.value = source["value"];
	        this.complete = source["complete"];
	        this.negative = source["negative"];
	        this.groups = this.convertValues(source["groups"], Group);
	        this.issues = source["issues"];
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class Card {
	    id: string;
	    name: string;
	    currency: string;
	    rateDate: string;
	    rate: string;
	    summary: Summary;

	    static createFrom(source: any = {}) {
	        return new Card(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.currency = source["currency"];
	        this.rateDate = source["rateDate"];
	        this.rate = source["rate"];
	        this.summary = this.convertValues(source["summary"], Summary);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class Point {
	    deposits: string;
	    depositsComplete: boolean;
	    depositIssues: string[];
	    date: string;
	    value: string;
	    complete: boolean;
	    issues: string[];

	    static createFrom(source: any = {}) {
	        return new Point(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deposits = source["deposits"];
	        this.depositsComplete = source["depositsComplete"];
	        this.depositIssues = source["depositIssues"];
	        this.date = source["date"];
	        this.value = source["value"];
	        this.complete = source["complete"];
	        this.issues = source["issues"];
	    }
	}
	export class Overview {
	    date: string;
	    total: Summary;
	    portfolios: Card[];
	    points: Point[];

	    static createFrom(source: any = {}) {
	        return new Overview(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.total = this.convertValues(source["total"], Summary);
	        this.portfolios = this.convertValues(source["portfolios"], Card);
	        this.points = this.convertValues(source["points"], Point);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

	export class ReturnPoint {
	    date: string;
	    annual: string;
	    cumulative: string;
	    issue: string;

	    static createFrom(source: any = {}) {
	        return new ReturnPoint(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.annual = source["annual"];
	        this.cumulative = source["cumulative"];
	        this.issue = source["issue"];
	    }
	}
	export class ReturnSeries {
	    id: string;
	    name: string;
	    startDate: string;
	    points: ReturnPoint[];

	    static createFrom(source: any = {}) {
	        return new ReturnSeries(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.startDate = source["startDate"];
	        this.points = this.convertValues(source["points"], ReturnPoint);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class ReturnComparison {
	    baseDate: string;
	    dates: string[];
	    series: ReturnSeries[];

	    static createFrom(source: any = {}) {
	        return new ReturnComparison(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.baseDate = source["baseDate"];
	        this.dates = source["dates"];
	        this.series = this.convertValues(source["series"], ReturnSeries);
	    }

		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}




}
