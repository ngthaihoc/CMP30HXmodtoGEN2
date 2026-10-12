export namespace main {
	
	export class EnvCheckItem {
	    name: string;
	    ok: boolean;
	    note: string;
	
	    static createFrom(source: any = {}) {
	        return new EnvCheckItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.ok = source["ok"];
	        this.note = source["note"];
	    }
	}
	export class EnvironmentReport {
	    gpuDetected: boolean;
	    gpuName: string;
	    pciBusId: string;
	    detectedModel: string;
	    deviceId: number;
	    is30HX: boolean;
	    is40HX: boolean;
	    gspActive: boolean;
	    isGen2: boolean;
	    driverStrategy: number;
	    autoHard: boolean;
	    retryCount: number;
	    retryInterval: number;
	    items: EnvCheckItem[];
	    recommendations: Record<string, boolean>;
	
	    static createFrom(source: any = {}) {
	        return new EnvironmentReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.gpuDetected = source["gpuDetected"];
	        this.gpuName = source["gpuName"];
	        this.pciBusId = source["pciBusId"];
	        this.detectedModel = source["detectedModel"];
	        this.deviceId = source["deviceId"];
	        this.is30HX = source["is30HX"];
	        this.is40HX = source["is40HX"];
	        this.gspActive = source["gspActive"];
	        this.isGen2 = source["isGen2"];
	        this.driverStrategy = source["driverStrategy"];
	        this.autoHard = source["autoHard"];
	        this.retryCount = source["retryCount"];
	        this.retryInterval = source["retryInterval"];
	        this.items = this.convertValues(source["items"], EnvCheckItem);
	        this.recommendations = source["recommendations"];
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
	export class HVCIReport {
	    enabled: boolean;
	    statusText: string;
	    guideText: string;
	    canAutoTurn: boolean;
	
	    static createFrom(source: any = {}) {
	        return new HVCIReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.statusText = source["statusText"];
	        this.guideText = source["guideText"];
	        this.canAutoTurn = source["canAutoTurn"];
	    }
	}
	export class RiotStatusReport {
	    model: string;
	    isWin11: boolean;
	    secureBootOn: boolean;
	    canPlayLeagueOfLegends: boolean;
	    canPlayValorant: boolean;
	    needsEFISigning: boolean;
	    hasTensorCore: boolean;
	    osDesc: string;
	    secureBootDesc: string;
	    gpuDesc: string;
	    lolDesc: string;
	    valorantDesc: string;
	    recommendation: string;
	    displayHostDesc: string;
	
	    static createFrom(source: any = {}) {
	        return new RiotStatusReport(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.isWin11 = source["isWin11"];
	        this.secureBootOn = source["secureBootOn"];
	        this.canPlayLeagueOfLegends = source["canPlayLeagueOfLegends"];
	        this.canPlayValorant = source["canPlayValorant"];
	        this.needsEFISigning = source["needsEFISigning"];
	        this.hasTensorCore = source["hasTensorCore"];
	        this.osDesc = source["osDesc"];
	        this.secureBootDesc = source["secureBootDesc"];
	        this.gpuDesc = source["gpuDesc"];
	        this.lolDesc = source["lolDesc"];
	        this.valorantDesc = source["valorantDesc"];
	        this.recommendation = source["recommendation"];
	        this.displayHostDesc = source["displayHostDesc"];
	    }
	}
	export class SubtoolResult {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new SubtoolResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class UninstallResult {
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UninstallResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
	export class UnlockResult {
	    success: boolean;
	    speed: number;
	    width: number;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UnlockResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.speed = source["speed"];
	        this.width = source["width"];
	        this.message = source["message"];
	    }
	}

}

