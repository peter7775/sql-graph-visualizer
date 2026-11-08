/* Reconnecting subscription to real benchmark samples, independent of page UI. */
(function () {
    'use strict';
    window.LiveBenchmarkFeed = {
        subscribe(onSample) {
            let socket, timer, attempt = 0, closed = false;
            function connect() {
                if (closed || !window.WebSocket) return;
                try {
                    socket = new WebSocket(`${location.protocol === 'https:' ? 'wss:' : 'ws:'}//${location.host}/ws/performance`);
                    socket.onopen = () => {
                        attempt = 0;
                        socket.send(JSON.stringify({ type: 'subscribe', topic: 'benchmark' }));
                    };
                    socket.onmessage = (event) => {
                        let message;
                        try { message = JSON.parse(event.data); } catch (e) { return; }
                        if (message.type === 'data' && message.topic === 'benchmark' && message.data && message.data.benchmark_id) {
                            onSample(message.data);
                        }
                    };
                    socket.onerror = () => { /* reconnect via onclose */ };
                    socket.onclose = reconnect;
                } catch (e) { reconnect(); }
            }
            function reconnect() {
                if (!closed) timer = setTimeout(connect, Math.min(30000, 1000 * 2 ** attempt++));
            }
            connect();
            return () => { closed = true; clearTimeout(timer); if (socket) socket.close(); };
        },
    };
})();
