package freedns

import (
	"testing"
)

func TestAppendDefaultPort(t *testing.T) {
	cases := []struct {
		i string
		o string
	}{
		{"127.0.0.1", "127.0.0.1:53"},
		{"114.114.114.114:5353", "114.114.114.114:5353"},
		{"::1", "::1"},
	}
	for _, c := range cases {
		if appendDefaultPort(c.i) != c.o {
			t.Errorf("Expected: %s", c.o)
		}
	}
}

func TestSmokingNewRunAndShutdown(t *testing.T) {
	s, err := NewServer(Config{
		FastDNS:  "114.114.114.114",
		CleanDNS: "1.1.1.1",
		Listen:   "127.0.0.1:52345",
		CacheCap: 1024 * 5,
	})
	if err != nil {
		t.Error(err)
	}
	go func() {
		err := s.Run()
		if err != nil {
			t.Error(err)
		}
		s.Shutdown()
	}()
}
