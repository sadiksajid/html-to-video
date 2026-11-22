export namespace app {
	
	export class BatchItem {
	    reelPath: string;
	    audioPath: string;
	    resolution: string;
	    fps: number;
	    quality: string;
	
	    static createFrom(source: any = {}) {
	        return new BatchItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.reelPath = source["reelPath"];
	        this.audioPath = source["audioPath"];
	        this.resolution = source["resolution"];
	        this.fps = source["fps"];
	        this.quality = source["quality"];
	    }
	}
	export class BatchPairResult {
	    pairs: BatchItem[];
	    unpairs: string[];
	
	    static createFrom(source: any = {}) {
	        return new BatchPairResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pairs = this.convertValues(source["pairs"], BatchItem);
	        this.unpairs = source["unpairs"];
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
	export class BrowserStatus {
	    edgeAvailable: boolean;
	    chromiumAvailable: boolean;
	    selectedEngine: string;
	
	    static createFrom(source: any = {}) {
	        return new BrowserStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.edgeAvailable = source["edgeAvailable"];
	        this.chromiumAvailable = source["chromiumAvailable"];
	        this.selectedEngine = source["selectedEngine"];
	    }
	}

}

export namespace iframe {
	
	export class CheckResult {
	    url: string;
	    working: boolean;
	    reason?: string;
	
	    static createFrom(source: any = {}) {
	        return new CheckResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.working = source["working"];
	        this.reason = source["reason"];
	    }
	}

}

export namespace jobs {
	
	export class JobOptions {
	    resolution: string;
	    fps: number;
	    quality: string;
	
	    static createFrom(source: any = {}) {
	        return new JobOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.resolution = source["resolution"];
	        this.fps = source["fps"];
	        this.quality = source["quality"];
	    }
	}
	export class Job {
	    id: string;
	    name: string;
	    status: string;
	    phase: string;
	    frame: number;
	    frames: number;
	    pct: number;
	    eta: number;
	    error?: string;
	    reelPath: string;
	    audioPath: string;
	    outPath: string;
	    options: JobOptions;
	    duration: number;
	    fileSize: number;
	    created: number;
	    completed: number;
	    logs: string[];
	    iframeReplacements?: Record<string, string>;
	
	    static createFrom(source: any = {}) {
	        return new Job(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	        this.phase = source["phase"];
	        this.frame = source["frame"];
	        this.frames = source["frames"];
	        this.pct = source["pct"];
	        this.eta = source["eta"];
	        this.error = source["error"];
	        this.reelPath = source["reelPath"];
	        this.audioPath = source["audioPath"];
	        this.outPath = source["outPath"];
	        this.options = this.convertValues(source["options"], JobOptions);
	        this.duration = source["duration"];
	        this.fileSize = source["fileSize"];
	        this.created = source["created"];
	        this.completed = source["completed"];
	        this.logs = source["logs"];
	        this.iframeReplacements = source["iframeReplacements"];
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

export namespace settings {
	
	export class Settings {
	    defaultOutputDir: string;
	    defaultResolution: string;
	    defaultFps: number;
	    defaultQuality: string;
	    maxParallelRenders: number;
	    keepIntermediateFiles: boolean;
	    browserChoice: string;
	    language: string;
	    filenameTemplate: string;
	    openWithEnabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.defaultOutputDir = source["defaultOutputDir"];
	        this.defaultResolution = source["defaultResolution"];
	        this.defaultFps = source["defaultFps"];
	        this.defaultQuality = source["defaultQuality"];
	        this.maxParallelRenders = source["maxParallelRenders"];
	        this.keepIntermediateFiles = source["keepIntermediateFiles"];
	        this.browserChoice = source["browserChoice"];
	        this.language = source["language"];
	        this.filenameTemplate = source["filenameTemplate"];
	        this.openWithEnabled = source["openWithEnabled"];
	    }
	}

}

