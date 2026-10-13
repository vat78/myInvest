export namespace desktop {

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
	    value: string;
	    found: boolean;

	    static createFrom(source: any = {}) {
	        return new CurrencyRateDTO(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
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
