package test

import (
	"runtime/debug"
	"testing"
	"time"
)

var DefaultTestTimeout = 20 * time.Second

type SystemTest struct {
	*testing.T
}

type Void struct{}

func executeTest(t *testing.T, testFunction func(t *testing.T)) (ret Void) {
	defer func() {
		if err := recover(); err != nil {
			t.Errorf("Test case exited due to panic - [%v] with stack trace [%v]", err, string(debug.Stack()))
			ret = Void{}
		}
	}()
	testFunction(t)
	return Void{}
}

func (w *SystemTest) RunWithCustomTimeout(name string, timeout time.Duration, f func(t *testing.T)) bool {
	timeoutWrappedTestCase := func(t *testing.T) {
		t.Logf("Test case [%s] start.", name)
		result := make(chan Void, 1)
		go func() {
			result <- executeTest(t, f)
		}()
		select {
		case <-time.After(timeout):
			t.Logf("Test case [%s] timed out after [%v]", name, timeout)
			t.Fatalf("Test case [%s] end.", name)
		case _ = <-result:
		}
		t.Logf("Test case [%s] end.", name)
	}

	return w.T.Run(name, timeoutWrappedTestCase)
}

func (w *SystemTest) Run(name string, testCaseFunction func(t *testing.T)) bool {
	return w.RunWithCustomTimeout(name, DefaultTestTimeout, testCaseFunction)
}
