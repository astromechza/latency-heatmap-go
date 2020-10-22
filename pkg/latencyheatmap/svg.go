package latencyheatmap

import (
	"fmt"
	"sort"
	"time"

	"golang.org/x/net/html"
)

type Datapoint struct {
	Time time.Time
	Latency time.Duration
}

const js = "" +
	"var details;" +
	"function a(evt) {details = document.getElementById(\"details\").firstChild;}" +
	"function b(rt, ls, le, v) {" +
	"details.nodeValue = \"at \" + rt + \", range: \" + ls + \"-\" + le + \", count: \" +  v;" +
	"}" +
	"function c() {details.nodeValue = \"(hover over a square for details)\";}"

type SVGOptions struct {
	DisableXMLDoctype bool
}

func RenderSVG(
	points []Datapoint,
	options SVGOptions,
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
	columnInterval = roundUpToFriendlyInterval(columnInterval)
	if columnInterval <= 0 {
		panic("wat")
	}
	columns := int(elapsed / columnInterval) + 1

	// TODO: handle out of range Inf
	minValue := time.Duration(0)
	maxValue := minValue
	for _, p := range points {
		if p.Latency > maxValue {
			maxValue = p.Latency
		}
	}

	// choose a block height so that we
	rowPeriod := (maxValue - minValue) / 100
	rowPeriod = roundUpToFriendlyInterval(rowPeriod)
	if rowPeriod <= 0 {
		panic("wat")
	}
	rows := int((maxValue - minValue) / rowPeriod) + 1

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
	padLeft := 60.0
	padTop := 30.0
	padRight := 10.0
	padBottom := 60.0

	svgNode := &html.Node{
		Type: html.ElementNode,
		Data: "svg",
		Attr: []html.Attribute{
			{Key: "version", Val: "1.1"},
			{Key: "width", Val: fmt.Sprint(imageWidth)},
			{Key: "height", Val: fmt.Sprint(imageHeight)},
			{Key: "viewBox", Val: fmt.Sprintf("0 0 %f %f", imageWidth, imageHeight)},
			{Key: "xmlns", Val: "http://www.w3.org/2000/svg"},
			{Key: "onload", Val: "a(evt)"},
		},
	}
	svgNode.AppendChild(&html.Node{
		Type: html.RawNode,
		Data: "<script type=\"text/ecmascript\"\n><![CDATA[\n" + js + "]]>\n</script>",
	})

	svgNode.AppendChild(&html.Node{
		Type: html.ElementNode,
		Data: "rect",
		Attr: []html.Attribute{
			{Key: "x", Val: "0"},
			{Key: "y", Val: "0"},
			{Key: "width", Val: fmt.Sprint(imageWidth)},
			{Key: "height", Val: fmt.Sprint(imageHeight)},
			{Key: "fill", Val: "rgb(255, 255, 255)"},
		},
	})

	widthBetweenColumns := (imageWidth - padLeft - padRight) / float64(columns)
	heightBetweenRows := (imageHeight - padTop - padBottom) / float64(rows)

	for i := 0; i < columns; i++ {
		for j := 0; j < rows; j++ {
			if blocks[i][rows - j - 1] > 0 {
				val := float64(blocks[i][rows-j-1]) / float64(largestFrequency)
				newNode := &html.Node{
					Type: html.ElementNode,
					Data: "rect",
					Attr: []html.Attribute{
						{Key: "x", Val: fmt.Sprint(padLeft + float64(i)*widthBetweenColumns)},
						{Key: "y", Val: fmt.Sprint(padTop + float64(j)*heightBetweenRows)},
						{Key: "width", Val: fmt.Sprint(widthBetweenColumns)},
						{Key: "height", Val: fmt.Sprint(heightBetweenRows)},
						{Key: "fill", Val: CividisRGB(val)},
						{Key: "onmouseover", Val: fmt.Sprintf(
							"b(%q, %q, %q, %d)",
								// the relative time on the left column
								time.Duration(i) * columnInterval,
								// the latency of the row start
								time.Duration(rows-j-1) * rowPeriod,
								time.Duration(rows-j) * rowPeriod,
								blocks[i][rows-j-1],
							)},
						{Key: "onmouseout", Val: "c()"},
					},
				}
				svgNode.AppendChild(newNode)
			}
		}
	}

	svgNode.AppendChild(boldLine(padLeft, padTop, imageWidth - padRight, padTop))
	svgNode.AppendChild(boldLine(imageWidth - padRight, padTop, imageWidth - padRight, imageHeight - padBottom))
	svgNode.AppendChild(boldLine(padLeft, padTop, padLeft, imageHeight - padBottom))
	svgNode.AppendChild(boldLine(padLeft, imageHeight - padBottom, imageWidth - padRight, imageHeight - padBottom))

	svgNode.AppendChild(text(padLeft, imageHeight - padBottom + 5, "0", "hanging", "start"))
	svgNode.AppendChild(text(imageWidth - padRight, imageHeight - padBottom + 5, fmt.Sprintf("+%s", roundTo2ndDecimal(columnInterval * time.Duration(columns))), "hanging", "end"))
	svgNode.AppendChild(text(padLeft / 2, imageHeight / 2, "Latency", "middle", "middle"))
	svgNode.LastChild.Attr = append(svgNode.LastChild.Attr, html.Attribute{Key: "transform", Val: fmt.Sprintf("rotate(90, %f, %f)", padLeft / 2, imageHeight / 2)})

	svgNode.AppendChild(text(padLeft - 5, imageHeight - padBottom, "0", "alphabetic", "end"))
	svgNode.AppendChild(text(padLeft - 5, padTop, fmt.Sprintf("%s", roundTo2ndDecimal(rowPeriod * time.Duration(rows))), "hanging", "end"))
	svgNode.AppendChild(text(imageWidth / 2, imageHeight - padBottom + 5, "Relative time", "hanging", "middle"))

	svgNode.AppendChild(text(imageWidth / 2, imageHeight - padBottom / 3, "(hover over a square for details)", "middle", "middle"));
	svgNode.LastChild.Attr = append(svgNode.LastChild.Attr, html.Attribute{Key: "id", Val: "details"})

	outputNode := svgNode
	if !options.DisableXMLDoctype {
		outputNode = &html.Node{
			Type: html.DocumentNode,
		}
		outputNode.AppendChild(&html.Node{
			Type: html.RawNode,
			Data: "<?xml version=\"1.0\" standalone=\"no\"?>",
		})
		outputNode.AppendChild(&html.Node{
			Type: html.DoctypeNode,
			Data: "svg",
			Attr: []html.Attribute{
				{"", "public", "-//W3C//DTD SVG 1.1//EN"},
				{"", "system", "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd"},
			},
		})
		outputNode.AppendChild(svgNode)
	}

	return outputNode, nil
}

// roundTo2ndDecimal will round a duration to an appropriate value so that when rendered it only shows 2 decimal places.
// eg: 45m12.53s -> 45m12s
func roundTo2ndDecimal(duration time.Duration) time.Duration {
	if duration > time.Hour {
		return duration.Round(time.Minute)
	}
	if duration > time.Minute {
		return duration.Round(time.Second)
	}
	if duration > time.Second {
		return duration.Round(time.Millisecond * 10)
	}
	if duration > time.Millisecond {
		return duration.Round(time.Microsecond * 10)
	}
	if duration > time.Microsecond {
		return duration.Round(time.Nanosecond * 10)
	}
	return duration
}

func roundUpToFriendlyInterval(duration time.Duration) time.Duration {
	for _, d := range []time.Duration{
		time.Hour, time.Minute * 10, time.Minute * 5, time.Minute, time.Second * 30, time.Second * 10, time.Second,
		time.Millisecond * 500, time.Millisecond * 200, time.Millisecond * 100, time.Millisecond * 50, time.Millisecond * 10,
		time.Millisecond,
	} {
		if duration > d {
			return duration.Round(d + d/2)
		}
	}
	return duration
}

func text(x, y float64, input string, baseline, anchor string) *html.Node {
	block := html.Node{
		Type: html.ElementNode,
		Data: "text",
		Attr: []html.Attribute{
			{Key: "text-anchor", Val: anchor},
			{Key: "dominant-baseline", Val: baseline},
			{Key: "x", Val: fmt.Sprint(x)},
			{Key: "y", Val: fmt.Sprint(y)},
			{Key: "fill", Val: "black"},
		},
	}
	block.AppendChild(&html.Node{Type: html.TextNode, Data: input})
	return &block
}

func boldLine(x1, y1, x2, y2 float64) *html.Node {
	return &html.Node{
		Type: html.ElementNode,
		Data: "line",
		Attr: []html.Attribute{
			{Key: "x1", Val: fmt.Sprint(x1)},
			{Key: "y1", Val: fmt.Sprint(y1)},
			{Key: "x2", Val: fmt.Sprint(x2)},
			{Key: "y2", Val: fmt.Sprint(y2)},
			{Key: "stroke", Val: "black"},
			{Key: "stroke-width", Val: "1"},
		},
	}
}
