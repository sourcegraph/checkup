package checkup

import (
	"testing"
)

func TestExecChecker(t *testing.T) {

	assert := func(ok bool, format string, args ...interface{}) {
		if !ok {
			t.Fatalf(format, args...)
		}
	}

	command := "testdata/exec.sh"

	// check non-zero exit code
	{
		testName := "Non-zero exit"
		hc := ExecChecker{Name: testName, Command: command, Arguments: []string{"1", testName}, Attempts: 2}

		result, err := hc.Check()
		assert(err == nil, "expected no error, got %v, %#v", err, result)
		assert(result.Title == testName, "expected result.Title == %s, got %s", testName, result.Title)
		assert(result.Down == true, "expected result.Down = true, got %v", result.Down)
	}

	// check zero exit code
	{
		testName := "Non-zero exit"
		hc := ExecChecker{Name: testName, Command: command, Arguments: []string{"0", testName}, Attempts: 2}

		result, err := hc.Check()
		t.Logf("%#v", result)
		assert(err == nil, "expected no error, got %v, %#v", err, result)
		assert(result.Title == testName, "expected result.Title == %s, got %s", testName, result.Title)
		assert(result.Down == false, "expected result.Down = false, got %v", result.Down)
	}
}
