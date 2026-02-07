/* ══════════════════════════════════════════════
   DeliverTrack — API Client
   ══════════════════════════════════════════════ */

class ApiClient {
    constructor() {
        this.baseUrl = '';
    }

    getToken() {
        return localStorage.getItem('dt_token') || '';
    }

    async request(method, path, body = null) {
        const headers = { 'Content-Type': 'application/json' };
        const token = this.getToken();
        if (token) {
            headers['Authorization'] = 'Bearer ' + token;
        }

        const opts = { method, headers };
        if (body !== null) {
            opts.body = JSON.stringify(body);
        }

        const resp = await fetch(this.baseUrl + path, opts);

        if (resp.status === 401) {
            localStorage.removeItem('dt_token');
            localStorage.removeItem('dt_user');
            if (window.Alpine) {
                Alpine.store('auth').token = '';
                Alpine.store('auth').user = null;
            }
            window.location.hash = '#/login';
            throw new Error('Session expired. Please log in again.');
        }

        if (!resp.ok) {
            let msg = resp.statusText;
            try {
                const err = await resp.json();
                msg = err.message || err.error || msg;
            } catch (_) {}
            throw new Error(msg);
        }

        const text = await resp.text();
        return text ? JSON.parse(text) : null;
    }

    get(path) {
        return this.request('GET', path);
    }

    post(path, body) {
        return this.request('POST', path, body);
    }

    put(path, body) {
        return this.request('PUT', path, body);
    }

    delete(path) {
        return this.request('DELETE', path);
    }
}

const api = new ApiClient();

/**
 * Decode a JWT token payload (no validation — just extracts claims).
 */
function parseJWT(token) {
    try {
        const base64Url = token.split('.')[1];
        const base64 = base64Url.replace(/-/g, '+').replace(/_/g, '/');
        const json = decodeURIComponent(
            atob(base64)
                .split('')
                .map(c => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
                .join('')
        );
        return JSON.parse(json);
    } catch {
        return null;
    }
}
