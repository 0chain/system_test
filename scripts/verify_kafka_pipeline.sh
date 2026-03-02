#!/bin/bash
# verify_kafka_pipeline.sh — Verify Kafka ↔ 0box data pipeline
#
# Usage:
#   bash verify_kafka_pipeline.sh pre-chain   # Before chain starts: Kafka + 0box up, consumer registered
#   bash verify_kafka_pipeline.sh post-chain  # After chain starts:  events flowing, 0box DB populated
#   bash verify_kafka_pipeline.sh             # Run both phases sequentially
#
# Returns 0 on success, 1 on failure.

set -euo pipefail

PHASE="${1:-both}"

# ── Kafka connection settings (SASL_PLAINTEXT) ──────────────────────────────
KAFKA_BROKER="198.19.0.99:9092"
KAFKA_USER="${KAFKA_USERNAME:-admin}"
KAFKA_PASS="${KAFKA_PASSWORD:-admin-secret}"
KAFKA_TOPIC="${KAFKA_TOPIC:-events}"
KAFKA_GROUP="${KAFKA_CONSUMER_GROUP:-events-consumer}"
KAFKA_CONTAINER="kafka"

# ── 0box settings ─────────────────────────────────────────────────────────────
ZBOX_API="http://localhost:9081"
ZBOX_PG_CONTAINER="${ZBOX_PG_CONTAINER:-postgres-0box}"
ZBOX_DB_USER="${ZBOX_DB_USER:-zbox_user}"
ZBOX_DB_PASS="${ZBOX_DB_PASS:-zbox_server}"
ZBOX_DB_NAME="${ZBOX_DB_NAME:-zbox}"

# ── Timing ────────────────────────────────────────────────────────────────────
MAX_WAIT_PRE=120    # seconds to wait for pre-chain readiness
MAX_WAIT_POST=300   # seconds to wait for post-chain event flow
POLL_INTERVAL=5

# ── Colours ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'
CYAN='\033[0;34m'; BOLD='\033[1m'; NC='\033[0m'

pass() { echo -e "  ${GREEN}[PASS]${NC} $*"; }
fail() { echo -e "  ${RED}[FAIL]${NC} $*"; }
warn() { echo -e "  ${YELLOW}[WARN]${NC} $*"; }
info() { echo -e "  ${CYAN}[INFO]${NC} $*"; }
header() { echo -e "\n${BOLD}${CYAN}══════════════════════════════════════════════════${NC}"; \
           echo -e "${BOLD}${CYAN}  $*${NC}"; \
           echo -e "${BOLD}${CYAN}══════════════════════════════════════════════════${NC}"; }

# ── Helper: run kafka-topics/consumer-groups via docker exec ─────────────────
kafka_cmd() {
    docker exec "$KAFKA_CONTAINER" bash -c "
cat > /tmp/client.properties << 'EOF'
security.protocol=SASL_PLAINTEXT
sasl.mechanism=PLAIN
sasl.jaas.config=org.apache.kafka.common.security.plain.PlainLoginModule required username=\"${KAFKA_USER}\" password=\"${KAFKA_PASS}\";
EOF
$*" 2>/dev/null
}

# ── Helper: query 0box postgres ───────────────────────────────────────────────
zbox_sql() {
    docker exec -e PGPASSWORD="${ZBOX_DB_PASS}" "${ZBOX_PG_CONTAINER}" \
        psql -U "${ZBOX_DB_USER}" -d "${ZBOX_DB_NAME}" -t -c "$1" 2>/dev/null | tr -d ' '
}

# ─────────────────────────────────────────────────────────────────────────────
# PHASE 1: Pre-chain — Kafka broker ready
# 0box starts AFTER the chain (requires 0dns which starts with the chain).
# Kafka retains all messages so 0box will consume full history when it connects.
# Must pass before chain is allowed to start.
# ─────────────────────────────────────────────────────────────────────────────
verify_pre_chain() {
    header "Kafka Pipeline: Pre-Chain Verification"
    info "Verifying Kafka broker is ready before chain starts..."
    info "Max wait: ${MAX_WAIT_PRE}s"

    local start; start=$(date +%s)
    local ok=false

    while true; do
        local elapsed=$(( $(date +%s) - start ))
        if [ "$elapsed" -ge "$MAX_WAIT_PRE" ]; then
            fail "Timed out after ${MAX_WAIT_PRE}s — Kafka not ready"
            return 1
        fi

        # 1. Kafka container running?
        if ! docker ps --format '{{.Names}}' | grep -q "^${KAFKA_CONTAINER}$"; then
            info "[${elapsed}s] Kafka container not running yet..."
            sleep "$POLL_INTERVAL"; continue
        fi

        # 2. Kafka broker accepting connections?
        if ! kafka_cmd "/opt/kafka/bin/kafka-topics.sh --bootstrap-server ${KAFKA_BROKER} --list --command-config /tmp/client.properties" &>/dev/null; then
            info "[${elapsed}s] Kafka broker not ready yet..."
            sleep "$POLL_INTERVAL"; continue
        fi

        ok=true; break
    done

    if ! $ok; then return 1; fi

    local elapsed=$(( $(date +%s) - start ))

    echo ""
    pass "Kafka container running"
    pass "Kafka broker accepting connections (${KAFKA_BROKER})"

    # Check if events topic already exists (informational only)
    local topics
    topics=$(kafka_cmd "/opt/kafka/bin/kafka-topics.sh --bootstrap-server ${KAFKA_BROKER} --list --command-config /tmp/client.properties" 2>/dev/null || echo "")
    if echo "$topics" | grep -q "^${KAFKA_TOPIC}$"; then
        pass "Kafka topic '${KAFKA_TOPIC}' exists"
    else
        warn "Kafka topic '${KAFKA_TOPIC}' not yet created (auto-created when chain starts)"
    fi

    echo ""
    pass "Pre-chain verification PASSED in ${elapsed}s — Kafka ready, safe to start chain"
    info "Note: 0box starts AFTER chain (requires 0dns). Kafka retains all events."
    return 0
}

# ─────────────────────────────────────────────────────────────────────────────
# PHASE 2: Post-chain — events flowing from chain → Kafka → 0box DB
# Must pass before blobbers and other services are set up.
# ─────────────────────────────────────────────────────────────────────────────
verify_post_chain() {
    header "Kafka Pipeline: Post-Chain Verification"
    info "Verifying events are flowing from chain → Kafka → 0box..."
    info "Max wait: ${MAX_WAIT_POST}s"

    local start; start=$(date +%s)
    local errors=0

    # ── Step 1: Wait for Kafka topic to have messages (chain is producing) ────
    info "Waiting for chain events to appear in Kafka topic '${KAFKA_TOPIC}'..."
    local topic_has_messages=false
    while true; do
        local elapsed=$(( $(date +%s) - start ))
        if [ "$elapsed" -ge "$MAX_WAIT_POST" ]; then
            fail "Timed out waiting for chain events in Kafka topic"
            errors=$((errors + 1))
            break
        fi

        local end_offset
        end_offset=$(kafka_cmd "/opt/kafka/bin/kafka-run-class.sh kafka.tools.GetOffsetShell \
            --broker-list ${KAFKA_BROKER} \
            --topic ${KAFKA_TOPIC} \
            --time -1 \
            --command-config /tmp/client.properties 2>/dev/null" 2>/dev/null \
            | awk -F: '{sum+=$3} END {print sum+0}')

        if [ "${end_offset:-0}" -gt 0 ] 2>/dev/null; then
            pass "Kafka topic '${KAFKA_TOPIC}': ${end_offset} messages produced by chain"
            topic_has_messages=true
            break
        fi
        info "[${elapsed}s] Waiting for chain to produce events to Kafka (current offset: ${end_offset:-0})..."
        sleep "$POLL_INTERVAL"
    done

    # ── Step 2: Wait for 0box consumer group to consume messages ─────────────
    if $topic_has_messages; then
        info "Waiting for 0box consumer group to consume messages..."
        local consumer_active=false
        local attempts=0
        local max_consumer_wait=60
        while true; do
            local elapsed=$(( $(date +%s) - start ))
            if [ "$elapsed" -ge "$MAX_WAIT_POST" ] || [ "$attempts" -ge $((max_consumer_wait / POLL_INTERVAL)) ]; then
                fail "0box consumer group '${KAFKA_GROUP}' not consuming messages"
                errors=$((errors + 1))
                break
            fi

            local lag_output
            lag_output=$(kafka_cmd "/opt/kafka/bin/kafka-consumer-groups.sh \
                --bootstrap-server ${KAFKA_BROKER} \
                --group ${KAFKA_GROUP} \
                --describe \
                --command-config /tmp/client.properties" 2>/dev/null || echo "")

            local committed
            committed=$(echo "$lag_output" | awk 'NR>2 && $5~/^[0-9]+$/ {sum+=$5} END {print sum+0}')

            if [ "${committed:-0}" -gt 0 ] 2>/dev/null; then
                local lag
                lag=$(echo "$lag_output" | awk 'NR>2 && $6~/^[0-9]+$/ {sum+=$6} END {print sum+0}')
                pass "0box consumer group active: committed=${committed}, lag=${lag:-?}"
                consumer_active=true
                break
            fi
            info "[${elapsed}s] 0box consumer group lag: checking... (committed=${committed:-0})"
            attempts=$((attempts + 1))
            sleep "$POLL_INTERVAL"
        done
    fi

    # ── Step 3: Verify 0box DB is being populated from chain events ───────────
    info "Verifying 0box database is populated from Kafka events..."
    local db_wait_start; db_wait_start=$(date +%s)
    local db_populated=false
    while true; do
        local elapsed=$(( $(date +%s) - start ))
        local db_elapsed=$(( $(date +%s) - db_wait_start ))
        if [ "$elapsed" -ge "$MAX_WAIT_POST" ]; then
            break
        fi

        local miner_count
        miner_count=$(zbox_sql "SELECT COUNT(*) FROM miners;" 2>/dev/null | tr -d ' \n' || echo "0")
        local sharder_count
        sharder_count=$(zbox_sql "SELECT COUNT(*) FROM sharders;" 2>/dev/null | tr -d ' \n' || echo "0")

        if [ "${miner_count:-0}" -gt 0 ] 2>/dev/null && [ "${sharder_count:-0}" -gt 0 ] 2>/dev/null; then
            pass "0box DB populated: ${miner_count} miners, ${sharder_count} sharders from Kafka events"
            db_populated=true
            break
        fi
        info "[${db_elapsed}s] 0box DB: miners=${miner_count:-0}, sharders=${sharder_count:-0} — waiting for Kafka events to arrive..."
        sleep "$POLL_INTERVAL"
    done

    if ! $db_populated; then
        local miner_count; miner_count=$(zbox_sql "SELECT COUNT(*) FROM miners;" 2>/dev/null | tr -d ' \n' || echo "0")
        local sharder_count; sharder_count=$(zbox_sql "SELECT COUNT(*) FROM sharders;" 2>/dev/null | tr -d ' \n' || echo "0")
        if [ "${miner_count:-0}" -eq 0 ] || [ "${sharder_count:-0}" -eq 0 ]; then
            fail "0box DB not populated from Kafka: miners=${miner_count:-0}, sharders=${sharder_count:-0}"
            errors=$((errors + 1))
        fi
    fi

    # ── Step 4: Check consumer lag is not growing (0box is keeping up) ────────
    local lag_output
    lag_output=$(kafka_cmd "/opt/kafka/bin/kafka-consumer-groups.sh \
        --bootstrap-server ${KAFKA_BROKER} \
        --group ${KAFKA_GROUP} \
        --describe \
        --command-config /tmp/client.properties" 2>/dev/null || echo "")

    local total_lag end_offset_total committed_total
    total_lag=$(echo "$lag_output" | awk 'NR>2 && $6~/^[0-9]+$/ {sum+=$6} END {print sum+0}')
    end_offset_total=$(echo "$lag_output" | awk 'NR>2 && $4~/^[0-9]+$/ {sum+=$4} END {print sum+0}')
    committed_total=$(echo "$lag_output" | awk 'NR>2 && $5~/^[0-9]+$/ {sum+=$5} END {print sum+0}')

    echo ""
    echo -e "  ${BOLD}Kafka Pipeline Summary:${NC}"
    echo "    Topic '${KAFKA_TOPIC}' end offset : ${end_offset_total:-0}"
    echo "    Consumer group committed offset  : ${committed_total:-0}"
    echo "    Consumer lag                     : ${total_lag:-0}"

    if [ "${total_lag:-0}" -lt 100 ] 2>/dev/null; then
        pass "Consumer lag acceptable (${total_lag:-0} messages behind)"
    else
        warn "Consumer lag is high: ${total_lag:-0} messages — 0box may be catching up"
    fi

    echo ""
    local elapsed=$(( $(date +%s) - start ))
    if [ $errors -eq 0 ]; then
        pass "Post-chain verification PASSED in ${elapsed}s — Kafka → 0box pipeline confirmed working"
        return 0
    else
        fail "Post-chain verification FAILED ($errors critical checks failed) — check Kafka/0box logs"
        return 1
    fi
}

# ─────────────────────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────────────────────
case "$PHASE" in
    pre-chain)
        verify_pre_chain
        ;;
    post-chain)
        verify_post_chain
        ;;
    both)
        verify_pre_chain && verify_post_chain
        ;;
    *)
        echo "Usage: $0 [pre-chain|post-chain|both]"
        exit 1
        ;;
esac
