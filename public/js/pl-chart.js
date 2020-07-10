'use strict';

var plChart = (function () {

    function plChart() {
        var div,
            svg,
            g,
            data,
            margin = {top: 50, right: 10, bottom: 10, left: 80},
            width = 720,  // isFitWidthがtrueのときは最小幅の意味になる
            curWidth = width,  // 実際のsvg幅
            barHeight,
            numMonths = 12,
            height = 800,
            maxX,
            expenseStacked,
            incomeStacked,
            xValue = function(d) { return d[0]; },
            yValue = function(d) { return d[1]; },
            xScale = d3.scaleLinear(),
            yScale = d3.scaleBand().paddingOuter(0.1).paddingInner(0.4),
            xAxis = d3.axisTop(xScale),
            yAxis = d3.axisLeft(yScale);

        function chart(selection) {
            window.addEventListener("resize", resize);
            div = selection;

            selection.each(function(data0) {
                data = data0;

                data.values.forEach(function (d) {
                    var y = Math.floor(d.month / 100),
                        m = d.month % 100;

                    if (m < 10) {
                        m = "0" + m;
                    } else {
                        m = "" + m;
                    }

                    d.month = y + "-" + m;
                });

                var expenseValues = createValues(data.values, function(v) { return -v; });
                expenseStacked = getStacked(data.keys, expenseValues);
                var maxExpenseX = getMaxX(expenseStacked);

                var incomeValues = createValues(data.values, function(v) { return +v; });
                incomeStacked = getStacked(data.keys, incomeValues);
                var maxIncomeX = getMaxX(incomeStacked);

                maxX = Math.max(maxExpenseX, maxIncomeX);

                yScale
                    .domain(data.values.map(function(d) { return d.month; }))
                    .range([0, height - margin.top - margin.bottom]);

                barHeight = yScale.bandwidth() / 2;

                var svg0 = div.selectAll("svg").data([data]);

                if (svg0.size() > 0) {
                    svg = svg0;
                    svg.attr("height", height);
                } else {
                    svg = svg0.enter().append("svg")
                        .attr("height", height);

                    g = svg.append("g")
                        .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

                    g.append("g").attr("class", "x axis");
                    g.append("g").attr("class", "y axis");

                    var color = d3.scaleOrdinal(d3.schemePastel1);

                    data.keys.forEach(function (d) {
                        color(d);
                    });

                    enterLayer(incomeStacked, "incomeLayers", color, 0);
                    enterLayer(expenseStacked, "expenseLayers", color, barHeight);
                }

                g.select(".y.axis").call(yAxis);

                resize();
            });
        }

        function resize() {
            curWidth = getWidth();
            redraw();
        }

        function redraw() {
            svg.attr("width", curWidth);

            var w1 = parseInt(svg.style("width")),
                w = w1 - margin.left - margin.right;

            xScale
                .domain([0, maxX])
                .range([0, w]);

            g.select(".x.axis").call(xAxis);

            updateLayer(incomeStacked, "incomeLayers");
            updateLayer(expenseStacked, "expenseLayers");
        }

        function getWidth() {
            if (typeof div === "undefined") {
                return width;
            }

            return Math.max(width, parseInt(div.style("width"), 10));
        }

        function createValues(values, f) {
            return values.map(function (d) {
                var obj = {};

                obj.month = d.month;

                data.keys.forEach(function (key) {
                    if (f(d[key]) > 0) {
                        obj[key] = f(d[key]);
                    } else {
                        obj[key] = 0;
                    }
                });

                return obj;
            });
        }

        function getStacked(keys, data) {
            var stack = d3.stack().keys(keys);
            return stack(data);
        }

        function getMaxX(stacked) {
            var numKeys = stacked.length;
            var lastXs = stacked[numKeys-1].map(function(d) { return d[1]; });
            return d3.max(lastXs);
        }

        function enterLayer(stacked, className, color, extraY) {
            var layers = g.selectAll("g." + className)
                .data(stacked, function(d) { return d.key; })
                .enter()
                .append("g")
                .attr("class", className)
                .attr("fill", function(d) { return color(d.key); });

            layers.selectAll("rect")
                .data(function(d) { return d; })
                .enter()
                .append("rect")
                .attr("y", function(d) { return yScale(d.data.month) + extraY; })
                .attr("height", barHeight);

            layers.selectAll("text")
                .data(function(d) { return d; })
                .enter()
                .append("text")
                .attr("y", function(d) { return yScale(d.data.month) + extraY + barHeight - 5; })
                .attr("fill", function(d) { return "black"; })
                .style("font-size", "12px")
                .style("text-anchor", "middle")
                .text(function(d) { return d.key; });
        }

        function updateLayer(stacked, className) {
            var layers = g.selectAll("g." + className)
                .data(stacked, function(d) { return d.key; })
                .each(function (d) {
                    for (var i = 0; i < d.length; i++) {
                        d[i].key = d.key;
                    }
                });

            layers.selectAll("rect")
                .data(function(d) { return d; })
                .attr("x", function(d) {
                    return xScale(d[0]);
                })
                .attr("width", function(d) {
                    return Math.max(0, xScale(d[1]) - xScale(d[0]));
                });

            layers.selectAll("text")
                .data(function(d) { return d; })
                .attr("x", function(d) { return xScale(d[0]) + (xScale(d[1]) - xScale(d[0])) / 2; })
                .each(function(d) {
                    this.textContent = d.key;

                    // 勘定科目名がバーに収まらなければ表示しない
                    if (xScale(d[1]) - xScale(d[0]) < this.getComputedTextLength()) {
                        this.textContent = "";
                    }
                });
        }

        return chart;
    }

    return plChart;
})();
