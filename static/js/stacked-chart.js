'use strict';

var TYPE_ASSETS = 1;
var TYPE_LIABILITIES = 2;
var TYPE_INCOME = 3;
var TYPE_EXPENSE = 4;

var StackedChart;

(function () {
StackedChart = function (selection) {

var WIDTH = 1400;
var ROW_HEIGHT = 60;
var ROWS_NUM = 12;

var margin = { top: 40, left: 100, bottom: 35, right: 50 };

var dataset = null; // dataset
var width = WIDTH;
var isAutoMaxAmount = true;
var maxAmount = 0;
var filterAccountId = null;
var showTypes = [TYPE_INCOME, TYPE_EXPENSE];
var messageBox = null;
var rowsNum = ROWS_NUM;

var startRow = 0;

function my() {
}

my.update = function () {
    var ds = getDataSubset(dataset);

    var w = width - margin.left - margin.right;
    var h = Math.min(rowsNum, ds.length) * ROW_HEIGHT;

    var height = h + margin.top + margin.bottom;

    selection.selectAll('svg.stacked-chart').remove();

    var svg = selection.append("svg")
        .attr("width", width)
        .attr("height", height)
        .attr("class", "stacked-chart")
        .append("g")
        .attr("transform", "translate(" + margin.left + "," + margin.top + ")");

    // calcTotalMaxAmount や setXYWH で使われる値を設定する
    setPrevTotalAmount(ds);

    if (isAutoMaxAmount) {
        maxAmount = calcTotalMaxAmount(ds);
    }

    var xScale = d3.scaleLinear()
        .domain([0, maxAmount])
        .range([0, w]);

    var yScale = d3.scaleBand()
        .domain(ds.map(function (d) { return d.date; }))
        .range([0, h])
        .padding(.2);

    var barHeight = yScale.bandwidth() / 2;

    setXYWH(ds, xScale, yScale, w, barHeight);

    drawBrush(svg, h);

    drawAxes(svg, xScale, yScale, w, h);

    drawBars(ds, svg);
}

function update() {
}

// dataset の中から表示に必要な情報のみ取り出す
function getDataSubset(dataset) {
    var i;
    var ds = dataset.slice(startRow, Math.min(startRow + rowsNum, dataset.length));
    var res = [];

    for (i = 0; i < Math.min(ds.length, rowsNum); i++) {
        var data = ds[i];

        var amounts = data.amounts.filter(function (d) {
            return (!filterAccountId || filterAccountId == d.accountId) &&
                d.amount != 0 && isShowType(d.accountType);
        });

        res.push({
            date: data.date,
            amounts: amounts
        });
    }

    return res;
}

function setPrevTotalAmount(ds) {
    var i, k;

    for (i = 0; i < ds.length; i++) {
        var data = ds[i];

        // 資産と負債、もしくは収入と費用の積み上がりを保存するための配列
        var prevTotalAmounts = [0, 0];

        for (k = 0; k < data.amounts.length; k++) {
            var d = data.amounts[k];

            var idx = (d.amount < 0) ? 0 : 1;

            d.prevTotalAmount = prevTotalAmounts[idx];

            prevTotalAmounts[idx] += Math.abs(d.amount);
        }
    }
}

function iterDatasetAmounts(ds, callback) {
    var i, k;

    for (i = 0; i < ds.length; i++) {
        var data = ds[i];

        for (k = 0; k < data.amounts.length; k++) {
            var d = data.amounts[k];

            callback(d, k, data, i);
        }
    }
}

function calcTotalMaxAmount(ds) {
    var maxAmount = 0;

    iterDatasetAmounts(ds, function (d) {
        var v = d.prevTotalAmount + Math.abs(d.amount);

        if (v > maxAmount) {
            maxAmount = v;
        }
    });

    return maxAmount;
}

function setXYWH(ds, x, y, w, barHeight) {
    iterDatasetAmounts(ds, function (d, k, data) {
        if (filterAccountId == d.accountId) {
            d.x = 0;
        } else {
            d.x = x(d.prevTotalAmount);
        }

        d.w = x(Math.abs(d.amount));

        // バーが横にはみ出してたら直す
        if (d.x >= w) {
            d.w = 0;
        } else if (d.x + d.w > w) {
            d.w = w - d.x;
        }

        d.y = y(data.date);

        if (d.amount > 0) {
            d.y += barHeight;
        }

        d.h = barHeight;
    });
}

function getAllAmounts(ds) {
    var amounts = [];
    var i;

    for (i = 0; i < ds.length; i++) {
        amounts = amounts.concat(ds[i].amounts);
    }

    return amounts;
}

var BrushWidth = 20;

function drawBrush(svg, h) {
    if (dataset.length <= rowsNum) {
        return;
    }

    var yScale = d3.scaleLinear()
        .domain([0, dataset.length])
        .range([0, h]);

    var yg = svg.append("g")
        .attr('transform', function (d) {
            return ['translate(', -margin.left + 2, ',0)'].join('');
        });

    yg.append('rect')
        .attr('x', 0)
        .attr('y', 0)
        .attr('width', BrushWidth)
        .attr('height', h)
        .attr('fill', '#eee')
        .attr('stroke', 'black')
        .attr('stroke-width', 1);

    var yBrush = d3.brushY()
        .extent([[0, 0], [BrushWidth, h]]);

    yBrush(yg);
    var endRow = Math.max(0, Math.min(startRow + rowsNum, dataset.length));
    yBrush.move(yg, [yScale(startRow), yScale(endRow)]);

    yBrush.on('start', function () {
        var brushPos = d3.brushSelection(yg.node());

        if (brushPos[0] == brushPos[1]) {
            var newStartRow = Math.round(yScale.invert(brushPos[0]));
            newStartRow = Math.max(0, Math.min(newStartRow - ROWS_NUM / 2, dataset.length - rowsNum));

            if (newStartRow == startRow) {
                return;
            }

            startRow = newStartRow;
            my.update();
        }
    });

    yBrush.on('end', function () {
        var brushPos = d3.brushSelection(yg.node());

        var newStartRow = Math.round(yScale.invert(brushPos[0]));
        newStartRow = Math.max(0, Math.min(newStartRow, dataset.length - rowsNum));

        if (newStartRow == startRow) {
            return;
        }

        startRow = newStartRow;
        my.update();
    });

    yg.selectAll('.overlay')
        .style("cursor", "default");
}

function drawAxes(svg, xScale, yScale, w, h) {
    var xAxis = d3.axisTop(xScale);

    svg.append("g")
        .attr("class", "x axis")
        .call(xAxis);

    var yAxis = d3.axisLeft(yScale);

    svg.append("g")
        .attr("class", "y axis")
        .call(yAxis);
}

var numFormat = d3.format(",");

function drawBars(ds, svg) {
    var amounts = getAllAmounts(ds);

    var bars = svg.selectAll(".bar")
        .data(amounts)
        .enter()
        .append("g")
        .attr("class", "bar")
        .attr('transform', function (d) {
            return ['translate(', d.x, ', ', d.y, ')'].join('');
        });

    bars.append("rect")
        .attr("width", function(d) { return d.w; })
        .attr("height", function(d) { return d.h; })
        .style("fill", function(d) { return d.color; })
        .style("stroke", "#444")
        .style("stroke-width", "1")
        .on("click", function(d) { filterAccountId = d.accountId; my.update(); })
        .on("mouseover", function(d) {
            if (messageBox) {
                messageBox.attr('value', d.accountName + ' ' + numFormat(Math.abs(d.amount)));
            }
        })
        .on("mouseout", function() {
            if (messageBox) {
                messageBox.attr('value', '');
            }
        });

    bars.append("text")
        .attr("x", function (d) { return d.w / 2; })
        .attr("y", function (d) { return d.h / 2 + 4; })
        .style("font-size", "12px")
        .style("text-anchor", "middle")
        .text(function(d) { return d.accountName; })
        .each(function(d) {
            // 勘定科目名がバーに収まらなければ表示しない
            if (d.w < this.getComputedTextLength()) {
                this.textContent = '';
            }
        });
}

function isShowType(type) {
    return showTypes.indexOf(type) != -1;
}

my.dataset = function(value) {
    if (!arguments.length) return dataset;
    dataset = value;
    startRow = 0;
    return my;
};

my.width = function(value) {
    if (!arguments.length) return width;
    width = value;
    return my;
};

my.isAutoMaxAmount = function(value) {
    if (!arguments.length) return isAutoMaxAmount;
    isAutoMaxAmount = value;
    return my;
};

my.maxAmount = function(value) {
    if (!arguments.length) return maxAmount;
    isAutoMaxAmount = false;
    maxAmount = value;
    return my;
};

my.filterAccountId = function(value) {
    if (!arguments.length) return filterAccountId;
    filterAccountId = value;
    return my;
};

my.showTypes = function(value) {
    if (!arguments.length) return showTypes;
    showTypes = value;
    filterAccountId = null;
    return my;
};

my.messageBox = function(value) {
    if (!arguments.length) return messageBox;
    messageBox = value;
    return my;
};

return my;
}
})();
