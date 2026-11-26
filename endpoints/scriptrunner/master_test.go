package scriptrunner

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

type countingFlusher struct{ flushes int }

func (f *countingFlusher) Flush() { f.flushes++ }

func TestStartFlushingOutputStopsWithoutWaitingForTick(t *testing.T) {
	done := make(chan bool)
	go startFlushingOutput(done, &countingFlusher{})
	time.Sleep(50 * time.Millisecond)
	start := time.Now()
	done <- true
	if elapsed := time.Since(start); elapsed > 200*time.Millisecond {
		t.Fatalf("done took %v to be received; want well under the 1s flush interval", elapsed)
	}
}

func TestRunScriptReturnsAllStdout(t *testing.T) {
	var want strings.Builder
	for i := 0; i < 20000; i++ {
		fmt.Fprintf(&want, "line %d\n", i)
	}
	for run := 0; run < 20; run++ {
		var got bytes.Buffer
		runScript("sh", []string{"-c", "i=0; while [ $i -lt 20000 ]; do echo \"line $i\"; i=$((i+1)); done"}, strings.NewReader(""), &got, "test-uuid")
		if got.String() != want.String() {
			t.Fatalf("run %d: got %d bytes of stdout, want %d", run, got.Len(), want.Len())
		}
	}
}
