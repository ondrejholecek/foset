function inputDataChanged(data_id, showOthers) {
	$("#inputdata").val(data_id).trigger("liszt:updated");
	plotGraphs(foset[data_id], showOthers);
}

function initScreen(showData, showOthers) {
	// delete all old data
	var last_active_tab = $("#tabs").tabs('option', 'active');
	$("#tabs").hide({effect: 'fade', duration: 100});
	$("#tabs").tabs("destroy");
	$("#tabs").empty();
	$("#tabs").append($.parseHTML("<ul></ul>"));

	// create tabs
	for (tab of showData["tabs"]) {
		var li  = $.parseHTML("<li><a href=\"#" + tab.id + "\">" + tab.title + "</a></li>");
		var div = $.parseHTML("<div id=\"" + tab.id + "\" class=\"graph-tab\"></div>");
	
		$("#tabs ul").append(li);
		$("#tabs").append(div);
		$(div).css("display", "grid");
	}

	// create graph elements
	for (data of showData["order"]) {
		var block = undefined;
		var div = document.createElement("div");
		div.setAttribute('id', data);
		div.setAttribute('class', 'graphContainer');
		if (data.startsWith("space_")) { div.className += " empty" };
		div.appendChild(document.createElement("h2"));
		var p = document.createElement("p");
		p.setAttribute('class', 'description');
		div.appendChild(p);
		var canvas = document.createElement("canvas");
		canvas.setAttribute('class', 'graph');
		div.appendChild(canvas);
		block = $(div);
		$("div #" + showData["data"][data]["tab"]).append(block);
	}
	
	//
	$("#tabs").tabs({hide: { effect: 'fade', duration: 100 }, show: {effect: 'fade', duration: 100} });
	$("#tabs").tabs({ active: last_active_tab });
	$("#tabs").show({effect: 'fade', duration: 100});
}

var existing_graphs = Object();
function plotGraphs(showData, showOthers) {
	// plot graphs
	for (data of showData["order"]) {
		var block = $("div #" + data);
		try {
			if (!data.startsWith("space_")) {
				// remove others?
				var g_labels = showData["data"][data]["labels"].slice();
				var g_data   = showData["data"][data]["data"].slice();
				var g_format = showData["data"][data]["format"].slice();

				if (g_labels[g_labels.length-1].endsWith(" others") && !showOthers) {
					g_labels.pop();
					g_data.pop();
				}

				// fill with some empty lines if there is not enough data
				while (g_labels.length < 10) {
					g_labels.push("");
				}

				//
				var canvas = block.find("canvas")[0];
				block.find("p.description").text(showData["data"][data]["desc"])
				block.find("h2").text(showData["data"][data]["title"])

				if (typeof existing_graphs[data] === 'undefined') {
					// first init
					existing_graphs[data] = get_horizontal_graph(canvas.getContext('2d'), g_labels, g_data, formatters[g_format]);
				} else {
					// redraw
					existing_graphs[data].data.labels = g_labels;
					existing_graphs[data].data.datasets[0].data = g_data;
					existing_graphs[data].update();
				}
			}
		} catch(err) {
			console.error("Error when processing \"" + data + "\" graph: ", err);
		}
	}
}

function loadData() {
	var inputdata = $('#inputdata');

	var language;
	if (window.navigator.languages) {
		language = window.navigator.languages[0];
	} else {
		language = window.navigator.userLanguage || window.navigator.language;
	}

	for (var i = 0; i < foset.length; i++) {
		var option = "<option value=\"" + i + "\">Generated: ";

		var d = new Date(foset[i].info.calculated*1000);
		option += d.toLocaleString(language);

		parseInt(foset[i].info.sessions_total).toLocaleString();

		option += " | file \"" + foset[i].info.filename + "\" [" + parseInt(foset[i].info.sessions_total).toLocaleString() + "]";
		option += " | filter [" + parseInt(foset[i].info.sessions_matched).toLocaleString() + "] \"" + foset[i].info.filter + "\"";
		//option += " | config \"" + foset[i].info.plugin_config + "\"";
		option += "</option>";
		inputdata.append($.parseHTML(option));
	}
}

var quickSwitch = Object();

function setupKeys() {
	document.onkeyup=function(e){
		if (e.ctrlKey && (e.key == "p" || e.key == "k")) {
			var to = parseInt($("#inputdata").val())-1;
			if (to < 0) return;
			$("#inputdata").val("" + to);
			$("#inputdata").trigger("chosen:updated");
			$("#inputdata").trigger("change");

			$('.notifyjs-corner').empty();
			$.notify("Dataset: " + $('#inputdata option[value='+to+']').text() + "", {className: "success", position: "top right", autoHideDelay: 1000});

		} else if (e.ctrlKey && (e.key == "n" || e.key == "j")) {
			var to = parseInt($("#inputdata").val())+1;
			if (to >= $("#inputdata")[0].length) return;
			$("#inputdata").val("" + to);
			$("#inputdata").trigger("chosen:updated");
			$("#inputdata").trigger("change");

			$('.notifyjs-corner').empty();
			$.notify("Dataset: " + $('#inputdata option[value='+to+']').text() + "", {className: "success", position: "top right", autoHideDelay: 1000});

		} else if (e.ctrlKey && e.key == "o") {
			if ($('#show-others').prop("checked")) {
				$('#show-others').prop("checked", false);
				$('.notifyjs-corner').empty();
				$.notify("Other records hidden", {className: "success", position: "top right", autoHideDelay: 1000});
			} else {
				$('#show-others').prop("checked", true);
				$('.notifyjs-corner').empty();
				$.notify("Showing other records", {className: "success", position: "top right", autoHideDelay: 1000});
			}
			$("#show-others").trigger("change");

		} else if (e.ctrlKey && e.key == "e") {
			$('#inputdata').trigger('chosen:open');

		} else if (e.key == "ArrowRight" || (e.ctrlKey && e.key == "l")) {
			var to = $("#tabs").tabs('option', 'active')+1;
			if (to >= $("#tabs >ul >li").length) to = 0;
			$("#tabs").tabs({active: to});

		} else if (e.key == "ArrowLeft" || (e.ctrlKey && e.key == "h")) {
			var to = $("#tabs").tabs('option', 'active')-1;
			if (to < 0) to = $("#tabs >ul >li").length - 1;
			$("#tabs").tabs({active: to});

		} else if (e.ctrlKey && e.altKey && e.key >= 1 && e.key <= 9) {
			// there can be only one quick key on dataset
			var setit = true;
			for (let [k, v] of Object.entries(quickSwitch)) {
				// if the current key is already used somewhere else
				// delete it
				if (e.key == k && v != $("#inputdata").val()) {
					delete quickSwitch[k];
				}
				// if the current key is used on the current object
				// delete it and forbid setting it
				if (e.key == k && v == $("#inputdata").val()) {
					delete quickSwitch[k];
					$('.notifyjs-corner').empty();
					$.notify("Dataset shortcut " + e.key + " removed from the current dataset.", {className: "success", position: "top right", autoHideDelay: 3000});
					setit = false;
				}
				// if another key is used on the current object
				// delete it
				if (e.key != k && v == $("#inputdata").val()) {
					delete quickSwitch[k];
				}
			}
			
			if (setit) {
				quickSwitch[e.key] = $("#inputdata").val();
				$('.notifyjs-corner').empty();
				$.notify("Dataset shortcut " + e.key + " set on the current dataset.", {className: "success", position: "top right", autoHideDelay: 3000});
			}

			// remove all key shortcuts from descriptions
			var re = /^(.*?)( \(\*[0-9]\))$/;
			for (option of $("#inputdata option")) {
				var x = re.exec($(option).text());
				if (!x) continue;
				$(option).text(x[1]);
			}

			// add all the current shortcuts
			for (let [k, v] of Object.entries(quickSwitch)) {
				var option = $('#inputdata option[value=' + v + ']');
				option.text(option.text() + " (*" + k + ")");
			}

			$("#inputdata").trigger("chosen:updated");
			$("#inputdata").trigger("change");


		} else if (e.ctrlKey && e.key >= 1 && e.key <= 9) {
			if (typeof quickSwitch[e.key] === 'undefined') return;
			$("#inputdata").val("" + quickSwitch[e.key]);
			$("#inputdata").trigger("chosen:updated");
			$("#inputdata").trigger("change");

			$('.notifyjs-corner').empty();
			$.notify("Dataset: " + $('#inputdata option[value='+quickSwitch[e.key]+']').text() + "", {className: "success", position: "top right", autoHideDelay: 1000});
		}
	}
}
