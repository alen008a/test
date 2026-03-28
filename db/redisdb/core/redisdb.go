package core

import (
	"context"
	"msgPushSite/config"
	"msgPushSite/utils"
	"strings"
	"time"

	"msgPushSite/mdata"

	"github.com/go-redis/redis/v8"
)

var redisDb *RedisDb

const RedisNil = redis.Nil

type RedisDb struct {
	pool redis.UniversalClient
}

func InitRedis() error {
	cfg := config.GetRWRedisConfig()
	pool, err := initRedis(strings.TrimSpace(cfg.Host), strings.TrimSpace(cfg.Auth), cfg.Master, cfg.PoolSize)
	if err != nil {
		return err
	}
	redisDb = &RedisDb{
		pool: pool,
	}
	return nil
}

func initRedis(host, auth, master string, poolSize int) (redis.UniversalClient, error) {
	auth = utils.GetRealString(config.GetConfig().DBSecretKey, auth)
	options := &redis.UniversalOptions{
		Addrs:              strings.Split(host, ","), // redis地址
		MaxRedirects:       0,                        // 放弃前最大重试次数,默认是不重试失败的命令,默认是3次
		ReadOnly:           false,                    // 在从库上打开只读命令
		RouteByLatency:     false,                    // 允许将只读命令路由到最近的主节点或从节点,自动启用只读
		RouteRandomly:      false,                    // 允许将只读命令路由到随机主节点或从节点。 它自动启用只读。
		Password:           auth,
		MaxRetries:         2,
		MinRetryBackoff:    8 * time.Millisecond,
		MaxRetryBackoff:    512 * time.Millisecond,
		DialTimeout:        5 * time.Second,
		ReadTimeout:        10 * time.Second,
		WriteTimeout:       20 * time.Second,
		PoolSize:           poolSize,
		MaxConnAge:         6 * time.Minute,
		PoolTimeout:        30 * time.Second,
		IdleTimeout:        5 * time.Minute,
		IdleCheckFrequency: 1 * time.Minute, //空闲连接检查频率
	}
	//哨兵模式
	if len(master) > 0 {
		options.SentinelPassword = auth
		options.MasterName = master
	}
	redisPool := redis.NewUniversalClient(options)
	_, err := redisPool.Ping(context.Background()).Result()
	if err != nil {
		return nil, err
	}

	return redisPool, nil
}

func getPool(slave bool) redis.UniversalClient {
	return redisDb.pool
}

func GetKey(slave bool, key string) (string, error) {
	value, err := getPool(slave).Get(context.Background(), key).Result()
	if err != nil && err != redis.Nil {
		return "", err
	}
	return value, nil
}

func GetKeyBytes(slave bool, key string) ([]byte, error) {
	return getPool(slave).Get(context.Background(), key).Bytes()
}

func GetKeyInt(slave bool, key string) (int, error) {
	return getPool(slave).Get(context.Background(), key).Int()
}

func GetKeyFloat(slave bool, key string) (float64, error) {
	return getPool(slave).Get(context.Background(), key).Float64()
}

// SetNotExpireKV 设置不过期的 key
func SetNotExpireKV(key, value string) error {
	err := getPool(false).Set(context.Background(), key, value, 0).Err()
	if err != nil {
		return err
	}
	return nil
}

// SetExpireKV 设置过期的 key
func SetExpireKV(key, value string, expire time.Duration) error {
	err := getPool(false).Set(context.Background(), key, value, expire).Err()
	if err != nil {
		return err
	}
	return nil
}

// SetExpireKey 设置 key 过期
func SetExpireKey(key string, expire time.Duration) error {
	err := getPool(false).Expire(context.Background(), key, expire).Err()
	if err != nil {
		return err
	}
	return nil
}

// SetNX 设置 key, value 以及过期时间
func SetNX(key string, value string, expire time.Duration) (bool, error) {
	flag, err := getPool(false).SetNX(context.Background(), key, value, expire).Result()
	if err != nil {
		return false, err
	}
	return flag, nil
}

// DelKey 删除 redis 的key
func DelKey(key ...string) error {
	return getPool(false).Del(context.Background(), key...).Err()
}

// ScanKeys 模糊匹配键值
func ScanKeys(match string, perCount int64) ([]string, error) {
	var (
		cursor = uint64(0)
		data   []string
	)
	for {
		keys, retCursor, err := getPool(false).Scan(context.Background(), cursor, match, perCount).Result()
		if err != nil {
			return data, err
		}
		if len(keys) == 0 {
			break
		}
		data = append(data, keys...)
		if retCursor == 0 {
			break
		}
		cursor = retCursor
	}
	return data, nil
}

// KeyExist 判断某一个key 是否存在
func KeyExist(slave bool, keys string) (bool, error) {

	count, err := getPool(slave).Exists(context.Background(), keys).Result()
	if err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	return false, nil
}

// HSet 设置 hash
func HSet(key, field string, value interface{}) error {
	return getPool(false).HSet(context.Background(), key, field, value).Err()
}

// HMSet 批量存储 hash
func HMSet(key string, fields map[string]interface{}) error {
	if len(fields) < 1 {
		return nil
	}

	err := getPool(false).HMSet(context.Background(), key, fields).Err()
	if err != nil {
		return err
	}

	return nil
}

// HGet 获取单个 hash
func HGet(slave bool, key, field string) (string, error) {
	return getPool(slave).HGet(context.Background(), key, field).Result()
}

func HKeys(slave bool, key string) ([]string, error) {
	return getPool(slave).HKeys(context.Background(), key).Result()
}

// HMGet 批量获取 hash
func HMGet(slave bool, key string, fields ...string) ([]interface{}, error) {
	res, err := getPool(slave).HMGet(context.Background(), key, fields...).Result()
	if err != nil {
		return nil, err
	}

	return res, nil
}

// HScan 获取 hash 键值树
func HScan(slave bool, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return getPool(slave).HScan(context.Background(), key, cursor, match, count).Result()
}

func SScan(slave bool, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	return getPool(slave).SScan(context.Background(), key, cursor, match, count).Result()
}

func HLen(slave bool, key string) (int, error) {
	res, err := getPool(slave).HLen(context.Background(), key).Result()
	if err != nil {
		return 0, err
	}

	return int(res), nil
}

// HDel 删除 hash key
func HDel(key string, fields ...string) error {
	err := getPool(false).HDel(context.Background(), key, fields...).Err()
	if err != nil {
		return err
	}

	return nil
}

// RPush 在名称为key的list尾添加一个值为value的元素
func RPush(key string, values ...interface{}) error {
	return getPool(false).RPush(context.Background(), key, values...).Err()
}

// LPush 在名称为key的list头添加一个值为value的 元素
func LPush(key string, values ...interface{}) error {
	return getPool(false).LPush(context.Background(), key, values...).Err()
}

// LTrim 保留在名称为key的list保留指定区间内的元素，不在指定区间之内的元素都将被删除。
func LTrim(key string, start, stop int64) error {
	return getPool(false).LTrim(context.Background(), key, start, stop).Err()
}

func Subscribe(channel string) *redis.PubSub {
	return getPool(false).Subscribe(context.Background(), channel)
}

// Publish 发布消息
func Publish(channel string, values interface{}) (int64, error) {
	return getPool(false).Publish(context.Background(), channel, values).Result()
}

// LLen 返回名称为key的list的长度
func LLen(key string, slave bool) (int64, error) {
	return getPool(slave).LLen(context.Background(), key).Result()
}

// LRange 返回名称为key的list中start至end之间的元素, start为0, end为-1 则是获取所有 list key
func LRange(slave bool, key string, start, end int64) ([]string, error) {
	return getPool(slave).LRange(context.Background(), key, start, end).Result()
}

// LSet 给名称为key的list中index位置的元素赋值
func LSet(key string, index int64, value interface{}) error {
	return getPool(false).LSet(context.Background(), key, index, value).Err()
}

// LRem 删除count个key的list中值为value的元素
func LRem(key string, count int64, value interface{}) error {
	return getPool(false).LRem(context.Background(), key, count, value).Err()
}

// ZAdd 有序集合中增加一个成员
func ZAdd(key, member string, score float64) error {
	z := redis.Z{
		Score:  score,
		Member: member,
	}
	_, err := getPool(false).ZAdd(context.Background(), key, &z).Result()
	if err != nil {
		return err
	}

	return nil
}

func ZIncr(key string, member string, score float64) (float64, error) {
	z := redis.Z{
		Score:  score,
		Member: member,
	}

	return getPool(false).ZIncr(context.Background(), key, &z).Result()
}

// ZCount  有序集合中 min-max中的成员数量
func ZCount(slave bool, key, min, max string) (int64, error) {
	count, err := getPool(slave).ZCount(context.Background(), key, min, max).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ZCARD 获取中元素的数量
func ZCARD(slave bool, key string) (int64, error) {
	count, err := getPool(slave).ZCard(context.Background(), key).Result()
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ZRange 通过索引区间返回有序集合成指定区间内的成员
func ZRange(slave bool, key string, start, stop int64) ([]string, error) {
	arr, err := getPool(slave).ZRange(context.Background(), key, start, stop).Result()
	if err != nil {
		return []string{}, err
	}

	return arr, nil
}

func ZScore(key, member string) (float64, error) {
	return getPool(false).ZScore(context.Background(), key, member).Result()
}

// ZRangeByScore 通过索引区间返回有序集合成指定区间内的成员
func ZRangeByScore(slave bool, key string, min, max string) ([]string, error) {
	opt := redis.ZRangeBy{
		Min: min,
		Max: max,
	}
	arr, err := getPool(slave).ZRangeByScore(context.Background(), key, &opt).Result()
	if err != nil {
		return []string{}, err
	}

	return arr, nil
}

func ZRem(key string, members ...string) error {
	return getPool(false).ZRem(context.Background(), key, members).Err()
}

func HGetBytesByField(slave bool, key, filed string) ([]byte, error) {
	return getPool(slave).HGet(context.Background(), key, filed).Bytes()
}

func SIsMember(key string, member interface{}) (bool, error) {
	return getPool(false).SIsMember(context.Background(), key, member).Result()
}

func Incr(key string) (int64, error) {
	return getPool(false).Incr(context.Background(), key).Result()
}

func IncrBy(key string, value int64) (int64, error) {
	return getPool(false).IncrBy(context.Background(), key, value).Result()
}

func IncrWithResult(key string) (int64, error) {
	return getPool(false).Incr(context.Background(), key).Result()
}

func DecrWithResult(key string) (int64, error) {
	return getPool(false).Decr(context.Background(), key).Result()
}

func SMembers(slave bool, key string) ([]string, error) {
	return getPool(slave).SMembers(context.Background(), key).Result()
}

func SAdd(key string, members ...interface{}) (int64, error) {
	return getPool(false).SAdd(context.Background(), key, members...).Result()
}

func SRem(key string, members ...interface{}) (int64, error) {
	return getPool(false).SRem(context.Background(), key, members...).Result()
}

func SisMember(key string, members interface{}) (bool, error) {
	return getPool(false).SIsMember(context.Background(), key, members).Result()
}

func LIndex(key string, index int64) (string, error) {
	return getPool(false).LIndex(context.Background(), key, index).Result()
}

// SetSscan 集合读取
func SetSscan(slave bool, key string, match string, perCount int64) ([]string, error) {
	var (
		cursor = uint64(0)
		data   []string
	)
	for {
		keys, retCursor, err := getPool(slave).SScan(context.Background(), key, cursor, match, perCount).Result()
		if err != nil {
			return data, err
		}
		if len(keys) == 0 {
			break
		}
		data = append(data, keys...)
		if retCursor == 0 {
			break
		}
		cursor = retCursor
	}
	return data, nil
}

// 获取键值,如不存在 则获取func 存入到键中
func GetOrSet(slave bool, key string, f func() (interface{}, error), expire time.Duration) ([]byte, error) {
	result, err := getPool(slave).Get(context.Background(), key).Bytes()
	if err != nil || len(result) == 0 {
		data, err := f()
		if err == nil {
			var value []byte
			value, err = mdata.Cjson.Marshal(data)
			if err != nil {
				return nil, err
			}
			err = getPool(false).Set(context.Background(), key, value, expire).Err()
			if err != nil {
				return nil, err
			}
			return value, nil
		}
		return nil, err
	}
	return result, nil
}

// HGetAll 获取单个 hash
func HGetAll(isSlave bool, key string) (map[string]string, error) {
	return getPool(isSlave).HGetAll(context.Background(), key).Result()
}

// GetTTl 获取某个key的 expire
func GetTTl(slave bool, key string) (time.Duration, error) {
	value, err := getPool(slave).TTL(context.Background(), key).Result()
	if err != nil && err != redis.Nil {
		return 0, err
	}
	return value, nil
}

// IsMember 根据key查看集合中是否存在指定数据
func IsMember(slave bool, key string, value string) (bool, error) {
	isMember, err := getPool(slave).SIsMember(context.Background(), key, value).Result()
	if err != nil && err != redis.Nil {
		return isMember, err
	}
	return isMember, nil
}

// AddMember 添加集合
func AddMember(slave bool, key string, value string) (int64, error) {
	isOk, err := getPool(slave).SAdd(context.Background(), key, value).Result()
	if err != nil && err != redis.Nil {
		return isOk, err
	}
	return isOk, nil
}

func GetTTL(key string) time.Duration {
	return getPool(false).TTL(context.Background(), key).Val()
}

func TxPipelined(ctx context.Context, f func(redis.Pipeliner) error) error {
	_, err := getPool(false).TxPipelined(ctx, f)
	return err
}

func Close() error {
	if redisDb.pool != nil {
		return redisDb.pool.Close()
	}
	return nil
}

func SCard(key string) (int64, error) {
	return getPool(false).SCard(context.Background(), key).Result()
}

func TTL(ctx context.Context, isSlave bool, key string) (time.Duration, error) {
	return getPool(isSlave).TTL(ctx, key).Result()
}
