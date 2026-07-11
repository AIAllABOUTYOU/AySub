import { describe, expect, it } from "vitest";
import {
  parseFingerprintSignalsToRows,
  serializeFingerprintRowsToJSON,
} from "../codexFingerprintSignals";

describe("codex fingerprint signals 行编解码", () => {
  it("解析变体数组为斜杠分隔字符串", () => {
    const rows = parseFingerprintSignalsToRows(
      '[{"type":"header_exact","match":["session-id","session_id"],"required":true}]',
    );
    expect(rows).toEqual([
      { type: "header_exact", match: "session-id / session_id", required: true },
    ]);
  });

  it("序列化斜杠分隔字符串并保留 required", () => {
    const json = serializeFingerprintRowsToJSON([
      { type: "header_prefix", match: "x-codex-", required: true },
      { type: "body_path", match: " a / b ", required: false },
    ]);
    expect(JSON.parse(json)).toEqual([
      { type: "header_prefix", match: ["x-codex-"], required: true },
      { type: "body_path", match: ["a", "b"], required: false },
    ]);
  });

  it("空值和非法 JSON 返回空数组", () => {
    expect(parseFingerprintSignalsToRows("")).toEqual([]);
    expect(parseFingerprintSignalsToRows("nope")).toEqual([]);
    expect(serializeFingerprintRowsToJSON([])).toBe("[]");
  });
});
