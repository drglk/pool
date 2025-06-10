package pool

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDynamicPool(t *testing.T) {

	tests := []struct {
		Name                  string
		StartWorkersAmount    int
		RemovedWorkers        int
		AddedWorkers          int
		ExpectedWorkersAmount int
	}{
		{
			Name:                  "NumCPU Started ,0 Removed workers, 0 Added",
			StartWorkersAmount:    runtime.NumCPU(),
			RemovedWorkers:        0,
			AddedWorkers:          0,
			ExpectedWorkersAmount: runtime.NumCPU(),
		},
		{
			Name:                  "NumCPU Started, 1 Removed workers, 0 Added",
			StartWorkersAmount:    runtime.NumCPU(),
			RemovedWorkers:        1,
			AddedWorkers:          0,
			ExpectedWorkersAmount: runtime.NumCPU() - 1,
		},
		{
			Name:                  "NumCPU Started, 2 Removed workers, 0 Added",
			StartWorkersAmount:    runtime.NumCPU(),
			RemovedWorkers:        2,
			AddedWorkers:          0,
			ExpectedWorkersAmount: runtime.NumCPU() - 2,
		},
		{
			Name:                  "NumCPU Started, 0 Removed workers, 2 Added",
			StartWorkersAmount:    runtime.NumCPU(),
			RemovedWorkers:        0,
			AddedWorkers:          2,
			ExpectedWorkersAmount: runtime.NumCPU() + 2,
		},
		{
			Name:                  "NumCPU Started, 2 Removed workers, 2 Added",
			StartWorkersAmount:    runtime.NumCPU(),
			RemovedWorkers:        2,
			AddedWorkers:          2,
			ExpectedWorkersAmount: runtime.NumCPU(),
		},
		{
			Name:                  "0 Started, 1 Removed workers, 0 Added",
			StartWorkersAmount:    0,
			RemovedWorkers:        1,
			AddedWorkers:          0,
			ExpectedWorkersAmount: 1,
		},
	}

	for _, test := range tests {
		c := make(chan string)

		pool := New(c, func(id int, s string) {
			time.Sleep(1 * time.Millisecond)
		})

		workersAmount := test.StartWorkersAmount

		pool.Run(workersAmount)

		for i := 0; i < 1000; i++ {

			if i == 10 {
				for j := 0; j < test.RemovedWorkers; j++ {
					if pool.RemoveWorker() {
						workersAmount--
					}
				}
			}

			if i == 11 {
				for j := 0; j < test.AddedWorkers; j++ {
					pool.AddWorker()
					workersAmount++
				}

			}

			c <- fmt.Sprint(i)
		}

		receivedWorkers := pool.WorkersCount()
		close(c)

		pool.Stop()

		if receivedWorkers != test.ExpectedWorkersAmount {
			t.Errorf("test: %s\nvalue: %v\nexpected: %v", test.Name, receivedWorkers, test.ExpectedWorkersAmount)
		}
	}
}

func TestSingleWorker(t *testing.T) {

	tests := []struct {
		Name   string
		Input  []string
		Output []string
	}{
		{
			Name:   "Send nothing",
			Input:  []string{},
			Output: []string{},
		},
		{
			Name:   "Send once",
			Input:  []string{"test1"},
			Output: []string{"1: test1"},
		},
		{
			Name:   "Send double",
			Input:  []string{"test1", "test2"},
			Output: []string{"1: test1", "1: test2"},
		},
	}

	for _, test := range tests {

		input := make(chan string)

		cancelChan := make(chan struct{})

		buf := &bytes.Buffer{}

		var counter atomic.Int64

		worker := &worker{
			id:         1,
			input:      input,
			cancelChan: cancelChan,
			callback: func(id int, s string) {
				buf.Write([]byte(fmt.Sprintf("%v: %s\n", id, s)))

				counter.Add(1)
			},
		}

		wg := &sync.WaitGroup{}

		wg.Add(1)
		go func() {
			worker.work(wg)
		}()

		for _, s := range test.Input {
			input <- s
		}

		close(input)

		wg.Wait()

		out := buf.String()

		outSlice := strings.Split(out, "\n")

		outSlice = outSlice[:len(outSlice)-1]

		if len(outSlice) == len(test.Output) && int(counter.Load()) == len(test.Input) {
			for i, v := range test.Output {
				if outSlice[i] != v {
					t.Errorf("\ntest: %s\nvalue: %s\nexpected: %s", test.Name, outSlice[i], v)
				}
			}
		} else {
			t.Errorf("\ntest: %s\nvalue: %v\nexpected: %v", test.Name, outSlice, test.Output)
		}
	}
}

func TestMultiplyWorkers(t *testing.T) {
	tests := []struct {
		Name   string
		Input  []string
		Output []string
	}{
		{
			Name:   "Send nothing",
			Input:  []string{},
			Output: []string{},
		},
		{
			Name:   "Send once",
			Input:  []string{"test1"},
			Output: []string{"1: test1"},
		},
		{
			Name:   "Send double",
			Input:  []string{"test1", "test2"},
			Output: []string{"1: test1", "1: test2"},
		},
	}

	for _, test := range tests {

		type work struct {
			workerID int
			value    string
		}

		type outputWork struct {
			mu sync.Mutex

			output []work
		}

		out := &outputWork{}

		callback := func(id int, s string) {
			out.mu.Lock()
			defer out.mu.Unlock()

			out.output = append(out.output, work{workerID: id, value: s})
		}

		input := make(chan string)

		cancelChan := make(chan struct{})

		worker1 := &worker{
			id:         1,
			input:      input,
			cancelChan: cancelChan,
			callback:   callback,
		}

		worker2 := &worker{
			id:         2,
			input:      input,
			cancelChan: cancelChan,
			callback:   callback,
		}

		worker3 := &worker{
			id:         3,
			input:      input,
			cancelChan: cancelChan,
			callback:   callback,
		}

		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			worker1.work(wg)
		}()

		wg.Add(1)
		go func() {
			worker2.work(wg)
		}()

		wg.Add(1)
		go func() {
			worker3.work(wg)
		}()

		for _, s := range test.Input {
			input <- s
		}

		close(input)

		wg.Wait()

		m := make(map[string]int)

		func() {
			out.mu.Lock()
			defer out.mu.Unlock()
			for _, v := range out.output {
				m[v.value] = v.workerID
			}

		}()

		for _, v := range test.Input {
			if _, ok := m[v]; !ok {
				t.Errorf("task %s is not handled", v)
			}
		}
	}
}
