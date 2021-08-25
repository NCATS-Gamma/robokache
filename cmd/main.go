package main

import "github.com/NCATS-Gamma/robokache/internal/robokache"

func main() {

	r := robokache.SetupRouter()
	robokache.AddGUI(r)
	r.Run(":8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
