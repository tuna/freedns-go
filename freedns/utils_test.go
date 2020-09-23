package freedns

import "testing"

func Test_normalizeDnsAddress(t *testing.T) {
	assertError := func(addr string) {
		if res, err := normalizeDnsAddress(addr); res != "" || err == nil {
			t.Errorf("%s should not be normalized as dns address", addr)
		}
	}
	assertResult := func(addr, expected string) {
		if res, err := normalizeDnsAddress(addr); res != expected || err != nil {
			t.Errorf("%s should be normalized as %s, got %s (%s)", addr, expected, res, err.Error())
		}
	}

	assertError("")
	assertError("hello world")
	assertError("123:456")
	assertError("/hello/world:filename")
	assertError("1.2.3.4.5")

	assertResult("1.2.3.4", "1.2.3.4:53")
	assertResult("127.0.0.1:3333", "127.0.0.1:3333")
	assertResult("::1", "[::1]:53")
	assertResult("[::]:5300", "[::]:5300")
	assertResult(":5300", "0.0.0.0:5300")
}
