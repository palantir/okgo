package fail

fail

/*
This is a non-compiling file that has been added to explicitly ensure that CI fails.
It also contains the command that caused the failure and its output.
Remove this file if debugging locally.

./godelw verify failed after updating godel plugins and assets

Command that caused error:
./godelw lint --fix

Output:
okgo/check/check_test.go:245:6: func toDuration is unused (unused)
func toDuration(timeToWait time.Duration) *time.Duration {
     ^
1 issues:
* unused: 1

*/
