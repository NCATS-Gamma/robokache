package main

import "github.com/NCATS-Gamma/robokache/internal/robokache"

func main() {
	robokache.SetupHashids()
	robokache.SetupDB()

	r := robokache.SetupRouter()
	robokache.AddGUI(r)
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}
