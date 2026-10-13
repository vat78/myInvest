export namespace desktop {

	export class AssetDTO {
	    id: string;
	    ticker: string;

	    static createFrom(source: any = {}) {
	        return new AssetDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ticker = source["ticker"];
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
