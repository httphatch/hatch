const http = require("node:http");

const port = parseInt(process.env.PORT, 10) || 3000;
const sessionName = process.env.HATCH_SESSION || "default";

const server = http.createServer((req, res) => {
  if (req.url === "/api/info") {
    res.writeHead(200, { "Content-Type": "application/json" });
    res.end(
      JSON.stringify({
        session: sessionName,
        port,
        pid: process.pid,
        uptime: process.uptime(),
        timestamp: new Date().toISOString(),
      })
    );
    return;
  }

  res.writeHead(200, { "Content-Type": "text/html" });
  res.end(`<!DOCTYPE html>
<html>
<head>
  <title>Hatch Session Demo</title>
  <style>
    body { font-family: system-ui, sans-serif; max-width: 600px; margin: 60px auto; padding: 0 20px; }
    h1 { font-size: 1.5rem; }
    .info { background: #f4f4f5; border-radius: 8px; padding: 16px; font-family: monospace; font-size: 0.9rem; line-height: 1.8; }
    .label { color: #71717a; }
  </style>
</head>
<body>
  <h1>Hatch Session Demo</h1>
  <div class="info">
    <div><span class="label">Session:</span> ${sessionName}</div>
    <div><span class="label">Port:</span> ${port}</div>
    <div><span class="label">PID:</span> ${process.pid}</div>
    <div><span class="label">Host:</span> ${req.headers.host}</div>
  </div>
  <p>Each session runs its own instance of this server on a different port with a unique subdomain.</p>
</body>
</html>`);
});

server.listen(port, () => {
  console.log(`Server running on port ${port} (session: ${sessionName})`);
});
