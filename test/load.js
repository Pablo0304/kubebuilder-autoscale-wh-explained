const http = require("http");

const targetUrl = process.env.TARGET_URL || "http://localhost:8080";
const totalRequests = Number.parseInt(process.env.REQ || "20000", 10);
const concurrency = Number.parseInt(process.env.CONC || "5", 10);

function doRequest() {
  return new Promise((resolve, reject) => {
    const req = http.get(targetUrl, (res) => {
      res.resume();
      if (res.statusCode && res.statusCode >= 200 && res.statusCode < 400) {
        resolve();
        return;
      }
      reject(new Error(`Status ${res.statusCode}`));
    });
    req.on("error", reject);
  });
}

async function run() {
  const queue = Array.from({ length: totalRequests }, () => doRequest);
  let inFlight = 0;
  let completed = 0;
  let failed = 0;

  return new Promise((resolve) => {
    function launchNext() {
      if (completed + failed >= totalRequests) {
        resolve({ completed, failed });
        return;
      }
      while (inFlight < concurrency && queue.length > 0) {
        const next = queue.shift();
        inFlight += 1;
        next()
          .then(() => {
            completed += 1;
          })
          .catch(() => {
            failed += 1;
          })
          .finally(() => {
            inFlight -= 1;
            launchNext();
          });
      }
    }
    launchNext();
  });
}

run().then(({ completed, failed }) => {
  // Simple summary for quick validation in Prometheus.
  console.log(`Done. OK=${completed} FAIL=${failed}`);
});
