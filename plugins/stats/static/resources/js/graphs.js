// GraphJS default values
Chart.defaults.global.plugins.datalabels.anchor = 'end';
Chart.defaults.global.plugins.datalabels.align = 'right';
Chart.defaults.global.plugins.datalabels.offset = 0;
Chart.defaults.global.plugins.datalabels.padding.top = 0;
Chart.defaults.global.plugins.datalabels.padding.bottom = 0;
Chart.defaults.global.plugins.datalabels.padding.left = 5;
Chart.defaults.global.plugins.datalabels.padding.right = 5;
Chart.defaults.global.plugins.datalabels.borderWidth = 1;
Chart.defaults.global.plugins.datalabels.borderColor = 'black';
Chart.defaults.global.plugins.datalabels.font.weight = 'bold';
Chart.defaults.global.plugins.datalabels.backgroundColor = 'white';

function graph_click(e, name) {
	var obj = existing_graphs[name];
	var yIndex = obj.scales['y-axis-0'].getValueForPixel(e.offsetY);
	var xIndex = obj.scales['x-axis-0'].getValueForPixel(e.offsetX);
	// no label means an empty line that we don't copy
	if (obj.data.labels[yIndex].length == 0) return;
	// negative xIndex is a label click, positive is a bar click
	var clipboard;
	if (xIndex < 0) {
		clipboard = obj.data.labels[yIndex];
	} else {
		clipboard = obj.options.plugins.datalabels.formatter(obj.data.datasets[0].data[yIndex]);
	}
	if (navigator.clipboard) {
		navigator.clipboard.writeText(clipboard);
		$(obj.canvas).notify("\"" + clipboard + "\" copied to the clipboard.", { position: "top left", autoHideDelay: 1000, className: "success"});
	} else {
		$(obj.canvas).notify("Unable to copy to clipboard.", { position: "top left", autoHideDelay: 1000, className: "error"});
	}
}

// Create graphs
function get_horizontal_graph(ctx, labels, data, formatter, name) {
	var myChart = new Chart(ctx, {
		type: 'horizontalBar',
		data: {
			labels: labels,
			datasets: [{
				barPercentage: 1.0,
				categoryPercentage: 1.0,
//				barThickness : 15,
				data: data,
				backgroundColor: palette('mpn65', data.length).map(function(hex) { return '#' + hex; }),
				borderWidth: 1
			}]
		},

		options: {
			onClick: (function(x_name) {
				return function(e) {
					graph_click(e, x_name);
				}
			})(name),
			animation: {
				duration: 0,
			},
			plugins: {
				datalabels: {
					formatter: formatter,
				},
			},
			layout: {
				padding: {
					left: 0,
					right: 100,
					top: 0,
					bottom: 0
				}
			},
			responsive: true,
			maintainAspectRatio: true,
			legend: {
				display: false
			},
			scales: {
				xAxes: [{
					display: false,
				}],
				yAxes: [{
					ticks: {
						beginAtZero: true,
  					}
				}]
			}
		}
	});

	return myChart;
}

// formatters
var formatters = Object();
formatters["number"] = function(value, context) {
	n = parseInt(value);
	return n.toLocaleString();
}
formatters["rate"] = function(value, context) {
	n = parseInt(value)*8;
	if (n < 1000) return n + " bps"
	if (n < 1000*1000) return (n/1000).toFixed(1) + " kbps"
	if (n < 1000*1000*1000) return (n/1000/1000).toFixed(1) + " Mbps"
	return (n/1000/1000/1000).toFixed(1) + " Gbps"
}
formatters["size"] = function(value, context) {
	n = parseInt(value);
	if (n < 1000) return n + " B"
	if (n < 1000*1000) return (n/1000).toFixed(1) + " kB"
	if (n < 1000*1000*1000) return (n/1000/1000).toFixed(1) + " MB"
	if (n < 1000*1000*1000*1000) return (n/1000/1000/1000).toFixed(1) + " GB"
	return (n/1000/1000/1000/1000).toFixed(1) + " TB"
}
