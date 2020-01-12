package freedns

import (
	"strings"

	goc "github.com/louchenyao/golang-cache"
	"github.com/miekg/dns"
	"github.com/sirupsen/logrus"
)

// Config stores the configuration for the Server
type Config struct {
	FastDNS  string
	CleanDNS string
	Listen   string
	CacheCap int // the maximum items can be cached
}

// Server is type of the freedns server instance
type Server struct {
	config Config

	udpServer *dns.Server
	tcpServer *dns.Server

	chinaDom     *goc.Cache
	recordsCache *dnsCache
}

var log = logrus.New()

// Error is the freedns error type
type Error string

func (e Error) Error() string {
	return string(e)
}

// append the 53 port number after the ip, if the ip does not has ip infomation.
// It only works for IPv4 addresses, since it's a little hard to check if a port
// is in the IPv6 string representation.
func appendDefaultPort(ip string) string {
	if strings.Contains(ip, ".") && !strings.Contains(ip, ":") {
		return ip + ":53"
	}
	return ip
}

// NewServer creates a new freedns server instance.
func NewServer(cfg Config) (*Server, error) {
	s := &Server{}

	if cfg.Listen == "" {
		cfg.Listen = "127.0.0.1"
	}
	cfg.Listen = appendDefaultPort(cfg.Listen)
	cfg.FastDNS = appendDefaultPort(cfg.FastDNS)
	cfg.CleanDNS = appendDefaultPort(cfg.CleanDNS)
	s.config = cfg

	s.udpServer = &dns.Server{
		Addr: s.config.Listen,
		Net:  "udp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			s.handle(w, req, "udp")
		}),
	}

	s.tcpServer = &dns.Server{
		Addr: s.config.Listen,
		Net:  "tcp",
		Handler: dns.HandlerFunc(func(w dns.ResponseWriter, req *dns.Msg) {
			s.handle(w, req, "tcp")
		}),
	}

	s.recordsCache = newDNSCache(cfg.CacheCap)

	return s, nil
}

// Run tcp and udp server.
func (s *Server) Run() error {
	errChan := make(chan error, 2)

	go func() {
		err := s.tcpServer.ListenAndServe()
		errChan <- err
	}()

	go func() {
		err := s.udpServer.ListenAndServe()
		errChan <- err
	}()

	select {
	case err := <-errChan:
		s.tcpServer.Shutdown()
		s.udpServer.Shutdown()
		return err
	}
}

// Shutdown shuts down the freedns server
func (s *Server) Shutdown() {
	s.tcpServer.Shutdown()
	s.udpServer.Shutdown()
}

func (s *Server) handle(w dns.ResponseWriter, req *dns.Msg, net string) {
	res := &dns.Msg{}

	if len(req.Question) < 1 {
		res.SetRcode(req, dns.RcodeBadName)
		w.WriteMsg(res)
		log.WithFields(logrus.Fields{
			"op":  "handle",
			"msg": "request without questions",
		}).Warn()
		return
	}

	res, upstream := s.lookup(req.Question[0], net)
	res.SetRcode(req, res.Rcode)
	w.WriteMsg(res)

	// logging
	l := log.WithFields(logrus.Fields{
		"op":       "handle",
		"domain":   req.Question[0].Name,
		"type":     dns.TypeToString[req.Question[0].Qtype],
		"upstream": upstream,
		"status":   dns.RcodeToString[res.Rcode],
	})
	if res.Rcode == dns.RcodeSuccess {
		l.Info()
	} else {
		l.Warn()
	}
}

// lookup queries the dns request `q` on either the local cache or upstreams,
// and returns the result and which upstream is used. It updates the local cache
// if necessary.
func (s *Server) lookup(q dns.Question, net string) (*dns.Msg, string) {
	return nil, ""
}
