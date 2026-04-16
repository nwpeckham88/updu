import http from 'node:http';

const port = Number.parseInt(process.env.UPDU_E2E_FIXTURE_PORT ?? '4011', 10);

const server = http.createServer((request, response) => {
    if (request.url === '/healthz') {
        response.writeHead(200, { 'content-type': 'text/plain; charset=utf-8' });
        response.end('ok');
        return;
    }

    if (request.url === '/ok') {
        response.writeHead(200, { 'content-type': 'application/json; charset=utf-8' });
        response.end(JSON.stringify({ ok: true }));
        return;
    }

    if (request.url === '/fail') {
        response.writeHead(503, { 'content-type': 'application/json; charset=utf-8' });
        response.end(JSON.stringify({ ok: false }));
        return;
    }

    if (request.url === '/slow') {
        setTimeout(() => {
            response.writeHead(200, { 'content-type': 'application/json; charset=utf-8' });
            response.end(JSON.stringify({ ok: true, delayMs: 1200 }));
        }, 1200);
        return;
    }

    response.writeHead(404, { 'content-type': 'text/plain; charset=utf-8' });
    response.end('not found');
});

server.listen(port, '127.0.0.1', () => {
    process.stdout.write(`fixture server listening on 127.0.0.1:${port}\n`);
});

for (const signal of ['SIGINT', 'SIGTERM']) {
    process.on(signal, () => {
        server.close(() => process.exit(0));
    });
}