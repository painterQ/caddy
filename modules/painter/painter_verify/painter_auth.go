package painter_verify

import (
	"crypto/rand"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp/caddyauth"
	_ "github.com/caddyserver/caddy/v2/modules/caddyhttp/caddyauth"
	"github.com/caddyserver/caddy/v2/modules/painter/sql_db/orm_auth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/painterQ/poplar/auth"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(&JWT{})
}

const CookieName = "peacock"

// 确保JWT结构体实现Caddy所需的核心接口
var (
	_ caddy.Module            = (*JWT)(nil) // 基础模块接口
	_ caddy.Provisioner       = (*JWT)(nil) // 初始化接口
	_ caddy.Validator         = (*JWT)(nil) // 配置校验接口
	_ caddyauth.Authenticator = (*JWT)(nil) // 认证器接口
)

var jwtSecret []byte //algo: HS512, HMac with SHA-512
var initJwtOnce sync.Once

func getJWTSecret() []byte {
	initJwtOnce.Do(func() {
		jwtSecret = make([]byte, 64)
		_, _ = rand.Read(jwtSecret)
	})
	return jwtSecret
}

type JWT struct {
	//AllowDomain 规定jwt token放置的token所属的域名
	logger *zap.Logger
}

type UserInJWT struct {
	jwt.RegisteredClaims
	auth.User `json:",inline"` //UserID和RegisteredClaims中的ID相同
}

func (j *JWT) Provision(ctx caddy.Context) error {
	j.logger = ctx.Logger(j)
	return nil
}

func (j *JWT) Validate() error {
	return nil
}

/*******************************
            Running
********************************/

func (j *JWT) Cleanup() error {
	return nil
}

func (j *JWT) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		// 模块ID：遵循http.authentication.providers.xxx命名规范
		ID: "http.authentication.providers.painter_verify",
		// 模块实例构造函数
		New: func() caddy.Module { return new(JWT) },
	}
}

// GenerateJwt 生成JWT令牌 登录成功后写入Cookie+更新用户表
func GenerateJwt(user *orm_auth.User) (string, error) {
	const JwtExpireDuration = 7 * 24 * time.Hour // JWT有效期：7天
	// 载荷信息
	now := time.Now()
	claims := UserInJWT{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "caddy",
			Subject:   user.Name,
			Audience:  nil,
			ExpiresAt: jwt.NewNumericDate(now.Add(JwtExpireDuration)),
			NotBefore: jwt.NewNumericDate(now.Add(-time.Second)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        strconv.Itoa(user.ID),
		},
		User: auth.User{
			UserID:   user.ID,
			Username: user.Name,
			Email:    user.EMail,
			Role:     auth.Role(user.Role),
			DingID:   user.DingID,
		},
	}
	// 生成token
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	t, err := token.SignedString(getJWTSecret())
	if err != nil {
		return "", err
	}
	return t, nil
}

func ParseAndCheckToken(r *http.Request, log *zap.Logger) (user caddyauth.User, user2 *auth.User, err error) {
	// 步骤1：从Cookie中获取JWT，并验证Cookie为HttpOnly
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		log.Warn("Authentication failed", zap.String("remote", r.RemoteAddr), zap.String("reason", "未找到指定Cookie"))
		return caddyauth.User{}, nil, fmt.Errorf("未找到token对应的Cookie")
	}

	tokenStr := cookie.Value
	if tokenStr == "" {
		log.Warn("Authentication failed", zap.String("remote", r.RemoteAddr), zap.String("reason", "tokenStr为空"))
		return caddyauth.User{}, nil, fmt.Errorf("tokenStr为空")
	}

	// 步骤2：解析并验证JWT（格式、签名、有效期）
	token, err := jwt.ParseWithClaims(tokenStr, &UserInJWT{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法（防止算法混淆攻击）
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("不支持的签名算法: %v", token.Header["alg"])
		}
		// 返回签名验证密钥
		return getJWTSecret(), nil
	},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS512.Name}), // 限定支持的HMAC算法
		jwt.WithExpirationRequired(),                                // 要求token包含exp有效期字段
		jwt.WithLeeway(10*time.Second),                              // 允许10秒时间误差（解决服务器时间偏差）
		jwt.WithIssuer("caddy"),                                     //验证Issuer与生成时一致
	)

	// 处理JWT验证错误
	if err != nil {
		handleOfficialValidationErr(err, r.RemoteAddr, log)
		return caddyauth.User{}, nil, err
	}

	// 验证token有效性，并提取用户信息
	// 从JWT Claims中提取用户信息（按需调整字段）
	claims, ok := token.Claims.(*UserInJWT)
	if !ok || !token.Valid {
		return caddyauth.User{}, nil, fmt.Errorf("JWT Claims type error: %T", token.Claims)
	}

	user = caddyauth.User{
		ID: claims.ID, // 用户唯一标识（按需替换）
		Metadata: map[string]string{ // 附加元数据
			"user_id":  strconv.Itoa(claims.UserID),
			"username": claims.Username,
			"email":    claims.Email,
			"roles":    string(claims.Role),
			"ding_id":  claims.DingID,
		},
	}
	authUser := claims.User
	return user, &authUser, nil
}

// Authenticate 验证jwt:
// 1.jwt需要放置在cookie，并且需为httponly（但是不能检查这个，因为浏览器不会告诉后端）
// 2.jwt格式合法
// 3.jwt有效期未过
// 4.jwt签名合法
// 返回值说明：
// - caddyauth.User: 认证成功时的用户信息
// - bool: 认证是否成功（true=成功，false=失败）
// - error: 系统级错误（非认证失败类错误）
//
// caddyauth.User怎么使用？
//  1. 后续中间件或者handler，可以通过`user, authenticated := caddyauth.GetAuthenticatedUser(r.Context())`获取
//  2. 反向代理后的后端服务，需要通过header_up指令搭配caddy专门为身份认证预留的占位符
//     2.1  {auth.user} 对应整个user对象的 JSON 字符串
//     2.2. {auth.user.id} 对应caddyauth.User中的ID
//     2.3  {auth.user.metadata.*} 对应caddyauth.User中的MetaData
func (j *JWT) Authenticate(w http.ResponseWriter, r *http.Request) (_ caddyauth.User, ret bool, err error) {
	defer func() {
		if err != nil {
			j.logger.Error("Authentication error", zap.String("remote", r.RemoteAddr), zap.String("error", err.Error()))
		}
	}()

	user, _, err := ParseAndCheckToken(r, j.logger)
	if err != nil {
		j.logger.Error("Authentication error", zap.String("remote", r.RemoteAddr), zap.String("error", err.Error()))
		return user, false, nil
	}

	// 从请求上下文获取Caddy的占位符解析器Replacer
	// 关于占位符的操作，见文档https://caddyserver.com/docs/conventions#placeholders
	repl, ok := r.Context().Value(caddy.ReplacerCtxKey).(*caddy.Replacer)
	if !ok {
		return user, false, fmt.Errorf("failed to get replacer from request context")
	}

	for k, v := range user.Metadata {
		repl.Set(fmt.Sprintf("painter_verify.%v", k), v)
	}

	// 提取用户名（示例，可替换为user_id等业务字段）
	// 可选：通过UserService验证用户是否存在（业务层校验）
	// if !j.us.UserExists(username) {
	// 	return caddyauth.User{}, false, nil
	// }

	// 认证成功，构造Caddy用户对象

	return user, true, nil
}

// handleOfficialValidationErr 基于官方预定义错误，处理核心业务验证失败
// 仅处理6类核心错误，忽略密钥/aud/iss/iat等你无需关注的场景
func handleOfficialValidationErr(valErr error, remote string, log *zap.Logger) {
	switch {
	// 核心1：Token已过期（业务最常用，官方ErrTokenExpired）
	case errors.Is(valErr, jwt.ErrTokenExpired):
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "Token已过期"), zap.String("err", valErr.Error()))
	// 核心2：签名无效（篡改/签名错误，官方ErrTokenSignatureInvalid）
	case errors.Is(valErr, jwt.ErrTokenSignatureInvalid):
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "签名错误"), zap.String("err", valErr.Error()))
	// 核心3：缺少必填字段（如exp，开启WithExpirationRequired后触发，官方ErrTokenRequiredClaimMissing）
	case errors.Is(valErr, jwt.ErrTokenRequiredClaimMissing):
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "缺少必填字段"), zap.String("err", valErr.Error()))
	// 核心4：Token无法验证（无签名/签名字段为空，官方ErrTokenUnverifiable）
	case errors.Is(valErr, jwt.ErrTokenUnverifiable):
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "Token无法验证"), zap.String("err", valErr.Error()))
	// 核心5：Token尚未生效（nbf字段未到时间，官方ErrTokenNotValidYet）
	case errors.Is(valErr, jwt.ErrTokenNotValidYet):
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "Token尚未生效"), zap.String("err", valErr.Error()))
	// 核心6：Claims无效（如字段类型错误，官方ErrTokenInvalidClaims）
	case errors.Is(valErr, jwt.ErrTokenInvalidClaims):
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "Claims无效"), zap.String("err", valErr.Error()))
	// 其他小众验证错误（aud/iss/jti等），合并为通用提示，无需单独处理
	default:
		log.Warn("Authentication failed", zap.String("remote", remote), zap.String("reason", "其他错误"), zap.String("err", valErr.Error()))
	}
}
