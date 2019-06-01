'use strict';

var NUM_MONTHS = 60;

var BAR_WIDTH = 12;
var MARGIN = { TOP: 50, RIGHT: 10, BOTTOM: 10, LEFT: 80 };
var WIDTH = (BAR_WIDTH + 4) * NUM_MONTHS - MARGIN.LEFT - MARGIN.RIGHT;
var HEIGHT = 200 - MARGIN.TOP - MARGIN.BOTTOM;

var dt = new Date();
var end_month = dt.getFullYear() * 100 + dt.getMonth() + 1;
dt.setDate(1);
dt.setMonth(dt.getMonth() - NUM_MONTHS + 1);
var start_month = dt.getFullYear() * 100 + dt.getMonth() + 1;

d3.json('/api/bp/' + start_month + '/' + end_month).then(function(data) {
    var parseMonth = d3.timeParse('%Y%m');

    data.forEach(function(d) {
        d.month = parseMonth(d.month);
    });

    var balances = data.map(function(d) { return d.balance; });
    var IQR = d3.quantile(balances, 0.75) - d3.quantile(balances, 0.25);
    // 外れ値
    var threshold = Math.abs(d3.quantile(balances, 0.75) + IQR) * 1.5;

    var x = d3.scaleBand()
        .padding(0.1)
        .domain(data.map(function(d) { return d.month; }))
        .range([0, WIDTH]);

    var max_balance = d3.max(data.filter(function(d) { return Math.abs(d.balance) < threshold; }),
        function(d) { return Math.abs(d.balance); });

    var y = d3.scaleLinear()
        .domain([-max_balance, max_balance])
        .range([HEIGHT, 0]);

    var svg = d3.select('svg.line-chart')
        .attr('width', WIDTH + MARGIN.LEFT + MARGIN.RIGHT)
        .attr('height', HEIGHT + MARGIN.TOP + MARGIN.BOTTOM)
        .append('g')
        .attr('transform', 'translate(' + MARGIN.LEFT + ',' + MARGIN.TOP + ')');

    var xAxis = d3.axisTop(x)
        .tickValues(x.domain().filter(function(d,i) { return !(i%6)}))
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

    svg.selectAll('rect')
        .data(data)
        .enter()
        .append('rect')
        .attr('x', function(d) { return x(d.month); })
        .attr('y', function(d) {
            if (d.balance >= 0) {
                if (Math.abs(d.balance) < threshold) {
                    return y(d.balance);
                } else {
                    return 0;
                }
            } else {
                return HEIGHT / 2;
            }})
        .attr('width', BAR_WIDTH)
        .attr('height', function(d) {
            if (d.balance >= 0) {
                if (Math.abs(d.balance) < threshold) {
                    return HEIGHT / 2 - y(d.balance);
                } else {
                    return HEIGHT / 2;
                }
            } else {
                if (Math.abs(d.balance) < threshold) {
                    return y(d.balance) - HEIGHT / 2;
                } else {
                    return y(-max_balance) - HEIGHT / 2;
                }
            }})
        .attr('fill', function(d) {
            if (d.balance >= 0) {
                if (Math.abs(d.balance) < threshold) {
                    return 'steelblue';
                } else {
                    return 'deepskyblue';
                }
            } else {
                if (Math.abs(d.balance) < threshold) {
                    return 'crimson';
                } else {
                    return 'deeppink';
                }
            }});
});
