package painter_sign

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

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
	httpcaddyfile.RegisterHandlerDirective("painter_sign", parseCaddyFileLoginURL)
}

// 确保JWT结构体实现Caddy所需的核心接口
var (
	_ caddy.Module      = (*JWT)(nil) // 基础模块接口
	_ caddy.Provisioner = (*JWT)(nil) // 初始化接口
	_ caddy.Validator   = (*JWT)(nil) // 配置校验接口
	// 注册 HTTP 处理模块（命名为 "http.handlers.painter_sign"，Caddyfile 中会用到）
	_ caddyhttp.MiddlewareHandler = (*JWT)(nil) // HTTP处理器接口
)

// login_url /auth
func parseCaddyFileLoginURL(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	var j JWT
	// 2. 解析指令后的参数（必须有且仅有一个域名参数）
	// h.Next() 确认当前指令是painter_sign（已由RegisterHandlerDirective保证）
	if !h.Next() {
		return nil, h.ArgErr() // 无指令，返回参数错误
	}

	// 3. 读取第一个（也是唯一的）参数：带通配符的域名
	if h.NextArg() {
		domain := h.Val()
		// 4. 校验域名格式合法性（通配符规则）
		if err := validateWildcardDomain(domain); err != nil {
			return nil, fmt.Errorf("invalid wildcard domain '%s': %v", domain, err)
		}
		j.AllowDomain = normalizeDomain(domain) // 标准化域名格式（统一为.开头）
	} else {
		// 无参数，返回错误
		return nil, h.ArgErr()
	}

	// 5. 检查是否有多余参数（只允许一个参数）
	if h.NextArg() {
		return nil, h.Errf("too many arguments: only one wildcard domain is allowed (got %s)", h.Val())
	}

	return &j, nil
}

// 合法域名字符的正则：字母、数字、-、.，且不能以-开头/结尾，不能有连续的.
var validDomainRegex = regexp.MustCompile(`^([a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?\.)+[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// validateWildcardDomain 校验域名格式是否合法（支持通配符域名/普通域名）
// 合法格式：
// - 普通域名：baidu.com、20210109.xyz（至少1个点，字符合法）
// - 通配符域名：*.baidu.com、.20210109.xyz（通配符仅允许在开头，且核心域名合法）
func validateWildcardDomain(domain string) error {
	// 1. 空域名不合法
	if domain == "" {
		return fmt.Errorf("domain cannot be empty")
	}

	var coreDomain string
	var isWildcard bool

	// 2. 处理通配符前缀，区分通配符域名和普通域名
	switch {
	// 情况1：以 *. 开头（如 *.baidu.com）
	case strings.HasPrefix(domain, "*"):
		isWildcard = true
		// 通配符后必须跟 .（如 *.baidu.com 合法，*baidu.com 非法）
		if len(domain) < 3 || domain[1] != '.' {
			return fmt.Errorf("wildcard domain must be in format '*.domain.com' (e.g., *.baidu.com)")
		}
		// 提取通配符后的核心域名（如 *.baidu.com → baidu.com）
		coreDomain = strings.TrimPrefix(domain, "*")
		// 去掉核心域名开头的 .（避免出现 .baidu.com → 空）
		coreDomain = strings.TrimPrefix(coreDomain, ".")

	// 情况2：以 . 开头（如 .baidu.com）
	case strings.HasPrefix(domain, "."):
		isWildcard = true
		// 提取核心域名（如 .baidu.com → baidu.com）
		coreDomain = strings.TrimPrefix(domain, ".")

	// 情况3：普通域名（无通配符，如 baidu.com）
	default:
		coreDomain = domain
	}

	// 3. 核心域名不能为空（如 *./. 这类非法格式）
	if coreDomain == "" {
		return fmt.Errorf("core domain cannot be empty (e.g., 'baidu.com' for wildcard '.baidu.com')")
	}

	// 4. 校验核心域名的合法性：
	//    - 至少包含1个点（如 baidu.com 合法，baidu 非法）
	//    - 符合域名字符规则（字母、数字、-、.，无连续.，无首尾-）
	if strings.Count(coreDomain, ".") < 1 {
		if isWildcard {
			return fmt.Errorf("wildcard domain's core part must contain at least one dot (e.g., '.baidu.com' instead of '.com')")
		}
		return fmt.Errorf("normal domain must contain at least one dot (e.g., 'baidu.com' instead of 'baidu')")
	}

	// 5. 用正则校验核心域名的字符和格式
	if !validDomainRegex.MatchString(coreDomain) {
		return errors.New(
			"invalid domain format: " +
				"only letters, numbers, '-' and '.' are allowed; " +
				"no consecutive dots; no leading/trailing '-'",
		)
	}

	return nil
}

// normalizeDomain 将域名标准化为 .domain.com 格式（统一通配符写法）
func normalizeDomain(domain string) string {
	// 将 *.domain.com 转换为 .domain.com
	domain = strings.TrimPrefix(domain, "*")
	// 确保以 . 开头
	if !strings.HasPrefix(domain, ".") {
		domain = "." + domain
	}
	return domain
}

type JWT struct {
	db          *gorm.DB
	us          *UserService
	ginEngine   *gin.Engine
	h           http.Handler
	log         *zap.Logger
	AllowDomain string `json:"allow_domain"` //必须大写，caddy的要求
}

func (j *JWT) ServeHTTP(writer http.ResponseWriter, request *http.Request, n caddyhttp.Handler) error {
	j.h.ServeHTTP(writer, request)
	return nil
}

func (j *JWT) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		// 模块ID：遵循http.handlers.xxx命名规范
		ID: "http.handlers.painter_sign",
		// 模块实例构造函数
		New: func() caddy.Module {
			caddy.Log().Named("painter_sign").Info("painter_sign模块New方法调用（实例化）")
			return new(JWT)
		},
	}
}
