package login

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
	j.us = NewUserService(j.db, j.log, "")
	j.ginEngine = gin.Default()
	j.us.Handle(j.ginEngine)
	j.h = j.ginEngine.Handler()
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
