package painter_sign

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"math/big"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/caddyserver/caddy/v2/modules/painter/painter_sign/frontend"
	painterAuth "github.com/caddyserver/caddy/v2/modules/painter/painter_verify"
	"github.com/caddyserver/caddy/v2/modules/painter/sql_db/orm_auth"
	"github.com/gin-gonic/gin"
	"github.com/painterQ/poplar/ip2region"
	"github.com/painterQ/poplar/logger"
	"github.com/painterQ/poplar/utils/web_helper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ChallengeExpire 连续三次错误需要10分钟后再试
const ChallengeExpire = 3 * time.Minute // 验证码有效期3分钟

type UserService struct {
	db           *gorm.DB
	zapLogger    *zap.Logger
	log          *logger.Logger
	challengeMap sync.Map //deviceID -> challenge
	allowDomain  string
}

func NewUserService(db *gorm.DB, log *zap.Logger, allowDomain, dingToken string) *UserService {
	return &UserService{
		db:          db,
		zapLogger:   log,
		log:         logger.GetLogger("auth", logger.GetDingTalkOpt(dingToken)),
		allowDomain: allowDomain,
	}
}

// HandleUserInfo 这个接口需要登录
// 因为login插件不经过auth的检查，所以这里需要自己获取cookie并解析检查token的有效性
func (u *UserService) HandleUserInfo(ctx *gin.Context) {
	_, user, err := painterAuth.ParseAndCheckToken(ctx.Request, u.zapLogger)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, "painter unauthorized")
		return
	}
	//ctx := caddyauth.WithAuthenticatedUser(r.Context(), user)
	//r = r.WithContext(ctx)

	userResponse, err := orm_auth.ModelQueryUserByID(u.db, user.UserID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": err.Error()})
		return
	}

	web_helper.JSON(ctx, http.StatusOK, userResponse)
}

// Handle 注册路由
func (u *UserService) Handle(ctx *gin.Engine) {
	//ctx.Use(func(context *gin.Context) {
	//	fmt.Println("login", context.Request.RequestURI)
	//})
	g := ctx.Group("/auth")
	g.GET("/user", u.HandleUserInfo)        // 获取用户信息
	g.POST("/login", u.HandleLogin)         // 登录
	g.POST("/v-code", u.HandleGetChallenge) // 获取验证码

	//静态文件
	content, _ := fs.Sub(frontend.StaticFiles, "dist")

	ctx.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/auth") {

			pathRelative := c.Request.URL.Path[len("/auth"):]
			pathRelative = strings.TrimLeft(pathRelative, "/") //我也不知道为啥 /index.html不行，但是index.html行
			if len(pathRelative) == 0 {
				pathRelative = "index.html"
			}

			f, err := content.Open(pathRelative)
			if err != nil {
				c.Redirect(http.StatusPermanentRedirect, "/auth/index.html")
				return
			}
			c.Status(http.StatusOK)

			//关键：mime
			mimeType := mime.TypeByExtension(filepath.Ext(pathRelative))
			c.Header("Content-Type", mimeType)

			defer f.Close()
			if pathRelative == "index.html" {
				replace2(c.Writer, f)
			}
			_, _ = io.Copy(c.Writer, f)
		}
	})
}

// replace 高效流式将fs.File拷贝到gin.ResponseWriter，同时将/ui替换为/auth
// 流式处理、KMP匹配、处理跨缓冲区边界、纯官方库、无正则
func replace2(writer gin.ResponseWriter, f fs.File) {
	c, _ := io.ReadAll(f)
	ret := bytes.Replace(c, []byte("/ui"), []byte("/auth"), -1)
	writer.Write(ret)
}

type loginRequest struct {
	Username          string `json:"username" binding:"required"`
	ChallengeResponse string `json:"challengeResponse"`
	DeviceID          string `json:"device_id" binding:"required"`
	DeviceInfo        string `json:"device_info"`
}

// HandleLogin 登录
func (u *UserService) HandleLogin(ctx *gin.Context) {
	// 输入为body中的loginRequest，json
	var req loginRequest
	var err error
	var loginLog orm_auth.LoginLog
	defer func() {
		_ = orm_auth.CreateLoginLog(u.db, loginLog)
	}()

	// 获取客户端真实IP + 登录归属地
	loginLog.ClientIP = ctx.ClientIP()
	ipv4 := net.ParseIP(loginLog.ClientIP).To4()
	loginLog.Region, err = ip2region.QueryRegion(ipv4)
	if err != nil {
		loginLog.Status = 5 // 5=参数错误
		loginLog.Msg = "登录失败：ip解析错误"
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数格式错误：" + err.Error()})
		return
	}

	// 解析参数
	if err := ctx.ShouldBindJSON(&req); err != nil {
		loginLog.Status = 5 // 5=参数错误
		loginLog.Msg = "登录失败：body参数解析错误"
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "参数格式错误：" + err.Error()})
		return
	}

	loginLog.DeviceID = req.DeviceID

	// 检查deviceID是否是hex并且符合md5长度
	if !isValidMD5Hex(req.DeviceID) {
		loginLog.Status = 5 // 5=参数错误
		loginLog.Msg = "登录失败：设备ID格式非法，必须是32位MD5十六进制字符串"
		ctx.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "设备ID格式非法，必须是32位MD5十六进制字符串"})
		return
	}

	// 调用orm_auth的方法，校验当前IP是否被锁定
	isIpLocked, err := orm_auth.CheckIpIsLocked(u.db, loginLog.ClientIP, req.DeviceID)
	if err != nil {
		loginLog.Status = 4 // 4=账号被锁定(IP锁定)
		loginLog.Msg = "内部错误:CheckIpIsLocked函数报错:" + err.Error()
		u.log.Err(fmt.Errorf("校验IP锁定状态失败: %w", err))
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "系统内部错误"})
		return
	}
	if isIpLocked {
		loginLog.Status = 4 // 4=账号被锁定(IP锁定)
		loginLog.Msg = "当前IP登录失败次数超限，已被锁定10分钟"
		ctx.JSON(http.StatusForbidden, gin.H{"code": 423, "msg": loginLog.Msg})
		return
	}

	// 检查用户是否存在
	user, err := orm_auth.ModelQueryUserByUsername(u.db, req.Username)
	if err != nil || user == nil {
		loginLog.Status = 3 // 3=用户不存在
		loginLog.Msg = "登录失败：用户名不存在"
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户名或验证码错误"})
		return
	}

	// 用户存在，补全日志的用户ID
	loginLog.UserID = user.ID

	// 根据DeviceID查询challengeMap中是否有验证码，如果没有，报错
	challengeVal, ok := u.challengeMap.Load(req.DeviceID)
	if !ok {
		loginLog.Status = 2
		loginLog.Msg = "登录失败：验证码已过期或不存在，请重新获取"
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": loginLog.Msg})
		return
	}

	challenge, ok := challengeVal.(*challenge)
	if !ok {
		u.challengeMap.Delete(req.DeviceID)
		loginLog.Status = 2
		loginLog.Msg = "登录失败：验证码格式异常"
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": loginLog.Msg})
		return
	}

	// 校验验证码是否过期
	if time.Now().After(challenge.timeTo) {
		u.challengeMap.Delete(req.DeviceID)
		loginLog.Status = 2
		loginLog.Msg = "登录失败：验证码已过期，请重新获取"
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": loginLog.Msg})
		return
	}

	// 验证ChallengeResponse和验证码是否匹配
	if strings.TrimSpace(req.ChallengeResponse) != strings.TrimSpace(challenge.challengeValue) {
		loginLog.Status = 2 // 2=验证码错误
		loginLog.Msg = "登录失败：验证码输入错误"
		ctx.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "用户名或验证码错误"})
		return
	}

	// ====== 登录成功 所有后续逻辑 ======
	// 清理当前设备的验证码缓存
	u.challengeMap.Delete(req.DeviceID)

	// 签发JWT
	tokenStr, err := painterAuth.GenerateJwt(user)
	if err != nil {
		u.log.Err(fmt.Errorf("生成JWT令牌失败: %w", err))
		loginLog.Status = 2
		loginLog.Msg = "登录失败：生成登录凭证异常"
		ctx.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "系统内部错误"})
		return
	}

	// 写入登录成功日志
	loginLog.Status = 1 // 1=登录成功
	loginLog.Msg = "登录成功"

	//签发JWT，放到cookie，httpOnly
	ctx.SetCookie(
		painterAuth.CookieName,
		tokenStr,
		int(orm_auth.JwtExpireDuration.Seconds()),
		"/",
		u.allowDomain,
		true, // Secure：仅HTTPS传输
		true, // HttpOnly：强制开启，防止前端JS读取，防XSS攻击，必须为true
	)

	// 登录成功响应
	web_helper.JSON(ctx, http.StatusOK, user)
}

type challenge struct {
	timeTo         time.Time
	challengeValue string
}

type challengeResponse struct {
	Until string `json:"until"`
}

func getRandomStr() string {
	const charset = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, 6)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := range result {
		// 从加密安全的随机源生成索引
		idx, _ := rand.Int(rand.Reader, charsetLen)
		result[i] = charset[idx.Int64()]
	}

	return string(result)
}

// HandleGetChallenge 获取验证码
func (u *UserService) HandleGetChallenge(ctx *gin.Context) {
	challengeValue := getRandomStr()
	until := time.Now().Add(ChallengeExpire)
	response := challengeResponse{Until: until.Format(time.RFC3339)}

	// 解析body中的loginRequest json参数
	var req loginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, response)
		u.log.Err(fmt.Errorf("HandleGetChallenge err: %w", err))
		return
	}

	// 检查用户是否存在，不存在则直接返回200
	user, err := orm_auth.ModelQueryUserByUsername(u.db, req.Username)
	if err != nil || user == nil {
		ctx.JSON(http.StatusOK, response)
		return
	}

	ip := ctx.ClientIP()
	loginInfo, err := orm_auth.ModelGetUserRecentLoginInfo(u.db, user.ID, req.DeviceID, ip)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, nil)
		return
	}

	// 将验证码存入challengeMap
	challenge := &challenge{
		timeTo:         until,
		challengeValue: challengeValue,
	}
	u.challengeMap.Store(req.DeviceID, challenge)

	region := "未知"
	remoteIP := net.ParseIP(ctx.Request.RemoteAddr)
	if remoteIP != nil {
		region, err = ip2region.QueryRegion(remoteIP)
		if err != nil {
			u.log.Err(fmt.Errorf("HandleGetChallenge QueryRegion (%v) err: %w", ctx.Request.RemoteAddr, err))
			region = "未知"
		}
	}
	if len(loginInfo) > 0 {
		u.log.DingMarkdown("异常登录", fmt.Sprintf("* 异常原因: %v\n设备: %v\n位置: %v\n* 验证码: **%v**", loginInfo, req.DeviceInfo, region, challengeValue), []string{user.DingID})
	} else {
		u.log.DingMarkdown("登录", fmt.Sprintf("验证码: %v，设备: %v\n位置: %v", challengeValue, req.DeviceInfo, region), []string{user.DingID})
	}

	ctx.JSON(http.StatusOK, response)
}

// 【辅助函数】校验字符串是否为32位MD5十六进制格式
func isValidMD5Hex(s string) bool {
	if len(s) != 32 {
		return false
	}
	_, err := hex.DecodeString(s)
	return err == nil
}
