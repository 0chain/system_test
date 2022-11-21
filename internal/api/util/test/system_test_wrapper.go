package test

import (
	"log"
	"sync"
	"testing"
	"time"
)

var DefaultTestTimeout = 20 * time.Second

type SystemTest struct {
	*testing.T
}

func (w *SystemTest) Run(name string, testCaseFunction func(t *testing.T)) bool {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic - [%v] occurred while executing test", err)
		}
	}()
	return w.RunWithCustomTimeout(name, DefaultTestTimeout, testCaseFunction)
}

func (w *SystemTest) RunWithCustomTimeout(name string, timeout time.Duration, testFunction func(t *testing.T)) bool {
	wg := sync.WaitGroup{}
	wg.Add(1)

	timeoutWrappedTestCase := func(t *testing.T) {
		defer func() {
			if err := recover(); err != nil {
				t.Errorf("Panic - [%v] occurred while timing out test", err)
			}
		}()

		t.Logf("Test case [%s] start.", name)

		testCaseChannel := make(chan struct{}, 1)

		go executeTest(t, testFunction, testCaseChannel, &wg)

		select {
		case <-time.After(timeout):
			t.Errorf("Test case [%s] timed out after [%v]", name, timeout)
		case _ = <-testCaseChannel:
		}

		t.Logf("Test case [%s] end.", name)
	}

	return w.T.Run(name, timeoutWrappedTestCase)
}

func executeTest(t *testing.T, testFunction func(t *testing.T), testCaseChannel chan struct{}, wg *sync.WaitGroup) {
	go func() {
		defer wg.Done()
		defer handlePanic(t)
		testFunction(t)
	}()
	wg.Wait()
	close(testCaseChannel)
}

func handlePanic(t *testing.T) {
	if err := recover(); err != nil {
		t.Errorf("Test case exited due to panic - [%v]", err)
	}
}
