package util

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"hash/crc32"
)

func HashCode(s string) int {
	if len(s) < 1 {
		return 0
	}
	v := int(crc32.ChecksumIEEE([]byte(s)))
	if v >= 0 {
		return v
	}
	if -v >= 0 {
		return -v
	}
	return 0
}

func MD5(s string) string {
	if len(s) < 1 {
		return ""
	}
	h := md5.New()
	h.Write([]byte(s))
	return hex.EncodeToString(h.Sum(nil))
}

func SafeInt(i interface{}) (int, error) {
	var t int
	switch i.(type) {
	case int:
		t = i.(int)
	case int8:
		t = int(i.(int8))
	case int16:
		t = int(i.(int16))
	case int32:
		t = int(i.(int32))
	case int64:
		t = int(i.(int64))
	default:
		return 0, errors.New("the input data is not an int[int8|int16|int32|int64] number")
	}
	return t, nil
}
