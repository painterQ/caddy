package sql_db

import (
	"database/sql"
	"fmt"
	"regexp"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/painter/sql_db/orm_auth"
	configMgr "github.com/painterQ/poplar/config"
	"gorm.io/gorm"
)

func init() {
	caddy.RegisterModule(MySQLApp{})
	httpcaddyfile.RegisterGlobalOption("sql_db", func(d *caddyfile.Dispenser, existingVal any) (any, error) {
		a := new(MySQLApp)
		err := a.UnmarshalCaddyfile(d)
		if err != nil {
			return nil, err
		}
		return httpcaddyfile.App{
			Name:  "sql_db",
			Value: caddyconfig.JSON(a, nil),
		}, nil
	})
}

// MySQLApp is a Caddy app module that manages a shared *gorm.DB instance.
type MySQLApp struct {
	Type   string `json:"type"`
	Addr   string `json:"addr"` //是/path/to/s.sock或者127.0.0.1:8080
	User   string `json:"user,omitempty"`
	Pwd    string `json:"pwd,omitempty"`
	DBName string `json:"dbname,omitempty"`
	// internal
	db     *gorm.DB
	sqlDB  *sql.DB
	closed bool
}

func (m *MySQLApp) GetDB() *gorm.DB {
	return m.db
}

// Compile-time check that MySQLApp implements necessary interfaces
var (
	_ caddy.Module          = (*MySQLApp)(nil)
	_ caddy.Provisioner     = (*MySQLApp)(nil)
	_ caddy.App             = (*MySQLApp)(nil)
	_ caddyfile.Unmarshaler = (*MySQLApp)(nil)
)

// CaddyModule returns module info.
func (MySQLApp) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "sql_db",
		New: func() caddy.Module { return new(MySQLApp) },
	}
}

// Default values
const (
	defaultMaxOpenConn     = 25
	defaultMaxIdleConn     = 5
	defaultConnMaxLifetime = 300 // seconds
)

var reIPPortOrLinuxPath = regexp.MustCompile(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?):(?:[1-9]\d{0,3}|[1-5]\d{4}|6[0-4]\d{3}|65[0-4]\d{2}|655[0-2]\d|6553[0-5])|/(?:[a-zA-Z0-9_.-@~+]+/)*(?:[a-zA-Z0-9_.-@~+]+/?)?$`)

// IsIPPortOrLinuxPath 校验字符串是否为合法IPv4+端口 或 Linux绝对文件路径
// 返回true：符合格式；false：不符合格式
func isIPPortOrLinuxPath(s string) bool {
	// 空字符串直接返回false
	if s == "" {
		return false
	}
	return reIPPortOrLinuxPath.MatchString(s)
}

// Provision sets up the GORM DB connection.
func (m *MySQLApp) Provision(ctx caddy.Context) error {
	if len(m.Addr) == 0 {
		return fmt.Errorf("DB: Addr must not be empty")
	}

	if !isIPPortOrLinuxPath(m.Addr) {
		return fmt.Errorf("DB: Addr invalid: '%s'", m.Addr)
	}

	if m.Type != "mysql" && m.Type != "postgres" && m.Type != "sqlite" {
		return fmt.Errorf("MySQLApp: type must be one of [mysql, postgres, sqlite]")
	}

	if m.Type != "sqlite" {
		if len(m.Pwd) == 0 {
			return fmt.Errorf("DB: Password must not be empty")
		}
		if len(m.User) == 0 {
			return fmt.Errorf("DB: User must not be empty")
		}
	}

	config := configMgr.DBConfig{
		Type: m.Type,
		Addr: m.Addr,
		User: m.User,
		Pwd:  m.Pwd,
	}

	db, err := config.GetDB(m.DBName, &orm_auth.User{}, &orm_auth.LoginLog{})
	if err != nil {
		return fmt.Errorf("GetDB err: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("mysql module: failed to get sql.DB from gorm: %w", err)
	}

	// configure pool
	sqlDB.SetMaxOpenConns(defaultMaxOpenConn)
	sqlDB.SetMaxIdleConns(defaultMaxIdleConn)
	sqlDB.SetConnMaxLifetime(time.Duration(defaultConnMaxLifetime) * time.Second)

	m.db = db
	m.sqlDB = sqlDB
	m.closed = false

	return nil
}

// Start is a noop; connection already established in Provision
func (m *MySQLApp) Start() error {
	return nil
}

// Stop closes the DB connection.
func (m *MySQLApp) Stop() error {
	if m.closed {
		return nil
	}
	if m.sqlDB != nil {
		_ = m.sqlDB.Close()
	}
	m.closed = true
	return nil
}

// UnmarshalCaddyfile allows configuring this module via the Caddyfile.
//
// Syntax:
//
//	mysql_db {
//	    type <type>
//	    addr <addr>
//	    user <user>
//	    pwd <pwd>
//	    dbname <dbname>
//	}
func (m *MySQLApp) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		// inside a block: iterate over block tokens
		for d.NextBlock(0) {
			switch d.Val() {
			case "type":
				if !d.NextArg() {
					return d.Err("type requires an argument")
				}
				m.Type = d.Val()
			case "addr":
				if !d.NextArg() {
					return d.Err("addr requires an argument")
				}
				m.Addr = d.Val()
			case "user":
				if !d.NextArg() {
					return d.Err("user requires an argument")
				}
				m.User = d.Val()
			case "pwd":
				if !d.NextArg() {
					return d.Err("pwd requires an argument")
				}
				m.Pwd = d.Val()
			case "dbname":
				if !d.NextArg() {
					return d.Err("dbname requires an argument")
				}
				m.DBName = d.Val()
			default:
				return d.Errf("unrecognized sql_db option: %s", d.Val())
			}
		}
	}
	return nil
}
