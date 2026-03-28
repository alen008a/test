package sqldb

import (
	"msgPushSite/utils"
	"time"

	"msgPushSite/config"
	"msgPushSite/internal/glog"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
)

var sqlDB *SqlDB

type SqlDB struct {
	//Site          *gorm.DB
	Live          *gorm.DB
	LiveSlave     *gorm.DB
	closeFunction []func() error
}

//func Site() *gorm.DB {
//	return sqlDB.Site
//}

func Live() *gorm.DB {
	return sqlDB.Live
}
func LiveSlave() *gorm.DB {
	return sqlDB.Live
}

// InitTiDB init mysql
func InitTiDB() (err error) {
	sqlDB = new(SqlDB)

	//sqlDB.Site, err = initSqlDB(config.GetTiDBConfig())
	//if err != nil {
	//	glog.Errorf("init Site db is err: %v", err)
	//	return err
	//}
	sqlDB.Live, err = initSqlDB(config.GetLive())
	if err != nil {
		glog.Errorf("init Live db is err: %v", err)
		return err
	}
	sqlDB.LiveSlave, err = initSqlDB(config.GetLiveSlave())
	if err != nil {
		glog.Errorf("init LiveSlave db is err: %v", err)
		return err
	}
	return nil
}

// 初始化数据库
func initSqlDB(c *config.Mysql) (*gorm.DB, error) {
	var logLevel logger.LogLevel
	if c.LogEnable {
		logLevel = logger.Info
	} else {
		logLevel = logger.Silent
	}
	address := utils.GetRealString(config.GetApplication().DBSecretKey, c.Address)
	db, err := gorm.Open(
		mysql.New(
			mysql.Config{
				DSN: address,
			},
		), &gorm.Config{
			SkipDefaultTransaction: true,
			Logger: glog.NewDBLog(
				logger.Config{
					SlowThreshold:             time.Millisecond * 400,
					Colorful:                  true,
					IgnoreRecordNotFoundError: true,
					LogLevel:                  logLevel,
				},
			),
			NamingStrategy: schema.NamingStrategy{
				SingularTable: true,
			},
		},
	)
	if err != nil {
		glog.Emergency("gorm Open error|addr=%s |err=%v", c.Address, err)
		return nil, err
	}

	dbp, err := db.DB()
	if err != nil {
		glog.Emergency("db.DB() error|addr=%s |err=%v", c.Address, err)
		return nil, err
	}

	dbp.SetMaxOpenConns(c.MaxConnect)
	dbp.SetMaxIdleConns(c.IdleConnect)
	dbp.SetConnMaxIdleTime(time.Duration(c.MaxLifeTime) * time.Second)
	dbp.SetConnMaxLifetime(time.Duration(c.MaxLifeTime) * time.Second)

	if err = dbp.Ping(); err != nil {
		glog.Emergency("db.Ping() error|addr=%s |err=%v", c.Address, err)
		return nil, err
	}

	sqlDB.closeFunction = append(
		sqlDB.closeFunction, func() error {
			if dbp != nil {
				return dbp.Close()
			}
			return nil
		},
	)

	return db, nil
}

func Close() error {
	for _, v := range sqlDB.closeFunction {
		err := v()
		if err != nil {
			return err
		}
	}
	return nil
}
