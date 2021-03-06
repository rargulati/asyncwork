package worker

import "sync"

// There is my implementation of the "pipeling". The original idea is described in
// the "Go Blog": https://blog.golang.org/pipelines

// TaskFunction is a function type for tasks to be performed.
// All incoming tasks have to conform to this function type.
type TaskFunction func() interface{}

// PerformTasks is a function which will be called by the client to perform
// multiple task concurrently.
// Input:
// tasks: the slice with functions (type TaskFunction)
// done:  the channel to trigger the end of task processing and return
// Output: the channel with results
func PerformTasks(tasks []TaskFunction, done chan struct{}) chan interface{} {

	// Create a worker for each incoming task
	workers := make([]chan interface{}, 0, len(tasks))

	for _, task := range tasks {
		resultChannel := newWorker(task, done)
		workers = append(workers, resultChannel)
	}

	// Merge results from all workers
	out := merge(workers, done)
	return out
}

func newWorker(task TaskFunction, done chan struct{}) chan interface{} {
	out := make(chan interface{})
	go func() {
		defer close(out)

		select {
		case <-done:
			// Received a signal to abandon further processing
			return
		case out <- task():
			// Got some result
		}
	}()

	return out
}

func merge(workers []chan interface{}, done chan struct{}) chan interface{} {
	// Merged channel with results
	out := make(chan interface{})

	// Synchronization over channels: do not close "out" before all tasks are completed
	var wg sync.WaitGroup

	// Define function which waits the result from worker channel
	// and sends this result to the merged channel.
	// Then it decreases the counter of running tasks via wg.Done().
	output := func(c <-chan interface{}) {
		defer wg.Done()
		for result := range c {
			select {
			case <-done:
				// Received a signal to abandon further processing
				return
			case out <- result:
				// some message or nothing
			}
		}
	}

	wg.Add(len(workers))
	for _, workerChannel := range workers {
		go output(workerChannel)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
