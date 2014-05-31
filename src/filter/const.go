package filter

const (
	STATUS_DDOS       = 2 << 0 //DDOS模式
	STATUS_BLACKLIST  = 2 << 1 //存在于黑名单
	STATUS_WHITELIST  = 2 << 2 //存在于白名单
	STATUS_CACHE_HITS = 2 << 3 //缓存命中

)
