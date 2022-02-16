"use strict";
const { ReportBase } = require("istanbul-lib-report");

function nodeMissing(metrics, fileCoverage) {
  const isEmpty = metrics.isEmpty();
  const lines = isEmpty ? 0 : metrics.lines.pct;
  let coveredLines;

  if (lines === 100) {
    const branches = fileCoverage.getBranchCoverageByLine();
    coveredLines = Object.entries(branches).map(([key, { coverage }]) => [
      key,
      coverage === 100,
    ]);
  } else {
    coveredLines = Object.entries(fileCoverage.getLineCoverage());
  }

  let newRange = true;
  const ranges = coveredLines
    .reduce((acum, [line, hit]) => {
      if (hit) newRange = true;
      else {
        line = parseInt(line);
        if (newRange) {
          acum.push([line]);
          newRange = false;
        } else acum[acum.length - 1][1] = line;
      }

      return acum;
    }, [])
    .map((range) => {
      const { length } = range;

      if (length === 1) return range[0];

      return `${range[0]}-${range[1]}`;
    });

  return [].concat(...ranges).join(",");
}

class JsonSummaryReport extends ReportBase {
  constructor(opts) {
    super();

    const { maxCols } = opts;

    this.maxCols = maxCols != null ? maxCols : process.stdout.columns || 80;
    this.file = opts.file || "coverage-merged.json";
    this.contentWriter = null;
    this.first = true;
  }

  onStart(root, context) {
    this.contentWriter = context.writer.writeFile(this.file);
    this.contentWriter.write("{");
  }

  writeSummary(filePath, sc, uncovered) {
    const cw = this.contentWriter;
    if (this.first) {
      this.first = false;
    } else {
      cw.write(",");
    }
    if (uncovered) {
      sc.data.uncovered_lines = uncovered;
    }
    cw.write(JSON.stringify(filePath));
    cw.write(": ");
    cw.write(JSON.stringify(sc));
    cw.println("");
  }

  onSummary(node) {
    if (!node.isRoot()) {
      return;
    }
    this.writeSummary("total", node.getCoverageSummary());
  }

  onDetail(node) {
    const metrics = node.getCoverageSummary();
    const fileCoverage = node.getFileCoverage();
    let missingLines;
    if (!node.isSummary()) {
      missingLines = nodeMissing(metrics, fileCoverage);
    }
    this.writeSummary(fileCoverage.path, metrics, missingLines);
  }

  onEnd() {
    const cw = this.contentWriter;
    cw.println("}");
    cw.close();
  }
}
module.exports = JsonSummaryReport;
