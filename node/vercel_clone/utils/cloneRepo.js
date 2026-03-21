const path = require("path");
const { spawn } = require("child_process");
// const { generate } = require("./genId");
function cloneRepo(url, id) {
  return new Promise((resolve, reject) => {
    const outputDir = path.join(__dirname, "..", "temp", id);
    // const repoName=path.basename(url,'.git');
    let gitProcess = spawn("git", ["clone", url, outputDir]);
    gitProcess.stderr.on("data", (data) => {
      const msg = data.toString();
      if (!msg.includes("Cloning into")) {
        console.error(`stderr: ${msg}`);
      }
    });

    gitProcess.on("close", (code) => {
      if (code === 0) {
        console.log(`Cloned repo to ${outputDir}`);
        resolve(outputDir);
      } else {
        reject(new Error(`git clone process exited with code ${code}`));
      }
    });
  });
}

module.exports = { cloneRepo };
