#!/usr/bin/env bash
# test-p2p-local.sh — Spin up two local updu instances and exercise P2P pairing.
#
# Node A = "VPS"   → port 3000, data in /tmp/updu-test/a
# Node B = "Agent" → port 3010, data in /tmp/updu-test/b  (agent mode)
#
# Usage:
#   ./scripts/test-p2p-local.sh           # full run (auto-stops)
#   ./scripts/test-p2p-local.sh --keep    # leave processes running for browser inspection
#
set -euo pipefail

BINARY="${BINARY:-./bin/updu}"
DIR_A="/tmp/updu-test/a"
DIR_B="/tmp/updu-test/b"
PORT_A=3000
PORT_B=3010
KEEP="${1:-}"

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${CYAN}[test]${NC} $*"; }
ok()   { echo -e "${GREEN}[ok]${NC}   $*"; }
warn() { echo -e "${YELLOW}[warn]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; exit 1; }

cleanup() {
  log "Stopping instances..."
  [[ -n "${PID_A:-}" ]] && { kill "$PID_A" 2>/dev/null; ok "Node A stopped (pid $PID_A)"; }
  [[ -n "${PID_B:-}" ]] && { kill "$PID_B" 2>/dev/null; ok "Node B stopped (pid $PID_B)"; }
  echo "  Logs: $DIR_A/updu.log  |  $DIR_B/updu.log"
}
trap cleanup EXIT

wait_ready() {
  local url="$1" label="$2" tries=0
  until curl -sf "$url/healthz" >/dev/null 2>&1; do
    tries=$((tries+1))
    [[ $tries -gt 40 ]] && fail "$label did not become ready after 20s"
    sleep 0.5
  done
  ok "$label is ready at $url"
}

# Helper: call API with current $SESSION cookie
api() {
  local method="$1" url="$2"; shift 2
  curl -sf -X "$method" "$url" \
    -H "Content-Type: application/json" \
    -H "Cookie: $SESSION" \
    "$@"
}

# ── Preflight ─────────────────────────────────────────────────────────────────
[[ -x "$BINARY" ]] || fail "Binary not found: $BINARY — run 'go build -o bin/updu ./cmd/updu' first"
rm -rf "$DIR_A" "$DIR_B"
mkdir -p "$DIR_A" "$DIR_B"

# ── Start Node A (full dashboard mode) ───────────────────────────────────────
log "Starting Node A (VPS / full mode) on :$PORT_A..."
UPDU_DB_PATH="$DIR_A/updu.db" \
UPDU_HOST=127.0.0.1 \
UPDU_PORT=$PORT_A \
UPDU_AUTH_SECRET="alpha-secret-1234" \
UPDU_NODE_NAME="VPS-Alpha" \
  "$BINARY" >"$DIR_A/updu.log" 2>&1 &
PID_A=$!

# ── Start Node B (agent mode, lightweight) ───────────────────────────────────
log "Starting Node B (HomeSrv / agent mode) on :$PORT_B..."
UPDU_DB_PATH="$DIR_B/updu.db" \
UPDU_HOST=127.0.0.1 \
UPDU_PORT=$PORT_B \
UPDU_AUTH_SECRET="bravo-secret-5678" \
UPDU_NODE_NAME="HomeSrv-Bravo" \
  "$BINARY" agent >"$DIR_B/updu.log" 2>&1 &
PID_B=$!

wait_ready "http://127.0.0.1:$PORT_A" "Node A"
wait_ready "http://127.0.0.1:$PORT_B" "Node B"

# ── Register admin accounts (first-user self-registration) ───────────────────
log "Registering admin accounts..."
curl -sf -X POST "http://127.0.0.1:$PORT_A/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"testpass123"}' >/dev/null && ok "Node A admin registered"
curl -sf -X POST "http://127.0.0.1:$PORT_B/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"testpass123"}' >/dev/null && ok "Node B admin registered"

# ── Login and capture session cookies ────────────────────────────────────────
log "Logging in to both nodes..."
curl -sf -X POST "http://127.0.0.1:$PORT_A/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"testpass123"}' \
  -c "$DIR_A/cookies.txt" >/dev/null
SESSION_A=$(grep 'updu_session' "$DIR_A/cookies.txt" | awk '{print "updu_session="$NF}' | head -1)
[[ -n "$SESSION_A" ]] && ok "Node A: ${SESSION_A:0:36}..." || fail "Failed to login to Node A"

curl -sf -X POST "http://127.0.0.1:$PORT_B/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"testpass123"}' \
  -c "$DIR_B/cookies.txt" >/dev/null
SESSION_B=$(grep 'updu_session' "$DIR_B/cookies.txt" | awk '{print "updu_session="$NF}' | head -1)
[[ -n "$SESSION_B" ]] && ok "Node B: ${SESSION_B:0:36}..." || fail "Failed to login to Node B"

# ── Fetch node identities ─────────────────────────────────────────────────────
log "Fetching node identities..."
ID_A=$(curl -sf "http://127.0.0.1:$PORT_A/api/v1/p2p/identity" | python3 -c "import sys,json; print(json.load(sys.stdin)['node_id'])")
ID_B=$(curl -sf "http://127.0.0.1:$PORT_B/api/v1/p2p/identity" | python3 -c "import sys,json; print(json.load(sys.stdin)['node_id'])")
NAME_A=$(curl -sf "http://127.0.0.1:$PORT_A/api/v1/p2p/identity" | python3 -c "import sys,json; print(json.load(sys.stdin).get('name','?'))")
NAME_B=$(curl -sf "http://127.0.0.1:$PORT_B/api/v1/p2p/identity" | python3 -c "import sys,json; print(json.load(sys.stdin).get('name','?'))")
ok "Node A: $NAME_A ($ID_A)"
ok "Node B: $NAME_B ($ID_B)"

# ── Create a monitor on Node A to demonstrate federation ─────────────────────
log "Creating a test monitor on Node A..."
SESSION="$SESSION_A"
MON_RESP=$(api POST "http://127.0.0.1:$PORT_A/api/v1/monitors" \
  -d '{"name":"Example HTTPS","type":"http","config":{"url":"https://example.com"},"interval_s":60,"enabled":true}')
echo "$MON_RESP" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('id','?'))" | xargs -I{} echo "[ok]   Monitor on A: {}" || warn "Monitor parse issue"

# ── Node A initiates pairing with Node B ─────────────────────────────────────
log "Node A connecting to Node B at 127.0.0.1:$PORT_B..."
SESSION="$SESSION_A"
PAIR=$(api POST "http://127.0.0.1:$PORT_A/api/v1/admin/peers/connect" \
  -d "{\"address\":\"127.0.0.1:$PORT_B\",\"name\":\"HomeSrv-Bravo\"}")
echo "$PAIR" | python3 -c "
import sys, json
d = json.load(sys.stdin)
print(f'  → peer id={d[\"id\"]}  status={d[\"status\"]}')
"
ok "Pair request dispatched"

# ── Check pending proposals on Node B (A sent a pair-request to B) ───────────
sleep 1
log "Peers on Node B (expecting pending from A)..."
SESSION="$SESSION_B"
api GET "http://127.0.0.1:$PORT_B/api/v1/admin/peers" | python3 -c "
import sys, json
resp = json.load(sys.stdin)
peers = resp.get('peers') or []
print(f'  {len(peers)} peer(s):')
for p in peers:
    print(f'    [{p[\"status\"]}] {p.get(\"name\",\"?\")} ({p[\"id\"]})')
"

# ── Node B approves Node A ────────────────────────────────────────────────────
log "Node B approving Node A ($ID_A)..."
SESSION="$SESSION_B"
api POST "http://127.0.0.1:$PORT_B/api/v1/admin/peers/approve" \
  -d "{\"id\":\"$ID_A\",\"role\":\"peer\"}" >/dev/null \
  && ok "Node A approved on Node B"

# ── Wait for the poll loop to collect federated state ────────────────────────
log "Waiting 14s for federation poll cycle (poll interval = 6s)..."
sleep 14

# ── Node A peers state ────────────────────────────────────────────────────────
log "Peer state on Node A:"
SESSION="$SESSION_A"
api GET "http://127.0.0.1:$PORT_A/api/v1/admin/peers" | python3 -c "
import sys, json
resp = json.load(sys.stdin)
local = resp.get('local') or {}
peers = resp.get('peers') or []
disc  = resp.get('discovered') or []
print(f'  local: {local.get(\"name\",\"?\")} ({local.get(\"node_id\",\"?\")})')
print(f'  peers: {len(peers)}  discovered: {len(disc)}')
for p in peers:
    mons = p.get('monitors') or []
    print(f'    [{p[\"status\"]}] {p.get(\"name\",\"?\")} ({p[\"id\"]}) — {len(mons)} federated monitors')
"

# ── Triage test: kill Node B, watch Node A detect it ─────────────────────────
log "Killing Node B to trigger survivor triage autopsy..."
kill "$PID_B" 2>/dev/null
unset PID_B

log "Waiting 20s for Node A to detect failure and run triage..."
sleep 20

log "Triage state on Node A:"
SESSION="$SESSION_A"
api GET "http://127.0.0.1:$PORT_A/api/v1/admin/peers" | python3 -c "
import sys, json
resp = json.load(sys.stdin)
peers = resp.get('peers') or []
for p in peers:
    triage = p.get('triage')
    if triage:
        print(f'  !! TRIAGE [{p.get(\"name\",\"?\")}]')
        print(f'     probable_cause : {triage.get(\"probable_cause\",\"?\")}')
        print(f'     summary        : {triage.get(\"summary\",\"?\")}')
        diag = triage.get('diagnostics') or {}
        for k, v in (diag.items() if isinstance(diag, dict) else []):
            print(f'     {k}: {v}')
    else:
        print(f'  [{p.get(\"status\",\"?\")}] {p.get(\"name\",\"?\")} — no triage yet')
" && ok "Triage check complete" || warn "Triage parse issue (check response above)"

echo ""
echo "======================================"
echo "  P2P federation test complete!"
echo "======================================"
echo ""
echo "  Node A:  http://127.0.0.1:$PORT_A  (admin / testpass123)"
echo "  Node B:  http://127.0.0.1:$PORT_B  (admin / testpass123)"
echo ""
echo "  Logs:  $DIR_A/updu.log   $DIR_B/updu.log"
echo ""

if [[ "$KEEP" == "--keep" ]]; then
  warn "--keep flag: processes kept running. Ctrl-C or 'kill $PID_A' to stop."
  trap - EXIT
  wait
fi
