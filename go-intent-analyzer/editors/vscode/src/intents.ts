export interface IntentVisual {
  id: string;
  label: string;
  description: string;
  color: string;
}

export const INTENT_VISUALS: IntentVisual[] = [
  { id: "simplicity", label: "Simplicity", description: "単純性を優先する意図", color: "#4CAF50" },
  { id: "change_localization", label: "Change localization", description: "変更の影響範囲を局所化する意図", color: "#26A69A" },
  { id: "extensibility", label: "Extensibility", description: "実装や振る舞いを追加可能にする意図", color: "#42A5F5" },
  { id: "reliability", label: "Reliability", description: "障害、再試行、復旧を考慮する意図", color: "#5C6BC0" },
  { id: "performance", label: "Performance", description: "CPU、メモリ、I/O、遅延を改善する意図", color: "#AB47BC" },
  { id: "security", label: "Security", description: "信頼境界、認可、入力、秘密情報を守る意図", color: "#EF5350" },
  { id: "backward_compatibility", label: "Backward compatibility", description: "既存の契約やデータ形式を維持する意図", color: "#FF7043" },
  { id: "testability", label: "Testability", description: "依存や副作用を制御してテストしやすくする意図", color: "#66BB6A" },
  { id: "operability", label: "Operability", description: "観測、設定、診断、復旧を容易にする意図", color: "#FFA726" },
  { id: "migration_safety", label: "Migration safety", description: "段階移行やロールバックを安全にする意図", color: "#8D6E63" },
  { id: "delivery_speed", label: "Delivery speed", description: "完全性より早期提供や検証を優先する意図", color: "#EC407A" }
];

export const INTENT_COLORS = Object.fromEntries(INTENT_VISUALS.map((intent) => [intent.id, intent.color])) as Record<string, string>;
