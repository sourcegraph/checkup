/**

FS Storage Adapter for Checkup.js

**/

var checkup = checkup || {};

checkup.storage = (function() {
	var url;

	// getCheckFileList gets the list of check files within
	// the given timeframe (as a unit of nanoseconds) to
	// download.
	function getCheckFileList(timeframe, callback) {
		var after = time.Now() - timeframe;
		checkup.getJSON(url+'/index.json', function(index) {
			var names = [];
			for (var name in index) {
				if (index[name] >= after) {
					names.push(name);
				}
			}
			callback(names);
		});
	};

	// setup prepares this storage unit to operate.
	this.setup = function(cfg) {
		url = cfg.url;
	};

	// getChecksWithin gets all the checks within timeframe as a unit
	// of nanoseconds, and executes callback for each check file.
	this.getChecksWithin = function(timeframe, fileCallback, doneCallback) {
		var checksLoaded = 0, resultsLoaded = 0;
		getCheckFileList(timeframe, function(list) {
			if (list.length == 0 && (typeof doneCallback === 'function')) {
				doneCallback(checksLoaded);
			} else {
				for (var i = 0; i < list.length; i++) {
					checkup.getJSON(url+'/'+list[i], function(filename) {
						return function(json, url) {
							checksLoaded++;
							resultsLoaded += json.length;
							if (typeof fileCallback === 'function')
								fileCallback(json, filename);
							if (checksLoaded >= list.length && (typeof doneCallback === 'function'))
								doneCallback(checksLoaded, resultsLoaded);
						};
					}(list[i]));
				}
			}
		});
	};

	// getNewChecks gets any checks since the timestamp on the file name
	// of the youngest check file that has been downloaded. If no check
	// files have been downloaded, no new check files will be loaded.
	this.getNewChecks = function(fileCallback, doneCallback) {
		if (!checkup.lastCheckTs == null)
			return;
		var timeframe = time.Now() - checkup.lastCheckTs;
		return this.getChecksWithin(timeframe, fileCallback, doneCallback);
	};

	return this;
})();
