package main

import (
	"fmt"
	"runtime"
	"vkintern/pool"
)

func main() {
	c := make(chan string)

	pool := pool.New(c, func(id int, s string) {
		fmt.Printf("%v: %s\n", id, s)
	})

	pool.Run(runtime.NumCPU())

	for i := 0; i < 100; i++ {

		if i == 10 {
			for j := 0; j < 2; j++ {
				pool.RemoveWorker()
			}
		}

		if i == 11 {
			for j := 0; j < 3; j++ {
				pool.AddWorker()
			}
		}
		c <- fmt.Sprint(i)
	}

	close(c)

	pool.Stop()
}
