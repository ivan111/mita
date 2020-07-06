'use strict';

var assetsChart = timeSeriesChart()
    .prefix("assets")
    .x(function(d) { return timeParser(d.month); })
    .y(function(d) { return +d.balance; })
    .xText(function(d) {
        var d = timeParser(d.month);
        return d.getFullYear() + "年" + (d.getMonth()+1) + "月";
    })
    .yText(function(d) { return (+d.balance).toLocaleString(); })
    .fitWidth();

var timeParser = d3.timeParse("%Y%m");

d3.json('/api/assets').then(function(data) {
    d3.select("#assets-chart")
        .datum(data)
        .call(assetsChart);
});
