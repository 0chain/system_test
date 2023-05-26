package test

import (
	"log"
	"runtime/debug"
	"strings"
	"sync"
	"testing"
	"time"
)

var DefaultTestTimeout = 40 * time.Second
var SmokeTestMode = false

type SystemTest struct {
	Unwrap             *testing.T
	testComplete       bool
	childTest          bool
	runAllTestsAsSmoke bool
	smokeTests         map[string]bool
}

func NewSystemTest(t *testing.T) *SystemTest {
	return &SystemTest{Unwrap: t, testComplete: false, childTest: false}
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
	if timeout < DefaultTestTimeout {
		timeout = DefaultTestTimeout
	}

	return s.run(name, timeout, testCaseFunction, true)
}

func (s *SystemTest) RunSequentiallyWithTimeout(name string, timeout time.Duration, testCaseFunction func(w *SystemTest)) bool {
	s.Unwrap.Helper()
	if timeout < DefaultTestTimeout {
		timeout = DefaultTestTimeout
	}
	return s.run(name, timeout, testCaseFunction, false)
}

func (s *SystemTest) TestSetup(label string, setupFunction func()) {
	s.TestSetupWithTimeout(label, DefaultTestTimeout, setupFunction)
}

func (s *SystemTest) TestSetupWithTimeout(label string, timeout time.Duration, setupFunction func()) {
	s.testSetup(label, timeout, setupFunction)
}

func (s *SystemTest) testSetup(label string, timeout time.Duration, setupFunction func()) {
	s.Unwrap.Helper()
	timeoutWrappedFunction := func() {
		defer handlePanic(s)

		wg := sync.WaitGroup{}
		wg.Add(1)

		s.Logf("Test setup [%s] scheduled at [%s] ", label, time.Now().Format("01-02-2006 15:04:05"))

		testSetupChannel := make(chan struct{}, 1)

		go func() {
			s.Unwrap.Helper()
			defer handlePanic(s)
			go func() {
				defer wg.Done()
				defer handlePanic(s)
				s.Logf("Test setup [%s] start at [%s] ", label, time.Now().Format("01-02-2006 15:04:05"))
				setupFunction()
			}()
			wg.Wait()
			close(testSetupChannel)
		}()

		select {
		case <-time.After(timeout):
			s.Fatalf("Test setup [%s] timed out after [%s]", label, timeout)
		case _ = <-testSetupChannel:
		}

		s.Logf("Test setup [%s] exit at [%s]", label, time.Now().Format("01-02-2006 15:04:05"))
	}
	timeoutWrappedFunction()
}

func (s *SystemTest) run(name string, timeout time.Duration, testFunction func(w *SystemTest), runInParallel bool) bool {
	s.Unwrap.Helper()
	timeoutWrappedTestCase := func(testSetup *testing.T) {
		t := &SystemTest{Unwrap: testSetup, testComplete: false, childTest: true}

		if SmokeTestMode && !s.runAllTestsAsSmoke && !s.smokeTests[name] {
			t.Skip("Test skipped as it is not a smoke test.")
		}

		testSetup.Helper()
		defer handlePanic(t)

		wg := sync.WaitGroup{}
		wg.Add(1)

		t.Logf("Test case [%s] scheduled at [%s] ", name, time.Now().Format("01-02-2006 15:04:05"))

		if SmokeTestMode && !s.runAllTestsAsSmoke && len(s.smokeTests) < 1 {
			t.Fatal("No smoke tests were defined for this test file.")
		}

		testCaseChannel := make(chan struct{}, 1)

		if runInParallel {
			if !s.childTest {
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
		s.Errorf("Test case exited due to panic - [%v], stack: [%v]", err, string(debug.Stack()))
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

func (s *SystemTest) Fail() {
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

func (s *SystemTest) EscapedName() string {
	s.Unwrap.Helper()
	defer handleTestCaseExit()
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(s.Unwrap.Name())
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

func (s *SystemTest) SetRunAllTestsAsSmokeTest() {
	s.runAllTestsAsSmoke = true
}

func (s *SystemTest) SetSmokeTests(smokeTests ...string) {
	s.smokeTests = make(map[string]bool)
	for _, v := range smokeTests {
		s.smokeTests[v] = true
	}
}
