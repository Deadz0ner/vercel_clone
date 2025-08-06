const blobServiceClient = require("./blob_connect");
const getAllFiles = require('./getAllFiles');
const path = require('path');



async function uploadToContainer(dir,id) {
  const containerClient = blobServiceClient.getContainerClient("sites-data");
  const filesArr = getAllFiles(`./downloaded/${id}/build`);
  for (let filePath of filesArr) {
    const relativePath = path.relative(dir,filePath).replace(/\\/g, "/");
    console.log(relativePath);
    const blobName = `${id}-build/${relativePath}`;
    const blockBlobClient = containerClient.getBlockBlobClient(blobName);

    // console.log(`Uploading ${filePath} as ${blobName}`);
    await blockBlobClient.uploadFile(filePath);
  }
  console.log("Upload Complete")
}
module.exports = uploadToContainer;