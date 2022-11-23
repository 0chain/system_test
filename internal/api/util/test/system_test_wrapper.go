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
	wg := sync.WaitGroup{}
	wg.Add(1)

	timeoutWrappedTestCase := func(t *testing.T) {
		t.Helper()
		ws := &SystemTest{T: t}
		defer func() {
			if err := recover(); err != nil {
				t.Errorf("Panic - [%v] occurred while timing out test", err)
			}
		}()

		t.Logf("Test case [%s] start.", name)

		testCaseChannel := make(chan struct{}, 1)

		go executeTest(ws, testFunction, testCaseChannel, &wg)

		select {
		case <-time.After(timeout):
			t.Errorf("Test case [%s] timed out after [%v]", name, timeout)
		case _ = <-testCaseChannel:
		}

		t.Logf("Test case [%s] end.", name)
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

func handleTestCaseExit(s *SystemTest) {
	if err := recover(); err != nil {
		s.Errorf("Test case already exited - [%v]", err)
	}
}

//Boilerplate due to go's very unhelpful lack of polymorphism...

func (s *SystemTest) Cleanup(f func()) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	s.T.Cleanup(f)
}

func (s *SystemTest) Error(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Error(args)
	} else {
		s.T.Error()
	}
}

func (s *SystemTest) Errorf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Errorf(format, args)
	} else {
		s.T.Errorf(format)
	}
}

func (s *SystemTest) Fai() {
	s.T.Helper()
	defer handleTestCaseExit(s)
	s.T.Fail()
}

func (s *SystemTest) FailNow() {
	s.T.Helper()
	defer handleTestCaseExit(s)
	s.T.FailNow()
}

func (s *SystemTest) Failed() bool {
	s.T.Helper()
	defer handleTestCaseExit(s)
	return s.T.Failed()
}

func (s *SystemTest) Fatal(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Fatal(args)
	} else {
		s.T.Fatal()
	}
}

func (s *SystemTest) Fatalf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Fatalf(format, args)
	} else {
		s.T.Fatalf(format)
	}
}

func (s *SystemTest) Log(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Log(args)
	} else {
		s.T.Log()
	}

}

func (s *SystemTest) Logf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Logf(format, args)
	} else {
		s.T.Logf(format)
	}

}

func (s *SystemTest) Name() string {
	s.T.Helper()
	defer handleTestCaseExit(s)
	return s.T.Name()
}

func (s *SystemTest) Setenv(key, value string) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	s.T.Setenv(key, value)

}

func (s *SystemTest) Skip(args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Skip(args)
	} else {
		s.T.Skip()
	}
}

func (s *SystemTest) SkipNow() {
	s.T.Helper()
	defer handleTestCaseExit(s)
	s.T.SkipNow()
}

func (s *SystemTest) Skipf(format string, args ...any) {
	s.T.Helper()
	defer handleTestCaseExit(s)
	if len(args) > 0 {
		s.T.Skipf(format, args)
	} else {
		s.T.Skipf(format)
	}
}

func (s *SystemTest) Skipped() bool {
	s.T.Helper()
	defer handleTestCaseExit(s)
	return s.T.Skipped()
}

func (s *SystemTest) TempDir() string {
	s.T.Helper()
	defer handleTestCaseExit(s)
	return s.T.TempDir()
}
