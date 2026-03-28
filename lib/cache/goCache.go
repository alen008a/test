package cache

import (
	"time"

	goCache "github.com/patrickmn/go-cache"
)

var GoCacheLocal = goCache.New(2*time.Minute, 5*time.Minute)

func GSet(keyName string, value interface{}, expire time.Duration) {
	GoCacheLocal.Set(keyName, value, expire)
}

func GGet(keyName string) (result interface{}, ok bool) {
	result, ok = GoCacheLocal.Get(keyName)
	return
}

func GDel(keyName string) {
	GoCacheLocal.Delete(keyName)
}

//func GetOrSet(keyName string, expire time.Duration, f func() (interface{}, error)) (interface{}, error) {
//	x, found := GoCacheLocal.Get(keyName)
//	if !found {
//		v, err := f()
//		if err != nil {
//			return nil, err
//		}
//
//		if v == nil {
//			return nil, errors.New("GetOrSet not found")
//		}
//
//		GoCacheLocal.Set(keyName, v, expire)
//		return v, nil
//	}
//
//	return x, nil
//}
