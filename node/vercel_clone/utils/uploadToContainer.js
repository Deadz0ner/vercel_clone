const blobServiceClient = require("./blob_connect");
const path = require('path');

async function uploadToContainer(dir, filesArr, id) {
  const containerClient = blobServiceClient.getContainerClient("sites-data");
  for (let filePath of filesArr) {
    const relativePath = path.relative(dir, filePath).replace(/\\/g, "/");
    console.log(relativePath);
    const blobName = `${id}/${relativePath}`;
    const blockBlobClient = containerClient.getBlockBlobClient(blobName);

    // console.log(`Uploading ${filePath} as ${blobName}`);
    await blockBlobClient.uploadFile(filePath);
  }
  console.log("Upload Complete")
}
module.exports = {uploadToContainer};