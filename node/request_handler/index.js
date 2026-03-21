const express = require("express");
const cookieParser = require("cookie-parser");
const { BlobServiceClient } = require("@azure/storage-blob");
const mime = require("mime-types");
require('dotenv').config();

const containerName = "sites-data";
const containerClient = BlobServiceClient.fromConnectionString(process.env.AZURE_STORAGE_CONNECTION_STRING).getContainerClient(containerName);

const app = express();
app.use(cookieParser());

// Request logging middleware with unique request ID
app.use((req, res, next) => {
  req.requestId = Math.random().toString(36).substr(2, 9);
  console.log(`[${new Date().toISOString()}] [${req.requestId}] ${req.method} ${req.url}`);
  console.log(`[${req.requestId}] Host: ${req.hostname}`);
  next();
});

app.get("*", async (req, res) => {
  const reqId = req.requestId;
  console.log(`\n=== [${reqId}] Processing Request ===`);
  
  // Get subdomain as siteId: e.g. wglk7.localhost:3001
  const host = req.hostname;
  console.log(`[${reqId}] Raw hostname: ${host}`);
  
  const siteId = host.split(".")[0]; // assumes format <siteId>.localhost
  console.log(`[${reqId}] Extracted siteId: ${siteId}`);

  if (!siteId) {
    console.log(`[${reqId}] ❌ No siteId found - returning 400`);
    return res.status(400).send("Invalid site ID");
  }

  let reqPath = req.path === "/" ? "/index.html" : req.path;
  
  // Strip /portfolio prefix if it exists (common when React apps are built with homepage setting)
  if (reqPath.startsWith('/portfolio/')) {
    reqPath = reqPath.replace('/portfolio', '');
    console.log(`[${reqId}] Stripped /portfolio prefix: ${req.path} -> ${reqPath}`);
  }
  
  console.log(`[${reqId}] Request path: ${req.path} -> Resolved path: ${reqPath}`);
  
  const blobPath = `${siteId}-build/${siteId}/build${reqPath}`;
  console.log(`[${reqId}] Blob path: ${blobPath}`);
  
  const blobClient = containerClient.getBlockBlobClient(blobPath);

  try {
    console.log(`[${reqId}] 🔍 Checking if blob exists...`);
    const startTime = Date.now();
    const exists = await blobClient.exists();
    const existsTime = Date.now() - startTime;
    console.log(`[${reqId}] Blob exists: ${exists} (checked in ${existsTime}ms)`);
    
    if (!exists) {
      console.log(`[${reqId}] ❌ Blob not found, throwing error`);
      throw new Error("Blob not found");
    }

    console.log(`[${reqId}] 📁 Determining content type...`);
    const contentType = mime.lookup(reqPath) || "application/octet-stream";
    console.log(`[${reqId}] Content type: ${contentType}`);
    
    console.log(`[${reqId}] ⬇️ Downloading blob to buffer...`);
    const downloadStart = Date.now();
    const buffer = await blobClient.downloadToBuffer();
    const downloadTime = Date.now() - downloadStart;
    console.log(`[${reqId}] ✅ Downloaded ${buffer.length} bytes in ${downloadTime}ms`);

    res.setHeader("Content-Type", contentType);
    console.log(`[${reqId}] 📤 Sending response with buffer`);
    res.send(buffer);
    
  } catch (err) {
    console.log(`[${reqId}] ❌ Error occurred: ${err.message}`);
    
    // Only fallback to index.html if it's an HTML request (i.e. SPA route)
    const accept = req.headers["accept"] || "";
    console.log(`[${reqId}] Accept header: ${accept}`);
    console.log(`[${reqId}] Accept includes text/html: ${accept.includes("text/html")}`);
    
    if (accept.includes("text/html")) {
      console.log(`[${reqId}] 🔄 Attempting fallback to index.html...`);
      const fallbackBlobPath = `${siteId}-build/${siteId}/build/index.html`;
      console.log(`[${reqId}] Fallback blob path: ${fallbackBlobPath}`);
      
      const fallbackBlob = containerClient.getBlockBlobClient(fallbackBlobPath);
      
      try {
        console.log(`[${reqId}] 🔍 Checking if fallback blob exists...`);
        const fallbackExistsStart = Date.now();
        const fallbackExists = await fallbackBlob.exists();
        const fallbackExistsTime = Date.now() - fallbackExistsStart;
        console.log(`[${reqId}] Fallback blob exists: ${fallbackExists} (checked in ${fallbackExistsTime}ms)`);

        if (fallbackExists) {
          console.log(`[${reqId}] ⬇️ Downloading fallback blob...`);
          const fallbackDownloadStart = Date.now();
          const fallbackBuffer = await fallbackBlob.downloadToBuffer();
          const fallbackDownloadTime = Date.now() - fallbackDownloadStart;
          console.log(`[${reqId}] ✅ Downloaded fallback ${fallbackBuffer.length} bytes in ${fallbackDownloadTime}ms`);
          
          res.setHeader("Content-Type", "text/html");
          console.log(`[${reqId}] 📤 Sending fallback response`);
          return res.send(fallbackBuffer);
        } else {
          console.log(`[${reqId}] ❌ Fallback blob not found`);
        }
      } catch (fallbackErr) {
        console.log(`[${reqId}] ❌ Fallback error: ${fallbackErr.message}`);
      }
    } else {
      console.log(`[${reqId}] ⏭️ Not an HTML request, skipping fallback`);
    }

    console.log(`[${reqId}] 🚫 Returning 404 Not Found`);
    return res.status(404).send("Not found");
  }
});

// Debug endpoint to list blobs for a site
app.get("/debug/list/:siteId", async (req, res) => {
  const { siteId } = req.params;
  const prefix = `${siteId}-build/`;
  
  console.log(`\n=== Listing blobs for siteId: ${siteId} ===`);
  console.log(`Prefix: ${prefix}`);
  
  try {
    const blobs = [];
    for await (const blob of containerClient.listBlobsFlat({ prefix })) {
      blobs.push(blob.name);
      console.log(`Found blob: ${blob.name}`);
    }
    
    res.json({
      siteId,
      prefix,
      totalBlobs: blobs.length,
      blobs: blobs.sort()
    });
  } catch (error) {
    console.error(`Error listing blobs: ${error.message}`);
    res.status(500).json({ error: error.message });
  }
});

app.listen(3001, () => {
  console.log("🚀 Server running on http://<siteId>.localhost:3001");
  console.log(`Container name: ${containerName}`);
  console.log(`Azure Storage configured: ${!!process.env.AZURE_STORAGE_CONNECTION_STRING}`);
  console.log("\n💡 Debug endpoint available at: http://localhost:3001/debug/list/<siteId>");
});