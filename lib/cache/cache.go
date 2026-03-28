package cache

import (
	"fmt"
	"msgPushSite/internal/glog"
	"sync"
	"time"

	"github.com/RussellLuo/timingwheel"
)

// 本地缓存方案
func init() {
	gTimingWheel.Start()
}

type (
	rotatePeriod  = time.Duration
	ValueFunction = func() (interface{}, error)
)

const (
	RotatePeriod10  = 10 * time.Second
	RotatePeriod30  = 30 * time.Second
	RotatePeriod60  = 60 * time.Second
	RotatePeriod120 = 120 * time.Second
)

var (
	gCache       = make(map[string]*cache)
	gTimingWheel = timingwheel.NewTimingWheel(time.Millisecond, 20)
	mux          = sync.RWMutex{}
)

type RotateScheduler struct{ Interval time.Duration }

func (s *RotateScheduler) Next(prev time.Time) time.Time {
	return prev.Add(s.Interval)
}
func DeleteCache(k string) {
	delete(gCache, k)
}

func getCache(k string, s rotatePeriod, f ValueFunction) (*cache, error) {
	mux.RLock()
	ca, ok := gCache[k]
	if ok {
		mux.RUnlock()
		return ca, nil
	}

	mux.RUnlock()

	mux.Lock()

	data, err := f()
	if err != nil {
		mux.Unlock()
		glog.Errorf("localCache |k=%s |err=%v", k, err)
		return nil, err
	}

	ca = &cache{
		k:      k,
		data:   data,
		rotate: s,
		f:      f,
		l:      new(sync.RWMutex),
	}

	gCache[k] = ca

	gCache[k].tw = gTimingWheel.ScheduleFunc(&RotateScheduler{Interval: s}, func() {
		mux.RLock()
		//定期更新
		_, err = gCache[k].load()
		if err != nil {
			fmt.Printf("重载失败:%+v\n", err)
		}

		mux.RUnlock()
	})

	mux.Unlock()

	return ca, nil
}

type cache struct {
	k      string
	data   interface{}
	rotate rotatePeriod
	f      ValueFunction
	l      *sync.RWMutex
	tw     *timingwheel.Timer
}

func GetOrSet(k string, s rotatePeriod, f ValueFunction) (interface{}, error) {
	ca, err := getCache(k, s, f)
	if err != nil {
		return nil, err
	}
	ca.l.RLock()
	data := ca.data
	if data == nil {
		ca.l.RUnlock()
		return ca.load()
	}
	ca.l.RUnlock()
	return data, nil
}

func (ca *cache) load() (interface{}, error) {
	data, err := ca.f()
	if err != nil {
		glog.Errorf("localCache |err=%v", err)
		return ca.data, err
	}
	ca.l.Lock()
	ca.data = data
	ca.l.Unlock()
	return data, nil
}

func Close() {
	mux.Lock()
	for _, c := range gCache {
		c.tw.Stop()
	}
	mux.Unlock()
	gTimingWheel.Stop()
}
