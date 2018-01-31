package freedns

import (
	"fmt"
	"net"
	"time"

	"github.com/Chenyao2333/golang-cache"
	"github.com/Sirupsen/logrus"
	"github.com/miekg/dns"
	geoip2 "github.com/oschwald/geoip2-golang"
)

type Config struct {
	FastDNS  string
	CleanDNS string
	Listen   string
}

type Server struct {
	config Config
	// s[0] servers on udp, s[1] servers on tcp
	s [2]*dns.Server

	chinaDom *goc.Cache
	cache    *goc.Cache
}

type Error string

var geodb *geoip2.Reader
var log = logrus.New()

func init() {
	var err error
	geodb, err = geoip2.Open("GeoLite2-Country.mmdb")

	if err != nil {
		log.Fatalln(err)
	}
}

func (e Error) Error() string {
	return string(e)
}

func NewServer(cfg Config) (*Server, error) {
	s := &Server{}

	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1:53"
	}
	s.config = cfg

	s.s[0] = &dns.Server{
		Addr:    s.config.Listen,
		Net:     "udp",
		Handler: dns.HandlerFunc(s.handle),
	}

	s.s[1] = &dns.Server{
		Addr:    s.config.Listen,
		Net:     "tcp",
		Handler: dns.HandlerFunc(s.handle),
	}

	var err error
	s.chinaDom, err = goc.NewCache("lru", 1024*20)
	if err != nil {
		log.Fatalln(err)
	}

	return s, nil
}

func (s *Server) Run() error {
	// Run tcp and udp servers in goroutines.
	errChan := make(chan error, 2)

	go func() {
		err := s.s[0].ListenAndServe()
		errChan <- err
	}()

	go func() {
		err := s.s[1].ListenAndServe()
		errChan <- err
	}()

	select {
	case err := <-errChan:
		return err
	}
}

func (s *Server) handle(w dns.ResponseWriter, req *dns.Msg) {
	res := &dns.Msg{}
	hit := false
	var err error

	if len(req.Question) < 1 {
		res.SetRcode(req, dns.RcodeBadName)
		w.WriteMsg(res)
		return
	}

	qname := req.Question[0].Name
	upstream := ""

	if res, hit = s.LookupHosts(req); hit {
		upstream = "hosts"
	} else if res, hit = s.LookupCache(req); hit {
		upstream = "cache"
	} else {
		upstream = "net"
		res, upstream, err = s.LookupNet(req)

		if err != nil {
			log.Error(err)
		}

		if res == nil {
			res = &dns.Msg{}
			res.Rcode = dns.RcodeServerFailure
		}
	}

	// SetRcode will set res as reply of req,
	// and also set rcode
	res.SetRcode(req, res.Rcode)

	l := log.WithFields(logrus.Fields{
		"domain":   qname,
		"type":     dns.TypeToString[req.Question[0].Qtype],
		"upstream": upstream,
		"status":   dns.RcodeToString[res.Rcode],
	})
	if res.Rcode == dns.RcodeSuccess {
		l.Info()
	} else {
		l.Warn()
	}

	//fmt.Println(res)
	w.WriteMsg(res)
}

// LookupNet resolve the the dns request through net.
// The first return value is answer,iff it's nil means failed in resolving.
// Due to implementation, now the error will always be nil,
// but don't do this assumpation in your code.
func (s *Server) LookupNet(req *dns.Msg) (*dns.Msg, string, error) {
	fastCh := make(chan *dns.Msg)
	cleanCh := make(chan *dns.Msg)

	Q := func(ch chan *dns.Msg, useClean bool) {
		upstream := s.config.FastDNS
		if useClean {
			upstream = s.config.CleanDNS
		}

		res, err := resolve(req, upstream, "udp")
		if res == nil {
			ch <- nil
			return
		}

		// if it's fastDNS upstream and maybe polluted, just return serverFailure
		if !useClean && (res.Rcode != dns.RcodeSuccess || err != nil || s.maybePolluted(res)) {
			ch <- nil
			return
		}

		ch <- res
	}

	go Q(cleanCh, true)
	go Q(fastCh, false)

	// ensure ch must will receive nil after timeout
	go func() {
		time.Sleep(2 * time.Second)
		fastCh <- nil
		cleanCh <- nil
	}()

	// first try to resolve by fastDNS
	res := <-fastCh
	if res != nil {
		return res, s.config.FastDNS, nil
	}

	// if fastDNS failed, just return result of cleanDNS
	res = <-cleanCh
	return res, s.config.CleanDNS, nil
}

func resolve(req *dns.Msg, upstream string, net string) (*dns.Msg, error) {
	r := new(dns.Msg)
	r.Id = dns.Id()
	r.Question = req.Question
	r.RecursionDesired = req.RecursionDesired

	c := &dns.Client{Net: net}

	res, _, err := c.Exchange(r, upstream)

	return res, err
}

func (s *Server) maybePolluted(res *dns.Msg) bool {
	if containA(res) {
		china := containChinaIP(res)
		s.chinaDom.Set(res.Question[0].Name, china)
		return !china
	}

	china, ok := s.chinaDom.Get(res.Question[0].Name)
	if ok {
		return !china.(bool)
	}
	return false
}

func containA(res *dns.Msg) bool {
	var rrs []dns.RR

	rrs = append(rrs, res.Answer...)
	rrs = append(rrs, res.Ns...)
	rrs = append(rrs, res.Extra...)

	for i := 0; i < len(rrs); i++ {
		_, ok := rrs[i].(*dns.A)
		if ok {
			return true
		}
	}
	return false
}

// containChinaIP judge answers whether contains IP belong to China.
func containChinaIP(res *dns.Msg) bool {
	var rrs []dns.RR

	rrs = append(rrs, res.Answer...)
	rrs = append(rrs, res.Ns...)
	rrs = append(rrs, res.Extra...)

	for i := 0; i < len(rrs); i++ {
		rr, ok := rrs[i].(*dns.A)
		if ok {
			ip := rr.A.String()
			if isChinaIP(ip) {
				return true
			}
		}
	}
	return false
}

func isChinaIP(ip string) bool {
	record, err := geodb.Country(net.ParseIP(ip))
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(record.Country.IsoCode)
	return record.Country.IsoCode == "CN"
}

func genCacheKey(q dns.Question) string {
	return q.Name + "_" + dns.TypeToString[q.Qtype]
}

type cacheEntry struct {
	putin time.Time
	reply *dns.Msg
}

func (s *Server) LookupCache(req *dns.Msg) (*dns.Msg, bool) {
	return nil, false
}

func (s *Server) LookupHosts(req *dns.Msg) (*dns.Msg, bool) {
	// TODO: implement needed
	return nil, false
}
