const istanbulCoverage = require('istanbul-lib-coverage');
const istanbulReport = require('istanbul-lib-report');
const istanbulReports = require('istanbul-reports');
const libSourceMaps = require('istanbul-lib-source-maps');
const map = istanbulCoverage.createCoverageMap();
const parser = require('yargs-parser');
const argv = parser(process.argv.slice(2));

if (!!!argv.commitDir || !!!argv.coverageFiles) {
  console.error('error while running merging coverage files');
  process.exit(-1);
}


const mapFileCoverage = (fileCoverage) => {
  fileCoverage.path = fileCoverage.path.replace(
      /(.*packages\/.*\/)(build)(\/.*)/,
      '$1src$3',
  );
  return fileCoverage;
};


for (const coverageFile of argv.coverageFiles.split(' ')) {
  console.log(coverageFile);
  try {
    const coverageJSON = require(coverageFile);
    Object.keys(coverageJSON).forEach((filename) =>
      map.addFileCoverage(mapFileCoverage(coverageJSON[filename])),
    );
  } catch (err) {
    console.error('error while loading ' + coverageFile + err);
    process.exit(-1);
  }
}

const checkCoverage = (summary, thresholds, file) => {
  console.log(thresholds);
  console.log(summary);
  Object.keys(thresholds).forEach((key) => {
    if (summary[key]) {
      const coverage = summary[key].pct;
      if (coverage < thresholds[key]) {
        if (file) {
          console.error('ERROR: Coverage for ' + key + ' (' + coverage + '%) does not meet threshold (' + thresholds[key] + '%) for ' + file);
        } else {
          console.error('ERROR: Coverage for ' + key + ' (' + coverage + '%) does not meet global threshold (' + thresholds[key] + '%)');
        }
      }
    }
  });
};
(async () => {
  const sourceMapStore = libSourceMaps.createSourceMapStore();
  const transformedMap = await sourceMapStore.transformCoverage(map);
  const context = istanbulReport.createContext({coverageMap: transformedMap, dir: argv.commitDir});
  [{name: '/scripts/custom-reporter.js', file: 'coverage-merged.json'}, {name: 'text'}].forEach((reporter) =>
    istanbulReports.create(reporter.name, {file: reporter.file}).execute(context),
  );

  if (argv.coverageManifest) {
    const manifestFile = require(argv.coverageManifest);
    const thresholds = manifestFile.coverage_threshold;
    if (thresholds) {
      if (thresholds.perfile) {
        transformedMap.files().forEach((file) => {
          checkCoverage(transformedMap.fileCoverageFor(file).toSummary(), thresholds, file);
        });
      } else {
        checkCoverage(transformedMap.getCoverageSummary(), thresholds);
      }
    }
  }
})();
