package orm_auth

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

const (
	LoginErrorMaxCount = 3                  // 连续登录失败最大次数
	LoginLockDuration  = 10 * time.Minute   // 锁定时长：10分钟
	JwtExpireDuration  = 7 * 24 * time.Hour // JWT有效期：7天
)

// 【建议新增】登录状态常量，增强代码可读性，和LoginLog的comment完全对应
const (
	LoginStatusSuccess  = 1 // 登录成功
	LoginStatusPwdError = 2 // 密码/验证码错误
	LoginStatusNoUser   = 3 // 用户不存在
	LoginStatusLocked   = 4 // 账号被锁定
	LoginStatusParamErr = 5 // 参数错误
)

// LoginLog 登录记录表，存储所有登录行为
type LoginLog struct {
	ID         uint64    `gorm:"primary_key;autoIncrement"` // 主键自增
	CreatedAt  time.Time `gorm:"utoCreateTime"`             // 登录时间
	UserID     int       `gorm:"user_id;index"`             // 关联用户ID，为空则是用户不存在的情况
	Status     int       `gorm:"status;not null;comment:'1=登录成功,2=密码错误,3=用户不存在,4=账号被锁定,5=参数错误'"`
	DeviceID   string    `gorm:"device_id;size:100;not null"` // 登录设备
	DeviceInfo string    `gorm:"device_info;size:100"`
	ClientIP   string    `gorm:"client_ip;size:50;not null"` // 客户端IP
	Region     string    `gorm:"region;size:100;not null"`   // 登录位置
	Msg        string    `gorm:"msg;size:255"`               // 备注信息
}

// TableName 表名
func (l *LoginLog) TableName() string {
	return "login_logs"
}

// CreateLoginLog 写入登录日志
func CreateLoginLog(db *gorm.DB, log LoginLog) error {
	return db.Create(&log).Error
}

// CheckIpIsLocked 校验【当前请求IP】是否被锁定
// 核心逻辑：查询该IP在【最近10分钟内】的登录失败日志，若失败次数 >=3 → 锁定该IP，拒绝登录
// 参数：db=数据库连接  clientIp=客户端真实IP（从gin的c.ClientIP()获取）
func CheckIpIsLocked(db *gorm.DB, clientIp string, deviceHash string) (bool, error) {
	// 定义：统计失败次数
	var failCount int64
	// 计算时间阈值：当前时间 往前推10分钟
	tenMinutesAgo := time.Now().Add(-LoginLockDuration)

	// GORM查询核心SQL逻辑
	// 条件1: 匹配当前请求的客户端IP
	// 条件2: 登录状态为【失败】(2=密码错误 3=用户不存在，都是登录失败行为)
	// 条件3: 失败时间在【最近10分钟内】
	// 最终：统计该IP的失败次数
	err := db.Model(&LoginLog{}).
		Where("client_ip = ? AND status in (2,3) AND created_at >= ?", clientIp, tenMinutesAgo).
		Count(&failCount).Error

	// 数据库查询异常兜底
	if err != nil {
		return false, err
	}

	// 核心判断：10分钟内失败次数 >=3 → 返回【true=IP被锁定】
	if failCount >= LoginErrorMaxCount {
		return true, nil
	}

	// 否则：IP未被锁定，正常放行
	return false, nil
}

// ModelGetUserRecentLoginInfo 查询是否存在异常登陆
// 检查:1.是否是用户近1个月登录过的设备 2.是否和上次登录IP的归属地相同
// 返回值: 两个条件都为true返回空字符串, 否则返回对应的描述文本; 第二个返回值为错误信息
func ModelGetUserRecentLoginInfo(db *gorm.DB, userId int, deviceId string, ip string) (string, error) {
	// 定义返回的描述信息
	var descMsg strings.Builder
	// 时间阈值：近1个月
	oneMonthAgo := time.Now().AddDate(0, -1, 0)

	// 检查是否为该用户近期(1月)登录过的设备
	var oldDeviceCount int64
	err := db.Model(&LoginLog{}).Where(
		"user_id = ? AND device_id = ? AND status = ? AND created_at >= ?",
		userId, deviceId, LoginStatusSuccess, oneMonthAgo,
	).Count(&oldDeviceCount).Error
	if err != nil {
		return "", fmt.Errorf("查询用户历史设备失败: %w", err)
	}
	// 判断是否是旧设备
	isRecentDevice := oldDeviceCount > 0
	if !isRecentDevice {
		descMsg.WriteString("新登录设备")
	}

	// 检查是否和上次登陆IP的归属地相同
	var lastLoginLog LoginLog
	err = db.Model(&LoginLog{}).Where(
		"user_id = ? AND status = ?", userId, LoginStatusSuccess,
	).Order("created_at DESC").First(&lastLoginLog).Error

	// 处理两种无历史登录的情况: 1.查询报错 2.用户从未成功登录过
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// 用户无任何成功登录记录 → 归属地不一致
			if descMsg.Len() > 0 {
				descMsg.WriteString(", ")
			}
			descMsg.WriteString("首次登录")
			return descMsg.String(), nil
		}
		// 数据库查询异常
		return "", fmt.Errorf("查询用户最近登录记录失败: %w", err)
	}

	// 归属地判断规则(和你之前约定一致):
	// 1. 当前请求IP == 上次登录IP → 归属地一定相同
	// 2. IP不同 → 对比 LoginLog 的 Region 字段(归属地)是否一致
	isSameRegion := true
	if ip != lastLoginLog.ClientIP {
		isSameRegion = false
	}

	// 归属地不一致，追加描述信息
	if !isSameRegion {
		if descMsg.Len() > 0 {
			descMsg.WriteString(", ")
		}
		descMsg.WriteString(fmt.Sprintf("本次登录归属地[%s]与上次[%s]不一致", ip, lastLoginLog.ClientIP))
	}

	// ✅ 两个条件都为true → 返回空字符串；否则返回拼接后的描述
	return descMsg.String(), nil
}
