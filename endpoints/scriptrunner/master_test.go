package scriptrunner

import (
	"bytes"
	"errors"
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

func TestRunScriptDoesNotWaitForBackgroundChildHoldingStdout(t *testing.T) {
	var got bytes.Buffer
	start := time.Now()
	runScript("sh", []string{"-c", "echo done; (sleep 5; echo late) &"}, strings.NewReader(""), &got, "test-uuid")
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("runScript took %v; it waited for the background child", elapsed)
	}
	if got.String() != "done\n" {
		t.Fatalf("got %q, want %q", got.String(), "done\n")
	}
}

type slowWriter struct{ bytes.Buffer }

func (w *slowWriter) Write(p []byte) (int, error) {
	time.Sleep(20 * time.Millisecond)
	return w.Buffer.Write(p)
}

func TestRunScriptReturnsAllStdoutToSlowWriter(t *testing.T) {
	var want strings.Builder
	for i := 0; i < 20000; i++ {
		fmt.Fprintf(&want, "line %d\n", i)
	}
	var got slowWriter
	runScript("sh", []string{"-c", "i=0; while [ $i -lt 20000 ]; do echo \"line $i\"; i=$((i+1)); done"}, strings.NewReader(""), &got, "test-uuid")
	if got.String() != want.String() {
		t.Fatalf("got %d bytes of stdout, want %d", got.Len(), want.Len())
	}
}

type cappedWriter struct{ bytes.Buffer }

func (w *cappedWriter) Write(p []byte) (int, error) {
	if w.Len() > 8<<20 {
		return 0, errors.New("runScript kept draining past 8 MiB")
	}
	return w.Buffer.Write(p)
}

func TestRunScriptStopsDrainingBackgroundChildThatKeepsWriting(t *testing.T) {
	var got cappedWriter
	start := time.Now()
	runScript("sh", []string{"-c", "echo done; (yes late &)"}, strings.NewReader(""), &got, "test-uuid")
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("runScript took %v; it kept draining the background child", elapsed)
	}
	if got.Len() > 8<<20 {
		t.Fatalf("runScript drained %d bytes from the background child", got.Len())
	}
	if !strings.HasPrefix(got.String(), "done\n") {
		t.Fatalf("output does not start with the script's own stdout: %.20q", got.String())
	}
}

func TestRunScriptStopsDrainingBackgroundChildThatTrickles(t *testing.T) {
	var got cappedWriter
	start := time.Now()
	runScript("sh", []string{"-c", "echo done; (while true; do echo late; sleep 0.05; done &)"}, strings.NewReader(""), &got, "test-uuid")
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("runScript took %v; it kept draining a trickling background child", elapsed)
	}
	if !strings.HasPrefix(got.String(), "done\n") {
		t.Fatalf("output does not start with the script's own stdout: %.20q", got.String())
	}
}
