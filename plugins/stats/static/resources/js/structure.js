function inputDataChanged(data_id, showOthers) {
	$("#inputdata").val(data_id).trigger("liszt:updated");
	initScreen(foset[data_id], showOthers);
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
	
	// plot graphs
	for (data of showData["order"]) {
		var block = undefined;
		try {
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
				get_horizontal_graph(block.find("canvas")[0].getContext('2d'), g_labels, g_data, formatters[g_format]);
				block.find("p.description").text(showData["data"][data]["desc"])
				block.find("h2").text(showData["data"][data]["title"])
			}
		} catch(err) {
			console.error("Error when processing \"" + data + "\" graph: ", err);
		}
	
		$("div #" + showData["data"][data]["tab"]).append(block);
	}
	
	$("#tabs").tabs({hide: { effect: 'fade', duration: 100 }, show: {effect: 'fade', duration: 100} });
	$("#tabs").tabs({ active: last_active_tab });
	$("#tabs").show({effect: 'fade', duration: 100});
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

		option += " | file \"" + foset[i].info.filename + "\"";
		option += " | config \"" + foset[i].info.plugin_config + "\"";
		option += " | filter \"" + foset[i].info.filter + "\"";
		option += "</option>";
		inputdata.append($.parseHTML(option));
	}
}