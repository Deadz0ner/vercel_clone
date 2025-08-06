const { createClient } = require('redis');

const redisClient = createClient({
  url: "redis://localhost:6379",
});
redisClient.on("error", (err) => {
  console.log(`${err}: Error From Redis`);
});
redisClient
  .connect()
  .then(() => {
    console.log("Redis connected");
  })
  .catch((err) => {
    console.error("Redis connection failed:", err);
  });
module.exports = redisClient;