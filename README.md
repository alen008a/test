# 注意事项
- config/config.json是备份文件，不要手动修改

# 动态配置使用

项目不用重启更改配置,所有配置都是动态配置 数据库初始化不是经常变动的配置，不要动态使用

# 技术说明

技术 | 说明 | 官网
----|----|----
gin | 轻量级MVC框架 | https://github.com/gin-gonic/gin
gorm | ORM框架  | https://github.com/jinzhu/gorm
redis | redis缓存 | https://github.com/go-redis/redis
grpc | grpc微服务 | https://grpc.io
log | 高性能日志 | https://github.com/uber-go/zap
elasticsearch | 分布式搜索引擎 | https://www.elastic.co/cn/products/elasticsearch
docker | 应用容器引擎 | https://www.docker.com

# 初始化项目

go mod init msgPushSite go mod tidy

## 分支命名规范示例

```
# [名字]_[JIRA号]_[dev/merge/...]_[branch-name]
# dev 开发 / merge 合并

name_dev_master

name_testing
```

## linux 编译

 ```
 ./build.sh build
 ``` 

## dev环境 docker 部署

```
./build.sh dockerDevDeploy
``` 

## docker 停止

```
./build.sh dockerStop
```

## 镜像清理

```
./build.sh dockerClean
```

## 开发启动

```
go build
msgPushSite.exe --config=./config/app.dev.ini
```