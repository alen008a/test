package utils

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

const (
	TimeBarFormat          = "2006-01-02 15:04:05"
	TimeBarFormatPM        = "2006-01-02 15:04:05 PM"
	TimeFormatHMS          = "20060102150405"
	TimeUnderlineYearMonth = "2006_01"
	TimeBarYYMMDD          = "2006-01-02"
	TimeMMDD               = "01-02"
	TimeHHMMSS             = "15:04:05"
	TimeHHMM               = "15:04"
	TimeYYMMDD             = "20060102"
	TimeYMDHM              = "200601021504"
	TimeBEIJINGFormat      = "2006-01-02 15:04:05 +08:00"
	TimeGDFormat           = "01/02/2006 15:04:05"
	TimeTFormat            = "2006-01-02T15:04:05"
	TimeTBjFormat          = "2006-01-02T15:04:05+08:00"

	Minute   = 60
	HourVal  = Minute * 60
	DayVal   = HourVal * 24
	MonthVal = DayVal * 30
	YearVal  = MonthVal * 365

	BeiJinAreaTime = "Asia/Shanghai"
)

func GetBjTimeLoc() *time.Location {
	// 获取北京时间, 在 windows系统上 time.LoadLocation 会加载失败, 最好的办法是用 time.FixedZone
	var bjLoc *time.Location
	var err error
	bjLoc, err = time.LoadLocation(BeiJinAreaTime)
	if err != nil {
		bjLoc = time.FixedZone("CST", 8*3600)
	}

	return bjLoc
}

func GetBjNowTime() time.Time {
	// 获取北京时间, 在 windows系统上 time.LoadLocation 会加载失败, 最好的办法是用 time.FixedZone
	var bjLoc *time.Location
	var err error
	bjLoc, err = time.LoadLocation(BeiJinAreaTime)
	if err != nil {
		bjLoc = time.FixedZone("CST", 8*3600)
	}

	return time.Now().In(bjLoc)
}

// 将北京时间 2006-01-02 15:04:05 类型的时间转换为时间
func BjTBarFmtTime(timeStr string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, errors.New("time is empty")
	}

	bjTimeLoc := GetBjTimeLoc()
	return time.ParseInLocation(TimeBarFormat, timeStr, bjTimeLoc)
}

// 将时间戳转换为北京时间
func FmtUnixToBjTime(timestamp int64) time.Time {
	bjTimeLoc := GetBjTimeLoc()

	utcTime := time.Unix(timestamp, 0)
	return utcTime.In(bjTimeLoc)
}

// 将 2019-08-15T16:00:00+08:00 类型的时间数据转化为多少小时或分钟前
func GetTimeInterval(timeStr string) string {
	if timeStr == "" {
		return ""
	}

	bjTime, err := time.ParseInLocation(TimeTBjFormat, timeStr, GetBjTimeLoc())
	if err != nil {
		return "30分钟前"
	}
	//fmt.Println("bjTime: ", bjTime.Format(TimeBarFormat))

	interval := GetBjNowTime().Unix() - bjTime.Unix()
	if interval < 60 {
		return "刚刚"
	}

	if interval/Minute > 0 && interval/Minute < Minute {
		return fmt.Sprintf("%v分钟前", interval/(Minute))
	} else if interval/HourVal > 0 && interval/HourVal < 24 {
		return fmt.Sprintf("%v小时前", interval/HourVal)
	} else if interval/DayVal > 0 && interval/DayVal < 30 {
		return fmt.Sprintf("%v天前", interval/DayVal)
	} else if interval/MonthVal > 0 && interval/MonthVal < 12 {
		return fmt.Sprintf("%v月前", interval/MonthVal)
	} else if interval/YearVal > 0 {
		return fmt.Sprintf("%v年前", interval/YearVal)
	}

	return "刚刚"
}

func ChangeToES(v string) string {
	sourceTime, _ := time.Parse("2006-01-02 15:04:05", v)
	return sourceTime.UTC().Format("2006-01-02T15:04:05+08:00")
}

// ParseTime 解析时间字符串
func ParseTime(sTime string) (time.Time, error) {

	loc, _ := time.LoadLocation("Asia/Shanghai")
	resTime, err := time.ParseInLocation(TimeBarYYMMDD, sTime, loc)
	if err != nil {
		resTime, err = time.ParseInLocation(TimeBarFormat, sTime, loc)
		if err != nil {
			sTime = strings.ReplaceAll(sTime, "T", " ")
			sTime = strings.ReplaceAll(sTime, "+08:00", "")
			resTime, err = time.ParseInLocation(TimeBarFormat, sTime, loc)
		}
	}
	return resTime, err
}

// 通过两个时间差,计算如果从这个时间去查询数据,需要从 end 的月份开始倒查哪些月份
func GetDataNeedMonToEsIndex(start, end, format string) (string, error) {
	t1, err := ParseTime(start)
	if err != nil {
		return "", err
	}
	t2, err := ParseTime(end)
	if err != nil {
		return "", err
	} else {
		var y1, y2, m1, m2 int
		var begin time.Time
		loc, _ := time.LoadLocation("Asia/Shanghai")

		if t1.Before(t2) { //如果时间 start 大
			y1 = t1.Year()
			y2 = t2.Year()
			m1 = int(t1.Month())
			m2 = int(t2.Month())
			begin = time.Date(t1.Year(), t1.Month(), 1, 0, 0, 0, 0, loc)
		} else {
			y1 = t2.Year()
			y2 = t1.Year()
			m1 = int(t2.Month())
			m2 = int(t1.Month())
			begin = time.Date(t2.Year(), t2.Month(), 1, 0, 0, 0, 0, loc)
		}

		yearInterval := y1 - y2
		// 如果 d1的 月 小于 d2的 月 那么 yearInterval-- 这样就得到了相差的年数
		if m1 < m2 {
			yearInterval--
		}
		// 获取月数差值
		monthInterval := (m1 + 12) - m2
		monthInterval %= 12
		month := int(math.Abs(float64(yearInterval*12)+float64(monthInterval))) + 1
		if month < 1 {
			return "", errors.New("时间计算错误")
		}
		indexStr := begin.Format(format)
		for i := 1; i < month; i++ {
			thisMon := begin.AddDate(0, i, 0)
			indexStr += thisMon.Format("," + format)
		}
		return indexStr, nil
	}
}

// GetESTimeFomat return 2019-01-14T19:00:33+08:00
func GetESTimeFomat(timestr string) string {
	return fmt.Sprintf("%s+08:00", strings.Replace(strings.TrimSpace(timestr), " ", "T", -1))
}

// GetUnTimeFormat return 2019-01-14 19:00:33
func GetUnTimeFormat(timestr string) string {
	timeMainPart := strings.Replace(strings.TrimSpace(timestr), "+08:00", "", -1)
	return fmt.Sprintf(strings.Replace(strings.TrimSpace(timeMainPart), "T", " ", -1))
}

func StrToTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05 -0700",
		"2006-01-02 15:04:05",
		"2006/01/02 15:04:05 -0700 MST",
		"2006/01/02 15:04:05 -0700",
		"2006/01/02 15:04:05",
		"2006-01-02 -0700 MST",
		"2006-01-02 -0700",
		"2006-01-02",
		"2006/01/02 -0700 MST",
		"2006/01/02 -0700",
		"2006/01/02",
		"2006-01-02 15:04:05 -0700 -0700",
		"2006/01/02 15:04:05 -0700 -0700",
		"2006-01-02 -0700 -0700",
		"2006/01/02 -0700 -0700",
		time.ANSIC,
		time.UnixDate,
		time.RubyDate,
		time.RFC822,
		time.RFC822Z,
		time.RFC850,
		time.RFC1123,
		time.RFC1123Z,
		time.RFC3339,
		time.RFC3339Nano,
		time.Kitchen,
		time.Stamp,
		time.StampMilli,
		time.StampMicro,
		time.StampNano,
	}

	var t time.Time
	var err error
	for _, layout := range layouts {
		t, err = time.Parse(layout, value)
		if err == nil {
			return t
		}
	}

	return t
}

// 获取北京时间区的地理位置
func GetLoctionBJ() *time.Location {
	var beiJinLocation *time.Location
	var err error
	beiJinLocation, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		beiJinLocation = time.FixedZone("CST", 8*3600)
	}
	return beiJinLocation
}

// BJNowTime 北京当前时间
func BJNowTime() time.Time {
	// 获取北京时间, 在 windows系统上 time.LoadLocation 会加载失败, 最好的办法是用 time.FixedZone, es 中的时间为: "2019-03-01T21:33:18+08:00"
	var beiJinLocation *time.Location
	var err error

	beiJinLocation, err = time.LoadLocation("Asia/Shanghai")
	if err != nil {
		beiJinLocation = time.FixedZone("CST", 8*3600)
	}

	nowTime := time.Now().In(beiJinLocation)

	return nowTime
}

// StrToBJNowTime  string 类型转换为北京当前时间
func StrToBJTime(timeStr string) (time.Time, error) {
	bjLoc := GetLoctionBJ()

	bjTime, err := time.ParseInLocation(TimeBarFormat, timeStr, bjLoc)
	if err != nil {
		return time.Time{}, err
	}

	return bjTime, nil
}

func BeginOfTime(t *time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

func EndOfTime(t *time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, t.Location())
}

func BeginOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func EndOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
}

func BeginOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
}

func EndOfMonth(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month()+1, 0, 23, 59, 59, 999999999, t.Location())
}

func BeginOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
}

func EndOfYear(t time.Time) time.Time {
	return time.Date(t.Year(), 12, 31, 23, 59, 59, 999999999, t.Location())
}

// 一周的开始时间是周一到周日
func BeginOfWeek(t time.Time) time.Time {
	date := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
	days := -int(date.Weekday()) + 1
	date = date.AddDate(0, 0, days)
	return date
}

func EndOfWeek(t time.Time) time.Time {
	date := time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, t.Location())
	days := 7 - int(date.Weekday())
	date = date.AddDate(0, 0, days)
	return date
}

// 2006-01-02T15:04:05+08:00  转  2006-01-02 15:04:05
func GetStrToTime(timeStr string) string {
	times, err := time.Parse(TimeTBjFormat, timeStr)
	if err != nil {
		return ""
	}
	return times.Format(TimeBarFormat)
}

func GetStrToTimeS(timeStr string) string {
	times, err := time.Parse(TimeTBjFormat, timeStr)
	if err != nil {
		return ""
	}
	return times.Format(TimeBarYYMMDD)
}

// 以当天为起点，获取某天00:00:00 offset -1为昨天
func GetStartDateTime(offset int) string {
	t := time.Now().AddDate(0, 0, offset)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location()).Format(TimeBarFormat)
}

// 以当天为起点，获取某天23:59:59 offset -1为昨天
func GetEndDateTime(offset int) string {
	t := time.Now().AddDate(0, 0, offset)
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location()).Format(TimeBarFormat)
}

// GetBetweenDates 根据开始日期和结束日期计算出时间段内所有日期 返回  TimeUnderlineYearMonth 格式
func GetBetweenDates(sdate, edate, prefix string) []string {
	d := make([]string, 0)
	timeFormatTpl := TimeBarFormat
	date, err := time.Parse(TimeBarFormat, sdate)
	if err != nil {
		return d
	}
	date2, err := time.Parse(TimeBarFormat, edate)
	if err != nil {
		return d
	}
	if date2.Before(date) {
		return d
	}
	// 输出日期格式固定
	timeFormatTpl = TimeUnderlineYearMonth
	date2Str := date2.Format(timeFormatTpl)
	//d = append(d, date.Format(timeFormatTpl))
	for {
		dateStr := date.Format(timeFormatTpl)
		d = append(d, fmt.Sprint(prefix, dateStr))
		if dateStr == date2Str {
			break
		}
		date = date.AddDate(0, 1, 0)
	}
	return d
}

// 将北京时间 2006-01-02 15:04:05 类型的时间转换为时间
func BjTBarFmtTimeFormat(timeStr string, timeFormat string) (time.Time, error) {
	if timeStr == "" {
		return time.Time{}, errors.New("time is empty")
	}

	bjTimeLoc := GetBjTimeLoc()
	return time.ParseInLocation(timeFormat, timeStr, bjTimeLoc)
}

// 判断当前时间是否在指定时间之内
func CurrentTimeBetween(timeStart, timeEnd string) (bool, error) {
	start, err := BjTBarFmtTimeFormat(TimeBarFormat, timeStart)
	if err != nil {
		return false, err
	}
	end, err := BjTBarFmtTimeFormat(TimeBarFormat, timeEnd)
	if err != nil {
		return false, err
	}
	now := BJNowTime()
	if now.Before(start) || now.After(end) {
		return false, nil
	}
	return true, nil
}

// 获取两个日期相差的月
func DiffMonth(t1, t2 time.Time) (month int) {
	y1 := t1.Year()
	y2 := t2.Year()
	m1 := int(t1.Month())
	m2 := int(t2.Month())
	d1 := t1.Day()
	d2 := t2.Day()

	yearInterval := y1 - y2
	// 如果 d1的 月-日 小于 d2的 月-日 那么 yearInterval-- 这样就得到了相差的年数
	if m1 < m2 || m1 == m2 && d1 < d2 {
		yearInterval--
	}
	// 获取月数差值
	monthInterval := (m1 + 12) - m2
	if d1 < d2 {
		monthInterval--
	}
	monthInterval %= 12
	month = yearInterval*12 + monthInterval
	return
}

// 获取指定月份的日期
func MinusMonths(t time.Time, monthCount int) time.Time {
	return time.Date(t.Year(), t.Month()-time.Month(monthCount), t.Day(), 0, 0, 0, 0, t.Location())
}

// 获取指定时间的日期
func GetDate(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// 获取两个时间相差的天数，0表同一天，正数表t1>t2，负数表t1<t2
func GetDiffDays(t1, t2 time.Time) int {
	t1 = time.Date(t1.Year(), t1.Month(), t1.Day(), 0, 0, 0, 0, time.Local)
	t2 = time.Date(t2.Year(), t2.Month(), t2.Day(), 0, 0, 0, 0, time.Local)

	return int(t1.Sub(t2).Hours() / 24)
}

// 返回到凌晨的秒数
func GetEarlyMorningSecond() time.Duration {
	location, _ := time.LoadLocation(BeiJinAreaTime)
	todayEndTime, _ := time.ParseInLocation(TimeBarFormat, time.Now().Format(TimeBarYYMMDD)+" 23:59:59", location)
	duration := time.Duration(todayEndTime.Unix()-time.Now().In(location).Unix()) * time.Second
	return duration
}

// 获取倒计时多少小时或分钟
func GetCutDownInterval(endTime time.Time) string {
	interval := endTime.Unix() - GetBjNowTime().Unix()
	if interval <= 0 {
		return "EOF"
	}
	if interval < 60 {
		return fmt.Sprintf("%d秒", interval)
	}
	day := interval / (24 * 60 * 60)
	hour := (interval - day*24*60*60) / (60 * 60)
	minute := math.Ceil(float64(interval-day*24*60*60-hour*60*60) / float64(60))
	if day > 0 {
		return fmt.Sprintf("%d天", day)
	}
	if hour > 0 {
		return fmt.Sprintf("%d小时", hour)
	}
	if minute > 0 {
		return fmt.Sprintf("%.0f分", minute)
	}
	return ""
}
