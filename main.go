package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/gin-contrib/pprof"
	"msgPushSite/db/sqldb"
	"msgPushSite/service/sego"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"

	"msgPushSite/config"
	"msgPushSite/db"
	"msgPushSite/internal/glog"
	"msgPushSite/internal/glog/log"
	"msgPushSite/lib"
	"msgPushSite/router"

	"github.com/gin-gonic/gin"
)

var (
	confPath = flag.String("config", "./config/app.local.ini", "profilePath")
	httpSrv  *http.Server
)

func Init() error {
	flag.Parse()

	err := config.InitConfig(*confPath)
	if err != nil {
		return fmt.Errorf("init config is err: %v", err)
	}

	err = log.InitLog()
	if err != nil {
		return fmt.Errorf("init log is err: %v", err)
	}
	fmt.Println("[WsServer]init config success!!!")
	return nil
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			var buf [4096]byte
			n := runtime.Stack(buf[:], false)
			tmpStr := fmt.Sprintf("err=%v panic ==> %s\n", err, string(buf[:n]))
			glog.Emergency(tmpStr)
			fmt.Println(tmpStr)
		}
	}()
	// 1. 初始化配置文件
	err := Init()
	if err != nil {
		fmt.Println("main init error:", err)
		return
	}
	// 2. 初始化数据库
	defer db.Close()
	err = db.InitDB()
	if err != nil {
		fmt.Println("init db error:", err)
		return
	}
	//启动加载敏感词
	sego.Init(sqldb.Live())
	// 3. 初始化kafka, 连接管理等
	defer lib.Close()
	err = lib.InitLib()
	if err != nil {
		fmt.Println("init lib error:", err)
		return
	}

	// 设置gin模式,生产模式使用release
	if strings.ToLower(config.GetApp().Env) == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}

	//gin.SetMode(gin.ReleaseMode)
	// 4. 初始化http engine
	engine := gin.New()
	initPprof(engine)
	//corsConfig := cors.DefaultConfig()
	//corsConfig.AllowAllOrigins = true
	engine.Use(gin.Recovery())
	router.ApiRouter(engine)
	appConf := config.GetConfig().Application
	httpSrv = &http.Server{
		Addr:    fmt.Sprintf("%s:%s", appConf.Address, appConf.Port),
		Handler: engine,
	}

	// 5. 启动HTTP服务
	glog.Infof("START WS Address:%s:%s", appConf.Address, appConf.Port)
	go func() {
		if err = httpSrv.ListenAndServe(); err != nil && !errors.Is(http.ErrServerClosed, err) {
			glog.Errorf("START AT:%s ERROR=%v", appConf.Port, err)
		}
	}()

	// 6. 监听程序退出信号
	quit := make(chan os.Signal)
	signal.Notify(quit, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	handleSignal(quit)

}

func initPprof(engine *gin.Engine) {
	hn, _ := os.Hostname()
	//只在02结尾的服务器上开启pprof
	if hn != "" && strings.HasSuffix(hn, "02") {
		pprof.Register(engine)
		glog.Infof("%s开启pprof")
	}
}

func handleSignal(c chan os.Signal) {
	switch a := <-c; a {
	case syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGHUP, syscall.SIGKILL:
		fmt.Println("Shutdown quickly, bye...", a)
		glog.Info("Shutdown quickly, bye...", a)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := httpSrv.Shutdown(ctx); err != nil {
			glog.Error("HTTP Server Shutdown err:", err, a)
		}
	default:
		os.Exit(0)
	}
}
