const { BlobServiceClient } = require("@azure/storage-blob");
const path = require('path');
const fs = require('fs');
require("dotenv").config();  //used for debug
const AZURE_STORAGE_CONNECTION_STRING =
  process.env.AZURE_STORAGE_CONNECTION_STRING;

if (!AZURE_STORAGE_CONNECTION_STRING) {
  throw new Error("Azure Storage connection string not found.");
}

const blobServiceClient = BlobServiceClient.fromConnectionString(
  AZURE_STORAGE_CONNECTION_STRING
);
// const containerClient = blobServiceClient.getContainerClient('sites-data');
async function downloadFromBlob(prefix,dir) {
  const containerClient = blobServiceClient.getContainerClient("sites-data");
  const iter = containerClient.listBlobsFlat({ prefix });
  const downloadDir = path.join(dir,prefix);
  if(!fs.existsSync(downloadDir)){
    fs.mkdirSync(downloadDir,{ recursive: true });
  }
  for await (const blob of iter) {
    const blobName = blob.name;
    const blockBlobClient = containerClient.getBlockBlobClient(blob.name);
    const localPathOfFile = path.join(downloadDir,blobName.replace(prefix,''));
    const dirOfLocalFile = path.dirname(localPathOfFile);
    fs.mkdirSync(dirOfLocalFile, {recursive:true});
    await blockBlobClient.downloadToFile(localPathOfFile);
    console.log(`Downloaded ${blobName} → ${localPathOfFile}`);
  }
}
module.exports = downloadFromBlob;
