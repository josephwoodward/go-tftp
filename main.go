package main

import (
	"flag"
	"github.com/go-tftp/tftp"
	"log"
	"time"
)

var (
	address = flag.String("a", "127.0.0.1:69", "listen address")
)

func main() {

	flag.Parse()

	s := tftp.NewServer(tftp.WithTimeout(10 * time.Second))
	log.Fatal(s.ListenAndServer(*address))

}
