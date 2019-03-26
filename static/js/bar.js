'use strict';

var drawBarChart;

(function () {
var margin = { top: 50, right: 10, bottom: 10, left: 80 };
var barHeight = 18;
var width = 1000 - margin.left - margin.right;

function getStacked(keys, data) {
    var stack = d3.stack().keys(keys);
    return stack(data);
}

function getMaxX(stacked) {
    var numKeys = stacked.length;
    var lastXs = stacked[numKeys-1].map(function(d) { return d[1]; });
    return d3.max(lastXs);
}

function enterLayer(svg, x, y, stacked, className, color, extraY) {
    var layers = svg.selectAll('g.' + className)
        .data(stacked, function(d) { return d.key; })
        .enter()
        .append('g')
        .attr('class', className)
        .attr('fill', function(d) { return color(d.key); })
        .each(function (d) {
            for (var i = 0; i < d.length; i++) {
                d[i].key = d.key;
            }
        });

    layers.selectAll('rect')
        .data(function(d) { return d; })
        .enter()
        .append('rect')
        .attr('y', function(d) { return y(d.data.month) + extraY; })
        .attr('height', barHeight)
        .attr('x', function(d) {
            return x(d[0]);
        })
        .attr('width', function(d) {
            return Math.max(0, x(d[1]) - x(d[0]));
        });

    layers.selectAll('text')
        .data(function(d) { return d; })
        .enter()
        .append('text')
        .attr('x', function(d) { return x(d[0]) + (x(d[1]) - x(d[0])) / 2; })
        .attr('y', function(d) { return y(d.data.month) + extraY + barHeight - 5; })
        .attr('fill', function(d) { return 'black'; })
        .style('font-size', '12px')
        .style('text-anchor', 'middle')
        .text(function(d) { return d.key; })
        .each(function(d) {
            // 勘定科目名がバーに収まらなければ表示しない
            if (x(d[1]) - x(d[0]) < this.getComputedTextLength()) {
                this.textContent = '';
            }
        });
}

drawBarChart = function (svgSelection, startMonth, endMonth, numMonths) {
d3.json('/api/summary/' + startMonth + '/' + endMonth).then(function(data) {
    var height = (barHeight * 2 + 10) * numMonths;

    var expenseStacked = getStacked(data.expense_keys, data.expense);
    var maxexpenseX = getMaxX(expenseStacked);

    var incomeStacked = getStacked(data.income_keys, data.income);
    var maxIncomeX = getMaxX(incomeStacked);

    var maxX = Math.max(maxexpenseX, maxIncomeX);

    var x = d3.scaleLinear()
        .domain([0, maxX])
        .range([0, width]);

    var y = d3.scaleBand()
        .padding(0.1)
        .domain(data.expense.map(function(d) { return d.month; }))
        .range([0, height]);

    var svg = d3.select(svgSelection)
        .attr('width', width + margin.left + margin.right)
        .attr('height', height + margin.top + margin.bottom)
        .append('g')
        .attr('transform', 'translate(' + margin.left + ',' + margin.top + ')');

    var xAxis = d3.axisTop(x);

    svg.append('g')
        .attr('class', 'x axis')
        .call(xAxis);

    var yAxis = d3.axisLeft(y);

    svg.append('g')
        .attr('class', 'y axis')
        .call(yAxis);

    var expenseColor = d3.scaleOrdinal(d3.schemeSet3);
    var incomeColor = d3.scaleOrdinal(d3.schemePastel1);

    data.expense_keys.forEach(function (d) {
        expenseColor(d);
    });

    data.income_keys.forEach(function (d) {
        incomeColor(d);
    });

    enterLayer(svg, x, y, incomeStacked, 'incomeLayers', incomeColor, 0);
    enterLayer(svg, x, y, expenseStacked, 'expenseLayers', expenseColor, barHeight);
});
};

})();
