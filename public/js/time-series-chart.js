'use strict';

/* based on https://bost.ocks.org/mike/chart/time-series-chart.js */

var timeSeriesChart = (function () {
    function timeSeriesChart() {
        var div,
            svg,
            g,
            zoom,
            data,
            margin = {top: 10, right: 10, bottom: 50, left: 80},
            width = 720,  // isFitWidthがtrueのときは最小幅の意味になる
            curWidth = width,  // 実際のsvg幅
            height = 300,
            tooltipWidth = 100,
            tooltipHeight = 50,
            isFitWidth = false,
            pos1Color = "lightsteelblue",
            pos2Color = "steelblue",
            neg1Color = "lightpink",
            neg2Color = "firebrick",
            prefix = "tsc",
            transform = d3.zoomIdentity,
            xValue = function(d) { return d[0]; },
            yValue = function(d) { return d[1]; },
            xValueText = function(d) { return d[0]; },
            yValueText = function(d) { return d[1]; },
            xScale = d3.scaleTime(),
            yScale = d3.scaleLinear(),
            xAxis = d3.axisBottom(xScale),
            yAxis = d3.axisLeft(yScale),
            area = d3.area().x(X).y1(Y),
            line = d3.line().x(X).y(Y),
            bisectDate = d3.bisector(function(d) { return d[0]; }).left;

        function chart(selection) {
            div = selection;

            selection.each(function(data0) {
                data = data0.map(function(d, i) {
                    return [xValue.call(data0, d, i), yValue.call(data0, d, i),
                        xValueText.call(data0, d, i), yValueText.call(data0, d, i)];
                });

                var yDomain = d3.extent(data, function(d) { return d[1]; });
                yDomain = [Math.min(0, yDomain[0]), Math.max(0, yDomain[1])];

                yScale
                    .domain(yDomain)
                    .range([height - margin.top - margin.bottom, 0]);

                zoom = d3.zoom()
                    .scaleExtent([1, 8])
                    .on("zoom", function () {
                        transform = d3.event.transform;

                        if (transform.k == 1) {
                            transform.x = 0;
                            transform.y = 0;
                        }

                        redraw();
                    });

                var svg0 = div.selectAll("svg").data([data]);

                if (svg0.size() > 0) {
                    svg = svg0;
                    svg.attr("height", height);
                } else {
                    // svg要素が見つからない場合は作る
                    svg = svg0.enter().append("svg")
                        .attr("height", height)
                        .call(zoom);

                    g = svg.append("g")
                        .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

                    g.append("clipPath")
                        .attr("id", prefix + "-clip")
                        .append("rect")
                        .attr("x", 0)
                        .attr("y", 0)
                        .attr("height", height - margin.top - margin.bottom);

                    var drawArea = g.append("g")
                        .attr("clip-path", "url(#" + prefix + "-clip)");

                    drawArea.append("path").attr("class", "area");
                    drawArea.append("path").attr("class", "line");
                    g.append("g").attr("class", "y axis");
                    g.append("g").attr("class", "x axis");
                    g.append("g").attr("class", "tooltip-g");
                    g.append("rect").attr("class", "overlay");

                    setupTooltip();
                }

                g.select(".area")
                    .attr("fill", "url(#" + prefix + "-gradient)")
                    .attr("stroke-width", 0);

                g.select(".x.axis")
                    .attr("transform", "translate(0," + yScale.range()[0] + ")");

                g.select(".y.axis")
                    .call(yAxis);

                g.select(".overlay")
                    .attr("height", height - margin.top - margin.bottom);

                setupLinearGradient(yDomain);

                resize();
            });
        }

        function orgRound(value, base) {
            return Math.round(value * base) / base;
        }

        function resize() {
            curWidth = getWidth();
            redraw();
        }

        function redraw() {
            svg.attr("width", curWidth);

            var w1 = parseInt(svg.style("width")),
                w = w1 - margin.left - margin.right;

            zoom
                .extent([[0, 0], [w, height]])
                .translateExtent([[0, 0], [w, height]]);

            xScale
                .domain(d3.extent(data, function(d) { return d[0]; }))
                .range([0, w]);

            xScale = transform.rescaleX(xScale);

            svg.select("#" + prefix + "-clip rect")
                .attr("width", w);

            g.select(".area")
                .data([data])
                .attr("d", area.y0(yScale(0)));

            g.select(".line")
                .data([data])
                .attr("d", line);

            g.select(".x.axis")
                .call(xAxis.scale(transform.rescaleX(xScale)));

            g.select(".overlay")
                .attr("width", w);
        };

        function setupTooltip() {
            var tooltipG = g.select(".tooltip-g")
                .style("display", "none");

            var circle = tooltipG.append("circle")
                .attr("r", 5);

            var line = tooltipG.append("line")
                .attr("x1", 0)
                .attr("x2", 0);

            var tooltip = tooltipG.append("g")
                .attr("class", "tooltip");

            tooltip.append("rect")
                .attr("class", "tooltip-box")
                .attr("width", tooltipWidth)
                .attr("height", tooltipHeight)
                .attr("rx", 4)
                .attr("ry", 4);

            tooltip.append("text")
                .attr("class", "tooltip-date")
                .attr("x", 8)
                .attr("y", 20);

            tooltip.append("text")
                .attr("class", "tooltip-value")
                .attr("x", 8)
                .attr("y", 40);

            var h = height - margin.top - margin.bottom,
                tw2 = tooltipWidth / 2;

            g.select(".overlay")
                .on("touchstart mouseenter", function() { tooltipG.style("display", null); })
                .on("touchend mouseleave", function() { tooltipG.style("display", "none"); })
                .on("touchmove mousemove", function() {
                    var x0 = xScale.invert(d3.mouse(this)[0]),
                        i = bisectDate(data, x0, 1, data.length-1),
                        d0 = data[i - 1],
                        d1 = data[i],
                        d = x0 - X(d0) > X(d1) - x0 ? d1 : d0;

                    tooltip.select(".tooltip-date").text(d[2]);
                    tooltip.select(".tooltip-value").text(d[3]);

                    var xt = X(d),
                        yt = Y(d);

                    tooltipG.attr("transform", "translate(" + xt + ",0)");
                    circle.attr("transform", "translate(0," + yt + ")");
                    line.attr("y2", yt);

                    var w = parseInt(svg.style("width")) - margin.left - margin.right,
                        x1;

                    if (xt + tw2 > w) {
                        x1 = w - xt - tooltipWidth;
                    } else if (xt < tw2) {
                        x1 = -xt;
                    } else {
                        x1 = -tw2;
                    }

                    if (yt > h/2) {
                        line.attr("y1", 10 + tooltipHeight);
                        tooltip.attr("transform", "translate(" + x1 + ",10)");
                    } else {
                        var y1 = h - tooltipHeight - 10;
                        line.attr("y1", y1);
                        tooltip.attr("transform", "translate(" + x1 + "," + y1 + ")");
                    }
                });

        }

        function setupLinearGradient(yDomain) {
            // 0を境に正負で領域を色分けする
            var gradData;

            if (yDomain[0] == 0) {
                gradData = [
                    {offset: "0%", color: pos1Color},
                    {offset: "100%", color: pos2Color}
                ];
            } else if (yDomain[1] == 0) {
                gradData = [
                    {offset: "0%", color: neg2Color},
                    {offset: "100%", color: neg1Color}
                ];
            } else {
                gradData = [
                    {offset: "0%", color: neg2Color},
                    {offset: "50%", color: neg1Color},
                    {offset: "49.999%", color: pos1Color},
                    {offset: "100%", color: pos2Color}
                ];
            }

            // yDomainが[-50, 100]の場合、-50の色が、100の色の濃さの半分になるように調整
            var y1, y2;

            if (yDomain[0] < 0 && yDomain[1] > 0) {
                if (-yDomain[0] > yDomain[1]) {
                    y1 = yScale(yDomain[0]);
                    y2 = yScale(0) * 2 - y1;
                } else {
                    y2 = yScale(yDomain[1]);
                    y1 = yScale(0) * 2 + y2;
                }
            } else {
                y1 = yScale(yDomain[0]);
                y2 = yScale(yDomain[1]);
            }

            svg.select("#" + prefix + "-gradient").remove();

            svg.append("linearGradient")
                .attr("id", prefix + "-gradient")
                .attr("gradientUnits", "userSpaceOnUse")
                .attr("x1", 0).attr("y1", y1)
                .attr("x2", 0).attr("y2", y2)
                .selectAll("stop")
                .data(gradData)
                .enter().append("stop")
                .attr("offset", function(d) { return d.offset; })
                .attr("stop-color", function(d) { return d.color; });
        };

        function getWidth() {
            if (!isFitWidth || typeof div === "undefined") {
                return width;
            }

            return Math.max(width, parseInt(div.style("width"), 10));
        }

        function X(d) {
            return xScale(d[0]);
        }

        function Y(d) {
            return yScale(d[1]);
        }

        chart.margin = function(_) {
            if (!arguments.length) return margin;
            margin = _;
            return chart;
        };

        chart.width = function(_) {
            if (!arguments.length) return width;
            width = _;
            return chart;
        };

        chart.height = function(_) {
            if (!arguments.length) return height;
            height = _;
            return chart;
        };

        chart.tooltipWidth = function(_) {
            if (!arguments.length) return tooltipWidth;
            tooltipWidth = _;
            return chart;
        };

        chart.posColor = function(_) {
            if (!arguments.length) return posColor;
            posColor = _;
            return chart;
        };

        chart.negColor = function(_) {
            if (!arguments.length) return negColor;
            negColor = _;
            return chart;
        };

        chart.zeroColor = function(_) {
            if (!arguments.length) return zeroColor;
            zeroColor = _;
            return chart;
        };

        chart.prefix = function(_) {
            if (!arguments.length) return prefix;
            prefix = _;
            return chart;
        };

        chart.x = function(_) {
            if (!arguments.length) return xValue;
            xValue = _;
            return chart;
        };

        chart.y = function(_) {
            if (!arguments.length) return yValue;
            yValue = _;
            return chart;
        };

        chart.xText = function(_) {
            if (!arguments.length) return xValueText;
            xValueText = _;
            return chart;
        };

        chart.yText = function(_) {
            if (!arguments.length) return yValueText;
            yValueText = _;
            return chart;
        };

        chart.fitWidth = function() {
            isFitWidth = true;
            window.addEventListener("resize", resize);
            return chart;
        };

        return chart;
    }

    return timeSeriesChart;
})();
