#!/usr/bin/env bash
# 음성 안내 WAV 재생성 (macOS `say` 한국어 voice → 16-bit PCM WAV).
# 문구·voice를 바꾸려면 아래를 수정하고 다시 실행하면 된다.
#
#   ./generate.sh           # 기본 voice(Yuna)
#   VOICE="Sora (Enhanced)" ./generate.sh
#
# 산출물: online.wav · suspect.wav · approved.wav (aplay/afplay 재생용)
set -euo pipefail
cd "$(dirname "$0")"
# Yuna (Premium) = 무대용 확정. Premium 미설치 환경이면 VOICE=Yuna 로 폴백.
VOICE="${VOICE:-Yuna (Premium)}"

gen() {  # gen <name> <text>
  say -v "$VOICE" -o "/tmp/_df_$1.aiff" "$2"
  afconvert -f WAVE -d LEI16@44100 -c 1 "/tmp/_df_$1.aiff" "$1.wav"
  rm -f "/tmp/_df_$1.aiff"
  echo "  ✅ $1.wav"
}

# 무대용 짧고 강한 문구.
gen online   "엣지 노드 가동. 라인 감시 시작."
gen suspect  "불량 의심. 스크래치 검출."
gen approved "격리 승인. 작업 지시 세 건, 격리합니다."
echo "완료 — voice: $VOICE"
