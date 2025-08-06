const { createClient } = require("redis");
const downloadFromBlob = require("./downloadFromBlob");
const buildProject = require('./build');
const uploadToContainer = require('./uploadToContainer');
require("dotenv").config();

const subscriber = createClient({
  url: "redis://localhost:6379", // Check your correct Redis port
});
const publisher = createClient({
  url: "redis://localhost:6379", // Check your correct Redis port
});

subscriber.on("error", (err) => {
  console.error("Redis Client Error:", err);
});

async function main() {
  await subscriber.connect();
  await publisher.connect();
  while (true) {
    const id = await subscriber.brPop("deployQueue", 0);
    console.log("Received from queue:", id);
    try {
      await downloadFromBlob(id.element, "./downloaded/");
      await buildProject(id.element);
      await uploadToContainer('./downloaded/',id.element);
      publisher.hSet("status",id.element,'deployed');
    } catch (err) {
      console.error("Failed to download files for ID:", id.element, err);
    }
  }
}

main();
