package latencyheatmap

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/html"
)

const OutOfRange = time.Duration(1<<63 - 1)

type Datapoint struct {
	Time time.Time
	Latency time.Duration
}

func RenderSVG(
	points []Datapoint,
) (*html.Node, error) {
	if points == nil || len(points) == 0 {
		return nil, fmt.Errorf("must provide at least one datapoint")
	}

	// start by sorting by time, this is slow but should make future operations faster
	sort.Slice(points, func(i, j int) bool {
		return points[i].Time.Before(points[j].Time)
	})

	// choose a col width so that we get 100 columns
	elapsed := points[len(points)-1].Time.Sub(points[0].Time)
	// start with the perfect width
	columnInterval := elapsed / 100
	// TODO: optimise so that we get human time sections
	if columnInterval <= 0 {
		panic("wat")
	}
	columns := int(elapsed / columnInterval)

	// TODO: handle out of range Inf
	minValue := time.Duration(0)
	maxValue := minValue
	for _, p := range points {
		if p.Latency > maxValue {
			maxValue = p.Latency
		}
	}

	// choose a block height so that we
	rowPeriod := (maxValue - minValue)/100
	// TODO: optimise so that we get human time sections
	if rowPeriod <= 0 {
		panic("wat")
	}
	rows := int((maxValue - minValue) / rowPeriod)

	blocks := make([][]int, columns)
	for i := 0; i < columns; i++ {
		blocks[i] = make([]int, rows)
	}

	largestFrequency := 0
	for _, p := range points {
		blockX := int(p.Time.Sub(points[0].Time) / columnInterval)
		if blockX == columns {
			blockX = columns - 1
		}
		blockY := int(p.Latency / rowPeriod)
		if blockY == rows {
			blockY = rows - 1
		}
		blocks[blockX][blockY] += 1
		if blocks[blockX][blockY] > largestFrequency {
			largestFrequency = blocks[blockX][blockY]
		}
	}

	imageWidth, imageHeight := 1024.0, 768.0

	svgNode := &html.Node{
		Type: html.ElementNode,
		Data: "svg",
		Attr: []html.Attribute{
			{"", "version", "1.1"},
			{"", "width", fmt.Sprint(imageWidth)},
			{"", "height", fmt.Sprint(imageHeight)},
			{"", "viewBox", fmt.Sprintf("0 0 %d %d", imageWidth, imageHeight)},
			{"", "xmlns", "http://www.w3.org/2000/svg"},
		},
	}
	svgNode.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "rect",
		Attr: []html.Attribute{
			{"", "x", "0"},
			{"", "y", "0"},
			{"", "width", fmt.Sprint(imageWidth)},
			{"", "height", fmt.Sprint(imageHeight)},
			{"", "fill", "rgb(255, 255, 255)"},
		},
	})

	svgNode.AppendChild(boldLine(0, 0, imageWidth, 0))
	svgNode.AppendChild(boldLine(imageWidth, 0, imageWidth, imageHeight))
	svgNode.AppendChild(boldLine(0, 0, 0, imageHeight))
	svgNode.AppendChild(boldLine(0, imageHeight, imageWidth, imageHeight))

	widthBetweenColumns := imageWidth / float64(columns)
	for i := 1.0; i < float64(columns); i++ {
		svgNode.AppendChild(faintLine(widthBetweenColumns * i, 0, widthBetweenColumns * i, imageHeight))
	}

	heightBetweenRows := imageHeight / float64(rows)
	for i := 1.0; i < float64(rows); i++ {
		svgNode.AppendChild(faintLine(0, heightBetweenRows * i, imageWidth, heightBetweenRows * i))
	}

	for i := 0; i < columns; i++ {
		for j := 0; j < rows; j++ {
			val := float64(blocks[i][j]) / float64(largestFrequency)
			svgNode.AppendChild(&html.Node{
				Type: html.ElementNode,
				Data: "rect",
				Attr: []html.Attribute{
					{"", "x", fmt.Sprint(float64(i) * widthBetweenColumns)},
					{"", "y", fmt.Sprint(float64(j) * heightBetweenRows)},
					{"", "width", fmt.Sprint(widthBetweenColumns)},
					{"", "height", fmt.Sprint(heightBetweenRows)},
					{"", "fill", fmt.Sprintf("rgba(255, 0, 0, %d)", int(255 * val))},
				},
			})
		}
	}

	return svgNode, nil
}

func boldLine(x1, y1, x2, y2 float64) *html.Node {
	return &html.Node{
		Type: html.ElementNode,
		Data: "line",
		Attr: []html.Attribute{
			{"", "x1", fmt.Sprint(x1)},
			{"", "y1", fmt.Sprint(y1)},
			{"", "x2", fmt.Sprint(x2)},
			{"", "y2", fmt.Sprint(y2)},
			{"", "stroke", "black"},
			{"", "stroke-width", "1"},
		},
	}
}

func faintLine(x1, y1, x2, y2 float64) *html.Node {
	return &html.Node{
		Type: html.ElementNode,
		Data: "line",
		Attr: []html.Attribute{
			{"", "x1", fmt.Sprint(x1)},
			{"", "y1", fmt.Sprint(y1)},
			{"", "x2", fmt.Sprint(x2)},
			{"", "y2", fmt.Sprint(y2)},
			{"", "stroke", "gray"},
			{"", "stroke-width", "1"},
		},
	}
}
