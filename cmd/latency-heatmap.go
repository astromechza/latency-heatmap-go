package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"

	"github.com/astromechza/latency-heatmap-go/pkg/latencyheatmap"
)

func mainInner() error {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	inputVar := fs.String("input", "", "The input source")
	inputFormatVar := fs.String("input-format", "auto", "The input source format")
	outputVar := fs.String("output", "-", "The destination file to write, or - for stdout")

	if err := fs.Parse(os.Args[1:]); err != nil {
		return err
	}

	points, err := readDataPoints(*inputVar, *inputFormatVar)
	if err != nil {
		return fmt.Errorf("failed to parse input: %v", err)
	}

	var f *os.File
	if *outputVar == "" {
		return fmt.Errorf("output destination cannot be empty ''")
	} else if *outputVar == "-" {
		f = os.Stdout
	} else {
		if f, err = os.Create("demo.svg"); err != nil {
			return fmt.Errorf("failed to open output file for writing: %v", err)
		}
		defer f.Close()
	}

	n, err := latencyheatmap.RenderSVG(points, latencyheatmap.SVGOptions{})
	if err != nil {
		return fmt.Errorf("failed to construct svg: %v", err)
	}
	if err = html.Render(f, n); err != nil {
		return fmt.Errorf("failed to render svg: %v", err)
	}
	return nil
}

var formatDetectors = map[string]*regexp.Regexp{
	"epoch-line": regexp.MustCompile("^\\d+[ \\t]+\\d+(?:\\.\\d+)?[ \\t]*(?:\\n|$)"),
	"epoch-csv": regexp.MustCompile("^\\d+[ \\t]*,[ \\t]*\\d+(?:\\.\\d+)?[ \\t]*(?:\\n|$)"),
	"json": regexp.MustCompile("^\\S+\\[\\S*[{\\]]"),
	"rfc3339nano-milliseconds-csv": regexp.MustCompile("^[\\d\\-+.:TZ]+,\\d+(?:\\.\\d+)?(?:\\n|$)"),
}

func readDataPoints(source string, sourceFormat string) ([]latencyheatmap.Datapoint, error) {
	if sourceFormat == "random" {
		rand.Seed(time.Now().UnixNano())
		baseTime := time.Now()
		points := make([]latencyheatmap.Datapoint, 0, 1000)
		for i := 0; i < 100000; i++ {
			v := time.Second * 2 + time.Duration(rand.NormFloat64() * float64(time.Second))
			if v <= 0 {
				v = 0
			}
			points = append(points, latencyheatmap.Datapoint{
				Time:    baseTime.Add(time.Duration(float64(time.Hour) * rand.Float64())),
				Latency: v,
			})
		}
		return points, nil
	}

	var err error
	var f *os.File
	if source == "" {
		return nil, fmt.Errorf("source cannot be empty ''")
	} else if source == "-" {
		f = os.Stdin
	} else {
		if f, err = os.Open(source); err != nil {
			return nil, fmt.Errorf("cannot open file: %v", err)
		}
		defer f.Close()
	}

	var r io.Reader = f
	if sourceFormat == "auto" {
		buff := make([]byte, 100)
		_, err := r.Read(buff)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}
		for k, v := range formatDetectors {
			if v.Match(buff) {
				sourceFormat = k
			}
		}
		if sourceFormat == "auto" {
			return nil, fmt.Errorf("failed to determine source format from first %d bytes", len(buff))
		}
		r = io.MultiReader(bytes.NewReader(buff), r)
	}
	if sourceFormat == "epoch-line" {
		panic("not implemented")
	} else if sourceFormat == "epoch-csv" {
		panic("not implemented")
	} else if sourceFormat == "rfc3339nano-milliseconds-csv" {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("failed to read: %v", err)
		}
		lines := strings.Split(string(data), "\n")
		points := make([]latencyheatmap.Datapoint, 0, len(lines))
		for i, line := range lines {
			parts := strings.Split(line, ",")
			if len(parts) == 2 {
				dt, err := time.Parse(time.RFC3339Nano, parts[0])
				if err != nil {
					return nil, fmt.Errorf("failed to parse time on line %d: %v", i, err)
				}
				ms, err := strconv.ParseFloat(parts[1], 64)
				if err != nil {
					return nil, fmt.Errorf("failed to parse float on line %d: %v", i, err)
				}
				points = append(points, latencyheatmap.Datapoint{
					Time: dt,
					Latency: time.Duration(ms * float64(time.Millisecond)),
				})
			}
		}
		return points, nil
	} else if sourceFormat == "json" {
		panic("not implemented")
	} else {
		return nil, fmt.Errorf("unknown source format '%s'", sourceFormat)
	}
}

func main() {
	if err := mainInner(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}
