package collector

import "testing"

func BenchmarkGetMemory(b *testing.B) {
	// Need a way to mock runVCGenCmd to just test the string parsing, but let's see how it works first.
	// Oh, `runVCGenCmd` calls `exec.Command("vcgencmd", ...)` which is actually using a stub in the tests!
	// Wait, the test uses the stub because of the Makefile putting bin/ in the PATH.
	for i := 0; i < b.N; i++ {
		GetMemory("arm")
	}
}
