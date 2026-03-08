package orm_auth

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/painterQ/poplar/logger"
	"gorm.io/gorm"
)

type User struct {
	ID               int       `gorm:"primary_key;auto_increment" json:"id"`
	Name             string    `gorm:"name;size:100;not null;unique_index" json:"name"`
	DingID           string    `gorm:"ding_id;not null;size:50;unique" json:"ding_id"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"-"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	Role             string    `gorm:"role;size:20" json:"role"`
	EMail            string    `gorm:"email;size:255;not null" json:"email"`
	MainDeviceHash   string    `gorm:"main_device_hash;size:100" json:"-"`
	AvatarSourcePath string    `gorm:"avatar_source_path;size:255" json:"-"`
	Favorite         string    `gorm:"type:text" json:"-"` // 存储JSON字符串
	FavoriteValue    Favorite  `gorm:"-" json:"favorite"`
}

func (u *User) TableName() string {
	return "users"
}

type Favorite struct {
	Colors []string `json:"main_color"` // 修正标签，应为"main_color"
}

var userInitList = []User{
	{
		ID:            1,
		DingID:        "painterqiao",
		Role:          "admin",
		Name:          "painter",
		EMail:         "painter_qiao@qq.com",
		FavoriteValue: Favorite{Colors: []string{"red", "green", "blue"}},
	},
	{
		ID:            2,
		DingID:        "zve2uth",
		Role:          "user",
		Name:          "xixi",
		EMail:         "1102329106@qq.com",
		FavoriteValue: Favorite{Colors: []string{"red", "green", "blue"}},
	},
}

func (u *User) Init(tx *gorm.DB, log *logger.Logger) (err error) {
	for _, user := range userInitList {
		if userExists(tx, user.Name) {
			return nil
		}
		u := &User{
			ID:     user.ID,
			Role:   user.Role,
			Name:   user.Name,
			EMail:  user.EMail,
			DingID: user.DingID,
		}
		log.Log("try register user %s", user.EMail)
		if err = ModelCreateUser(tx, u); err != nil {
			return fmt.Errorf("create user err: %w", err)
		}
	}
	log.Log("finish register new users")
	return nil
}

// 检查用户是否已存在
func userExists(db *gorm.DB, name string) bool {
	var count int64
	db.Model(&User{}).Where("name = ?", name).Count(&count)
	return count > 0
}

func ModelCreateUser(db *gorm.DB, user *User) error {
	// 将Favorite结构体转换为JSON字符串
	fav := Favorite{
		Colors: []string{"#FF0000", "#00FF00"},
	}

	favJSON, err := json.Marshal(fav)
	if err != nil {
		return err
	}

	// 设置Favorite字段为JSON字符串
	user.Favorite = string(favJSON)

	// 插入用户
	return db.Create(&user).Error
}

// ModelQueryUserByID queries a user by GroupName and returns the Favorite struct
func ModelQueryUserByID(db *gorm.DB, id int) (*User, error) {
	var user User

	// Query user from database
	if err := db.First(&user, "id = ?", id).Error; err != nil {
		return nil, err
	}

	// Convert Favorite string to struct
	if err := convertFavorite(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

func ModelQueryUserByUsername(db *gorm.DB, username string) (*User, error) {
	var user User

	// Query user from database
	if err := db.First(&user, "name = ?", username).Error; err != nil {
		return nil, err
	}

	// Convert Favorite string to struct
	if err := convertFavorite(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// Helper function to convert Favorite string to struct
func convertFavorite(user *User) error {
	if user.Favorite == "" {
		return nil
	}

	if err := json.Unmarshal([]byte(user.Favorite), &user.FavoriteValue); err != nil {
		return err
	}

	return nil
}
