package painter_verify

import (
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/caddyauth"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("painter_verify", parseCaddyfile)
}

// parseCaddyfile 解析painter_verify指令的Caddyfile配置
// 语法：painter_verify <带通配符的域名> （例如：painter_verify .20210109.xyz）
func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	// 1. 初始化JWT配置结构体
	var j JWT

	// 6. 返回认证中间件，将JWT配置注入
	return caddyauth.Authentication{
		ProvidersRaw: caddy.ModuleMap{
			"painter_verify": caddyconfig.JSON(j, nil),
		},
	}, nil
}
