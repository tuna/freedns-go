package chinaip

import (
	"strconv"
	"strings"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

// IP2Int converts ip from string format to int format
func IP2Int(ip string) (int, error) {
	strs := strings.Split(ip, ".")
	if len(strs) != 4 {
		return -1, Error("isn't ipv4 addr")
	}
	ret := 0
	mul := 1
	for i := 3; i >= 0; i-- {
		a, err := strconv.Atoi(strs[i])
		if err != nil {
			return -1, err
		}
		ret += a * mul
		mul *= 256
	}
	return ret, nil
}

// IsChinaIP returns whether a IPv4 address belong to China
func IsChinaIP(ip string) bool {
	var i, _ = IP2Int(ip)
	var l = 0
	var r = len(chinaIPs) - 1
	for l <= r {
		var mid = int((l + r) / 2)
		if i < chinaIPs[mid][0] {
			r = mid - 1
		} else if i > chinaIPs[mid][1] {
			l = mid + 1
		} else {
			return true
		}
	}
	return false
}
