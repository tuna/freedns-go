package freedns

import (
	"fmt"
	"time"

	"github.com/miekg/dns"
	"github.com/prometheus/common/log"
)

type Config struct {
	FastDNS  string
	CleanDNS string
	Listen   string
}

type Server struct {
	config Config
	s      [2]*dns.Server
}

type Error string

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
	res, err := s.resolve(req)
	if err != nil {
		log.Error(err)
	}
	if res == nil {
		res = new(dns.Msg)
		res.Rcode = dns.RcodeServerFailure
	}

	// SetRcode will set res as reply of req,
	// and also set rcode
	res.SetRcode(req, res.Rcode)

	fmt.Println(res)
	w.WriteMsg(res)
}

func (s *Server) resolve(req *dns.Msg) (*dns.Msg, error) {
	if len(req.Question) < 1 {
		return nil, Error("Empty Question section")
	}

	resChan := make(chan *dns.Msg)

	Q := func(useClean bool) {
		upstream := s.config.FastDNS
		if useClean {
			upstream = s.config.CleanDNS
			time.Sleep(1 * time.Second)
		}

		res, err := resolveBy(req, upstream, "udp")
		if res == nil {
			return
		}

		if !useClean && (maybePolluted(res) || res.Rcode != dns.RcodeSuccess || err != nil) {
			return
		}

		resChan <- res
	}

	go Q(true)
	go Q(false)

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	select {
	case res := <-resChan:
		return res, nil
	case <-ticker.C:
		//timeout
		return nil, Error("No upstream can answer \"" + req.Question[0].Name + "\"")
	}
}

func resolveBy(req *dns.Msg, upstream string, net string) (*dns.Msg, error) {
	r := new(dns.Msg)
	r.Id = dns.Id()
	r.Question = req.Question
	r.RecursionDesired = req.RecursionDesired

	c := &dns.Client{Net: net}

	res, rtt, err := c.Exchange(r, upstream)

	fmt.Println(rtt)
	return res, err
}

func maybePolluted(res *dns.Msg) bool {
	// TODO: implement needed
	return false
}
