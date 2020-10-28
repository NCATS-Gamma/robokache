package main

import "github.com/NCATS-Gamma/robokache/internal/robokache"

func main() {

	r := robokache.SetupRouter()
	robokache.AddGUI(r)
	r.Run(":80") // listen and serve on 0.0.0.0:80 (for windows "localhost:80")
}
