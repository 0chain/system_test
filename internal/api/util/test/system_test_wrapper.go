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

func NewSystemTest(t *testing.T) *SystemTest {
	return &SystemTest{T: t}
}

func (s *SystemTest) Run(name string, testCaseFunction func(w *SystemTest)) bool {
	s.T.Helper()
	defer func() {
		if err := recover(); err != nil {
			log.Printf("Panic - [%v] occurred while executing test", err)
		}
	}()
	return s.RunWithCustomTimeout(name, DefaultTestTimeout, testCaseFunction)
}

func (s *SystemTest) RunWithCustomTimeout(name string, timeout time.Duration, testFunction func(w *SystemTest)) bool {
	s.T.Helper()
	timeoutWrappedTestCase := func(t *testing.T) {
		t.Helper()
		wg := sync.WaitGroup{}
		wg.Add(1)
		ws := &SystemTest{T: t}
		defer func() {
			if err := recover(); err != nil {
				ws.Errorf("Panic - [%v] occurred while timing out test", err)
			}
		}()

		dt := time.Now()
		ws.Logf("Test case [%s] start at [%s] ", name, dt.Format("01-02-2006 15:04:05"))

		testCaseChannel := make(chan struct{}, 1)

		go executeTest(ws, testFunction, testCaseChannel, &wg)

		select {
		case <-time.After(timeout):
			dt = time.Now()
			ws.Errorf("Test case [%s] timed out after [%s] at [%s]", name, timeout, dt.Format("01-02-2006 15:04:05"))
		case _ = <-testCaseChannel:
		}

		dt = time.Now()
		ws.Logf("Test case [%s] end at ", name, dt.Format("01-02-2006 15:04:05"))
	}

	return s.T.Run(name, timeoutWrappedTestCase)
}

func executeTest(s *SystemTest, testFunction func(w *SystemTest), testCaseChannel chan struct{}, wg *sync.WaitGroup) {
	s.Helper()
	go func() {
		defer wg.Done()
		defer handlePanic(s)
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
		log.Printf("Test case already exited - [%v]", err)
	}
}

//Boilerplate due to go's very unhelpful lack of polymorphism...

func (s *SystemTest) Cleanup(f func()) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Cleanup(f)
}

func (s *SystemTest) Error(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Error(args...)
}

func (s *SystemTest) Errorf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Errorf(format, args...)
}

func (s *SystemTest) Fai() {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Fail()
}

func (s *SystemTest) FailNow() {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.FailNow()
}

func (s *SystemTest) Failed() bool {
	s.T.Helper()
	defer handleTestCaseExit()
	return s.T.Failed()
}

func (s *SystemTest) Fatal(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Fatal(args...)
}

func (s *SystemTest) Fatalf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Fatalf(format, args...)
}

func (s *SystemTest) Log(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Log(args...)
}

func (s *SystemTest) Logf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Logf(format, args...)
}

func (s *SystemTest) Name() string {
	s.T.Helper()
	defer handleTestCaseExit()
	return s.T.Name()
}

func (s *SystemTest) Setenv(key, value string) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Setenv(key, value)

}

func (s *SystemTest) Skip(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Skip(args...)
}

func (s *SystemTest) SkipNow() {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.SkipNow()
}

func (s *SystemTest) Skipf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit()
	s.T.Skipf(format, args...)
}

func (s *SystemTest) Skipped() bool {
	s.T.Helper()
	defer handleTestCaseExit()
	return s.T.Skipped()
}

func (s *SystemTest) TempDir() string {
	s.T.Helper()
	defer handleTestCaseExit()
	return s.T.TempDir()
}
