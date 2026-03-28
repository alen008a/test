package db

import (
	"fmt"
	"msgPushSite/db/redisdb/core"
	"msgPushSite/db/sqldb"
)

// InitDB 初始化 db
func InitDB() error {
	var err error

	// mysql 的初始化
	err = sqldb.InitTiDB()
	if err != nil {
		return err
	}

	// 初始化 redis
	err = core.InitRedis()
	if err != nil {
		return err
	}
	fmt.Println("[WsServer]init db success!!!")
	return nil
}

// Close 关闭数据库
func Close() error {
	err := core.Close()
	if err != nil {
		return err
	}
	err = sqldb.Close()
	if err != nil {
		return err
	}
	return nil
}
