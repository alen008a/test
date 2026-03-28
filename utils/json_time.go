package utils

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type JSONTime struct {
	time.Time
}

// MarshalJSON on JSONTime format Time field with %Y-%m-%d %H:%M:%S
func (t JSONTime) MarshalJSON() ([]byte, error) {
	if t.Time.Year() == 1 {
		return []byte("\"\""), nil
	}
	formatted := fmt.Sprintf("\"%s\"", t.Format("2006-01-02 15:04:05"))
	return []byte(formatted), nil
}

// Value insert timestamp into mysql need this function.
func (t JSONTime) Value() (driver.Value, error) {
	var zeroTime time.Time
	if t.Time.UnixNano() == zeroTime.UnixNano() {
		return nil, nil
	}
	return t.Time, nil
}

func (t *JSONTime) UnmarshalJSON(data []byte) error {
	// Ignore null, like in the main JSON package.
	if len(data) == 0 || string(data) == "null" {
		return nil
	}
	temp := string(data)
	temp = strings.ReplaceAll(temp, "\"", "")
	ut, err := time.ParseInLocation("2006-01-02 15:04:05", temp, time.Now().Location())
	if err == nil {
		*t = JSONTime{Time: ut}
		return nil
	}
	return err
}

// Scan valueof time.Time
func (t *JSONTime) Scan(v interface{}) error {
	uints, ok := v.([]uint8)
	if ok {
		bytes := make([]byte, len(uints))
		for i, uint := range uints {
			bytes[i] = uint
		}
		str := string(bytes)
		ut, err := time.ParseInLocation("2006-01-02 15:04:05", str, time.Now().Location())
		if err == nil {
			*t = JSONTime{Time: ut}
			return nil
		}
		return fmt.Errorf("can not convert %v to timestamp", v)
	}
	value, ok := v.(time.Time)
	if ok {
		*t = JSONTime{Time: value}
		return nil
	}
	return fmt.Errorf("can not convert %v to timestamp", v)
}
