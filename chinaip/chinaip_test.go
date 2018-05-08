package chinaip_test

import "testing"
import "github.com/Chenyao2333/freedns-go/chinaip"

func TestIsChinaIP(t *testing.T) {
	var cn_ips = []string{"114.114.114.114", "220.181.57.216"}
	var non_cn_ips = []string{"8.8.8.8", "172.217.14.78"}

	for _, ip := range cn_ips {
		if !chinaip.IsChinaIP(ip) {
			t.Errorf("%s is China IP!", ip)
		}
	}

	for _, ip := range non_cn_ips {
		if chinaip.IsChinaIP(ip) {
			t.Errorf("%s isn't China IP!", ip)
		}
	}
}
