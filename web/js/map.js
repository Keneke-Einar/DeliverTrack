/* ══════════════════════════════════════════════
   DeliverTrack — Leaflet Map Helpers
   ══════════════════════════════════════════════ */

const MapHelper = {
    courierIcon: null,
    pickupIcon: null,
    dropoffIcon: null,
    _initialized: false,

    init() {
        if (this._initialized) return;
        this._initialized = true;

        this.courierIcon = L.divIcon({
            html: '<div class="courier-marker">🚚</div>',
            className: 'custom-marker',
            iconSize: [32, 32],
            iconAnchor: [16, 16],
        });

        this.pickupIcon = L.divIcon({
            html: '<div class="pickup-marker">📦</div>',
            className: 'custom-marker',
            iconSize: [32, 32],
            iconAnchor: [16, 32],
        });

        this.dropoffIcon = L.divIcon({
            html: '<div class="dropoff-marker">📍</div>',
            className: 'custom-marker',
            iconSize: [32, 32],
            iconAnchor: [16, 32],
        });
    },

    /**
     * Create a Leaflet map in the given element.
     */
    create(elementId, center, zoom) {
        center = center || [40.7128, -74.0060];
        zoom = zoom || 13;

        const map = L.map(elementId).setView(center, zoom);
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '&copy; <a href="https://openstreetmap.org/copyright">OpenStreetMap</a>',
            maxZoom: 19,
        }).addTo(map);

        // Fix tile rendering when map is in a hidden container
        setTimeout(() => map.invalidateSize(), 200);

        return map;
    },

    /**
     * Add a marker to the map.
     */
    addMarker(map, lat, lng, icon, popupText) {
        const marker = L.marker([lat, lng], { icon: icon }).addTo(map);
        if (popupText) {
            marker.bindPopup(popupText);
        }
        return marker;
    },

    /**
     * Update an existing marker's position.
     */
    updateMarker(marker, lat, lng) {
        marker.setLatLng([lat, lng]);
    },

    /**
     * Fit map bounds to include all given markers.
     */
    fitMarkers(map, markers) {
        if (!markers || markers.length === 0) return;
        const group = L.featureGroup(markers);
        map.fitBounds(group.getBounds().pad(0.15));
    },

    /**
     * Draw a polyline route on the map.
     */
    drawRoute(map, points) {
        if (!points || points.length < 2) return null;
        return L.polyline(points, {
            color: '#4f46e5',
            weight: 3,
            opacity: 0.7,
            smoothFactor: 1,
        }).addTo(map);
    },

    /**
     * Parse the "(lng,lat)" location string format used by the backend.
     * Returns { lat, lng } or null.
     */
    parseLocation(locStr) {
        if (!locStr) return null;
        const match = locStr.match(/\(?\s*([-\d.]+)\s*,\s*([-\d.]+)\s*\)?/);
        if (!match) return null;
        // Backend stores as (longitude, latitude) — PostGIS convention
        return { lat: parseFloat(match[2]), lng: parseFloat(match[1]) };
    }
};
