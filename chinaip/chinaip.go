package chinaip

import (
	"strconv"
	"strings"
)

// IP2Int converts ip from string format to int format
func IP2Int(ip string) int {
	var strs = strings.Split(ip, ".")
	var a, b, c, d int
	a, _ = strconv.Atoi(strs[0])
	b, _ = strconv.Atoi(strs[1])
	c, _ = strconv.Atoi(strs[2])
	d, _ = strconv.Atoi(strs[3])
	return a*256*256*256 + b*256*256 + c*256 + d
}

// IsChinaIP returns whether a IPv4 address belong to China
func IsChinaIP(ip string) bool {
	var i = IP2Int(ip)
	var l = 0
	var r = len(chinaIPs)
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
