package task

import (
	"sync"
	"time"
)

type unsafeSlideWindow struct {
	start      uint32        //开始索引
	end        uint32        //结束索引
	interval   time.Duration //间隔时间
	limitCount uint32        //间隔时间内限制的最大数量
	timeFlow   []time.Time   //左闭右开 循环队列
}

type safeSlideWindow struct {
	*unsafeSlideWindow
	mutex sync.Mutex
}

func NewSlideWindow(intervalTime time.Duration, count uint32) *safeSlideWindow {
	return &safeSlideWindow{
		unsafeSlideWindow: &unsafeSlideWindow{
			start:      0,
			end:        0,
			interval:   intervalTime,
			limitCount: count,
			timeFlow:   make([]time.Time, count+1),
		},
	}
}

// 原子性的增加1
func (ss *safeSlideWindow) Incr() bool {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	return ss.incr()
}

// 原子性的阻塞增加1
func (ss *safeSlideWindow) BlockIncr() bool {
	ss.mutex.Lock()
	defer ss.mutex.Unlock()
	return ss.blockIncr()
}

// 阻塞的增加1
func (us *unsafeSlideWindow) blockIncr() bool {
	for !us.incr() {
		nowTime := time.Now()
		tmp := nowTime.Sub(us.timeFlow[us.start])
		if us.interval-tmp > 0 {
			<-time.After(us.interval - tmp)
		}
	}
	return true
}

//增加1
func (us *unsafeSlideWindow) incr() bool {
	nowTime := time.Now()
	//查看记录的数据是否已经超过了limitCount
	count := us.end - us.start
	if us.end < us.start {
		count = uint32(len(us.timeFlow)) - uint32(us.start)
		count += us.end
	}
	if count >= us.limitCount && nowTime.Sub(us.timeFlow[us.start]) <= us.interval {
		return false
	}

	if count >= us.limitCount {
		us.start = (us.start + 1) % uint32(len(us.timeFlow))
	}
	us.timeFlow[us.end] = nowTime
	us.end = (us.end + 1) % uint32(len(us.timeFlow))
	return true
}
