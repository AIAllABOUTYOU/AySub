#!/usr/bin/env bash
set -euo pipefail

UPSTREAM_REMOTE="${UPSTREAM_REMOTE:-upstream}"
UPSTREAM_BRANCH="${UPSTREAM_BRANCH:-main}"
UPSTREAM_REF="${UPSTREAM_REMOTE}/${UPSTREAM_BRANCH}"

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" ]]; then
  printf '错误: 请在 Git 仓库中运行此脚本。\n' >&2
  exit 1
fi
cd "$repo_root"

usage() {
  cat <<'EOF'
AySub Sub2API 上游同步助手

用法:
  tools/upstream-sync.sh fetch
  tools/upstream-sync.sh inspect [--base <sha>]
  tools/upstream-sync.sh merge-preview [--base <sha>]
  tools/upstream-sync.sh baseline [--target <sha>] [--execute]
  tools/upstream-sync.sh status

命令:
  fetch          拉取 upstream/main，然后显示新增提交摘要
  inspect        列出基线后的提交、文件统计、与当前工作区重叠文件
  merge-preview  使用 git merge-tree 只读模拟合并并列出真实冲突
  baseline       为已完成并验证的同步创建 ours 策略桥接提交
  status         显示 HEAD、上游 HEAD、自动识别的基线和工作区状态

说明:
  - 建立桥接提交后，脚本会自动使用最近的共同祖先作为同步基线。
  - 首次建立桥接前，inspect/merge-preview 可通过 --base 指定上次已核销 SHA。
  - baseline 默认只预览；必须加 --execute，且工作区必须完全干净。
EOF
}

die() {
  printf '错误: %s\n' "$*" >&2
  exit 1
}

require_upstream_ref() {
  git rev-parse --verify "$UPSTREAM_REF" >/dev/null 2>&1 || \
    die "未找到 $UPSTREAM_REF，请先运行 tools/upstream-sync.sh fetch"
}

resolve_base() {
  local explicit_base="${1:-}"
  local base

  if [[ -n "$explicit_base" ]]; then
    git rev-parse --verify "${explicit_base}^{commit}" 2>/dev/null || \
      die "无效的基线提交: $explicit_base"
    return
  fi

  base="$(git merge-base HEAD "$UPSTREAM_REF" 2>/dev/null || true)"
  if [[ -z "$base" ]]; then
    die "HEAD 与 $UPSTREAM_REF 尚无共同祖先；请使用 --base <已核销上游 SHA>，或在同步完成后建立 baseline"
  fi
  printf '%s\n' "$base"
}

parse_base_option() {
  BASE_ARG=""
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --base)
        [[ $# -ge 2 ]] || die "--base 缺少 SHA"
        BASE_ARG="$2"
        shift 2
        ;;
      *) die "未知参数: $1" ;;
    esac
  done
}

print_range_summary() {
  local base="$1"
  local commit_count file_count

  require_upstream_ref
  git merge-base --is-ancestor "$base" "$UPSTREAM_REF" || \
    die "基线 $base 不是 $UPSTREAM_REF 的祖先"

  commit_count="$(git rev-list --count --no-merges "$base..$UPSTREAM_REF")"
  file_count="$(git diff --name-only "$base..$UPSTREAM_REF" | wc -l | tr -d ' ')"

  printf '基线:       %s\n' "$base"
  printf '上游:       %s\n' "$(git rev-parse "$UPSTREAM_REF")"
  printf '新增提交:   %s 个（不含 merge）\n' "$commit_count"
  printf '影响文件:   %s 个\n' "$file_count"

  if [[ "$commit_count" -gt 0 ]]; then
    printf '\n新增提交:\n'
    git log --format='  %h %s' --no-merges "$base..$UPSTREAM_REF"
  fi
}

fetch_upstream() {
  git remote get-url "$UPSTREAM_REMOTE" >/dev/null 2>&1 || \
    die "未配置 remote: $UPSTREAM_REMOTE"
  git fetch "$UPSTREAM_REMOTE" "$UPSTREAM_BRANCH"
  require_upstream_ref

  printf '\n拉取完成:\n'
  git log -1 --format='  %H %s' "$UPSTREAM_REF"

  local base
  base="$(git merge-base HEAD "$UPSTREAM_REF" 2>/dev/null || true)"
  if [[ -n "$base" ]]; then
    printf '\n'
    print_range_summary "$base"
  else
    printf '\n尚未建立 Git 上游基线。使用以下命令检查指定范围:\n'
    printf '  tools/upstream-sync.sh inspect --base <上次已核销上游 SHA>\n'
  fi
}

inspect_range() {
  local base="$1"
  local tmp_local tmp_upstream

  print_range_summary "$base"

  tmp_local="$(mktemp)"
  tmp_upstream="$(mktemp)"

  git status --porcelain=v1 | sed -E 's/^.. //' | sort -u >"$tmp_local"
  git diff --name-only "$base..$UPSTREAM_REF" | sort -u >"$tmp_upstream"

  printf '\n当前工作区与新增上游范围重叠:\n'
  if ! comm -12 "$tmp_local" "$tmp_upstream"; then
    die "计算重叠文件失败"
  fi
  local overlap_count
  overlap_count="$(comm -12 "$tmp_local" "$tmp_upstream" | wc -l | tr -d ' ')"
  printf '合计: %s 个重叠文件\n' "$overlap_count"
  rm -f "$tmp_local" "$tmp_upstream"
}

merge_preview() {
  local base="$1"
  local detected_base output_file conflict_file conflict_count

  require_upstream_ref
  detected_base="$(git merge-base HEAD "$UPSTREAM_REF" 2>/dev/null || true)"
  if [[ -z "$detected_base" ]]; then
    die "当前历史未桥接，拒绝模拟全仓 unrelated merge。先完成同步并运行 baseline；当前范围可用 inspect --base $base 查看"
  fi

  output_file="$(mktemp)"
  conflict_file="$(mktemp)"

  git merge-tree --messages --name-only HEAD "$UPSTREAM_REF" >"$output_file" || true
  sed -n 's/^CONFLICT ([^)]*): Merge conflict in //p' "$output_file" | sort -u >"$conflict_file"
  conflict_count="$(wc -l <"$conflict_file" | tr -d ' ')"

  print_range_summary "$detected_base"
  printf '\n真实文本冲突: %s 个文件\n' "$conflict_count"
  if [[ "$conflict_count" -gt 0 ]]; then
    sed 's/^/  /' "$conflict_file"
  fi
  rm -f "$output_file" "$conflict_file"
}

create_baseline() {
  local target="" execute="false"
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --target)
        [[ $# -ge 2 ]] || die "--target 缺少 SHA"
        target="$2"
        shift 2
        ;;
      --execute)
        execute="true"
        shift
        ;;
      *) die "未知参数: $1" ;;
    esac
  done

  require_upstream_ref
  target="${target:-$(git rev-parse "$UPSTREAM_REF")}"
  target="$(git rev-parse --verify "${target}^{commit}" 2>/dev/null)" || die "无效目标: $target"
  git merge-base --is-ancestor "$target" "$UPSTREAM_REF" || \
    die "目标 $target 不是当前 $UPSTREAM_REF 的祖先"

  if git merge-base --is-ancestor "$target" HEAD 2>/dev/null; then
    printf '目标 %s 已经是 HEAD 的祖先，无需重复建立基线。\n' "$target"
    return
  fi

  printf '将执行（保留 AySub 当前文件内容，仅建立 Git 祖先关系）:\n'
  printf '  git merge -s ours --allow-unrelated-histories %s -m %q\n' \
    "$target" "chore: establish Sub2API upstream sync baseline at ${target:0:10}"

  if [[ "$execute" != "true" ]]; then
    printf '\n当前为预览模式。确认目标范围已 100%% 适配并通过测试后，追加 --execute。\n'
    return
  fi

  [[ -z "$(git status --porcelain)" ]] || \
    die "工作区不干净。请先提交全部已适配代码，再建立 baseline"

  git merge -s ours --allow-unrelated-histories "$target" \
    -m "chore: establish Sub2API upstream sync baseline at ${target:0:10}"
  printf '基线建立完成: %s\n' "$(git rev-parse HEAD)"
}

show_status() {
  require_upstream_ref
  printf 'HEAD:        %s\n' "$(git rev-parse HEAD)"
  printf '上游:        %s\n' "$(git rev-parse "$UPSTREAM_REF")"
  local base
  base="$(git merge-base HEAD "$UPSTREAM_REF" 2>/dev/null || true)"
  printf '共同祖先:    %s\n' "${base:-未建立}"
  printf '工作区改动:  %s 个文件\n' "$(git status --porcelain | wc -l | tr -d ' ')"
}

main() {
  local command="${1:-}"
  [[ -n "$command" ]] || { usage; exit 0; }
  shift

  case "$command" in
    fetch) [[ $# -eq 0 ]] || die "fetch 不接受参数"; fetch_upstream ;;
    inspect)
      parse_base_option "$@"
      require_upstream_ref
      inspect_range "$(resolve_base "$BASE_ARG")"
      ;;
    merge-preview)
      parse_base_option "$@"
      require_upstream_ref
      merge_preview "$(resolve_base "$BASE_ARG")"
      ;;
    baseline) create_baseline "$@" ;;
    status) [[ $# -eq 0 ]] || die "status 不接受参数"; show_status ;;
    -h|--help|help) usage ;;
    *) usage >&2; die "未知命令: $command" ;;
  esac
}

main "$@"
