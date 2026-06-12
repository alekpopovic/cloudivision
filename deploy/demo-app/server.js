const http = require('node:http');

const port = Number(process.env.PORT || 3000);

const server = http.createServer((_req, res) => {
  res.writeHead(200, { 'content-type': 'application/json' });
  res.end(JSON.stringify({ ok: true, app: 'cloudivision-demo-app' }));
});

server.listen(port, () => {
  console.log(`demo app listening on ${port}`);
});
