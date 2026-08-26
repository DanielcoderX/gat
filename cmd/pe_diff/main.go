package main

import (
	"fmt"
	"os"
)

func main() {
	f, _ := os.ReadFile("bin/gatc-v2.exe")
	fmt.Printf("Disassembly of bb_write_u32 (0x1450 to 0x1700):\n")
	for i := 0x1450; i < 0x1700; i += 16 {
		fmt.Printf("%04X: ", i)
		for j := 0; j < 16 && i+j < len(f); j++ {
			fmt.Printf("%02X ", f[i+j])
		}
		fmt.Println()
	}
}
