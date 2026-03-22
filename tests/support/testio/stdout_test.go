package testio

import (
	"fmt"
	"os"
	"testing"
)

func TestCaptureStdout_CapturesOutput(t *testing.T) {
	output := CaptureStdout(t, func() {
		fmt.Print("hello")
		fmt.Println(" world")
	})

	if output != "hello world\n" {
		t.Fatalf("expected captured output %q, got %q", "hello world\n", output)
	}
}

func TestCaptureStdout_RestoresStdout(t *testing.T) {
	original := os.Stdout

	_ = CaptureStdout(t, func() {
		fmt.Print("temporary output")
	})

	if os.Stdout != original {
		t.Fatal("expected os.Stdout to be restored after capture")
	}
}
