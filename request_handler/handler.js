const express = require("express");
const path = require("path");
const { BlobServiceClient } = require("@azure/storage-blob");
require("dotenv").config();
const cookieParser = require("cookie-parser");
const mime = require("mime-types");

const app = express();
app.use(cookieParser());
const PORT = process.env.PORT || 3001;

const AZURE_STORAGE_CONNECTION_STRING =
  process.env.AZURE_STORAGE_CONNECTION_STRING;
if (!AZURE_STORAGE_CONNECTION_STRING)
  throw new Error("Azure Storage connection string not found");

const blobServiceClient = BlobServiceClient.fromConnectionString(
  AZURE_STORAGE_CONNECTION_STRING
);
const containerClient = blobServiceClient.getContainerClient("sites-data");

app.get("*", async (req, res) => {
  let id = req.cookies['site-id']; 
  if (!id) {
    id = req.url.replace('/', '');
  }
  
  let file = req.url;
  if (file === '/' || file === '') {
    file = '/index.html';
  }
  
  // Remove query parameters if any
  file = file.split('?')[0];
  
  await serveFromBlob(file, id, res);
});

async function serveFromBlob(file, id, res) {
  try {
    const cleanFile = file.startsWith('/') ? file.substring(1) : file;
    const blobName = path.join(`${id}-build/${id}/build`, cleanFile);
    const blockBlobClient = containerClient.getBlockBlobClient(blobName);

    const buffer = await blockBlobClient.downloadToBuffer();

    const contentType = mime.lookup(file) || "application/octet-stream";

    res.setHeader("Content-Type", contentType);
    res.setHeader("Content-Length", buffer.length);
    res.send(buffer);
  } catch (error) {
    console.error("Error serving from blob:", error);
    console.error("Attempted blob name:", `${id}-build/${id}/build/${file}`);
    res.status(404).send("File not found");
  }
}

app.listen(PORT, () => {
  console.log(`Server is running on http://localhost:${PORT}`);
});