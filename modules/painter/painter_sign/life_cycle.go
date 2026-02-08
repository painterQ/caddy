package painter_sign

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/painter/sql_db"
	"github.com/gin-gonic/gin"
)

func (j *JWT) Provision(ctx caddy.Context) error {
	sqlDB, err := ctx.App("sql_db")
	if err != nil {
		return fmt.Errorf("get sql_db app err: %w", err)
	}

	sqlDBApp, ok := sqlDB.(*sql_db.MySQLApp)
	if !ok {
		return fmt.Errorf("sql_db app is not a MySQLApp")
	}

	j.db = sqlDBApp.GetDB()

	j.log = ctx.Logger(j)
	const dingToken = "714ccb8d24df499d7f04a2c9f3181f5b9c8a5d239038a4662eec2cd8119c6c32"

	j.us = NewUserService(j.db, j.log, j.AllowDomain, dingToken)
	j.ginEngine = gin.Default()
	j.us.Handle(j.ginEngine)
	j.h = j.ginEngine.Handler()
	return nil
}

func (j *JWT) Validate() error {
	//这是正确的获取logger的方法
	if err := validateWildcardDomain(j.AllowDomain); err != nil {
		return fmt.Errorf("painter_sign: invalid allow domain: %w", err)
	}
	return nil
}

/*******************************
            Running
********************************/

func (j *JWT) Cleanup() error {
	return nil
}
