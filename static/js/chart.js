'use strict';

var chart;
var monthlyDs;
var yearlyDs;

d3.tsv("/accounts.tsv").then(function(accounts) {
    d3.tsv("/monthly_amount.tsv").then(function(data) {
        var ds = createDataset(data, accounts);
        ds.reverse();

        setColor(ds, TYPE_ASSETS, TYPE_LIABILITIES);
        setColor(ds, TYPE_INCOME, TYPE_EXPENSE);

        monthlyDs = ds;
        yearlyDs = monthly2yearly(ds);

        chart = StackedChart(d3.select("#chart"))
            .messageBox(d3.select("#msg"))
            .dataset(monthlyDs)
            .showTypes([TYPE_INCOME, TYPE_EXPENSE]);

        if (DEFAULT_MAX_AMOUNT) {
            chart.maxAmount(DEFAULT_MAX_AMOUNT);
        } else {
            chart.isAutoMaxAmount(true);
        }

        chart.update();
    });
});

function createDataset(data, accounts) {
    var ds = [];

    var i, k;

    for (i = 0; i < data.length; i++) {
        var d = data[i];
        var amounts = [];

        for (k = 0; k < accounts.length; k++) {
            var account = accounts[k];

            amounts.push({
                accountId: Number(account.id),
                accountType: Number(account.type),
                accountName: account.name,
                amount: Number(d[account.id])
            });
        }

        ds.push({
            date: d.ym,
            amounts: amounts
        });
    }

    return ds;
}

function setColor(ds, type1, type2) {
    var i, k;
    var colors = d3.scaleOrdinal(d3.schemePastel1.concat(d3.schemePastel2));
    var id2color = {};
    var colorIndex = 0;

    for (i = 0; i < ds.length; i++) {
        var data = ds[i];

        for (k = 0; k < data.amounts.length; k++) {
            var d = data.amounts[k];

            if (d.accountType == type1 || d.accountType == type2) {
                if (!(d.accountId in id2color)) {
                    id2color[d.accountId] = colors(colorIndex++);
                }

                d.color = id2color[d.accountId];
            }
        }
    }
}

function monthly2yearly(ds) {
    var res = [];

    if (ds.length == 0) {
        return res;
    }

    var i, k, m;
    var curYear = '#####';
    var amounts;

    for (i = 0; i < ds.length; i++) {
        var data = ds[i];
        var year = data.date.slice(0, 4);

        if (curYear != year) {
            amounts = [];

            res.push({
                date: year,
                amounts: amounts
            });

            curYear = year;
        }

        for (k = 0; k < data.amounts.length; k++) {
            var d = data.amounts[k];

            if (amounts[k]) {
                // 集計するのは収入・費用だけでいい。
                // 資産・負債は、最後の月の値にする。
                // データセットはで新しい順になっているので、先頭の値のままでいい。
                var type = amounts[k].accountType;

                if (type == TYPE_INCOME || type == TYPE_EXPENSE) {
                    amounts[k].amount += d.amount;
                }
            } else {
                amounts[k] = JSON.parse(JSON.stringify(d));
            }
        }
    }

    return res;
}

function setBS() {
    var a = chart.showTypes();

    if (a.indexOf(TYPE_ASSETS) != -1 &&
        a.indexOf(TYPE_LIABILITIES) != -1) {
        return;
    }

    chart.showTypes([TYPE_ASSETS, TYPE_LIABILITIES]);

    chart.update();
}

function setPL() {
    var a = chart.showTypes();

    if (a.indexOf(TYPE_INCOME) != -1 &&
        a.indexOf(TYPE_EXPENSE) != -1) {
        return;
    }

    chart.showTypes([TYPE_INCOME, TYPE_EXPENSE]);

    chart.update();
}

function setIncome() {
    var a = chart.showTypes();

    if (a.length == 1 && a[0] == TYPE_INCOME) {
        return;
    }

    chart.showTypes([TYPE_INCOME]);

    chart.update();
}

function setExpense() {
    var a = chart.showTypes();

    if (a.length == 1 && a[0] == TYPE_EXPENSE) {
        return;
    }

    chart.showTypes([TYPE_EXPENSE]);

    chart.update();
}

function setAssets() {
    var a = chart.showTypes();

    if (a.length == 1 && a[0] == TYPE_ASSETS) {
        return;
    }

    chart.showTypes([TYPE_ASSETS]);

    chart.update();
}

function setLiabilities() {
    var a = chart.showTypes();

    if (a.length == 1 && a[0] == TYPE_LIABILITIES) {
        return;
    }

    chart.showTypes([TYPE_LIABILITIES]);

    chart.update();
}

function setMonthly() {
    chart.dataset(monthlyDs)
        .update();
}

function setYearly() {
    chart.dataset(yearlyDs)
        .update();
}

function setAutoMaxAmount() {
    if (chart.isAutoMaxAmount()) {
        return;
    }

    chart.isAutoMaxAmount(true);

    chart.update();
}

function changeMaxAmount(maxAmount) {
    if (chart.maxAmount() == maxAmount) {
        return;
    }

    chart.maxAmount(maxAmount);

    chart.update();
}

function unsetAccountFilter() {
    if (!chart.filterAccountId()) {
        return;
    }

    chart.filterAccountId(null);

    chart.update();
}
