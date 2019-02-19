package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"

	"github.com/google/prog-edu-assistant/server"
)

var (
	port      = flag.Int("port", 8000, "The port to serve HTTP.")
	uploadDir = flag.String("upload_dir", "uploads", "The directory to write uploaded notebooks.")
)

func main() {
	flag.Parse()
	err := run()
	if err != nil {
		log.Fatal(err)
	}
}

func run() error {
	s := server.New(server.Options{
		UploadDir: *uploadDir,
	})
	addr := ":" + strconv.Itoa(*port)
	fmt.Printf("\n  Serving on http://localhost%s\n\n", addr)
	return s.ListenAndServe(addr)
}
