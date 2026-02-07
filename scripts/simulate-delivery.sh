#!/bin/bash
# ═══════════════════════════════════════════════════════════════
# DeliverTrack — Live Delivery Simulator
# Creates a demo customer + courier, an in-transit delivery,
# then streams courier location updates along a realistic
# NYC route so you can watch real-time tracking at:
#   http://localhost:3000/#/customer/track/<delivery_id>
# ═══════════════════════════════════════════════════════════════

set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:3000}"
INTERVAL="${INTERVAL:-2}"          # seconds between location updates
GREEN='\033[0;32m'; YELLOW='\033[1;33m'; BLUE='\033[0;34m'
RED='\033[0;31m'; CYAN='\033[0;36m'; NC='\033[0m'; BOLD='\033[1m'

log()  { echo -e "${BLUE}[simulator]${NC} $1"; }
ok()   { echo -e "${GREEN}  ✓${NC} $1"; }
warn() { echo -e "${YELLOW}  ⚠${NC} $1"; }
err()  { echo -e "${RED}  ✗${NC} $1"; }

# ── Unique names so the script is re-runnable ───────────────
TS=$(date +%s)
CUST_USER="democust_${TS}"
COUR_USER="democour_${TS}"
PASSWORD="password123"

# ── Step 1 — Register customer ──────────────────────────────
log "Registering customer ${CYAN}${CUST_USER}${NC}…"
CUST_REG=$(curl -sf "${BASE_URL}/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${CUST_USER}\",\"email\":\"${CUST_USER}@test.com\",\"password\":\"${PASSWORD}\",\"role\":\"customer\"}")
ok "Customer registered"

# ── Step 2 — Register courier ───────────────────────────────
log "Registering courier ${CYAN}${COUR_USER}${NC}…"
COUR_REG=$(curl -sf "${BASE_URL}/register" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${COUR_USER}\",\"email\":\"${COUR_USER}@test.com\",\"password\":\"${PASSWORD}\",\"role\":\"courier\"}")
ok "Courier registered"

# ── Step 3 — Login both ─────────────────────────────────────
log "Logging in customer…"
CUST_LOGIN=$(curl -sf "${BASE_URL}/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${CUST_USER}\",\"password\":\"${PASSWORD}\"}")
CUST_TOKEN=$(echo "$CUST_LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
CUST_ID=$(echo "$CUST_LOGIN" | python3 -c "import sys,json; u=json.load(sys.stdin)['user']; print(u.get('customer_id') or u.get('CustomerID') or '')")
ok "Customer token obtained (customer_id=${CUST_ID})"

log "Logging in courier…"
COUR_LOGIN=$(curl -sf "${BASE_URL}/login" \
  -H 'Content-Type: application/json' \
  -d "{\"username\":\"${COUR_USER}\",\"password\":\"${PASSWORD}\"}")
COUR_TOKEN=$(echo "$COUR_LOGIN" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])")
COUR_ID=$(echo "$COUR_LOGIN" | python3 -c "import sys,json; u=json.load(sys.stdin)['user']; print(u.get('courier_id') or u.get('CourierID') or '')")
ok "Courier token obtained (courier_id=${COUR_ID})"

# ── Step 4 — Create delivery ────────────────────────────────
# Pickup: Times Square area  →  Dropoff: Central Park area
PICKUP_LAT=40.7580;  PICKUP_LNG=-73.9855
DROPOFF_LAT=40.7812;  DROPOFF_LNG=-73.9665

log "Creating delivery (Times Square → Central Park)…"
DEL_RESP=$(curl -sf "${BASE_URL}/api/delivery/deliveries" \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${CUST_TOKEN}" \
  -d "{
    \"customer_id\": ${CUST_ID},
    \"pickup_location\": \"Times Square, New York, NY\",
    \"delivery_location\": \"Central Park, New York, NY\",
    \"notes\": \"Live tracking demo\"
  }")
DEL_ID=$(echo "$DEL_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin).get('ID') or json.load(sys.stdin).get('id'))" 2>/dev/null || echo "$DEL_RESP" | grep -oP '"(?:ID|id)":\s*\K\d+' | head -1)
ok "Delivery created: #${DEL_ID}"

# ── Step 5 — Assign courier and set in_transit ──────────────
log "Assigning courier and setting status to in_transit…"
# Assign courier (pending → assigned, auto-assigns courier_id)
curl -sf "${BASE_URL}/api/delivery/deliveries/${DEL_ID}/status" \
  -X PUT \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${COUR_TOKEN}" \
  -d "{\"status\":\"assigned\"}" > /dev/null
ok "Status → assigned (courier_id=${COUR_ID})"

# In transit (assigned → in_transit)
curl -sf "${BASE_URL}/api/delivery/deliveries/${DEL_ID}/status" \
  -X PUT \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${COUR_TOKEN}" \
  -d '{"status":"in_transit"}' > /dev/null
ok "Status → in_transit"

# ── Route: realistic NYC path from Times Square to Central Park ──
# ~20 waypoints along Broadway / 7th Ave / Central Park South (20-second demo)
read -r -d '' ROUTE << 'EOF' || true
40.7580,-73.9855
40.7590,-73.9848
40.7600,-73.9840
40.7610,-73.9832
40.7620,-73.9824
40.7630,-73.9816
40.7640,-73.9808
40.7650,-73.9800
40.7660,-73.9792
40.7670,-73.9784
40.7680,-73.9776
40.7690,-73.9768
40.7700,-73.9760
40.7710,-73.9752
40.7720,-73.9744
40.7730,-73.9736
40.7740,-73.9728
40.7750,-73.9720
40.7760,-73.9712
40.7770,-73.9704
40.7780,-73.9696
40.7790,-73.9688
40.7800,-73.9680
40.7812,-73.9665
EOF

TOTAL=$(echo "$ROUTE" | wc -l)

echo ""
echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
echo -e "${BOLD} 🚚 Live Delivery Simulation Ready${NC}"
echo -e "${BOLD}═══════════════════════════════════════════════════════${NC}"
echo ""
echo -e "  Delivery ID:   ${CYAN}#${DEL_ID}${NC}"
echo -e "  Customer:      ${CYAN}${CUST_USER}${NC} (password: ${PASSWORD})"
echo -e "  Courier:       ${CYAN}${COUR_USER}${NC} (password: ${PASSWORD})"
echo -e "  Route:         Times Square → Central Park (${TOTAL} waypoints)"
echo -e "  Duration:      ~${TOTAL} seconds (${INTERVAL}s intervals)"
echo ""
echo -e "  ${GREEN}▶ Open this URL to watch live tracking:${NC}"
echo -e "    ${BOLD}${BASE_URL}/#/customer/track/${DEL_ID}${NC}"
echo -e "    (log in as ${CYAN}${CUST_USER}${NC} / ${CYAN}${PASSWORD}${NC})"
echo ""
echo -e "${YELLOW}  Starting in 3 seconds… (Ctrl+C to cancel)${NC}"
sleep 3

# ── Step 6 — Stream location updates ────────────────────────
STEP=0
echo "$ROUTE" | while IFS=',' read -r LAT LNG; do
  STEP=$((STEP + 1))

  # Add slight randomness to simulate real GPS jitter (±0.0001°)
  JITTER_LAT=$(python3 -c "import random; print(f'{random.uniform(-0.0001, 0.0001):.6f}')")
  JITTER_LNG=$(python3 -c "import random; print(f'{random.uniform(-0.0001, 0.0001):.6f}')")
  ACTUAL_LAT=$(python3 -c "print(f'{${LAT} + ${JITTER_LAT}:.6f}')")
  ACTUAL_LNG=$(python3 -c "print(f'{${LNG} + ${JITTER_LNG}:.6f}')")

  # Simulated speed ~25 km/h in city
  SPEED=$(python3 -c "import random; print(f'{random.uniform(20.0, 30.0):.1f}')")

  RESP=$(curl -sf -o /dev/null -w "%{http_code}" \
    "${BASE_URL}/api/tracking/locations" \
    -H 'Content-Type: application/json' \
    -H "Authorization: Bearer ${COUR_TOKEN}" \
    -d "{
      \"delivery_id\": ${DEL_ID},
      \"courier_id\": ${COUR_ID},
      \"latitude\": ${ACTUAL_LAT},
      \"longitude\": ${ACTUAL_LNG},
      \"speed\": ${SPEED},
      \"accuracy\": 5.0,
      \"heading\": 0
    }" 2>/dev/null || echo "ERR")

  if [ "$RESP" = "201" ]; then
    printf "  ${GREEN}📍${NC} [%02d/%d] lat=%-11s lng=%-11s speed=%s km/h\n" \
      "$STEP" "$TOTAL" "$ACTUAL_LAT" "$ACTUAL_LNG" "$SPEED"
  else
    printf "  ${RED}⚠${NC}  [%02d/%d] HTTP %s — lat=%s lng=%s\n" \
      "$STEP" "$TOTAL" "$RESP" "$ACTUAL_LAT" "$ACTUAL_LNG"
  fi

  sleep "$INTERVAL"
done

# ── Step 7 — Mark delivered ──────────────────────────────────
echo ""
log "Marking delivery #${DEL_ID} as delivered…"
curl -sf "${BASE_URL}/api/delivery/deliveries/${DEL_ID}/status" \
  -X PUT \
  -H 'Content-Type: application/json' \
  -H "Authorization: Bearer ${COUR_TOKEN}" \
  -d '{"status":"delivered"}' > /dev/null
ok "Delivery #${DEL_ID} marked as delivered! 🎉"
echo ""
echo -e "${BOLD}Simulation complete.${NC}"
