import http from 'node:http';

export function handler(request, response) {
  if (request.url === '/healthz') {
    response.writeHead(200, { 'content-type': 'application/json' });
    response.end(JSON.stringify({ status: 'ok' }));
    return;
  }
  response.writeHead(404).end();
}

if (process.argv[1] === new URL(import.meta.url).pathname) {
  http.createServer(handler).listen(8080);
}
