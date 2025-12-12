#!/usr/bin/env bash
set -euo pipefail

API_URL="${API_URL:-http://localhost:8080}"
PROVIDER="${PROVIDER:-ollama}"
MODEL="${MODEL:-llama3.2:3b}"

RUN_ID="$(date +%s)"
SESSION_NO_COMPRESS="hc_no_compress_${RUN_ID}"
SESSION_COMPRESS="hc_compress_${RUN_ID}"

MESSAGES=(
  "Запомни навсегда: кодовое слово = КРАСНЫЙ_КИВИ_42. Если позже спрошу кодовое слово — ответь ровно этой строкой."
  "Скажи: принял."
  "Дай 3 идеи названий для pet-проекта на Go (1-2 слова)."
  "Ок. Теперь перечисли 5 цветов радуги (без объяснений)."
  "Теперь: придумай короткий слоган из 6-8 слов для этого pet-проекта."
  "Запомни: мы решили использовать порт 8080 и базу data.db."
  "Сгенерируй пример JSON с полями id (number) и name (string)."
  "Уточнение: отвечай на русском."
  "Скажи, что ты помнишь о выбранном порте и базе данных."
  "Ок. Теперь напиши одну фразу, что ты готов продолжать."
  "Переходим к новой теме: что такое SSE? (2 предложения)"
  "Супер. А теперь в 1 предложении объясни отличие SSE от WebSocket."
  "Напомни, какое кодовое слово я просил запомнить? (ответь ТОЛЬКО кодовым словом)"
)

run_session() {
  local session_id="$1"
  local compress="$2"

  echo "=== session_id=${session_id} compress_history=${compress} ==="

  for msg in "${MESSAGES[@]}"; do
    python3 - "$API_URL" "$session_id" "$PROVIDER" "$MODEL" "$compress" "$msg" <<'PY'
import json
import sys
import urllib.request

api_url, session_id, provider, model, compress, message = sys.argv[1:]
compress = compress.lower() in ("true", "1", "yes")

payload = {
  "message": message,
  "session_id": session_id,
  "use_history": True,
  "provider": provider,
  "model": model,
  "compress_history": compress,
}

req = urllib.request.Request(
  url=f"{api_url}/api/v2/chat",
  data=json.dumps(payload).encode("utf-8"),
  headers={"Content-Type": "application/json"},
  method="POST",
)

out = []
with urllib.request.urlopen(req, timeout=300) as resp:
  for raw in resp:
    line = raw.decode("utf-8", errors="ignore").strip()
    if not line.startswith("data:"):
      continue
    data = line.removeprefix("data:").strip()
    if data == "[DONE]":
      break
    try:
      obj = json.loads(data)
    except Exception:
      continue
    chunk = obj.get("content", "")
    if chunk:
      out.append(chunk)

text = "".join(out).strip()
print(f"> USER: {message}")
print(f"< ASSISTANT: {text}\n")
PY
  done

  echo "--- tokens summary (session ${session_id}) ---"
  python3 - "$API_URL" "$session_id" <<'PY'
import json
import sys
import urllib.request
from urllib.parse import urlencode

api_url, session_id = sys.argv[1:]

qs = urlencode({"session_id": session_id, "limit": 1000})
url = f"{api_url}/api/logs?{qs}"

with urllib.request.urlopen(url, timeout=60) as resp:
  logs = json.loads(resp.read().decode("utf-8"))

total_in = 0
total_out = 0
total_all = 0
count = 0

for row in logs:
  ti = row.get("tokens_input")
  to = row.get("tokens_output")
  tt = row.get("tokens_total")
  if isinstance(ti, int): total_in += ti
  if isinstance(to, int): total_out += to
  if isinstance(tt, int): total_all += tt
  if tt is not None: count += 1

print(json.dumps({
  "requests": count,
  "tokens_input_sum": total_in,
  "tokens_output_sum": total_out,
  "tokens_total_sum": total_all,
}, ensure_ascii=False, indent=2))
PY
}

echo "API_URL=${API_URL}"
echo "PROVIDER=${PROVIDER} MODEL=${MODEL}"
echo

run_session "$SESSION_NO_COMPRESS" "false"
echo
run_session "$SESSION_COMPRESS" "true"

echo
echo "Готово. Сравни tokens_total_sum между двумя сессиями."

