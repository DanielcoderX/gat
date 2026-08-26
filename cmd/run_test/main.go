package main

import (
	"fmt"
	"os/exec"
)

func main() {
	cmd := exec.Command("bin/hello.exe")
	out, err := cmd.CombinedOutput()
	fmt.Printf("Output: %q\n", string(out))
	fmt.Printf("Error: %v\n", err)
	if exitErr, ok := err.(*exec.ExitError); ok {
		fmt.Printf("Exit code: %d (0x%X)\n", exitErr.ExitCode(), uint32(exitErr.ExitCode()))
	}
}
