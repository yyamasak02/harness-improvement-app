export namespace applier {
	
	export class Entry {
	    id: string;
	    path: string;
	    op: string;
	    time: string;
	    existed: boolean;
	    oldHash: string;
	    newHash: string;
	    rollbackOf?: string;
	    before: string;
	    after: string;
	
	    static createFrom(source: any = {}) {
	        return new Entry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.op = source["op"];
	        this.time = source["time"];
	        this.existed = source["existed"];
	        this.oldHash = source["oldHash"];
	        this.newHash = source["newHash"];
	        this.rollbackOf = source["rollbackOf"];
	        this.before = source["before"];
	        this.after = source["after"];
	    }
	}
	export class Manifest {
	    id: string;
	    path: string;
	    op: string;
	    time: string;
	    existed: boolean;
	    oldHash: string;
	    newHash: string;
	    rollbackOf?: string;
	
	    static createFrom(source: any = {}) {
	        return new Manifest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.path = source["path"];
	        this.op = source["op"];
	        this.time = source["time"];
	        this.existed = source["existed"];
	        this.oldHash = source["oldHash"];
	        this.newHash = source["newHash"];
	        this.rollbackOf = source["rollbackOf"];
	    }
	}

}

export namespace collector {
	
	export class Stats {
	    Scanned: number;
	    Ingested: number;
	    Skipped: number;
	    Removed: number;
	    BrokenLines: number;
	
	    static createFrom(source: any = {}) {
	        return new Stats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Scanned = source["Scanned"];
	        this.Ingested = source["Ingested"];
	        this.Skipped = source["Skipped"];
	        this.Removed = source["Removed"];
	        this.BrokenLines = source["BrokenLines"];
	    }
	}

}

export namespace config {
	
	export class MaskPattern {
	    name: string;
	    regex: string;
	
	    static createFrom(source: any = {}) {
	        return new MaskPattern(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.regex = source["regex"];
	    }
	}
	export class Config {
	    claudePath: string;
	    model: string;
	    excludedProjects: string[];
	    correctionPatterns: string[];
	    maskPatterns: MaskPattern[];
	    timeoutSeconds: number;
	    dailyReportEnabled: boolean;
	    dailyReportTime: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.claudePath = source["claudePath"];
	        this.model = source["model"];
	        this.excludedProjects = source["excludedProjects"];
	        this.correctionPatterns = source["correctionPatterns"];
	        this.maskPatterns = this.convertValues(source["maskPatterns"], MaskPattern);
	        this.timeoutSeconds = source["timeoutSeconds"];
	        this.dailyReportEnabled = source["dailyReportEnabled"];
	        this.dailyReportTime = source["dailyReportTime"];
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

export namespace extensions {

	export class Hook {
	    event: string;
	    matcher: string;
	    type: string;
	    command: string;

	    static createFrom(source: any = {}) {
	        return new Hook(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.event = source["event"];
	        this.matcher = source["matcher"];
	        this.type = source["type"];
	        this.command = source["command"];
	    }
	}
	export class Skill {
	    name: string;
	    description: string;
	    path: string;

	    static createFrom(source: any = {}) {
	        return new Skill(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.path = source["path"];
	    }
	}
	export class Plugin {
	    id: string;
	    version: string;
	    scope: string;
	    enabled: boolean;

	    static createFrom(source: any = {}) {
	        return new Plugin(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.version = source["version"];
	        this.scope = source["scope"];
	        this.enabled = source["enabled"];
	    }
	}
	export class MCPServer {
	    name: string;
	    scope: string;
	    project: string;
	    type: string;
	    command: string;
	    args: string[];
	    url: string;
	    envKeys: string[];

	    static createFrom(source: any = {}) {
	        return new MCPServer(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.scope = source["scope"];
	        this.project = source["project"];
	        this.type = source["type"];
	        this.command = source["command"];
	        this.args = source["args"];
	        this.url = source["url"];
	        this.envKeys = source["envKeys"];
	    }
	}
	export class Report {
	    mcp: MCPServer[];
	    plugins: Plugin[];
	    skills: Skill[];
	    hooks: Hook[];
	    warnings: string[];

	    static createFrom(source: any = {}) {
	        return new Report(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.mcp = (source["mcp"] ?? []).map((e: any) => new MCPServer(e));
	        this.plugins = (source["plugins"] ?? []).map((e: any) => new Plugin(e));
	        this.skills = (source["skills"] ?? []).map((e: any) => new Skill(e));
	        this.hooks = (source["hooks"] ?? []).map((e: any) => new Hook(e));
	        this.warnings = source["warnings"];
	    }
	}

}

export namespace service {
	
	export class ReportInfo {
	    date: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ReportInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.path = source["path"];
	    }
	}
	export class ClaudeStatus {
	    found: boolean;
	    path: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ClaudeStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.found = source["found"];
	        this.path = source["path"];
	        this.error = source["error"];
	    }
	}
	export class DailyPoint {
	    date: string;
	    sessions: number;
	    messages: number;
	
	    static createFrom(source: any = {}) {
	        return new DailyPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.date = source["date"];
	        this.sessions = source["sessions"];
	        this.messages = source["messages"];
	    }
	}
	export class ModelStat {
	    model: string;
	    messages: number;
	    inputTokens: number;
	    outputTokens: number;
	
	    static createFrom(source: any = {}) {
	        return new ModelStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.messages = source["messages"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	    }
	}
	export class ProjectStat {
	    project: string;
	    cwd: string;
	    sessions: number;
	    lastAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectStat(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.project = source["project"];
	        this.cwd = source["cwd"];
	        this.sessions = source["sessions"];
	        this.lastAt = source["lastAt"];
	    }
	}
	export class MemoryItem {
	    target: targets.Target;
	    project: string;
	    size: number;
	    modTime: string;
	    stale: boolean;
	    duplicateOf: string[];

	    static createFrom(source: any = {}) {
	        return new MemoryItem(source);
	    }

	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = this.convertValues(source["target"], targets.Target);
	        this.project = source["project"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.stale = source["stale"];
	        this.duplicateOf = source["duplicateOf"];
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
	export class Overview {
	    sessions: number;
	    messages: number;
	    inputTokens: number;
	    outputTokens: number;
	    corrections: number;
	    toolErrors: number;
	    memoryFiles: number;
	    daily: DailyPoint[];
	    models: ModelStat[];
	    projects: ProjectStat[];
	
	    static createFrom(source: any = {}) {
	        return new Overview(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.sessions = source["sessions"];
	        this.messages = source["messages"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.corrections = source["corrections"];
	        this.toolErrors = source["toolErrors"];
	        this.memoryFiles = source["memoryFiles"];
	        this.daily = this.convertValues(source["daily"], DailyPoint);
	        this.models = this.convertValues(source["models"], ModelStat);
	        this.projects = this.convertValues(source["projects"], ProjectStat);
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
	
	export class Proposal {
	    targetPath: string;
	    baseHash: string;
	    original: string;
	    output: string;
	    rationale: string;
	    digest: string;
	    jobId: string;
	
	    static createFrom(source: any = {}) {
	        return new Proposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetPath = source["targetPath"];
	        this.baseHash = source["baseHash"];
	        this.original = source["original"];
	        this.output = source["output"];
	        this.rationale = source["rationale"];
	        this.digest = source["digest"];
	        this.jobId = source["jobId"];
	    }
	}
	export class ProposalRequest {
	    targetPath: string;
	    project: string;
	    days: number;
	    instruction: string;
	
	    static createFrom(source: any = {}) {
	        return new ProposalRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.targetPath = source["targetPath"];
	        this.project = source["project"];
	        this.days = source["days"];
	        this.instruction = source["instruction"];
	    }
	}
	export class SessionDetail {
	    session: store.Session;
	    messages: store.Message[];
	
	    static createFrom(source: any = {}) {
	        return new SessionDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.session = this.convertValues(source["session"], store.Session);
	        this.messages = this.convertValues(source["messages"], store.Message);
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
	export class SkillApplyFile {
	    path: string;
	    content: string;
	    baseHash: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillApplyFile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.content = source["content"];
	        this.baseHash = source["baseHash"];
	    }
	}
	export class SkillFileProposal {
	    path: string;
	    absPath: string;
	    original: string;
	    content: string;
	    baseHash: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillFileProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.absPath = source["absPath"];
	        this.original = source["original"];
	        this.content = source["content"];
	        this.baseHash = source["baseHash"];
	    }
	}
	export class SkillProposal {
	    name: string;
	    dir: string;
	    files: SkillFileProposal[];
	    rationale: string;
	    jobId: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.dir = source["dir"];
	        this.files = this.convertValues(source["files"], SkillFileProposal);
	        this.rationale = source["rationale"];
	        this.jobId = source["jobId"];
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
	export class SkillRequest {
	    name: string;
	    memoryPaths: string[];
	    project: string;
	    days: number;
	    instruction: string;
	
	    static createFrom(source: any = {}) {
	        return new SkillRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.memoryPaths = source["memoryPaths"];
	        this.project = source["project"];
	        this.days = source["days"];
	        this.instruction = source["instruction"];
	    }
	}
	export class TargetContent {
	    target: targets.Target;
	    content: string;
	    hash: string;
	
	    static createFrom(source: any = {}) {
	        return new TargetContent(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.target = this.convertValues(source["target"], targets.Target);
	        this.content = source["content"];
	        this.hash = source["hash"];
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

export namespace store {
	
	export class Message {
	    id: number;
	    sessionId: string;
	    project: string;
	    ts: string;
	    role: string;
	    kind: string;
	    tool: string;
	    text: string;
	    isError: boolean;
	    isCorrection: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionId = source["sessionId"];
	        this.project = source["project"];
	        this.ts = source["ts"];
	        this.role = source["role"];
	        this.kind = source["kind"];
	        this.tool = source["tool"];
	        this.text = source["text"];
	        this.isError = source["isError"];
	        this.isCorrection = source["isCorrection"];
	    }
	}
	export class SearchHit {
	    messageId: number;
	    sessionId: string;
	    project: string;
	    ts: string;
	    role: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new SearchHit(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messageId = source["messageId"];
	        this.sessionId = source["sessionId"];
	        this.project = source["project"];
	        this.ts = source["ts"];
	        this.role = source["role"];
	        this.text = source["text"];
	    }
	}
	export class Session {
	    id: string;
	    project: string;
	    filePath: string;
	    cwd: string;
	    gitBranch: string;
	    startedAt: string;
	    endedAt: string;
	    userMsgs: number;
	    assistantMsgs: number;
	    toolErrors: number;
	    corrections: number;
	    inputTokens: number;
	    outputTokens: number;
	    cacheRead: number;
	    cacheCreate: number;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.project = source["project"];
	        this.filePath = source["filePath"];
	        this.cwd = source["cwd"];
	        this.gitBranch = source["gitBranch"];
	        this.startedAt = source["startedAt"];
	        this.endedAt = source["endedAt"];
	        this.userMsgs = source["userMsgs"];
	        this.assistantMsgs = source["assistantMsgs"];
	        this.toolErrors = source["toolErrors"];
	        this.corrections = source["corrections"];
	        this.inputTokens = source["inputTokens"];
	        this.outputTokens = source["outputTokens"];
	        this.cacheRead = source["cacheRead"];
	        this.cacheCreate = source["cacheCreate"];
	    }
	}

}

export namespace targets {
	
	export class Target {
	    kind: string;
	    path: string;
	    label: string;
	    exists: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Target(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.kind = source["kind"];
	        this.path = source["path"];
	        this.label = source["label"];
	        this.exists = source["exists"];
	    }
	}

}

