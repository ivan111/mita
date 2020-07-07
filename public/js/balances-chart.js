'use strict';

var balancesChart = timeSeriesChart()
    .prefix("balances")
    .x(function(d) { return timeParser(d.month); })
    .y(function(d) { return +d.balance; })
    .xText(function(d) {
        var d = timeParser(d.month);
        return d.getFullYear() + "年" + (d.getMonth()+1) + "月";
    })
    .yText(function(d) { return (+d.balance).toLocaleString(); })
    .fitWidth();

d3.json('/api/balances').then(function(data) {
    d3.select("#balances-chart")
        .datum(data)
        .call(balancesChart);
});
