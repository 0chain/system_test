package test

import (
	"log"
	"sync"
	"testing"
	"time"
)

var DefaultTestTimeout = 20 * time.Second

type SystemTest struct {
	Unwrap       *testing.T
	testComplete bool
	isChildTest  bool
}

func NewSystemTest(t *testing.T) *SystemTest {
	return &SystemTest{Unwrap: t, testComplete: false, isChildTest: false}
}

func (s *SystemTest) Run(name string, testCaseFunction func(w *SystemTest)) bool {
	s.Unwrap.Helper()
	return s.RunWithTimeout(name, DefaultTestTimeout, testCaseFunction)
}

func (s *SystemTest) RunSequentially(name string, testCaseFunction func(w *SystemTest)) bool {
	s.Unwrap.Helper()
	return s.RunSequentiallyWithTimeout(name, DefaultTestTimeout, testCaseFunction)
}

func (s *SystemTest) RunWithTimeout(name string, timeout time.Duration, testCaseFunction func(w *SystemTest)) bool {
	s.Unwrap.Helper()
	return s.run(name, timeout, testCaseFunction, true)
}

func (s *SystemTest) RunSequentiallyWithTimeout(name string, timeout time.Duration, testCaseFunction func(w *SystemTest)) bool {
	s.Unwrap.Helper()
	return s.run(name, timeout, testCaseFunction, false)
}

func (s *SystemTest) run(name string, timeout time.Duration, testFunction func(w *SystemTest), runInParallel bool) bool {
	s.Unwrap.Helper()
	timeoutWrappedTestCase := func(testSetup *testing.T) {
		t := &SystemTest{Unwrap: testSetup, testComplete: false, isChildTest: true}
		testSetup.Helper()
		defer handlePanic(t)

		wg := sync.WaitGroup{}
		wg.Add(1)

		t.Logf("Test case [%s] scheduled at [%s] ", name, time.Now().Format("01-02-2006 15:04:05"))

		testCaseChannel := make(chan struct{}, 1)

		if runInParallel {
			if !s.isChildTest {
				t.Parallel()
			} else {
				t.Logf("[WARN] Not running test case [%s] in parallel as it is a child test. Use t.Unwrap.run() then t.Parallel() if you wish to do this.", name)
			}
		}
		go executeTest(t, name, testFunction, testCaseChannel, &wg)

		select {
		case <-time.After(timeout):
			t.Errorf("Test case [%s] timed out after [%s]", name, timeout)
		case _ = <-testCaseChannel:
		}

		t.Logf("Test case [%s] exit at [%s]", name, time.Now().Format("01-02-2006 15:04:05"))
		t.testComplete = true
	}

	return s.Unwrap.Run(name, timeoutWrappedTestCase)
}

func executeTest(s *SystemTest, name string, testFunction func(w *SystemTest), testCaseChannel chan struct{}, wg *sync.WaitGroup) {
	s.Unwrap.Helper()
	defer handlePanic(s)
	go func() {
		defer wg.Done()
		defer handlePanic(s)
		s.Logf("Test case [%s] start at [%s] ", name, time.Now().Format("01-02-2006 15:04:05"))
		testFunction(s)
	}()
	wg.Wait()
	close(testCaseChannel)
}

func handlePanic(s *SystemTest) {
	if err := recover(); err != nil {
		s.Errorf("Test case exited due to panic - [%v]", err)
	}
}

func handleTestCaseExit() {
	if err := recover(); err != nil {
		log.Printf("Suppressed test function panic - [%v]", err)
	}
}

// Boilerplate due to go's very unhelpful lack of polymorphism...

func (s *SystemTest) Cleanup(f func()) {
	s.Unwrap.Helper()
	defer handleTestCaseExit()
	s.Unwrap.Cleanup(f)
}

func (s *SystemTest) Error(args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Error(args...)
	}
}

func (s *SystemTest) Errorf(format string, args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Errorf(format, args...)
	}
}

func (s *SystemTest) Fai() {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Fail()
	}
}

func (s *SystemTest) FailNow() {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.FailNow()
	}
}

func (s *SystemTest) Failed() bool {
	s.Unwrap.Helper()
	defer handleTestCaseExit()
	return s.Unwrap.Failed()
}

func (s *SystemTest) Fatal(args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Fatal(args...)
	}
}

func (s *SystemTest) Fatalf(format string, args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Fatalf(format, args...)
	}
}

func (s *SystemTest) Log(args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Log(args...)
	}
}

func (s *SystemTest) Logf(format string, args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Logf(format, args...)
	}
}

func (s *SystemTest) Name() string {
	s.Unwrap.Helper()
	defer handleTestCaseExit()
	return s.Unwrap.Name()
}

func (s *SystemTest) Setenv(key, value string) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Setenv(key, value)
	}
}

func (s *SystemTest) Skip(args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Skip(args...)
	}
}

func (s *SystemTest) SkipNow() {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.SkipNow()
	}
}

func (s *SystemTest) Skipf(format string, args ...any) {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Skipf(format, args...)
	}
}

func (s *SystemTest) Skipped() bool {
	s.Unwrap.Helper()
	defer handleTestCaseExit()
	return s.Unwrap.Skipped()
}

func (s *SystemTest) TempDir() string {
	s.Unwrap.Helper()
	defer handleTestCaseExit()
	return s.Unwrap.TempDir()
}

func (s *SystemTest) Parallel() {
	if !s.testComplete {
		s.Unwrap.Helper()
		defer handleTestCaseExit()
		s.Unwrap.Parallel()
	}
}
