package main

import (
	"math/rand"
	"os"
	"time"

	"golang.org/x/net/html"

	"github.com/astromechza/latency-heatmap-go/pkg/latencyheatmap"
)

func main() {
	f, err := os.Create("demo.svg")
	if err != nil {
		panic(err)
	}
	defer f.Close()

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

	n, err := latencyheatmap.RenderSVG(points)
	if err != nil {
		panic(err)
	}

	html.Render(f, &html.Node{
		Type: html.RawNode,
		Data: "<?xml version=\"1.0\" standalone=\"no\"?>",
	})
	html.Render(f, &html.Node{
		Type: html.DoctypeNode,
		Data: "svg",
		Attr: []html.Attribute{
			{"", "public", "-//W3C//DTD SVG 1.1//EN"},
			{"", "system", "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd"},
		},
	})

	err = html.Render(f, n)
	if err != nil {
		panic(err)
	}
}
