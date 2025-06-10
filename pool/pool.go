package pool

import (
	"container/list"
	"sync"
)

// Pool is a set of goroutines, where the number of concurrent
// goroutines processing requests does not exceed the specified.
type Pool struct {
	currentWorkerID uint
	input           <-chan string
	running         bool
	workers         list.List
	mu              sync.Mutex
	callback        func(id int, s string)
}

// New creates a pool of workers.
func New(input <-chan string, callback func(id int, s string)) *Pool {
	return &Pool{input: input, currentWorkerID: 1, callback: callback}
}

// Run starts a pool of workers with specified amount.
func (p *Pool) Run(workerAmount int) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if workerAmount < 1 {
		workerAmount = 1
	}

	if p.running {
		return
	}
	for i := 1; i <= workerAmount; i++ {
		p.addWorker()
	}

	p.running = true
}

// Stop stops a pool of workers.
func (p *Pool) Stop() {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.running {
		return
	}

	for e := p.workers.Back(); e != nil; {
		w := e.Value.(*worker)

		if w.cancelChanClosed {
			w.cancelChan <- struct{}{}
		}

		prev := e
		e = e.Prev()
		p.workers.Remove(prev)
	}

	p.running = false
}

// AddWorker adding worker into a pool.
func (p *Pool) AddWorker() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.addWorker()
}

// RemoveWorker removing worker from pool.
func (p *Pool) RemoveWorker() bool {

	var w *worker

	ok := func() bool {
		p.mu.Lock()
		defer p.mu.Unlock()

		if p.workers.Len() == 1 {
			return false
		}

		backElement := p.workers.Back()

		ok1 := true

		w, ok1 = backElement.Value.(*worker)
		if !ok1 {
			return false
		}

		p.workers.Remove(backElement)

		return true
	}()

	if !ok {
		return false
	}

	w.cancelChan <- struct{}{}

	return true
}

// WorkersCount returns count of current workers.
func (p *Pool) WorkersCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.workers.Len()
}

func (p *Pool) addWorker() {
	worker := newWorker(p.input, int(p.currentWorkerID), p.callback)

	p.currentWorkerID++

	p.workers.PushBack(worker)
}

type worker struct {
	id         int
	input      <-chan string
	cancelChan chan struct{}
	callback   func(id int, s string)

	mu               sync.Mutex
	cancelChanClosed bool
}

func newWorker(input <-chan string, id int, callback func(id int, s string)) *worker {

	c := make(chan struct{})

	worker := &worker{input: input, id: id, callback: callback, cancelChan: c}

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go worker.work(wg)

	go func() {
		wg.Wait()

		worker.mu.Lock()
		defer worker.mu.Unlock()
		close(c)

		worker.cancelChanClosed = true
	}()

	return worker
}

func (w *worker) work(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case s, ok := <-w.input:
			if !ok {
				return
			}
			w.callback(w.id, s)
		case <-w.cancelChan:
			return
		}
	}
}
