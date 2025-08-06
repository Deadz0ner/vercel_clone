const express = require("express");
const cors = require("cors");
const path = require("path");
const { generate } = require("./utils/genId");
const { cloneRepo } = require("./utils/cloneRepo");
const { uploadToContainer } = require("./utils/uploadToContainer");
const { getAllFiles } = require("./utils/getAllfiles");
const redis = require('./utils/redisConnxon');
const subscriber = require('./utils/redisConnxon');
require("dotenv").config();

const app = express();

app.use(cors());
app.use(express.json()); //

app.post("/deploy", async (req, res) => {
  const id = generate();
  const repoUrl = req.body.repoUrl; //    https://github.com//Deadz0ner/test
  console.log(repoUrl);
  try {
    const clonedDir = await cloneRepo(repoUrl, id);
    const filesArr = getAllFiles(clonedDir);
    await uploadToContainer(clonedDir,filesArr,id);
    await redis.lPush("deployQueue", id);
    redis.hSet("status",id,"uploaded");
    res.json({
        msg:  "uploading done",
        id
    })
  } catch (err) {
    res.status(500).json({ error: err.message });
  }
  
});

app.get('status/', async (req, res) => {
  const id = req.params.id;
  const status = await subscriber.hGet("status", id);
  res.json({
    id,
    status: status || "not found"
  })
})

app.listen("3000", () => {
  console.log("listening on 3000");
});
