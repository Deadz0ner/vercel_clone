const fs = require('fs');
const path = require('path');

function getAllFiles(dir,fileList = []) {
  
  const entries = fs.readdirSync(dir, { withFileTypes: true });
//   console.log(entries)

  for (const entry of entries) {
    const fullPath = path.join(dir, entry.name);

    if (entry.name === '.git') continue;

    if (entry.isDirectory()) {
      getAllFiles(fullPath, fileList);
    } else {
      fileList.push(fullPath);
    }
  }

  return fileList;
}
module.exports = {getAllFiles}
