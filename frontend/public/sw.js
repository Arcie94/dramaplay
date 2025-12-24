const CACHE_NAME = 'dramaplay-v8';
// ... (keep urlsToCache same if not changing)

// ... (install and activate same)

self.addEventListener('fetch', event => {
    // Navigation (HTML) -> Network First
    if (event.request.mode === 'navigate') {
        event.respondWith(
            fetch(event.request)
                .then(response => {
                    return response;
                })
                .catch(() => {
                    return caches.match(event.request);
                })
        );
        return;
    }

    // Assets -> Cache First
    event.respondWith(
        caches.match(event.request)
            .then(response => {
                if (response) {
                    return response;
                }
                return fetch(event.request);
            })
    );
});
