// Command extract exports decoded sprites and native definition tables.
package main

import (
	"flag"
	"github.com/olivierh59500/go-turrican32/internal/extract"
	"log"
	"os"
)

func main() {
	input := flag.String("input", "", "unpacked Windows executable")
	output := flag.String("output", "assets", "asset directory")
	flag.Parse()
	b, e := os.ReadFile(*input)
	if e != nil {
		log.Fatal(e)
	}
	if e = extract.ExportImage(b, *output); e != nil {
		log.Fatal(e)
	}
}
