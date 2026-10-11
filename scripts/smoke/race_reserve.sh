
#!/usr/bin/env bash
set -euo pipefail

BASE_URL="http://localhost:8080"
DB_NAME="postgres_test"
CONTAINER="postgres"
ID_FILE="/tmp/race_reserve_part_id.txt"
CONSTRAINT="reserve_events_test_insert_failure"

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
DIM='\033[2m'
RESET='\033[0m'

PASS=0
FAIL=0

# ─── PostgreSQL ───────────────────────────────────────────────────────────────

pg() {
  docker exec -i "$CONTAINER" sh -c \
    'exec psql -X -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d "$1" "${@:2}"' \
    sh "$DB_NAME" "$@"
}

db_query() {
  local sql="$1"

  echo -e "  ${DIM}psql: ${sql}${RESET}" >&2

  pg -At -c "$sql" | tr -d '[:space:]'
}

# ─── Helpers ──────────────────────────────────────────────────────────────────

section() {
  echo ""
  echo -e "${CYAN}${BOLD}══════════════════════════════════════════${RESET}"
  echo -e "${CYAN}${BOLD}  $1${RESET}"
  echo -e "${CYAN}${BOLD}══════════════════════════════════════════${RESET}"
}

check() {
  local description="$1"
  local result="$2"

  if [ "$result" = "true" ]; then
    echo -e "  ${GREEN}✓${RESET}  ${description}"
    PASS=$((PASS + 1))
  else
    echo -e "  ${RED}✗${RESET}  ${description}"
    FAIL=$((FAIL + 1))
  fi
}

summary() {
  echo ""
  echo -e "${CYAN}${BOLD}══════════════════════════════════════════${RESET}"

  local total=$((PASS + FAIL))

  if [ "$FAIL" -eq 0 ]; then
    echo -e "${GREEN}${BOLD}  ВСЕ ТЕСТЫ ПРОШЛИ${RESET} (${PASS}/${total})"
  else
    echo -e "${RED}${BOLD}  ЕСТЬ УПАВШИЕ ТЕСТЫ${RESET}"
    echo -e "  Прошло: ${PASS}, упало: ${FAIL}"
  fi

  echo -e "${CYAN}${BOLD}══════════════════════════════════════════${RESET}"

  [ "$FAIL" -eq 0 ]
}

cleanup() {
  # Удаляем только наше временное ограничение.
  pg -c "
    ALTER TABLE reserve_events
    DROP CONSTRAINT IF EXISTS ${CONSTRAINT};
  " >/dev/null 2>&1 || true
}

preflight() {
  if [ "$DB_NAME" != "postgres_test" ]; then
    echo "Разрешена только тестовая база postgres_test"
    exit 1
  fi

  if ! pg -At -c "SELECT 1;" >/dev/null; then
    echo "PostgreSQL недоступен"
    exit 1
  fi

  if ! curl -fsS "${BASE_URL}/healthz" >/dev/null; then
    echo "HTTP-сервер недоступен на ${BASE_URL}"
    exit 1
  fi
}

usage() {
  echo "Использование: $0 <команда>"
  echo ""
  echo "Команды:"
  echo "  setup — очистка БД и создание детали"
  echo "  crash — имитация сбоя INSERT"
  exit 1
}

# ─── Setup ────────────────────────────────────────────────────────────────────

cmd_setup() {
  section "Подготовка — создаём деталь с quantity=10"

  preflight

  cleanup

  # Очищаем тестовые данные.
  pg -c "
    TRUNCATE TABLE reserve_events, parts CASCADE;
  " >/dev/null

  PART=$(curl -fsS -X POST "${BASE_URL}/parts" \
    -H 'Content-Type: application/json' \
    -d '{"name":"Тестовый болт","quantity":10,"price":500}')

  ID=$(echo "$PART" | python3 -c \
    "import sys,json; print(json.load(sys.stdin)['id'])")

  echo "$ID" > "$ID_FILE"

  QUANTITY=$(db_query \
    "SELECT quantity FROM parts WHERE id='${ID}';")

  RESERVED=$(db_query \
    "SELECT reserved FROM parts WHERE id='${ID}';")

  EVENTS=$(db_query \
    "SELECT COUNT(*) FROM reserve_events WHERE part_id='${ID}';")

  echo ""
  echo -e "  Создана деталь: ${ID}"
  echo ""
  echo -e "  ${BOLD}Состояние БД:${RESET}"
  echo "  quantity = ${QUANTITY}"
  echo "  reserved = ${RESERVED}"
  echo "  events   = ${EVENTS}"

  AVAILABLE=$((QUANTITY - RESERVED))

  echo -e "  ${YELLOW}Доступно для резерва: ${AVAILABLE}${RESET}"

  check "quantity = 10" "$([ "$QUANTITY" = "10" ] && echo true || echo false)"
  check "reserved = 0" "$([ "$RESERVED" = "0" ] && echo true || echo false)"
  check "events = 0" "$([ "$EVENTS" = "0" ] && echo true || echo false)"

  summary
}

# ─── Crash ────────────────────────────────────────────────────────────────────

cmd_crash() {
  if [ ! -f "$ID_FILE" ]; then
    echo -e "${RED}Сначала выполни: $0 setup${RESET}"
    exit 1
  fi

  preflight

  ID=$(cat "$ID_FILE")

  if [[ ! "$ID" =~ ^[0-9a-fA-F-]{36}$ ]]; then
    echo "Некорректный UUID"
    exit 1
  fi

  section "Имитация частичного сбоя"

  # Убираем ограничение от предыдущего запуска.
  cleanup

  # Проверяем наличие детали.
  EXISTS=$(db_query \
    "SELECT COUNT(*) FROM parts WHERE id='${ID}';")

  if [ "$EXISTS" != "1" ]; then
    echo "Тестовая деталь не найдена. Выполни setup."
    exit 1
  fi

  # Сбрасываем состояние тестовой детали.
  pg -c "
    DELETE FROM reserve_events WHERE part_id='${ID}';
    UPDATE parts SET reserved=0 WHERE id='${ID}';
  " >/dev/null

  echo "  reserved сброшен в 0"

  # Временное ограничение запрещает INSERT.
  # NOT VALID позволяет добавить его без проверки старых строк,
  # но новые INSERT должны удовлетворять CHECK.
  pg -c "
    ALTER TABLE reserve_events
    ADD CONSTRAINT ${CONSTRAINT}
    CHECK (false) NOT VALID;
  " >/dev/null

  # Удаляем ограничение при завершении скрипта,
  # даже если запрос завершится с ошибкой.
  trap cleanup EXIT

  echo ""
  echo "  INSERT в reserve_events теперь завершится ошибкой."
  echo "  Отправляем запрос резервирования..."

  HTTP=$(curl -sS -o /dev/null -w '%{http_code}' \
    -X POST "${BASE_URL}/parts/reserve" \
    -H 'Content-Type: application/json' \
    -d "{\"id\":\"${ID}\",\"quantity\":3}")

  RESERVED=$(db_query \
    "SELECT reserved FROM parts WHERE id='${ID}';")

  EVENTS=$(db_query \
    "SELECT COUNT(*) FROM reserve_events WHERE part_id='${ID}';")

  section "Результат"

  echo "  HTTP     = ${HTTP}"
  echo "  reserved = ${RESERVED}"
  echo "  events   = ${EVENTS}"

  echo ""

  if [ "$HTTP" = "500" ] && [ "$RESERVED" -gt 0 ]; then
    echo -e "${RED}${BOLD}  ОБНАРУЖЕН ЧАСТИЧНЫЙ СБОЙ${RESET}"
    echo "  UPDATE прошёл, но INSERT завершился ошибкой."
    echo "  Данные рассинхронизированы."

  elif [ "$HTTP" = "500" ] && [ "$RESERVED" = "0" ]; then
    echo -e "${GREEN}${BOLD}  Частичного изменения нет${RESET}"
    echo "  reserved остался 0."
    echo "  Это совместимо с корректным откатом транзакции."

  else
    echo -e "${YELLOW}  Неожиданный результат.${RESET}"
    echo "  Проверь обработчик /parts/reserve."
  fi

  echo ""

  check "HTTP = 500" \
    "$([ "$HTTP" = "500" ] && echo true || echo false)"

  check "событие не записано" \
    "$([ "$EVENTS" = "0" ] && echo true || echo false)"

  # Этот сценарий проверяет именно наличие частичного сбоя.
  # После исправления кода на транзакции reserved должен быть 0.
  check "частичный сбой воспроизведён" \
    "$([ "$RESERVED" -gt 0 ] && echo true || echo false)"

  cleanup
  trap - EXIT

  summary
}

# ─── Main ─────────────────────────────────────────────────────────────────────

case "${1:-}" in
  setup)
    cmd_setup
    ;;
  crash)
    cmd_crash
    ;;
  *)
    usage
    ;;
esac
