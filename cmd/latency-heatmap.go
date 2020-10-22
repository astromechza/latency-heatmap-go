package main

import (
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

	n, err := latencyheatmap.RenderSVG([]latencyheatmap.Datapoint{
		{time.Now(), time.Second},
		{time.Now(), time.Hour},
	})
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
