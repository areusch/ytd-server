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

	coord := NewDownloadCoordinator()
	http.Handle("/dl", NewDownloadHandler(*basePath, coord))
//	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(*basePath))))
	http.Handle("/", http.FileServer(http.Dir("fs/")))
	if e := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); e != nil {
		fmt.Printf("Cannot listen! Error: %s\n", e.String())
	}
}
