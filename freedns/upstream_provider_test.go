package freedns

import (
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestInvalidUpstreamProvider(t *testing.T) {
	cases := []string{
		"asdfasdf",
		"/dev/null",
	}
	for _, name := range cases {
		if _, err := newUpstreamProvider(name); err == nil {
			t.Errorf("Should not create provider for %s", name)
		}
	}
}

func TestStaticUpstreamProvider(t *testing.T) {
	provider, err := newUpstreamProvider("8.8.4.4")
	if err != nil || provider == nil {
		t.Errorf("Cannot create static upstream provider")
		return
	}
	if upstream := provider.GetUpstream(); upstream != "8.8.4.4:53" {
		t.Errorf("Static upstream provider invalid result %s", upstream)
	}
}

func TestResolvconfUpstreamProvider(t *testing.T) {
	tempfile, err := ioutil.TempFile("", "test_resolvconf")
	if err != nil {
		t.Errorf("Cannot create temp file: %s", err.Error())
		return
	}
	filename := tempfile.Name()
	tempfile.Close()

	WriteContent := func(content string) {
		tempfile, err := os.Create(filename)
		if err != nil {
			t.Errorf("Cannot open temp file: %s", err.Error())
			return
		}
		defer tempfile.Close()
		tempfile.Write([]byte(content))
	}

	defer os.Remove(filename)

	WriteContent("nameserver 1.2.3.4\n")

	provider, err := newUpstreamProvider(filename)
	if err != nil {
		t.Errorf("Cannot create resolvconf upstream provider for %s: %s", filename, err.Error())
		return
	}

	if upstream := provider.GetUpstream(); upstream != "1.2.3.4:53" {
		t.Errorf("Bad result %s", upstream)
	}

	WriteContent("nameserver 8.8.8.8\nnameserver8.8.4.4\n")
	time.Sleep(100 * time.Millisecond)
	// should be updated
	if upstream := provider.GetUpstream(); upstream != "8.8.8.8:53" {
		t.Errorf("Bad result %s", upstream)
	}

	WriteContent("some invalid content")
	time.Sleep(100 * time.Millisecond)
	// should not be updated
	if upstream := provider.GetUpstream(); upstream != "8.8.8.8:53" {
		t.Errorf("Bad result %s", upstream)
	}
}
