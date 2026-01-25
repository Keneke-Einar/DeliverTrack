#!/bin/bash
# Test MongoDB Geospatial Setup

set -e

echo "🔍 Testing MongoDB Geospatial Setup..."
echo

# Check if MongoDB is running
echo "1. Checking MongoDB connection..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --eval "db.adminCommand('ping')" --quiet
echo "✓ MongoDB is running"
echo

# Check collections
echo "2. Checking collections..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --eval "db.getCollectionNames()" --quiet
echo

# Check indexes on courier_locations
echo "3. Checking courier_locations indexes..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --eval "db.courier_locations.getIndexes()" --quiet
echo

# Check indexes on delivery_zones
echo "4. Checking delivery_zones indexes..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --eval "db.delivery_zones.getIndexes()" --quiet
echo

# Test inserting a courier location
echo "5. Testing courier location insertion..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --quiet <<EOF
db.courier_locations.insertOne({
    courier_id: 999,
    location: {
        type: "Point",
        coordinates: [-122.4194, 37.7749]
    },
    timestamp: new Date(),
    speed: 30.5,
    heading: 90.0,
    accuracy: 5.0,
    created_at: new Date()
});
print("✓ Inserted test courier location");
EOF
echo

# Test geospatial query
echo "6. Testing geospatial query (find nearby locations)..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --quiet <<EOF
var result = db.courier_locations.find({
    location: {
        \$near: {
            \$geometry: {
                type: "Point",
                coordinates: [-122.4194, 37.7749]
            },
            \$maxDistance: 5000
        }
    }
}).limit(5).toArray();
print("✓ Found " + result.length + " courier locations within 5km");
EOF
echo

# Test zone intersection
echo "7. Testing zone intersection query..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --quiet <<EOF
var zones = db.delivery_zones.find({
    geometry: {
        \$geoIntersects: {
            \$geometry: {
                type: "Point",
                coordinates: [-122.4144, 37.7799]
            }
        }
    }
}).toArray();
print("✓ Found " + zones.length + " zones containing the test point");
for (var i = 0; i < zones.length; i++) {
    print("  - " + zones[i].name);
}
EOF
echo

# Clean up test data
echo "8. Cleaning up test data..."
mongosh mongodb://admin:admin123@localhost:27017/delivertrack?authSource=admin --eval "db.courier_locations.deleteOne({courier_id: 999})" --quiet
echo "✓ Cleaned up test data"
echo

echo "✅ All MongoDB geospatial tests passed!"
