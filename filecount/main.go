package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

func fileReadAll(path string, channel chan<- byte) {
	defer close(channel)
	if data, err := ioutil.ReadFile(path); err == nil {
		for _, datum := range data {
			channel <- datum
		}
	} else {
		panic(err)
	}
}

func fileReadBuf(path string, channel chan<- byte) {
	defer close(channel)
	if f, err := os.Open(path); err == nil {
		defer f.Close()
		const BUFSIZ = 4096
		buf := make([]byte, BUFSIZ)
		readCount := 0
		for n, err := f.Read(buf); err == nil; n, err = f.Read(buf) {
			readCount++
			for i := 0; i < n; i++ {
				channel <- buf[i]
			}
		}
		fmt.Fprintf(os.Stderr, "%s: read count: %d\n", path, readCount)
	} else {
		panic(err)
	}
}

func fileBytes(path string) chan byte {
	channel := make(chan byte)
	go fileReadBuf(path, channel)
	return channel
}

type count struct {
	Path        string
	Chars       int
	Words       int
	Lines       int
	CharPerWord int
	CharPerLine int
}

func scoreFile(path string, wg *sync.WaitGroup, result chan<- count) {
	defer wg.Done()
	score := count{
		Path:        path,
		Chars:       0,
		Words:       0,
		Lines:       0,
		CharPerWord: 0,
		CharPerLine: 0,
	}
	inWord := false
	seeWhite := func() {
		if inWord {
			score.Words++
		}
		inWord = false
	}
	for datum := range fileBytes(path) {
		score.Chars++
		switch datum {
		case ' ', '\t':
			seeWhite()
		case '\n':
			score.Lines++
			seeWhite()
		default:
			inWord = true
		}
	}
	result <- score
}

func makeVisitor(wg *sync.WaitGroup, results chan<- count) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err == nil {
			switch mode := info.Mode(); {
			case mode.IsRegular():
				wg.Add(1)
				go scoreFile(path, wg, results)
			default:
				// do nothing
			}
			return nil
		} else {
			return err
		}
	}
}

func sortedKeys(m map[string]count) (keys []string) {
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func aggregator(raw <-chan count, agg chan<- count) {
	defer close(agg)
	for item := range raw {
		item.CharPerWord = item.Chars / item.Words
		item.CharPerLine = item.Chars / item.Lines
		agg <- item
	}
}

func main() {
	fChars := flag.Bool("c", false, "Count chars.")
	fWords := flag.Bool("w", false, "Count words.")
	fLines := flag.Bool("l", false, "Count lines.")
	fVerbose := flag.Bool("v", false, "Run verbosely.")
	flag.Parse()
	results := make(chan count)
	wg := &sync.WaitGroup{}
	for i, arg := range flag.Args() {
		if *fVerbose {
			fmt.Printf("%d:\t%s\n", i, arg)
		}
		filepath.Walk(arg, makeVisitor(wg, results))
	}
	go func() {
		wg.Wait()
		close(results)
	}()
	aggregates := make(chan count)
	go aggregator(results, aggregates)
	tally := make(map[string]count)
	for result := range aggregates {
		tally[result.Path] = result
	}
	for _, k := range sortedKeys(tally) {
		if *fLines {
			fmt.Printf("%5d ", tally[k].Lines)
		}
		if *fWords {
			fmt.Printf("%5d ", tally[k].Words)
		}
		if *fChars {
			fmt.Printf("%5d ", tally[k].Chars)
		}
		fmt.Printf("%d %d %s\n", tally[k].CharPerWord, tally[k].CharPerLine, k)
	}
}
