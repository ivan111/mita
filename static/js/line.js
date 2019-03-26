'use strict';

var MARGIN = { TOP: 50, RIGHT: 10, BOTTOM: 10, LEFT: 80 };
var WIDTH = 1000 - MARGIN.LEFT - MARGIN.RIGHT;
var HEIGHT = 200 - MARGIN.TOP - MARGIN.BOTTOM;

var dt = new Date();
var end_month = dt.getFullYear() * 100 + dt.getMonth() + 1;
dt.setMonth(dt.getMonth() - 12);
var start_month = dt.getFullYear() * 100 + dt.getMonth() + 1;

d3.json('/api/bp/' + start_month + '/' + end_month).then(function(data) {
    var parseMonth = d3.timeParse('%Y%m');

    data.forEach(function(d) {
        d.month = parseMonth(d.month);
    });

    var x = d3.scaleTime()
        .domain(d3.extent(data, function(d) { return d.month; }))
        .range([0, WIDTH]);

    var max_balance = d3.max(data, function(d) { return Math.abs(d.balance); });

    var y = d3.scaleLinear()
        .domain([-max_balance, max_balance])
        .range([HEIGHT, 0]);

    var value_line = d3.line()
        .x(function(d) { return x(d.month); })
        .y(function(d) { return y(d.balance); });

    var svg = d3.select('svg.line-chart')
        .attr('width', WIDTH + MARGIN.LEFT + MARGIN.RIGHT)
        .attr('height', HEIGHT + MARGIN.TOP + MARGIN.BOTTOM)
        .append('g')
        .attr('transform', 'translate(' + MARGIN.LEFT + ',' + MARGIN.TOP + ')');

    var xAxis = d3.axisTop(x)
        .tickFormat(d3.timeFormat("%Y-%m"));

    svg.append('g')
        .attr('class', 'x axis')
        .call(xAxis);

    var yAxis = d3.axisLeft(y)
        .ticks(7)
        .tickSize(-WIDTH);

    svg.append('g')
        .attr('class', 'y axis')
        .call(yAxis);

    svg.append('path')
        .data([data])
        .attr('class', 'line')
        .attr('d', value_line)
        .attr('fill', 'none')
        .attr('stroke', 'steelblue')
        .attr('stoke-width', 2);
});
