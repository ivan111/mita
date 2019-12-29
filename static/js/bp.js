'use strict';

(function () {

var NUM_MONTHS = 90;

var BAR_WIDTH = 10;
var BAR_MARGIN = 1;

var MARGIN = { TOP: 50, RIGHT: 10, BOTTOM: 10, LEFT: 80 };
var WIDTH = (BAR_WIDTH + BAR_MARGIN) * NUM_MONTHS - MARGIN.LEFT - MARGIN.RIGHT;
var HEIGHT = 300 - MARGIN.TOP - MARGIN.BOTTOM;

d3.json('/api/bp').then(function(data) {
    var vWidth = (BAR_WIDTH + BAR_MARGIN) * data.length;
    var maxTranslateX = Math.max(vWidth - WIDTH, 0);

    parseMonth(data);

    var threshold = getThreshold(data)

    var x = d3.scaleBand()
        .padding(0.1)
        .domain(data.map(function(d) { return d.month; }))
        .range([0, (BAR_WIDTH + BAR_MARGIN) * data.length]);

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

    var stage = svg.append('g');

    stage.append('rect')
        .attr('x', 0)
        .attr('y', 0)
        .attr('width', (BAR_WIDTH + BAR_MARGIN) * data.length)
        .attr('height', HEIGHT)
        .attr('fill', 'white');

    var xAxis = d3.axisTop(x)
        .tickValues(x.domain().filter(function(d,i) { return !(i%6)}))
        .tickFormat(d3.timeFormat('%Y-%m'));

    stage.append('g')
        .attr('class', 'x axis')
        .call(xAxis);

    var yAxis = d3.axisLeft(y)
        .ticks(7)
        .tickSize(-WIDTH);

    var gYAxis = svg.append('g');

    gYAxis.append('rect')
        .attr('x', -MARGIN.LEFT)
        .attr('y', -MARGIN.TOP)
        .attr('width', MARGIN.LEFT)
        .attr('height', 300)
        .attr('fill', 'white');

    gYAxis.append('g')
        .attr('class', 'y axis')
        .call(yAxis);

    stage.selectAll('rect.bp-bar')
        .data(data)
        .enter()
        .append('rect')
        .attr('class', 'bp-bar')
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

    var zoom = d3.zoom()
        .on('zoom', function(){
            var x = d3.event.transform.x;

            if (x < -maxTranslateX) {
                x = -maxTranslateX;
                d3.event.transform.x = -maxTranslateX;
            } else if (x > 0) {
                x = 0;
                d3.event.transform.x = 0;
            }

            stage.attr('transform', 'translate(' + x + ',0)');
        });

    svg.call(zoom)
        .call(zoom.transform, d3.zoomIdentity.translate(-maxTranslateX, 0));
});

function parseMonth(data) {
    var parser = d3.timeParse('%Y%m');

    data.forEach(function(d) {
        d.month = parser(d.month);
    });
}

function getThreshold(data) {
    var balances = data.map(function(d) { return d.balance; });
    balances = balances.sort(function(a, b){return a-b});

    var Q1 = d3.quantile(balances, 0.25);
    var Q3 = d3.quantile(balances, 0.75);

    var IQR = Q3 - Q1;
    if (Q3 > 0 && Q1 < 0) {
        IQR = Math.max(Q3, Math.abs(Q1));
    }

    // 外れ値
    return Math.abs(Q3 + IQR) * 2;
}

})();
