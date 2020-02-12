package services

import (
	"sync"
	"testing"
	"time"

	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
)

type testWorker struct {
	output chan struct{}
}

func (w *testWorker) Work() {
	w.output <- struct{}{}
}

type sleepyWorker struct {
	output chan struct{}
}

func (w *sleepyWorker) Work() {
	time.Sleep(time.Second)
	w.output <- struct{}{}
}

func (w *sleepyWorker) Output() struct{} {
	return <-w.output
}

func TestSleeperTask(t *testing.T) {
	worker := testWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.Start()
	sleeper.WakeUp()

	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()
}

func TestSleeperTask_WakeupBeforeStarted(t *testing.T) {
	worker := testWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.WakeUp()
	sleeper.Start()

	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()
}

func TestSleeperTask_Restart(t *testing.T) {
	worker := testWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.Start()
	sleeper.WakeUp()

	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()

	sleeper.Start()
	sleeper.WakeUp()

	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()
}

func TestSleeperTask_SenderNotBlockedWhileWorking(t *testing.T) {
	worker := testWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.Start()

	sleeper.WakeUp()
	sleeper.WakeUp()

	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()
}

func TestSleeperTask_StopWaitsUntilWorkFinishes(t *testing.T) {
	worker := sleepyWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.Start()
	sleeper.WakeUp()
	sleeper.Stop()

	assert.Equal(t, worker.Output(), struct{}{})
}

func TestSleeperTask_StopWithoutStartNonBlocking(t *testing.T) {
	worker := testWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.Start()
	sleeper.WakeUp()
	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()
	sleeper.Stop()
}

type slowWorker struct {
	mutex  sync.Mutex
	output chan struct{}
}

func (t *slowWorker) Work() {
	t.output <- struct{}{}
	t.mutex.Lock()
	t.mutex.Unlock()
}

func TestSleeperTask_WakeWhileWorkingRepeatsWork(t *testing.T) {
	worker := slowWorker{output: make(chan struct{})}
	sleeper := NewSleeperTask(&worker)

	sleeper.Start()

	// Lock the worker's mutex so it's blocked *after* sending to the output
	// channel, this guarantees that the worker blocks till we unlock the mutex
	worker.mutex.Lock()
	sleeper.WakeUp()
	// Make sure an item is received in the channel so we know the worker is blocking
	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	// Wake up the sleeper
	sleeper.WakeUp()
	// Now release the worker
	worker.mutex.Unlock()
	gomega.NewGomegaWithT(t).Eventually(worker.output).Should(gomega.Receive(&struct{}{}))

	sleeper.Stop()
}
