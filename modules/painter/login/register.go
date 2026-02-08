package login

import (
	"net/http"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/caddyauth"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func init() {
	caddy.RegisterModule(&JWT{})
	// 注册对login_url指令的解析器
	httpcaddyfile.RegisterHandlerDirective("painter", parseCaddyFileLoginURL)
}

// 确保JWT结构体实现Caddy所需的核心接口
var (
	_ caddy.Module      = (*JWT)(nil) // 基础模块接口
	_ caddy.Provisioner = (*JWT)(nil) // 初始化接口
	_ caddy.Validator   = (*JWT)(nil) // 配置校验接口
	// 注册 HTTP 处理模块（命名为 "http.handlers.painter"，Caddyfile 中会用到）
	_ caddyhttp.MiddlewareHandler = (*JWT)(nil) // HTTP处理器接口
)

// login_url /auth
func parseCaddyFileLoginURL(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	return &JWT{}, nil
}

type JWT struct {
	db        *gorm.DB
	us        *UserService
	ginEngine *gin.Engine
	h         http.Handler
	log       *zap.Logger
}

func (j *JWT) ServeHTTP(writer http.ResponseWriter, request *http.Request, n caddyhttp.Handler) error {
	j.h.ServeHTTP(writer, request)
	return nil
}

func (j *JWT) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		// 模块ID：遵循http.handlers.xxx命名规范
		ID: "http.handlers.painter",
		// 模块实例构造函数
		New: func() caddy.Module {
			caddy.Log().Named("painter").Info("Painter模块New方法调用（实例化）")
			return new(JWT)
		},
	}
}
