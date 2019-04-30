package util

import (
	"crypto/md5"
	"encoding/hex"
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
