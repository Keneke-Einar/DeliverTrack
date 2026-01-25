// Initialize MongoDB database and create geospatial collections
db = db.getSiblingDB('delivertrack');

// Create courier_locations collection with 2dsphere index for geospatial queries
db.createCollection('courier_locations');
db.courier_locations.createIndex({ location: '2dsphere' });
db.courier_locations.createIndex({ courier_id: 1 });
db.courier_locations.createIndex({ timestamp: -1 });
db.courier_locations.createIndex({ courier_id: 1, timestamp: -1 });

print('✓ Created courier_locations collection with geospatial indexes');

// Create delivery_zones collection with 2dsphere index for polygon queries
db.createCollection('delivery_zones');
db.delivery_zones.createIndex({ geometry: '2dsphere' });
db.delivery_zones.createIndex({ name: 1 }, { unique: true });
db.delivery_zones.createIndex({ active: 1 });

print('✓ Created delivery_zones collection with geospatial indexes');

// Insert sample delivery zones for testing
db.delivery_zones.insertMany([
    {
        name: 'Downtown Zone',
        geometry: {
            type: 'Polygon',
            coordinates: [[
                [-122.4194, 37.7749],  // San Francisco example
                [-122.4094, 37.7749],
                [-122.4094, 37.7849],
                [-122.4194, 37.7849],
                [-122.4194, 37.7749]
            ]]
        },
        active: true,
        description: 'Downtown delivery zone',
        created_at: new Date(),
        updated_at: new Date()
    },
    {
        name: 'Suburbs Zone',
        geometry: {
            type: 'Polygon',
            coordinates: [[
                [-122.4294, 37.7649],
                [-122.4094, 37.7649],
                [-122.4094, 37.7749],
                [-122.4294, 37.7749],
                [-122.4294, 37.7649]
            ]]
        },
        active: true,
        description: 'Suburban delivery zone',
        created_at: new Date(),
        updated_at: new Date()
    }
]);

print('✓ Inserted sample delivery zones');
print('MongoDB initialization complete!');
