// rssreader manages feeds' locations and show items for a particular feed.

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
)

func main() {
	url := flag.String("a", "", "add a feed's URL to fetch")
	list := flag.Bool("l", false, "show a list of current feeds")
	show := flag.String("s", "", "show a list of items for a feed with the specified index")

	cpuprof := flag.String("cpu", "", "write cpu profile to the file")
	memprof := flag.String("mem", "", "write memory profile to the file")

	flag.Parse()

	fdb := FileDatabase{
		Location: fdbLocation,
		Urls:     []string{},
	}

	var err error

	// pprof

	if *cpuprof != "" {
		f, err := os.Create(*cpuprof)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err = pprof.StartCPUProfile(f); err != nil {
			log.Fatal(err)
		}
		defer pprof.StopCPUProfile()
	}

	if *memprof != "" {
		f, err := os.Create(*memprof)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		if err = pprof.WriteHeapProfile(f); err != nil {
			log.Fatal(err)
		}
	}

	// app logic

	if err = fdb.init(); err != nil {
		fmt.Println(err)
	}

	if len(*url) > 0 {
		if err = fdb.save(*url); err != nil {
			fmt.Println(err)
		}
	}

	if *list {
		if err = fdb.list(); err != nil {
			fmt.Println(err)
		}
	}

	if len(*show) > 0 {
		if err = fdb.listAt(*show); err != nil {
			fmt.Println(err)
		}
	}
}
