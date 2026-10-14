export namespace cloud {
	
	export class Config {
	    publicOrigin: string;
	    profileId: string;
	    transportBase: string;
	    relayCapability: string;
	    remoteDesktop: boolean;
	    version: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.publicOrigin = source["publicOrigin"];
	        this.profileId = source["profileId"];
	        this.transportBase = source["transportBase"];
	        this.relayCapability = source["relayCapability"];
	        this.remoteDesktop = source["remoteDesktop"];
	        this.version = source["version"];
	    }
	}

}

export namespace securestore {
	
	export class Session {
	    token: string;
	    refreshToken: string;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.token = source["token"];
	        this.refreshToken = source["refreshToken"];
	    }
	}

}

