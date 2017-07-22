package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"sync"
)

type count struct {
	Path  string
	Chars int
	Words int
	Lines int
}

func fileBytes(path string) chan byte {
	c := make(chan byte)
	go func() {
		dat, err := ioutil.ReadFile(path)
		if err != nil {
			panic(err)
		}
		for _, b := range dat {
			c <- b
		}
		close(c)
	}()
	return c
}

func countFile(path string, wg *sync.WaitGroup, result chan<- count) {
	defer wg.Done()
	score := count{
		Path:  path,
		Chars: 0,
		Words: 0,
		Lines: 0,
	}
	inWord := false
	for b := range fileBytes(path) {
		score.Chars++
		switch b {
		case ' ', '\t':
			if inWord {
				score.Words++
			}
			inWord = false
		case '\n':
			score.Lines++
			if inWord {
				score.Words++
			}
			inWord = false
		default:
			inWord = true
		}
	}
	result <- score
}

func main() {
	fChars := flag.Bool("c", false, "Count chars.")
	fWords := flag.Bool("w", false, "Count words.")
	fLines := flag.Bool("l", false, "Count lines.")
	fVerbose := flag.Bool("v", false, "Run verbosely.")
	flag.Parse()
	results := make(chan count)
	wg := &sync.WaitGroup{}
	wg.Add(flag.NArg())
	for i, arg := range flag.Args() {
		if *fVerbose {
			fmt.Printf("%d:\t%s\n", i, arg)
		}
		go countFile(arg, wg, results)
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	tally := make(map[string]count)
	for result := range results {
		tally[result.Path] = result
	}
	for _, arg := range flag.Args() {
		if *fLines {
			fmt.Printf("%4d ", tally[arg].Lines)
		}
		if *fWords {
			fmt.Printf("%4d ", tally[arg].Words)
		}
		if *fChars {
			fmt.Printf("%4d ", tally[arg].Chars)
		}
		fmt.Printf("%s\n", arg)
	}
}
