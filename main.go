package main

import(
	"flag"
	"fmt"
	"http"
)

var help *bool = flag.Bool("help", false, "Display this message")

var basePath *string = flag.String("output_base", ".", "Base directory to write")
var port *int = flag.Int("port", 6565, "Port to listen on")

func main() {
	flag.Parse()
	if *help {
		flag.PrintDefaults()
		return
	}

	http.Handle("/dl", NewDownloadHandler(*basePath))
	http.Handle("/", http.FileServer(http.Dir("fs/")))
	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
	fmt.Printf("Now serving on port %d", *port)
}
