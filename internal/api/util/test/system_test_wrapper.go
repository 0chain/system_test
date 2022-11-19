package test

import (
	"runtime/debug"
	"testing"
	"time"
)

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
		result := make(chan Void, 1)
		go func() {
			result <- executeTest(t, f)
		}()
		select {
		case <-time.After(timeout):
			t.Fatalf("Test case timed out after [%v]", timeout)
		case _ = <-result:
			t.Logf("Test case completed successfully")
		}
	}

	return w.T.Run(name, timeoutWrappedTestCase)
}

func (w *SystemTest) Run(name string, testCaseFunction func(t *testing.T)) bool {
	defaultTimeout := 60 * time.Second

	return w.RunWithCustomTimeout(name, defaultTimeout, testCaseFunction)
}
