export namespace audio {
	
	export class AudioDevice {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new AudioDevice(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

export namespace services {
	
	export class AccessRequestInfo {
	    id: string;
	    pubKey: string;
	    displayName: string;
	    message: string;
	    isOnline: boolean;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new AccessRequestInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pubKey = source["pubKey"];
	        this.displayName = source["displayName"];
	        this.message = source["message"];
	        this.isOnline = source["isOnline"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class AppSettings {
	    theme: string;
	    notifySound: boolean;
	    notifyDesktop: boolean;
	    minimizeToTray: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AppSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.notifySound = source["notifySound"];
	        this.notifyDesktop = source["notifyDesktop"];
	        this.minimizeToTray = source["minimizeToTray"];
	    }
	}
	export class AppState {
	    phase: string;
	    loadingMsg: string;
	    progress: number;
	
	    static createFrom(source: any = {}) {
	        return new AppState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phase = source["phase"];
	        this.loadingMsg = source["loadingMsg"];
	        this.progress = source["progress"];
	    }
	}
	export class AuditEntry {
	    id: string;
	    action: string;
	    actorKey: string;
	    actorName: string;
	    details: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.action = source["action"];
	        this.actorKey = source["actorKey"];
	        this.actorName = source["actorName"];
	        this.details = source["details"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class AuditPage {
	    entries: AuditEntry[];
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new AuditPage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.entries = this.convertValues(source["entries"], AuditEntry);
	        this.hasMore = source["hasMore"];
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
	export class Ban {
	    id: string;
	    pubKey: string;
	    reason: string;
	    bannedByPubKey: string;
	    expiresAt: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Ban(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.pubKey = source["pubKey"];
	        this.reason = source["reason"];
	        this.bannedByPubKey = source["bannedByPubKey"];
	        this.expiresAt = source["expiresAt"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Category {
	    id: string;
	    name: string;
	    position: number;
	    type: string;
	
	    static createFrom(source: any = {}) {
	        return new Category(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.position = source["position"];
	        this.type = source["type"];
	    }
	}
	export class Channel {
	    id: string;
	    categoryId: string;
	    name: string;
	    type: string;
	    position: number;
	    opusBitrate: number;
	    roleIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new Channel(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.categoryId = source["categoryId"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.position = source["position"];
	        this.opusBitrate = source["opusBitrate"];
	        this.roleIds = source["roleIds"];
	    }
	}
	export class DMConversationInfo {
	    id: string;
	    isGroup: boolean;
	    label: string;
	    participants: string[];
	    createdAt: string;
	    lastActivity: string;
	
	    static createFrom(source: any = {}) {
	        return new DMConversationInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.isGroup = source["isGroup"];
	        this.label = source["label"];
	        this.participants = source["participants"];
	        this.createdAt = source["createdAt"];
	        this.lastActivity = source["lastActivity"];
	    }
	}
	export class DMMessageOut {
	    id: string;
	    convId: string;
	    senderKey: string;
	    content: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new DMMessageOut(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.convId = source["convId"];
	        this.senderKey = source["senderKey"];
	        this.content = source["content"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class DMMessagePage {
	    messages: DMMessageOut[];
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DMMessagePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], DMMessageOut);
	        this.hasMore = source["hasMore"];
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
	export class FileInfo {
	    id: string;
	    fileName: string;
	    mimeType: string;
	    size: number;
	    url: string;
	    thumbnail?: string;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.fileName = source["fileName"];
	        this.mimeType = source["mimeType"];
	        this.size = source["size"];
	        this.url = source["url"];
	        this.thumbnail = source["thumbnail"];
	    }
	}
	export class InviteInfo {
	    code: string;
	    createdBy: string;
	    usesLeft?: number;
	    expiresAt?: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new InviteInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.code = source["code"];
	        this.createdBy = source["createdBy"];
	        this.usesLeft = source["usesLeft"];
	        this.expiresAt = source["expiresAt"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class Message {
	    id: string;
	    channelId: string;
	    authorPubKey: string;
	    content: string;
	    fileIds: string[];
	    editedAt: string;
	    createdAt: string;
	
	    static createFrom(source: any = {}) {
	        return new Message(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.channelId = source["channelId"];
	        this.authorPubKey = source["authorPubKey"];
	        this.content = source["content"];
	        this.fileIds = source["fileIds"];
	        this.editedAt = source["editedAt"];
	        this.createdAt = source["createdAt"];
	    }
	}
	export class MessagePage {
	    messages: Message[];
	    hasMore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MessagePage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], Message);
	        this.hasMore = source["hasMore"];
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
	export class MessageSearchParams {
	    text: string;
	    channelId: string;
	    fromPubKey: string;
	    has: string;
	    before: string;
	    after: string;
	
	    static createFrom(source: any = {}) {
	        return new MessageSearchParams(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.text = source["text"];
	        this.channelId = source["channelId"];
	        this.fromPubKey = source["fromPubKey"];
	        this.has = source["has"];
	        this.before = source["before"];
	        this.after = source["after"];
	    }
	}
	export class PerServerSettings {
	    serverId: number;
	    showAvatars: boolean;
	    showBios: boolean;
	    showStatuses: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PerServerSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverId = source["serverId"];
	        this.showAvatars = source["showAvatars"];
	        this.showBios = source["showBios"];
	        this.showStatuses = source["showStatuses"];
	    }
	}
	export class Profile {
	    publicKey: string;
	    displayName: string;
	    avatarHash: string;
	    bio: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.publicKey = source["publicKey"];
	        this.displayName = source["displayName"];
	        this.avatarHash = source["avatarHash"];
	        this.bio = source["bio"];
	    }
	}
	export class Role {
	    id: string;
	    name: string;
	    color: string;
	    position: number;
	    isDefault: boolean;
	    permissions: number;
	
	    static createFrom(source: any = {}) {
	        return new Role(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.color = source["color"];
	        this.position = source["position"];
	        this.isDefault = source["isDefault"];
	        this.permissions = source["permissions"];
	    }
	}
	export class SearchResult {
	    messages: Message[];
	    totalCount: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messages = this.convertValues(source["messages"], Message);
	        this.totalCount = source["totalCount"];
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
	export class ServerEntry {
	    id: number;
	    address: string;
	    name: string;
	    iconHash: string;
	    isRelayOnly: boolean;
	    isOwner: boolean;
	    connected: boolean;
	    lastConnectedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.address = source["address"];
	        this.name = source["name"];
	        this.iconHash = source["iconHash"];
	        this.isRelayOnly = source["isRelayOnly"];
	        this.isOwner = source["isOwner"];
	        this.connected = source["connected"];
	        this.lastConnectedAt = source["lastConnectedAt"];
	    }
	}
	export class ServerHello {
	    serverPubKey: string;
	    serverName: string;
	    accessMode: string;
	    trustStatus: string;
	    storedPubKey: string;
	
	    static createFrom(source: any = {}) {
	        return new ServerHello(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.serverPubKey = source["serverPubKey"];
	        this.serverName = source["serverName"];
	        this.accessMode = source["accessMode"];
	        this.trustStatus = source["trustStatus"];
	        this.storedPubKey = source["storedPubKey"];
	    }
	}
	export class ServerInfo {
	    name: string;
	    description: string;
	    iconId: string;
	    iconHash: string;
	    accessMode: string;
	    memberCount: number;
	    maxFileSize: number;
	    totalStorageLimit: number;
	
	    static createFrom(source: any = {}) {
	        return new ServerInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.description = source["description"];
	        this.iconId = source["iconId"];
	        this.iconHash = source["iconHash"];
	        this.accessMode = source["accessMode"];
	        this.memberCount = source["memberCount"];
	        this.maxFileSize = source["maxFileSize"];
	        this.totalStorageLimit = source["totalStorageLimit"];
	    }
	}
	export class User {
	    pubKey: string;
	    displayName: string;
	    avatarHash: string;
	    bio: string;
	    status: string;
	    roleIds: string[];
	
	    static createFrom(source: any = {}) {
	        return new User(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pubKey = source["pubKey"];
	        this.displayName = source["displayName"];
	        this.avatarHash = source["avatarHash"];
	        this.bio = source["bio"];
	        this.status = source["status"];
	        this.roleIds = source["roleIds"];
	    }
	}
	export class VoiceParticipant {
	    pubKey: string;
	    displayName: string;
	    isMuted: boolean;
	    isDeafened: boolean;
	    isSpeaking: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VoiceParticipant(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.pubKey = source["pubKey"];
	        this.displayName = source["displayName"];
	        this.isMuted = source["isMuted"];
	        this.isDeafened = source["isDeafened"];
	        this.isSpeaking = source["isSpeaking"];
	    }
	}

}

