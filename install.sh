#!/usr/bin/env bash
# ccsm-go Installer
# Builds from source (requires Go 1.21+) or uses pre-built binary
set -e

BOLD='\033[1m'
GREEN='\033[32m'
YELLOW='\033[33m'
RED='\033[31m'
CYAN='\033[36m'
RESET='\033[0m'

INSTALL_DIR="${HOME}/.local/bin"
HOOK_DIR="${HOME}/.claude/hooks"
SETTINGS_FILE="${HOME}/.claude/settings.json"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Detect update vs fresh install
IS_UPDATE=false
if command -v ccsm &>/dev/null || [ -f "$INSTALL_DIR/ccsm" ]; then
    IS_UPDATE=true
fi

# Language detection
L="en"
DETECTED_LOCALE="${CCSM_LANG:-${LC_ALL:-${LANG:-}}}"
if [[ "$DETECTED_LOCALE" == de* ]]; then
    L="de"
fi

# Check if existing Go ccsm has language configured
if $IS_UPDATE && [ -f "${HOME}/.claude/ccsm.db" ]; then
    # Language from DB is handled by the binary itself
    true
elif $IS_UPDATE && [ -f "${HOME}/.claude/ccsm.conf" ]; then
    saved_lang=$(grep -E '^CCSM_LANG=' "${HOME}/.claude/ccsm.conf" 2>/dev/null | cut -d= -f2 | tr -d ' ')
    [ "$saved_lang" = "de" ] && L="de"
elif [[ "$DETECTED_LOCALE" == de* ]]; then
    echo -e "${BOLD}Sprache / Language${RESET}"
    echo -e "  Deutsche Locale erkannt (${DETECTED_LOCALE})"
    echo -ne "  ${BOLD}Deutsche Sprache verwenden?${RESET} [j/n] "
    read -r -n 1 lang_choice
    echo ""
    if [[ "${lang_choice,,}" != "j" ]] && [[ -n "$lang_choice" ]]; then
        L="en"
    fi
    echo ""
fi

# Translations
if [[ "$L" == "de" ]]; then
    T_TITLE="ccsm Installer (Go-Version)"
    T_CHECKING="Prüfe Abhängigkeiten..."
    T_BUILDING="Baue ccsm..."
    T_UPDATE="ccsm ist bereits installiert — wird aktualisiert."
    T_RESTART="Hinweis: Falls ccsm gerade läuft, bitte beenden und neu starten."
    T_HOOK_CONFIG="Konfiguriere Claude Code Hook..."
    T_HOOK_EXISTS="Hook bereits in settings.json vorhanden"
    T_HOOK_ADDED="Hook zu settings.json hinzugefügt"
    T_HOOK_MANUAL="Konnte Hook nicht automatisch hinzufügen. Bitte manuell eintragen:"
    T_HOOK_CREATED="erstellt"
    T_DONE="Installation abgeschlossen!"
    T_NOT_IN_PATH="ist nicht im PATH."
    T_ADD_PATH="Füge folgende Zeile zu ~/.bashrc oder ~/.zshrc hinzu:"
    T_START="Starte mit"
    T_MIGRATE="Vorhandene Bash-Daten migrieren mit:"
else
    T_TITLE="ccsm Installer (Go version)"
    T_CHECKING="Checking dependencies..."
    T_BUILDING="Building ccsm..."
    T_UPDATE="ccsm is already installed — updating."
    T_RESTART="Note: If ccsm is currently running, please quit and restart it."
    T_HOOK_CONFIG="Configuring Claude Code hook..."
    T_HOOK_EXISTS="Hook already in settings.json"
    T_HOOK_ADDED="Hook added to settings.json"
    T_HOOK_MANUAL="Could not add hook automatically. Please add manually:"
    T_HOOK_CREATED="created"
    T_DONE="Installation complete!"
    T_NOT_IN_PATH="is not in PATH."
    T_ADD_PATH="Add the following line to ~/.bashrc or ~/.zshrc:"
    T_START="Start with"
    T_MIGRATE="Migrate existing bash data with:"
fi

echo -e "${BOLD}${T_TITLE}${RESET}"
echo ""

# Check dependencies
echo -e "${BOLD}${T_CHECKING}${RESET}"

HAS_GO=false
if command -v go &>/dev/null; then
    echo -e "  ${GREEN}✓${RESET} Go ($(go version | awk '{print $3}'))"
    HAS_GO=true
else
    echo -e "  ${RED}✗${RESET} Go (not installed)"
fi

if command -v claude &>/dev/null; then
    echo -e "  ${GREEN}✓${RESET} Claude Code"
else
    echo -e "  ${YELLOW}!${RESET} Claude Code (recommended)"
fi

if command -v jq &>/dev/null; then
    echo -e "  ${GREEN}✓${RESET} jq (for hook)"
else
    echo -e "  ${YELLOW}!${RESET} jq (needed for session hook)"
fi

if $IS_UPDATE; then
    echo ""
    echo -e "${YELLOW}${T_UPDATE}${RESET}"
fi

echo ""

# Build
if [ -f "$SCRIPT_DIR/Makefile" ] && $HAS_GO; then
    echo -e "${BOLD}${T_BUILDING}${RESET}"
    cd "$SCRIPT_DIR"
    make build
    BINARY="$SCRIPT_DIR/ccsm"
elif [ -f "$SCRIPT_DIR/ccsm" ]; then
    # Pre-built binary in repo
    BINARY="$SCRIPT_DIR/ccsm"
else
    echo -e "${RED}Error: No Go compiler and no pre-built binary found.${RESET}"
    echo "Install Go: https://go.dev/dl/"
    exit 1
fi

# Install binary
mkdir -p "$INSTALL_DIR" "$HOOK_DIR"
cp "$BINARY" "$INSTALL_DIR/ccsm"
chmod +x "$INSTALL_DIR/ccsm"
echo -e "  ${GREEN}✓${RESET} ${INSTALL_DIR}/ccsm"

# Install hook script (reuse bash hook — it's called by Claude Code)
HOOK_SCRIPT="${HOOK_DIR}/ccsm_session_end.sh"
if [ -f "$SCRIPT_DIR/hooks/session_end.sh" ]; then
    cp "$SCRIPT_DIR/hooks/session_end.sh" "$HOOK_SCRIPT"
elif [ -f "${HOME}/.claude/hooks/ccsm_session_end.sh" ]; then
    echo -e "  ${YELLOW}○${RESET} Hook script already exists"
else
    # Create minimal hook script
    cat > "$HOOK_SCRIPT" <<'HOOKEOF'
#!/usr/bin/env bash
# ccsm SessionEnd Hook — saves session metadata to temp file
CCSM_TMPDIR="/tmp/ccsm"
mkdir -p "$CCSM_TMPDIR"
for cmd in jq python3; do
    command -v "$cmd" &>/dev/null || exit 0
done
INPUT=$(cat)
[ -z "$INPUT" ] && exit 0
SESSION_ID=$(echo "$INPUT" | jq -r '.session_id // empty')
CWD=$(echo "$INPUT" | jq -r '.cwd // empty')
TRANSCRIPT=$(echo "$INPUT" | jq -r '.transcript_path // empty')
[ -z "$SESSION_ID" ] && exit 0
TMPFILE="${CCSM_TMPDIR}/session-${SESSION_ID}.json"
BETREFF=""
TOKENS="0/0"
if [ -n "$TRANSCRIPT" ] && [ -f "$TRANSCRIPT" ]; then
    local_output=$(python3 - "$TRANSCRIPT" <<'PYEOF'
import json, sys
transcript_path = sys.argv[1]
betreff = ""
input_tokens = 0
output_tokens = 0
cache_read = 0
cache_create = 0
try:
    with open(transcript_path) as f:
        for line in f:
            try:
                obj = json.loads(line)
                if not betreff and obj.get('type') == 'user' and 'message' in obj:
                    msg = obj['message']
                    if isinstance(msg, dict) and msg.get('role') == 'user':
                        for c in msg.get('content', []):
                            if isinstance(c, dict) and c.get('type') == 'text':
                                betreff = c['text'].strip().replace('\n', ' ').replace('\t', ' ')[:120]
                                break
                msg = obj.get('message', {})
                if isinstance(msg, dict) and 'usage' in msg:
                    u = msg['usage']
                    input_tokens += u.get('input_tokens', 0)
                    output_tokens += u.get('output_tokens', 0)
                    cache_read += u.get('cache_read_input_tokens', 0)
                    cache_create += u.get('cache_creation_input_tokens', 0)
            except (json.JSONDecodeError, KeyError):
                pass
except Exception:
    pass
total_in = input_tokens + cache_read + cache_create
print(f"{betreff}\t{total_in}/{output_tokens}")
PYEOF
    )
    BETREFF=$(echo "$local_output" | cut -f1)
    TOKENS=$(echo "$local_output" | cut -f2)
fi
BETREFF=$(echo "$BETREFF" | tr '\t' ' ')
[ -z "$TOKENS" ] && TOKENS="0/0"
jq -n --arg sid "$SESSION_ID" --arg cwd "$CWD" --arg betreff "$BETREFF" \
    --arg transcript "$TRANSCRIPT" --arg tokens "$TOKENS" \
    '{"session_id": $sid, "cwd": $cwd, "betreff": $betreff, "transcript": $transcript, "tokens": $tokens}' > "$TMPFILE"
find "$CCSM_TMPDIR" -name "session-*.json" -mtime +1 -delete 2>/dev/null
HOOKEOF
    echo -e "  ${GREEN}✓${RESET} ${HOOK_SCRIPT} (${T_HOOK_CREATED})"
fi
chmod +x "$HOOK_SCRIPT"

# Zsh completion
if command -v zsh &>/dev/null; then
    COMP_DIR="${HOME}/.local/share/zsh/site-functions"
    mkdir -p "$COMP_DIR"
    "$INSTALL_DIR/ccsm" completion zsh > "$COMP_DIR/_ccsm" 2>/dev/null || true
    echo -e "  ${GREEN}✓${RESET} ${COMP_DIR}/_ccsm"
fi

# Configure hook in settings.json
echo ""
echo -e "${BOLD}${T_HOOK_CONFIG}${RESET}"

if [ -f "$SETTINGS_FILE" ]; then
    if grep -q "ccsm_session_end" "$SETTINGS_FILE" 2>/dev/null; then
        echo -e "  ${YELLOW}○${RESET} ${T_HOOK_EXISTS}"
    else
        if command -v jq &>/dev/null; then
            tmp_settings="${SETTINGS_FILE}.tmp"
            jq '.hooks.SessionEnd += [{"matcher": "", "hooks": [{"type": "command", "command": "bash '"${HOOK_DIR}"'/ccsm_session_end.sh", "timeout": 5}]}]' "$SETTINGS_FILE" > "$tmp_settings" 2>/dev/null
            if [ $? -eq 0 ]; then
                mv "$tmp_settings" "$SETTINGS_FILE"
                echo -e "  ${GREEN}✓${RESET} ${T_HOOK_ADDED}"
            else
                rm -f "$tmp_settings"
                echo -e "  ${YELLOW}!${RESET} ${T_HOOK_MANUAL}"
                echo '    "hooks": {"SessionEnd": [{"matcher": "", "hooks": [{"type": "command", "command": "bash ~/.claude/hooks/ccsm_session_end.sh", "timeout": 5}]}]}'
            fi
        else
            echo -e "  ${YELLOW}!${RESET} jq not found — ${T_HOOK_MANUAL}"
            echo '    "hooks": {"SessionEnd": [{"matcher": "", "hooks": [{"type": "command", "command": "bash ~/.claude/hooks/ccsm_session_end.sh", "timeout": 5}]}]}'
        fi
    fi
else
    cat > "$SETTINGS_FILE" <<'SETTINGSEOF'
{
  "hooks": {
    "SessionEnd": [
      {
        "matcher": "",
        "hooks": [
          {
            "type": "command",
            "command": "bash ~/.claude/hooks/ccsm_session_end.sh",
            "timeout": 5
          }
        ]
      }
    ]
  }
}
SETTINGSEOF
    echo -e "  ${GREEN}✓${RESET} ${SETTINGS_FILE} (${T_HOOK_CREATED})"
fi

# PATH check + final message
echo ""
if echo "$PATH" | tr ':' '\n' | grep -q "$INSTALL_DIR"; then
    echo -e "${GREEN}${BOLD}${T_DONE}${RESET}"
    if $IS_UPDATE; then
        echo ""
        echo -e "${YELLOW}${T_RESTART}${RESET}"
    fi
else
    echo -e "${YELLOW}${BOLD}${T_DONE}${RESET}"
    echo ""
    echo -e "${YELLOW}${INSTALL_DIR} ${T_NOT_IN_PATH}${RESET}"
    echo "${T_ADD_PATH}"
    echo ""
    echo "  export PATH=\"\$HOME/.local/bin:\$PATH\""
    echo ""
fi

# Migration hint (if bash data exists and no Go DB yet)
if [ -f "${HOME}/.claude/session_log.tsv" ] && [ ! -f "${HOME}/.claude/ccsm.db" ]; then
    echo ""
    echo -e "${CYAN}${T_MIGRATE}${RESET}"
    echo "  ccsm migrate"
    echo ""
fi

echo -e "${T_START}: ${CYAN}ccsm${RESET}"
