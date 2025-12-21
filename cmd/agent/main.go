package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	loc = mustLoadLocation("Asia/Seoul")
)
var counter atomic.Int64

func mustLoadLocation(src string) *time.Location {
	l, err := time.LoadLocation(src)
	if err != nil {
		panic(err)
	}
	return l
}

func worker() {
	v := counter.Add(1)
	fmt.Printf("%-6d ", v)

	fmt.Println(time.Now().In(loc).Format(time.RFC3339Nano))
}

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	config := flag.String("config", "", "config file path")
	once := flag.Bool("once", false, "run once")

	flag.Parse()

	fmt.Println("config: ", *config)
	fmt.Println("once: ", *once)

	fmt.Print("Agent Start.\n")

	if *once {
		worker()
		fmt.Println("Agent Stop.")
		return
	}

	for {
		select {
		case sig := <-sigCh:
			fmt.Printf("received: %v\n", sig)
			fmt.Println("Agent Stop.")
			return
		case <-ticker.C:
			worker()
		}
	}
}
