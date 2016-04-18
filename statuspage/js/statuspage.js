// TODO: Put into an external JSON file?
var config = {
	"timeframe": 1 * time.Day,
	"storage": {
		"AccessKeyID": "AKIAIQKTZO465CR5YMTQ",
		"SecretAccessKey": "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",
		"BucketName": "srcgraph-monitor-test"
	},
	"status_text": {
		"healthy": "Situation Normal",
		"degraded": "Minor Hiccup",
		"down": "Service Disruption"
	}
};

checkup.storage.setup(config.storage);

// Once the DOM is loaded, go ahead and render the graphs
// (if it hasn't been done already).
document.addEventListener('DOMContentLoaded', function() {
	checkup.domReady = true;

	checkup.dom.favicon = document.getElementById("favicon");
	checkup.dom.status = document.getElementById("overall-status");
	checkup.dom.statustext = document.getElementById("overall-status-text");
	checkup.dom.timeframe = document.getElementById("info-timeframe");
	checkup.dom.checkcount = document.getElementById("info-checkcount");
	checkup.dom.lastcheck = document.getElementById("info-lastcheck");
	checkup.dom.timeline = document.getElementById("timeline");

	if (!checkup.graphsMade) makeGraphs();
}, false);

// Immediately begin downloading check files, and keep page updated 
checkup.storage.getChecksWithin(config.timeframe, processNewCheckFile, allCheckFilesLoaded);
setInterval(function() {
	checkup.storage.getChecksWithin(60 * time.Second, processNewCheckFile, allCheckFilesLoaded);
}, 60000);


function processNewCheckFile(json, filename) {
	checkup.checks.push(json);

	for (var j = 0; j < json.length; j++) {
		var result = json[j];

		checkup.orderedResults.push(result); // will sort later, more efficient that way

		if (!checkup.results[result.timestamp])
			checkup.results[result.timestamp] = [result];
		else
			checkup.results[result.timestamp].push(result);

		var chart = checkup.charts[result.endpoint] || checkup.makeChart(result.title);
		chart.results.push(result);

		var stats = checkup.computeStats(result);
		var ts = new Date(result.timestamp * 1e-6);
		chart.series.min.push({ timestamp: ts, rtt: Math.round(stats.min) });
		chart.series.med.push({ timestamp: ts, rtt: Math.round(stats.median) });
		chart.series.max.push({ timestamp: ts, rtt: Math.round(stats.max) });
		chart.series.threshold.push({ timestamp: ts, rtt: result.threshold });

		checkup.charts[result.endpoint] = chart;

		for (var s in chart.series) {
			chart.series[s].sort(function(a, b) {
			  return a.timestamp - b.timestamp;
			});
		}

		if (!checkup.lastCheck || ts > checkup.lastCheck) {
			checkup.lastCheck = ts;
			checkup.dom.lastcheck.innerHTML = checkup.timeSince(checkup.lastCheck) + " ago";
		}
	}

	if (checkup.domReady)
		makeGraphs();
}

function allCheckFilesLoaded(numChecksLoaded) {
	checkup.orderedResults.sort(function(a, b) { return a.timestamp - b.timestamp; });

	// Create events for the timeline
	var statuses = {}; // keyed by endpoint
	for (var i = 0; i < checkup.orderedResults.length; i++) {
		var result = checkup.orderedResults[i];

		// TODO: Change status to a single field in the results struct?
		// Could make processing here a bit easier...
		var status = "healthy";
		if (result.degraded) status = "degraded";
		else if (result.down) status = "down";

		if (status != statuses[result.endpoint]) {
			// New event because status changed
			checkup.events.push({
				result: result,
				status: status
			});
		}
		if (result.message) {
			// New event because message posted
			checkup.events.push({
				result: result,
				status: status,
				message: result.message
			});
		}

		statuses[result.endpoint] = status;
	}

	function renderTime(ns) {
		var d = new Date(ns * 1e-6);
		var hours = d.getHours();
		var ampm = "AM";
		if (hours > 12) {
			hours -= 12;
			ampm = "PM";
		}
		return hours+":"+d.getMinutes()+" "+ampm;
	}

	// Render events
	// TODO: replace class color names with status names so we don't have to map like this
	var color = {healthy: "green", degraded: "yellow", down: "red"};
	for (var i = 0; i < checkup.events.length && i < numChecksLoaded; i++) {
		var e = checkup.events[checkup.events.length-i-1];

		var evtElem = document.createElement("div");
		if (e.message) {
			evtElem.className = "message "+color[e.status];
			evtElem.innerHTML = '<div class="message-head">'+checkup.timeSince(e.result.timestamp*1e-6)+'</div>';
			evtElem.innerHTML += '<div class="message-body">'+e.message+'</div>'; // TODO: Sanitize?
		} else {
			evtElem.className = "event "+color[e.status];
			// TODO: Even time should have the timeframe, like begin and end time (12:42 PM&mdash;12:45 PM)
			evtElem.innerHTML = '<span class="time">'+renderTime(e.result.timestamp)+'</span> '+e.result.title+" "+e.status;
		}
		checkup.dom.timeline.appendChild(evtElem);
	}

	// Update DOM now that we have the whole picture
	checkup.dom.favicon.href = "images/status-green.png";
	checkup.dom.status.className = "green"; // TODO: Color based on result of loading stats
	checkup.dom.statustext.innerHTML = config.status_text.healthy || "Healthy";

	var bigGap = false;
	var lastTimeDiff;
	for (var key in checkup.charts) {
		for (var k = 1; k < checkup.charts[key].results.length; k++) {
			var timeDiff = Math.abs(checkup.charts[key].results[k].timestamp - checkup.charts[key].results[k-1].timestamp);
			bigGap = lastTimeDiff && timeDiff > lastTimeDiff * 10;
			lastTimeDiff = timeDiff;
			if (bigGap)
			{
				document.getElementById("big-gap").style.display = 'block';
				break;
			}
		}
		if (bigGap) break;
	}
	if (!bigGap) {
		document.getElementById("big-gap").style.display = 'none';
	}
}

function makeGraphs() {
	checkup.dom.timeframe.innerHTML = checkup.formatDuration(config.timeframe);
	checkup.dom.checkcount.innerHTML = checkup.checks.length;

	if (!checkup.placeholdersRemoved && checkup.checks.length > 0) {
		// Remove placeholder to make way for the charts;
		// placeholder necessary to give space in absense of charts.
		if (phElem = document.getElementById("chart-placeholder"))
			phElem.remove();
		checkup.placeholdersRemoved = true;
	}

	for (var endpoint in checkup.charts)
		makeGraph(checkup.charts[endpoint]);

	checkup.graphsMade = true;
}

function makeGraph(chart) {
	// Render chart to page if first time seeing this endpoint
	if (!chart.elem) {
		renderChart(chart);
	}

	chart.xScale.domain([
		d3.min(chart.data, function(c) { return d3.min(c, function(d) { return d.timestamp; }); }),
		d3.max(chart.data, function(c) { return d3.max(c, function(d) { return d.timestamp; }); })
	]);
	chart.yScale.domain([
		0,
		d3.max(chart.data, function(c) { return d3.max(c, function(d) { return d.rtt; }); })
	]);

	chart.xAxis = d3.svg.axis()
		.scale(chart.xScale)
		.ticks(5)
		.outerTickSize(0)
		.orient("bottom");

	chart.yAxis = d3.svg.axis()
		.scale(chart.yScale)
		.tickFormat(checkup.formatDuration)
		.outerTickSize(0)
		.ticks(2)
		.orient("left");

	if (chart.svg.selectAll(".x.axis")[0].length == 0) {
		chart.svg.insert("g", ":first-child")
			.attr("class", "x axis")
			.attr("transform", "translate(0," + chart.height + ")")
			.call(chart.xAxis);
	} else {
		chart.svg.selectAll(".x.axis")
			.transition()
			.duration(checkup.animDuration)
			.call(chart.xAxis);
	}
	if (chart.svg.selectAll(".y.axis")[0].length == 0) {
		chart.svg.insert("g", ":first-child")
			.attr("class", "y axis")
			.call(chart.yAxis);
	} else {
		chart.svg.selectAll(".y.axis")
			.transition()
			.duration(checkup.animDuration)
			.call(chart.yAxis);
	}

	chart.lines = chart.lineGroup.selectAll(".line")
		.data(chart.data);

	// transition from old paths to new paths
	chart.lines
		.transition()
		.duration(checkup.animDuration)
		.attr("d", chart.line);

	// enter any new data
	chart.lines.enter()
	  .append("path")
		.attr("class", function(d) {
			if (d == chart.series.min) return "min line";
			else if (d == chart.series.med) return "main line";
			else if (d == chart.series.max) return "max line";
			else if (d == chart.series.threshold) return "tolerance line";
			else return "line";
		})
		.attr("d", chart.line);

	// exit any old data
	chart.lines
		.exit()
		.remove();
}


function renderChart(chart) {
	// Outer div is a wrapper that we use for layout
	var el = document.createElement('div');
	var containerSize = "grid-50";
	if (document.getElementsByClassName('chart-container').length == 0) {
		containerSize = "grid-100";
	} else {
		// It's possible that a chart was created that, at the time,
		// was the only one, but now it is too wide, since there are
		// at least two charts. Resize the wide one to be smaller.
		var tooWide = document.querySelector('.chart-container.grid-100');
		if (tooWide)
			tooWide.className = "chart-container grid-50";
	}
	el.className = "chart-container "+containerSize;

	// Div to contain the endpoint / title
	var el2 = document.createElement('div');
	el2.className = "chart-title";
	el2.appendChild(document.createTextNode(chart.title));
	el.appendChild(el2);

	// Inner div is used to contain the actual svg tag
	var el3 = document.createElement('div');
	el3.className = "chart";
	el.appendChild(el3);

	// Inject elements into DOM
	document.getElementById('chart-grid').appendChild(el);

	// Save it with the chart and use D3 to set up its svg element.
	chart.elem = el3;
	chart.svgTag = d3.select(chart.elem)
	  .append("svg")
	    .attr("id", chart.id)
		.attr("preserveAspectRatio", "xMinYMin meet")
		.attr("viewBox", "0 0 "+checkup.CHART_WIDTH+" "+checkup.CHART_HEIGHT);

	chart.margin = {top: 20, right: 20, bottom: 40, left: 75};
	chart.width = checkup.CHART_WIDTH - chart.margin.left - chart.margin.right;
	chart.height = checkup.CHART_HEIGHT - chart.margin.top - chart.margin.bottom;

	// chart.svgTag is the actual svg tag, but
	// chart.svg is the group where we actually
	// put the lines.
	chart.svg = chart.svgTag
	  .append("g")
	  	.attr("class", "chart-data")
		.attr("transform", "translate(" + chart.margin.left + "," + chart.margin.top + ")");

	chart.xScale = d3.time.scale()
		.range([0, chart.width]);

	chart.yScale = d3.scale.linear()
		.range([chart.height, 0]);

	// TODO: Do we need closures here to preserve reference to the correct chart?
	chart.line = d3.svg.line()
		.x(function(d) { return chart.xScale(d.timestamp); })
		.y(function(d) { return chart.yScale(d.rtt); })
		.interpolate("monotone"); // linear, monotone, or basis


	chart.lineGroup = chart.svg
	  .append("g")
		.attr("class", "lines");

	var focus = chart.svg
	  .append("g")
		.attr("class", "focus")
		.style("display", "none");

	focus.append("circle")
		.attr("r", 6);

	var text = focus.append("text")
		.attr("x", 9)
		.attr("dy", ".35em")
		.attr("class", "focus-text");


	// Next we build an overlay to cover the data area,
	// so when the mouse hovers it we can show the point.
	var bisectDate = d3.bisector(function(d) { return d.timestamp; }).left;
	var overlay;
	var mousemove = function() {
		var x0 = chart.xScale.invert(d3.mouse(this)[0]),
			i = bisectDate(chart.series.med, x0, 1),
			d0 = chart.series.med[i - 1],
			d1 = chart.series.med[i],
			d = (d0 && d1) ? (x0 - d0.timestamp > d1.timestamp - x0 ? d1 : d0) : (d0 || d1);
		var xloc = chart.xScale(d.timestamp);
		focus.attr("transform", "translate(" + xloc + "," + chart.yScale(d.rtt) + ")");
		if (xloc > overlay.width.animVal.value - 50)
			text.attr("transform", "translate(-60, 10)");
		else
			text.attr("transform", "translate(0, 10)");
		focus.select("text").text(checkup.formatDuration(d.rtt));
	};
	chart.svg.append("rect")
		.attr("class", "overlay")
		.attr("width", chart.width)
		.attr("height", chart.height)
		.on("mouseover", function() { focus.style("display", null); })
		.on("mouseout", function() { focus.style("display", "none"); })
		.on("mousemove", mousemove);
	overlay = document.querySelector("#"+chart.id+" .overlay");
}
