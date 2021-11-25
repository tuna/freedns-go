package chinaip

import (
	"bufio"
	"encoding/binary"
	"log"
	"net"
	"os"
	"sort"
)

type Error string

func (e Error) Error() string {
	return string(e)
}

// IsChinaIP returns whether an IPv4 address belongs to China
func IsChinaIP(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	i := binary.BigEndian.Uint32(parsed.To4())
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

var chinaIPs = [][]uint32{}

// IPv4 only
func LoadChinaIP(name string) {
	file, err := os.Open(name)
	if err != nil {
		log.Fatalln(err)
		os.Exit(-1)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		_, cidr, err := net.ParseCIDR(line)
		if err != nil {
			continue
		}
		start := binary.BigEndian.Uint32(cidr.IP)
		end := start + ^uint32(0) - binary.BigEndian.Uint32(cidr.Mask)
		chinaIPs = append(chinaIPs, []uint32{start, end})
	}
	// sort by start is ok, assuming there's no overlap
	sort.Slice(chinaIPs, func(i, j int) bool {
		return chinaIPs[i][0] < chinaIPs[j][0]
	})
}
