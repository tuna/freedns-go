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
		Addr: s.config.Listen,
		Net:  "udp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			s.handle(w, req, "udp")
		}),
	}

	s.s[1] = &dns.Server{
		Addr: s.config.Listen,
		Net:  "tcp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			s.handle(w, req, "tcp")
		}),
	}

	var err error
	s.chinaDom, err = goc.NewCache("lru", 1024*20)
	if err != nil {
		log.Fatalln(err)
	}

	s.cache, err = goc.NewCache("lru", 1024*20)
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

func (s *Server) handle(w dns.ResponseWriter, req *dns.Msg, net string) {
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
	} else if res, hit = s.LookupCache(req, net); hit {
		upstream = "cache"
	} else {
		upstream = "net"
		res, upstream, err = s.LookupNet(req, net)

		if err != nil {
			log.Error(err)
		}

		if res == nil {
			res = serverFailureMsg(req)
		}
	}

	// SetRcode will set res as reply of req,
	// and also set rcode
	res.SetRcode(req, res.Rcode)

	l := log.WithFields(logrus.Fields{
		"op":       "handle-LookupNet",
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
func (s *Server) LookupNet(req *dns.Msg, net string) (*dns.Msg, string, error) {
	fastCh := make(chan *dns.Msg)
	cleanCh := make(chan *dns.Msg)

	Q := func(ch chan *dns.Msg, useClean bool) {
		upstream := s.config.FastDNS
		if useClean {
			upstream = s.config.CleanDNS
		}

		//fmt.Println(req.Id)
		res, err := resolve(req, upstream, net)

		if err != nil {
			log.WithFields(logrus.Fields{
				"op":       "Resolve",
				"upstream": upstream,
				"domain":   req.Question[0].Name,
			}).Error(err)
		}

		if res == nil {
			ch <- nil
			return
		}

		// if it's fastDNS upstream and maybe polluted, just return serverFailure
		//fmt.Println(upstream, res)
		if !useClean && (res.Rcode != dns.RcodeSuccess || err != nil || s.maybePolluted(res)) {
			// fmt.Println("114 failed", err)
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
		s.setCache(res, net)
		return res, s.config.FastDNS, nil
	}

	// if fastDNS failed, just return result of cleanDNS
	res = <-cleanCh
	if res == nil {
		res = serverFailureMsg(req)
	}
	if res.Rcode == dns.RcodeSuccess {
		s.setCache(res, net)
	}
	return res, s.config.CleanDNS, nil
}

func serverFailureMsg(req *dns.Msg) *dns.Msg {
	res := &dns.Msg{}
	res.SetRcode(req, dns.RcodeServerFailure)
	return res
}

func resolve(req *dns.Msg, upstream string, net string) (*dns.Msg, error) {
	r := req.Copy()
	r.Id = dns.Id()

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
	//fmt.Println(record.Country.IsoCode)
	return record.Country.IsoCode == "CN"
}

func genCacheKey(r *dns.Msg, net string) string {
	q := r.Question[0]
	s := q.Name + "_" + dns.TypeToString[q.Qtype]
	if r.RecursionDesired {
		s += "_1"
	} else {
		s += "_0"
	}
	s += "_" + net
	fmt.Println(s)
	return s
}

type cacheEntry struct {
	putin time.Time
	reply *dns.Msg
}

func subTTL(res *dns.Msg, delta int) bool {
	needUpdate := false
	S := func(rr []dns.RR) {
		for i := 0; i < len(rr); i++ {
			newTTL := int(rr[i].Header().Ttl)
			newTTL -= delta

			if newTTL <= 0 {
				newTTL = 3
				needUpdate = true
			}

			rr[i].Header().Ttl = uint32(newTTL)
		}
	}

	S(res.Answer)
	S(res.Ns)
	S(res.Extra)

	return needUpdate
}

func (s *Server) LookupCache(req *dns.Msg, net string) (*dns.Msg, bool) {
	key := genCacheKey(req, net)
	ci, ok := s.cache.Get(key)

	if ok {
		c := ci.(cacheEntry)
		delta := time.Now().Sub(c.putin).Seconds()

		r := c.reply.Copy()
		needUpdate := subTTL(r, int(delta))
		if needUpdate {
			go func() {
				res, upstream, _ := s.LookupNet(req, net)

				l := log.WithFields(logrus.Fields{
					"op":       "LookupCache-LookupNet",
					"domain":   req.Question[0].Name,
					"type":     dns.TypeToString[req.Question[0].Qtype],
					"upstream": upstream,
					"status":   dns.RcodeToString[res.Rcode],
				})

				if res.Rcode != dns.RcodeSuccess {
					l.Warn()
				} else {
					l.Info()
				}
			}()
		}

		return r, true
	}

	return nil, false
}

func (s *Server) setCache(res *dns.Msg, net string) {
	key := genCacheKey(res, net)

	s.cache.Set(key, cacheEntry{
		putin: time.Now(),
		reply: res,
	})
}

func (s *Server) LookupHosts(req *dns.Msg) (*dns.Msg, bool) {
	// TODO: implement needed
	return nil, false
}
