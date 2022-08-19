package main

import (
	"flag"
	"log"
	"time"

	"github.com/go-tftp/tftp"
)

var (
	address = flag.String("a", "127.0.0.1:69", "listen address")
)

func main() {
	flag.Parse()

	s := tftp.NewServer(tftp.WithTimeout(60 * time.Minute))
	log.Fatal(s.ListenAndServer(*address))
}
