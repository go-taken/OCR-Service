package main

import (
	"fmt"
	"log"

	"app/internal/server"
)

func main() {

	mem := 50 << 20
	fmt.Println(mem)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
