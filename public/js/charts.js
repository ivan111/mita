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

var myPLChart = plChart();

d3.json("/api/pl").then(function(data) {
    d3.select("#pl-chart")
        .datum(data)
        .call(myPLChart);
});

var isCurCash = false;
var curShowExtra = false;

function changeMode(isCash, showExtra) {
    if (isCash == null) {
        isCash = isCurCash;
    }

    if (showExtra == null) {
        showExtra = curShowExtra;
    }

    if (isCash != isCurCash || showExtra != curShowExtra) {
        isCurCash = isCash;
        curShowExtra = showExtra;

        var params = "";

        if (isCash) {
            params = "?cash=true";
        }

        if (showExtra) {
            if (params == "") {
                params = "?";
            } else {
                params += "&";
            }

            params += "extraordinary=true";
        }

        d3.json("/api/balances" + params).then(function(data) {
            d3.select("#balances-chart")
                .datum(data)
                .call(balancesChart);
        });

        d3.json("/api/pl" + params).then(function(data) {
            d3.select("#pl-chart")
                .datum(data)
                .call(myPLChart);
        });
    }
}
