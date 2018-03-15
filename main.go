package main

import (
	"log"
	"os"

	"github.com/Chenyao2333/freedns-go/freedns"
)

func main() {
	s, err := freedns.NewServer(freedns.Config{
		FastDNS:   "114.114.114.114:53",
		CleanDNS:  "8.8.8.8:53",
		Listen:    "127.0.0.1:53",
		CacheSize: 1024 * 2,
	})
	if err != nil {
		log.Fatalln(err)
		os.Exit(-1)
	}

	log.Fatalln(s.Run())
	os.Exit(-1)
}
