/* ══════════════════════════════════════════════
   DeliverTrack — Alpine.js Application
   ══════════════════════════════════════════════ */

document.addEventListener('alpine:init', () => {

    /* ── Auth Store ─────────────────────────── */
    Alpine.store('auth', {
        token: localStorage.getItem('dt_token') || '',
        user: JSON.parse(localStorage.getItem('dt_user') || 'null'),

        isAuthenticated() {
            return !!this.token;
        },

        get role()       { return this.user?.role || ''; },
        get userId()     { return this.user?.user_id || this.user?.id || 0; },
        get customerId() { return this.user?.customer_id || 0; },
        get courierId()  { return this.user?.courier_id || 0; },
        get username()   { return this.user?.username || ''; },

        persist() {
            localStorage.setItem('dt_token', this.token);
            localStorage.setItem('dt_user', JSON.stringify(this.user));
        },

        clear() {
            this.token = '';
            this.user = null;
            localStorage.removeItem('dt_token');
            localStorage.removeItem('dt_user');
        }
    });

    /* ── Router Store ───────────────────────── */
    Alpine.store('router', {
        page: '',
        params: {},

        init() {
            this.resolve();
            window.addEventListener('hashchange', () => this.resolve());
        },

        resolve() {
            const auth = Alpine.store('auth');
            const hash = window.location.hash.slice(1) || '';
            const parts = hash.split('/').filter(Boolean);

            if (!auth.isAuthenticated()) {
                this.page = 'login';
                return;
            }

            if (parts.length === 0) {
                this.navigate('/' + auth.role + '/dashboard');
                return;
            }

            // Extract params (e.g., /customer/track/42 → id=42)
            this.params = parts.length >= 3 ? { id: parts[2] } : {};
            this.page = parts.slice(0, 2).join('-');
        },

        navigate(path) {
            window.location.hash = '#' + path;
        },

        isActive(page) {
            return this.page === page;
        }
    });

    /* ── Toast Store ────────────────────────── */
    Alpine.store('toast', {
        items: [],
        _id: 0,

        show(message, type, duration) {
            type = type || 'info';
            duration = duration || 4000;
            const id = ++this._id;
            this.items.push({ id, message, type });
            setTimeout(() => {
                this.items = this.items.filter(t => t.id !== id);
            }, duration);
        },
        success(msg) { this.show(msg, 'success'); },
        error(msg)   { this.show(msg, 'error', 6000); },
        info(msg)    { this.show(msg, 'info'); }
    });

    /* ──────────────────────────────────────────
       Helpers shared across components
       ────────────────────────────────────────── */

    function statusColor(status) {
        var colors = {
            pending:    'bg-amber-100 text-amber-800',
            assigned:   'bg-blue-100 text-blue-800',
            picked_up:  'bg-purple-100 text-purple-800',
            in_transit:  'bg-indigo-100 text-indigo-800',
            delivered:  'bg-green-100 text-green-800',
            cancelled:  'bg-red-100 text-red-800'
        };
        return colors[status] || 'bg-gray-100 text-gray-800';
    }

    function formatDate(dateStr) {
        if (!dateStr) return '—';
        return new Date(dateStr).toLocaleDateString('en-US', {
            month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit'
        });
    }

    function nextStatusFor(current) {
        var flow = {
            pending: 'assigned',
            assigned: 'picked_up',
            picked_up: 'in_transit',
            in_transit: 'delivered'
        };
        return flow[current] || null;
    }

    /* ──────────────────────────────────────────
       LOGIN PAGE
       ────────────────────────────────────────── */
    Alpine.data('loginPage', () => ({
        mode: 'login',
        username: '',
        email: '',
        password: '',
        role: 'customer',
        loading: false,
        error: '',

        async doLogin() {
            this.loading = true;
            this.error = '';
            try {
                const data = await api.post('/login', {
                    username: this.username,
                    password: this.password
                });
                const auth = Alpine.store('auth');
                auth.token = data.token;
                auth.user = data.user;
                // Merge JWT claims for extra fields
                const claims = parseJWT(data.token);
                if (claims) {
                    auth.user = Object.assign({}, auth.user, claims);
                }
                auth.persist();
                Alpine.store('router').navigate('/' + auth.role + '/dashboard');
            } catch (e) {
                this.error = e.message;
            } finally {
                this.loading = false;
            }
        },

        async doRegister() {
            this.loading = true;
            this.error = '';
            try {
                await api.post('/register', {
                    username: this.username,
                    email: this.email,
                    password: this.password,
                    role: this.role
                });
                Alpine.store('toast').success('Account created — logging in…');
                await this.doLogin();
            } catch (e) {
                this.error = e.message;
            } finally {
                this.loading = false;
            }
        }
    }));

    /* ──────────────────────────────────────────
       CUSTOMER — Dashboard
       ────────────────────────────────────────── */
    Alpine.data('customerDashboard', () => ({
        deliveries: [],
        loading: true,

        async init() {
            try {
                const auth = Alpine.store('auth');
                const data = await api.get('/api/delivery/deliveries?customer_id=' + auth.customerId);
                this.deliveries = Array.isArray(data) ? data : [];
            } catch (e) {
                Alpine.store('toast').error('Failed to load deliveries');
            } finally {
                this.loading = false;
            }
        },

        statusColor: statusColor,
        formatDate: formatDate,

        track(id) {
            Alpine.store('router').navigate('/customer/track/' + id);
        }
    }));

    /* ──────────────────────────────────────────
       CUSTOMER — Create Delivery
       ────────────────────────────────────────── */
    Alpine.data('customerCreate', () => ({
        form: {
            customer_id: 0,
            pickup_location: '',
            delivery_location: '',
            notes: '',
            scheduled_date: ''
        },
        loading: false,

        init() {
            this.form.customer_id = Alpine.store('auth').customerId;
        },

        async submit() {
            this.loading = true;
            try {
                const body = Object.assign({}, this.form);
                if (!body.scheduled_date) delete body.scheduled_date;
                if (!body.notes) delete body.notes;
                await api.post('/api/delivery/deliveries', body);
                Alpine.store('toast').success('Delivery created!');
                Alpine.store('router').navigate('/customer/dashboard');
            } catch (e) {
                Alpine.store('toast').error(e.message);
            } finally {
                this.loading = false;
            }
        }
    }));

    /* ──────────────────────────────────────────
       CUSTOMER — Track Delivery
       ────────────────────────────────────────── */
    Alpine.data('customerTrack', () => ({
        delivery: null,
        locations: [],
        currentLocation: null,
        eta: null,
        map: null,
        courierMarker: null,
        ws: null,
        loading: true,

        async init() {
            const id = Alpine.store('router').params.id;
            if (!id) { this.loading = false; return; }
            try {
                this.delivery = await api.get('/api/delivery/deliveries/' + id);

                var trackData = null;
                try { trackData = await api.get('/api/tracking/deliveries/' + id + '/track'); } catch (_) {}
                this.locations = (trackData && trackData.locations) ? trackData.locations : [];

                try { this.currentLocation = await api.get('/api/tracking/deliveries/' + id + '/location'); } catch (_) {}
                try {
                    var etaResp = await api.get('/api/tracking/deliveries/' + id + '/eta');
                    if (etaResp) {
                        // eta is in nanoseconds — convert to minutes
                        this.eta = {
                            minutes: Math.round((etaResp.eta || 0) / 60000000000),
                            distance: (etaResp.distance_km || 0).toFixed(1),
                            speed: (etaResp.average_speed_kmh || 0).toFixed(0)
                        };
                    }
                } catch (_) {}

                this.$nextTick(() => this.initMap());
                this.connectWS(id);
            } catch (e) {
                Alpine.store('toast').error('Failed to load delivery');
            } finally {
                this.loading = false;
            }
        },

        initMap() {
            var el = document.getElementById('tracking-map');
            if (!el || this.map) return;
            MapHelper.init();

            var center = this.currentLocation
                ? [this.currentLocation.Latitude, this.currentLocation.Longitude]
                : [40.7128, -74.0060];

            this.map = MapHelper.create('tracking-map', center, 14);

            // Pickup/dropoff markers from delivery location strings
            if (this.delivery) {
                var pickup = MapHelper.parseLocation(this.delivery.PickupLocation);
                var dropoff = MapHelper.parseLocation(this.delivery.DeliveryLocation);
                if (pickup) MapHelper.addMarker(this.map, pickup.lat, pickup.lng, MapHelper.pickupIcon, 'Pickup');
                if (dropoff) MapHelper.addMarker(this.map, dropoff.lat, dropoff.lng, MapHelper.dropoffIcon, 'Dropoff');
            }

            if (this.currentLocation) {
                this.courierMarker = MapHelper.addMarker(
                    this.map,
                    this.currentLocation.Latitude, this.currentLocation.Longitude,
                    MapHelper.courierIcon, 'Courier'
                );
            }

            if (this.locations.length > 1) {
                var pts = this.locations.map(function (l) { return [l.Latitude, l.Longitude]; });
                MapHelper.drawRoute(this.map, pts);
            }
        },

        connectWS(deliveryId) {
            var self = this;
            var token = Alpine.store('auth').token;
            var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            var url = protocol + '//' + window.location.host + '/ws/deliveries/' + deliveryId + '/track?token=' + token;

            this.ws = new WebSocket(url);
            this.ws.onmessage = function (event) {
                try {
                    var data = JSON.parse(event.data);
                    if (data.location) {
                        self.currentLocation = data.location;
                        if (self.courierMarker) {
                            MapHelper.updateMarker(self.courierMarker, data.location.Latitude, data.location.Longitude);
                        } else if (self.map) {
                            MapHelper.init();
                            self.courierMarker = MapHelper.addMarker(
                                self.map, data.location.Latitude, data.location.Longitude,
                                MapHelper.courierIcon, 'Courier'
                            );
                        }
                        if (self.map) self.map.panTo([data.location.Latitude, data.location.Longitude]);
                    }
                } catch (_) {}
            };
            this.ws.onclose = function () {
                setTimeout(function () {
                    if (Alpine.store('router').page === 'customer-track') {
                        self.connectWS(deliveryId);
                    }
                }, 3000);
            };
        },

        statusColor: statusColor,
        formatDate: formatDate,

        destroy() {
            if (this.ws) { this.ws.close(); this.ws = null; }
            if (this.map) { this.map.remove(); this.map = null; }
        }
    }));

    /* ──────────────────────────────────────────
       CUSTOMER — Notifications
       ────────────────────────────────────────── */
    Alpine.data('customerNotifications', () => ({
        notifications: [],
        loading: true,
        ws: null,

        async init() {
            try {
                var auth = Alpine.store('auth');
                var data = await api.get('/api/notification/notifications?user_id=' + auth.userId);
                this.notifications = Array.isArray(data) ? data : [];
            } catch (e) {
                Alpine.store('toast').error('Failed to load notifications');
            } finally {
                this.loading = false;
            }
            this.connectWS();
        },

        connectWS() {
            var self = this;
            var token = Alpine.store('auth').token;
            var protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
            var url = protocol + '//' + window.location.host + '/ws/notifications?token=' + token;

            this.ws = new WebSocket(url);
            this.ws.onmessage = function (event) {
                try {
                    var data = JSON.parse(event.data);
                    self.notifications.unshift({
                        ID: Date.now(),
                        Type: data.type || 'push',
                        Message: data.message,
                        Status: 'pending',
                        CreatedAt: new Date().toISOString()
                    });
                    Alpine.store('toast').info(data.message);
                } catch (_) {}
            };
            this.ws.onclose = function () {
                setTimeout(function () {
                    if (Alpine.store('router').page === 'customer-notifications') {
                        self.connectWS();
                    }
                }, 3000);
            };
        },

        async markRead(id) {
            try {
                await api.post('/api/notification/notifications/mark-read', { notification_id: id });
                var n = this.notifications.find(function (n) { return n.ID === id; });
                if (n) n.Status = 'read';
            } catch (_) {}
        },

        formatDate: formatDate,

        destroy() {
            if (this.ws) { this.ws.close(); this.ws = null; }
        }
    }));

    /* ──────────────────────────────────────────
       COURIER — Dashboard
       ────────────────────────────────────────── */
    Alpine.data('courierDashboard', () => ({
        deliveries: [],
        loading: true,

        async init() {
            try {
                var data = await api.get('/api/delivery/deliveries');
                this.deliveries = Array.isArray(data) ? data : [];
            } catch (e) {
                Alpine.store('toast').error('Failed to load deliveries');
            } finally {
                this.loading = false;
            }
        },

        async updateStatus(id, status) {
            try {
                await api.put('/api/delivery/deliveries/' + id + '/status', { status: status });
                Alpine.store('toast').success('Status updated to ' + status.replace('_', ' '));
                this.loading = true;
                var data = await api.get('/api/delivery/deliveries');
                this.deliveries = Array.isArray(data) ? data : [];
                this.loading = false;
            } catch (e) {
                Alpine.store('toast').error(e.message);
            }
        },

        statusColor: statusColor,
        nextStatusFor: nextStatusFor,
        formatDate: formatDate
    }));

    /* ──────────────────────────────────────────
       COURIER — Location Update
       ────────────────────────────────────────── */
    Alpine.data('courierLocation', () => ({
        position: null,
        map: null,
        marker: null,
        autoUpdate: false,
        intervalId: null,
        deliveryId: '',
        sending: false,

        init() {
            var self = this;
            this.$nextTick(function () {
                var el = document.getElementById('courier-map');
                if (el && !self.map) {
                    MapHelper.init();
                    self.map = MapHelper.create('courier-map');
                    self.getPosition();
                }
            });
        },

        getPosition() {
            var self = this;
            if (!navigator.geolocation) {
                Alpine.store('toast').error('Geolocation not supported');
                return;
            }
            navigator.geolocation.getCurrentPosition(
                function (pos) {
                    self.position = {
                        latitude: pos.coords.latitude,
                        longitude: pos.coords.longitude,
                        accuracy: pos.coords.accuracy,
                        speed: pos.coords.speed,
                        heading: pos.coords.heading,
                        altitude: pos.coords.altitude
                    };
                    self.updateMapView();
                },
                function (err) {
                    Alpine.store('toast').error('Location error: ' + err.message);
                },
                { enableHighAccuracy: true }
            );
        },

        updateMapView() {
            if (!this.map || !this.position) return;
            var lat = this.position.latitude;
            var lng = this.position.longitude;
            if (this.marker) {
                MapHelper.updateMarker(this.marker, lat, lng);
            } else {
                MapHelper.init();
                this.marker = MapHelper.addMarker(this.map, lat, lng, MapHelper.courierIcon, 'Your Location');
            }
            this.map.setView([lat, lng], 15);
        },

        toggleAutoUpdate() {
            var self = this;
            this.autoUpdate = !this.autoUpdate;
            if (this.autoUpdate) {
                this.sendLocation();
                this.intervalId = setInterval(function () {
                    self.getPosition();
                    self.sendLocation();
                }, 10000);
            } else {
                if (this.intervalId) clearInterval(this.intervalId);
                this.intervalId = null;
            }
        },

        async sendLocation() {
            if (!this.position || this.sending) return;
            this.sending = true;
            try {
                var auth = Alpine.store('auth');
                var body = {
                    courier_id: auth.courierId,
                    latitude: this.position.latitude,
                    longitude: this.position.longitude
                };
                if (this.position.accuracy) body.accuracy = this.position.accuracy;
                if (this.position.speed) body.speed = this.position.speed;
                if (this.position.heading) body.heading = this.position.heading;
                if (this.position.altitude) body.altitude = this.position.altitude;
                if (this.deliveryId) body.delivery_id = parseInt(this.deliveryId);
                await api.post('/api/tracking/locations', body);
                if (!this.autoUpdate) Alpine.store('toast').success('Location sent');
            } catch (e) {
                Alpine.store('toast').error(e.message);
            } finally {
                this.sending = false;
            }
        },

        destroy() {
            if (this.intervalId) clearInterval(this.intervalId);
            if (this.map) { this.map.remove(); this.map = null; }
        }
    }));

    /* ──────────────────────────────────────────
       ADMIN — Dashboard
       ────────────────────────────────────────── */
    Alpine.data('adminDashboard', () => ({
        stats: null,
        recentDeliveries: [],
        loading: true,

        async init() {
            try {
                var results = await Promise.allSettled([
                    api.get('/api/analytics/analytics/delivery-stats'),
                    api.get('/api/delivery/deliveries')
                ]);
                this.stats = results[0].status === 'fulfilled' ? results[0].value : null;
                var deliveries = results[1].status === 'fulfilled' ? results[1].value : [];
                this.recentDeliveries = Array.isArray(deliveries) ? deliveries.slice(0, 10) : [];
            } catch (e) {
                Alpine.store('toast').error('Failed to load dashboard');
            } finally {
                this.loading = false;
            }
        },

        statCards() {
            if (!this.stats) return [];
            return [
                { label: 'Total Deliveries',     value: this.stats.TotalDeliveries || 0,     icon: '📦', color: 'bg-blue-500' },
                { label: 'Completed',             value: this.stats.CompletedDeliveries || 0, icon: '✅', color: 'bg-green-500' },
                { label: 'Pending',               value: this.stats.PendingDeliveries || 0,   icon: '⏳', color: 'bg-amber-500' },
                { label: 'Cancelled',             value: this.stats.CancelledDeliveries || 0, icon: '❌', color: 'bg-red-500' },
            ];
        },

        statusColor: statusColor,
        formatDate: formatDate
    }));

    /* ──────────────────────────────────────────
       ADMIN — Deliveries Management
       ────────────────────────────────────────── */
    Alpine.data('adminDeliveries', () => ({
        deliveries: [],
        statusFilter: '',
        loading: true,

        async init() {
            await this.load();
        },

        async load() {
            this.loading = true;
            try {
                var url = '/api/delivery/deliveries';
                if (this.statusFilter) url += '?status=' + this.statusFilter;
                var data = await api.get(url);
                this.deliveries = Array.isArray(data) ? data : [];
            } catch (e) {
                Alpine.store('toast').error('Failed to load deliveries');
            } finally {
                this.loading = false;
            }
        },

        async updateStatus(id, status) {
            try {
                await api.put('/api/delivery/deliveries/' + id + '/status', { status: status });
                Alpine.store('toast').success('Status updated');
                await this.load();
            } catch (e) {
                Alpine.store('toast').error(e.message);
            }
        },

        statusColor: statusColor,
        nextStatusFor: nextStatusFor,
        formatDate: formatDate
    }));

    /* ──────────────────────────────────────────
       ADMIN — Live Map
       ────────────────────────────────────────── */
    Alpine.data('adminMap', () => ({
        map: null,
        loading: true,
        deliveryCount: 0,

        init() {
            var self = this;
            this.$nextTick(function () {
                self.loadMap();
            });
        },

        async loadMap() {
            try {
                MapHelper.init();
                this.map = MapHelper.create('admin-map', [40.7128, -74.0060], 11);

                var deliveries = await api.get('/api/delivery/deliveries');
                var all = Array.isArray(deliveries) ? deliveries : [];
                var active = all.filter(function (d) {
                    return d.Status === 'in_transit' || d.Status === 'picked_up' || d.Status === 'assigned';
                });
                this.deliveryCount = active.length;

                var markers = [];
                for (var i = 0; i < active.length; i++) {
                    var d = active[i];
                    try {
                        var loc = await api.get('/api/tracking/deliveries/' + d.ID + '/location');
                        if (loc && loc.Latitude) {
                            var m = MapHelper.addMarker(
                                this.map, loc.Latitude, loc.Longitude,
                                MapHelper.courierIcon,
                                'Delivery #' + d.ID + ' — ' + d.Status
                            );
                            markers.push(m);
                        }
                    } catch (_) {}
                }
                if (markers.length > 0) MapHelper.fitMarkers(this.map, markers);
            } catch (e) {
                Alpine.store('toast').error('Failed to load map data');
            } finally {
                this.loading = false;
            }
        },

        destroy() {
            if (this.map) { this.map.remove(); this.map = null; }
        }
    }));

    /* ──────────────────────────────────────────
       ADMIN — Send Notifications
       ────────────────────────────────────────── */
    Alpine.data('adminNotifications', () => ({
        form: {
            user_id: '',
            type: 'push',
            subject: '',
            message: '',
            recipient: ''
        },
        sending: false,
        sent: [],

        async send() {
            this.sending = true;
            try {
                var body = {
                    user_id: parseInt(this.form.user_id),
                    type: this.form.type,
                    subject: this.form.subject,
                    message: this.form.message,
                    recipient: this.form.recipient
                };
                await api.post('/api/notification/notifications/send', body);
                Alpine.store('toast').success('Notification sent!');
                this.sent.unshift({
                    Subject: body.subject,
                    Message: body.message,
                    Type: body.type,
                    recipient: body.recipient,
                    CreatedAt: new Date().toISOString()
                });
                this.form.subject = '';
                this.form.message = '';
            } catch (e) {
                Alpine.store('toast').error(e.message);
            } finally {
                this.sending = false;
            }
        },

        formatDate: formatDate
    }));

});
