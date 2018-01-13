package freedns

import "github.com/miekg/dns"

type Config struct {
	FastDNS  string
	CleanDNS string
	IP       string
	Port     int
}

type Server struct {
	config Config
	s      *dns.Server
}

type Error string

func (e Error) Error() string {
	return string(e)
}

func NewServer(cfg Config) (*Server, error) {
	s := &Server{}

	if cfg.IP == "" {
		cfg.IP = "127.0.0.1"
	}
	if cfg.Port == 0 {
		cfg.Port = 53
	}
	s.config = cfg

	s.s = &dns.Server{
		Addr: cfg.IP + ":" + string(cfg.Port),
		Net:  "udp",
	}

	dns.HandleFunc(".", handle)

	return s, nil
}

func (s *Server) Run() {
	s.s.ListenAndServe()
}

func handle(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)

	//TODO

	w.WriteMsg(m)
}
