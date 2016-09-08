// config.js must be included BEFORE this file!

// Configure access to storage
checkup.storage.setup(checkup.config.storage);

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
checkup.storage.getChecksWithin(checkup.config.timeframe, processNewCheckFile, allCheckFilesLoaded);
setInterval(function() {
	checkup.storage.getNewChecks(processNewCheckFile, allCheckFilesLoaded);
}, checkup.config.refresh_interval * 1000);

// Update "time ago" tags every so often
setInterval(function() {
	var times = document.querySelectorAll("time.dynamic");
	for (var i = 0; i < times.length; i++) {
		var timeEl = times[i];
		var ms = Date.parse(timeEl.getAttribute("datetime"));
		timeEl.innerHTML = checkup.timeSince(ms);
	}
}, 5000);


function processNewCheckFile(json, filename) {
	checkup.checks.push(json);

	// update the timestamp of the last check file's timestamp
	var dashLoc = filename.indexOf("-");
	if (dashLoc > 0) {
		var checkTs = Number(filename.substr(0, dashLoc));
		if (checkTs > checkup.lastCheckTs) {
			checkup.lastCheckTs = checkTs;
		}
	}

	// iterate each result and store/process it
	for (var j = 0; j < json.length; j++) {
		var result = json[j];

		// Save stats with the result so we don't have to recompute them later
		result.stats = checkup.computeStats(result);

		checkup.orderedResults.push(result); // will sort later, more efficient that way

		if (!checkup.groupedResults[result.timestamp])
			checkup.groupedResults[result.timestamp] = [result];
		else
			checkup.groupedResults[result.timestamp].push(result);

		if (!checkup.results[result.endpoint])
			checkup.results[result.endpoint] = [result];
		else
			checkup.results[result.endpoint].push(result);

		var chart = checkup.charts[result.endpoint] || checkup.makeChart(result.title);
		chart.results.push(result);

		var ts = checkup.unixNanoToD3Timestamp(result.timestamp);
		chart.series.min.push({ timestamp: ts, rtt: Math.round(result.stats.min) });
		chart.series.med.push({ timestamp: ts, rtt: Math.round(result.stats.median) });
		chart.series.max.push({ timestamp: ts, rtt: Math.round(result.stats.max) });
		if (result.threshold)
			chart.series.threshold.push({ timestamp: ts, rtt: result.threshold });

		checkup.charts[result.endpoint] = chart;
		checkup.charts[result.endpoint].endpoint = result.endpoint;

		for (var s in chart.series) {
			chart.series[s].sort(function(a, b) {
			  return a.timestamp - b.timestamp;
			});
		}

		if (!checkup.lastResultTs || ts > checkup.lastResultTs) {
			checkup.lastResultTs = ts;
			checkup.dom.lastcheck.innerHTML = checkup.makeTimeTag(checkup.lastResultTs)+" ago";
		}
	}

	if (checkup.domReady)
		makeGraphs();
}

function allCheckFilesLoaded(numChecksLoaded, numResultsLoaded) {
	// Sort the result lists
	checkup.orderedResults.sort(function(a, b) { return a.timestamp - b.timestamp; });
	for (var endpoint in checkup.results)
		checkup.results[endpoint].sort(function(a, b) { return a.timestamp - b.timestamp; });

	// Create events for the timeline

	var newEvents = [];
	var statuses = {}; // keyed by endpoint

	// First load the last known status of each endpoint
	for (var i = checkup.events.length-1; i >= 0; i--) {
		var result = checkup.events[i].result;
		if (!statuses[result.endpoint])
			statuses[result.endpoint] = checkup.events[i].status;
	}

	// Then go through the new results and look for new events
	for (var i = checkup.orderedResults.length-numResultsLoaded; i < checkup.orderedResults.length; i++) {
		var result = checkup.orderedResults[i];

		var status = "healthy";
		if (result.degraded) status = "degraded";
		else if (result.down) status = "down";

		if (status != statuses[result.endpoint]) {
			// New event because status changed
			newEvents.push({
				id: checkup.eventCounter++,
				result: result,
				status: status
			});
		}
		if (result.message) {
			// New event because message posted
			newEvents.push({
				id: checkup.eventCounter++,
				result: result,
				status: status,
				message: result.message
			});
		}

		statuses[result.endpoint] = status;
	}

	checkup.events = checkup.events.concat(newEvents);

	function renderTime(ns) {
		var d = new Date(ns * 1e-6);
		var hours = d.getHours();
		var ampm = "AM";
		if (hours > 12) {
			hours -= 12;
			ampm = "PM";
		}
		return hours+":"+checkup.leftpad(d.getMinutes(), 2, "0")+" "+ampm;
	}

	// Render events
	for (var i = 0; i < newEvents.length; i++) {
		var e = newEvents[i];

		// Save this event to the chart's event series so it will render on the graph
		var imgFile = "ok.png", imgWidth = 15, imgHeight = 15; // the different icons look smaller/larger because of their shape
		if (e.status == "down") { imgFile = "incident.png"; imgWidth = 20; imgHeight = 20; }
		else if (e.status == "degraded") { imgFile = "degraded.png"; imgWidth = 25; imgHeight = 25; }
		var chart = checkup.charts[e.result.endpoint];
		chart.series.events.push({
			timestamp: checkup.unixNanoToD3Timestamp(e.result.timestamp),
			rtt: e.result.stats.median,
			eventid: e.id,
			imgFile: imgFile,
			imgWidth: imgWidth,
			imgHeight: imgHeight
		});

		// Render event to timeline
		var evtElem = document.createElement("div");
		evtElem.setAttribute("data-eventid", e.id);
		evtElem.classList.add("event-item");
		evtElem.classList.add("event-id-"+e.id);
		evtElem.classList.add(checkup.color[e.status]);
		if (e.message) {
			evtElem.classList.add("message");
			evtElem.innerHTML = '<div class="message-head">'+checkup.makeTimeTag(e.result.timestamp*1e-6)+' ago</div>';
			evtElem.innerHTML += '<div class="message-body">'+e.message+'</div>';
		} else {
			evtElem.classList.add("event");
			evtElem.innerHTML = '<span class="time">'+renderTime(e.result.timestamp)+'</span> '+e.result.title+" "+e.status;
		}
		checkup.dom.timeline.insertBefore(evtElem, checkup.dom.timeline.childNodes[0]);
	}

	// Update DOM now that we have the whole picture

	// Update overall status
	var overall = "healthy";
	for (var endpoint in checkup.results) {
		if (overall == "down") break;
		var lastResult = checkup.results[endpoint][checkup.results[endpoint].length-1];
		if (lastResult) {
			if (lastResult.down)
				overall = "down";
			else if (lastResult.degraded)
				overall = "degraded";
		}
	}

	if (overall == "healthy") {
		checkup.dom.favicon.href = "images/status-green.png";
		checkup.dom.status.className = "green";
		checkup.dom.statustext.innerHTML = checkup.config.status_text.healthy || "System Nominal";
	} else if (overall == "degraded") {
		checkup.dom.favicon.href = "images/status-yellow.png";
		checkup.dom.status.className = "yellow";
		checkup.dom.statustext.innerHTML = checkup.config.status_text.degraded || "Sub-Optimal";
	} else if (overall == "down") {
		checkup.dom.favicon.href = "images/status-red.png";
		checkup.dom.status.className = "red";
		checkup.dom.statustext.innerHTML = checkup.config.status_text.down || "Outage";
	} else {
		checkup.dom.favicon.href = "images/status-gray.png";
		checkup.dom.status.className = "gray";
		checkup.dom.statustext.innerHTML = checkup.config.status_text.unknown || "Status Unknown";
	}


	// Detect big gaps in any of the charts, and if there is one, show an explanation.
	var bigGap = false;
	var lastTimeDiff;
	for (var key in checkup.charts) {
		// We expect results to be chronologically ordered, but since they are downloaded
		// in an arbitrary order due to network conditions, we have to sort to be sure.
		checkup.charts[key].results.sort(function(a, b) {
			return a.timestamp - b.timestamp;
		});
		for (var k = 1; k < checkup.charts[key].results.length; k++) {
			var timeDiff = Math.abs(checkup.charts[key].results[k].timestamp - checkup.charts[key].results[k-1].timestamp);
			bigGap = lastTimeDiff && timeDiff > lastTimeDiff * 10;
			lastTimeDiff = timeDiff;
			if (bigGap) {
				document.getElementById("big-gap").style.display = 'block';
				break;
			}
		}
		if (bigGap) break;
	}
	if (!bigGap) {
		document.getElementById("big-gap").style.display = 'none';
	}

	makeGraphs(); // must render graphs again after we've filled in the event series
}

function makeGraphs() {
	checkup.dom.timeframe.innerHTML = checkup.formatDuration(checkup.config.timeframe);
	checkup.dom.checkcount.innerHTML = checkup.checks.length;

	if (!checkup.placeholdersRemoved && checkup.checks.length > 0) {
		// Remove placeholder to make way for the charts;
		// placeholder necessary to give space in absense of charts.
		if (phElem = document.getElementById("chart-placeholder"))
			phElem.remove();
		checkup.placeholdersRemoved = true;
	}

	for (var endpoint in checkup.charts) {
		makeGraph(checkup.charts[endpoint], endpoint);
  }

	checkup.graphsMade = true;
}

function makeGraph(chart, endpoint) {
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
	chart.events = chart.eventGroup.selectAll("image")
		.data(chart.series.events);

	// transition from old paths to new paths
	chart.lines
		.transition()
		.duration(checkup.animDuration)
		.attr("d", chart.line);
	chart.events
		.transition()
		.duration(checkup.animDuration);

	// enter any new data (lines)
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

	// enter any new data (events)
	chart.events.enter().append("svg:image")
		.attr("width", function(d, i) { return d.imgWidth || 0; })
		.attr("height", function(d, i) { return d.imgHeight || 0; })
		.attr("xlink:href", function(d, i) { return "images/"+d.imgFile; })
		.attr("x", function(d, i) { return chart.xScale(d.timestamp) - (d.imgWidth/2); })
		.attr("y", function(d, i) { return chart.yScale(d.rtt) - (d.imgHeight/2); })
		.attr("data-eventid", function(d, i) { return d.eventid; })
		.attr("class", function(d, i) { return "event-item event-id-"+d.eventid; })
		.on("mouseover", highlightSameEvent)
		.on("mouseout", unhighlightSameEvent);

	// exit any old data
	chart.lines
		.exit()
		.remove();
}


function renderChart(chart) {
	// Outer div is a wrapper that we use for layout
	var el = document.createElement('div');
	var containerSize = "chart-50";
	if (document.getElementsByClassName('chart-container').length == 0) {
		containerSize = "chart-100";
	} else {
		// It's possible that a chart was created that, at the time,
		// was the only one, but now it is too wide, since there are
		// at least two charts. Resize the wide one to be smaller.
		var tooWide = document.querySelector('.chart-container.chart-100');
		if (tooWide)
			tooWide.className = "chart-container chart-50";
	}
	el.className = "chart-container "+containerSize;

	// Div to contain the endpoint / title
	var el2 = document.createElement('div');
	el2.className = "chart-title";
	var el2b = document.createElement('a'); el2b.setAttribute("href", chart.endpoint);
	el2b.appendChild(document.createTextNode(chart.title));
	el2.appendChild(el2b);
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

	// chart.svgTag is the svg tag, but chart.svg
	// is the group where we actually put the lines.
	chart.svg = chart.svgTag
	  .append("g")
	  	.attr("class", "chart-data")
		.attr("transform", "translate(" + chart.margin.left + "," + chart.margin.top + ")");

	chart.xScale = d3.time.scale()
		.range([0, chart.width]);

	chart.yScale = d3.scale.linear()
		.range([chart.height, 0]);

	chart.line = d3.svg.line()
		.x(function(d) { return chart.xScale(d.timestamp); })
		.y(function(d) { return chart.yScale(d.rtt); })
		.interpolate("step-after"); // linear, monotone, or basis


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

	chart.eventGroup = chart.svg
	  .append("g")
		.attr("class", "events");
}


function highlightSameEvent() {
	var elems = document.querySelectorAll(".event-item:not(.event-id-"+this.getAttribute("data-eventid")+")");
	for (var i = 0; i < elems.length; i++) {
		elems[i].style.opacity = ".25";
	}
}

function unhighlightSameEvent() {
	var elems = document.querySelectorAll(".event-item:not(.event-id-"+this.getAttribute("data-eventid")+")");
	for (var i = 0; i < elems.length; i++) {
		elems[i].style.opacity = "";
	}
}
