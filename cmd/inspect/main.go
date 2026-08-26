package main

import (
	"fmt"
	"os"
)

func main() {
	f, err := os.Open("bin/gatc-v2.exe")
	if err != nil {
		fmt.Printf("Open err: %v\n", err)
		return
	}
	defer f.Close()
	b, _ := os.ReadFile("bin/gatc-v2.exe")
	targetOff := 0x165EA - 32
	fmt.Printf("Bytes around 0x165EA (inside ir_build_expr):\n")
	for i := 0x400 + targetOff; i < 0x400+targetOff+64; i++ {
		fmt.Printf("%02X ", b[i])
		if (i-0x400-targetOff+1)%16 == 0 {
			fmt.Println()
		}
	}
	fmt.Println()
}
