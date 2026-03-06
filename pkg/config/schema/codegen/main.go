package main

import (
	"fmt"
	"os"
)

func main() {
	if err := Run(); err != nil {
		fmt.Println(err)
		os.Exit(-2)
	}
}
