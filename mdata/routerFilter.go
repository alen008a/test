package mdata

//不需要日志的接口过滤
var RouterFilterLogPath = map[string]bool{
	"/stream/api/v1/health": true,
}

var SimpleRouterLogPath = map[string]struct{}{
	"/stream/api/v1/history": {},
}
