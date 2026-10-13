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
