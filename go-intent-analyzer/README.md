# go-intent-analyzer

Goコードから読み取れる設計意図を、Jevの選択式判定として出力するCLIです。

実現度、技術的負債、設計品質、想定時間軸は評価しません。証拠が足りない意図は`insufficient`として扱い、`evidenceFit`を`null`にします。

## 構成

```text
analyzer/
├── analyzer.go                # state構築・回答変換
├── extractor.go               # 現行Go解析器のAST・型・Git抽出
├── types.go                   # Reportと解析コンテキスト
├── jev.go                     # Jev transport
├── questions/
│   └── questions.go           # 共通のquestion builder
└── languages/
    └── golang/
        └── questions.go       # Go専用の質問文・証拠キーワード
```

言語ディレクトリには、まず質問内容だけを置きます。questionの組み立て、Jev通信、回答からReportへの変換など、同じ処理は`analyzer/`で共有します。別言語で構文解析方法や回答mappingが本当に異なる場合に限り、その差分を追加します。

## 判定単位

- function
- method
- type
- package

指定ディレクトリ以下のGoパッケージを再帰的に解析します。局所コンテキストとして、宣言、doc comment、同一パッケージ内の呼び出し関係、暗黙的なinterface実装、関連テスト、パッケージ内宣言、ファイルまたはディレクトリのGit履歴を収集します。

## 判定する意図

`simplicity`、`change_localization`、`extensibility`、`reliability`、`performance`、`security`、`backward_compatibility`、`testability`、`operability`、`migration_safety`、`delivery_speed`の11種です。

各意図は独立したJev `choice`質問です。

- `supported`: 複数の独立した証拠が意図を支持
- `weakly_supported`: 限定的または曖昧な証拠が支持
- `contradicted`: 意図に関する証拠同士が矛盾
- `insufficient`: 意図を推定できる証拠が不足

`contradicted`は実装品質の評価には使用しません。

## 実行

必要なのはTypeSafeのAPIキーです。`OPENAI_API_KEY`は使用しません。

```bash
export TYPESAFE_API_KEY=...
go run ./cmd -dir ./path/to/repository > analysis.json
```

オプション:

```text
-dir               解析対象ディレクトリ
-model             Jevモデル名。既定値はjev-latest
-min-evidencefit   evidenceFitが指定値未満の意図を除外
```

## 出力

```json
{
  "schemaVersion": "1.0",
  "analyzerVersion": "0.1.0",
  "generatedAt": "2026-09-26T00:00:00Z",
  "repository": {
    "root": "/absolute/path/to/repository",
    "revision": "abc123..."
  },
  "units": [
    {
      "id": "internal/api/handler.go#function:HandleRequest:42",
      "uri": "/absolute/path/to/internal/api/handler.go",
      "range": {
        "start": { "line": 41, "character": 0 },
        "end": { "line": 57, "character": 1 }
      },
      "kind": "function",
      "name": "HandleRequest",
      "intents": [
        {
          "intent": "backward_compatibility",
          "assessment": "supported",
          "evidenceFit": 0.88,
          "confidence": 0.91,
          "probabilities": {
            "supported": 0.82,
            "weakly_supported": 0.12,
            "contradicted": 0.02,
            "insufficient": 0.04
          },
          "evidence": [
            {
              "id": "e-...",
              "kind": "source",
              "uri": "/absolute/path/to/internal/api/handler.go",
              "range": {
                "start": { "line": 41, "character": 0 },
                "end": { "line": 57, "character": 1 }
              },
              "signal": "判定対象の宣言"
            }
          ]
        }
      ]
    }
  ]
}
```

`confidence`はJevの`ChoiceAnswer.confidence`です。`evidenceFit`は表示用の派生値で、次の式により計算します。

```text
P(supported) + 0.5 × P(weakly_supported)
```

`evidence`は判定に与えた関連証拠候補です。Jevの内部推論や引用を表すものではありません。

位置情報はLSPと同様に0始まりで、`character`はUTF-16コード単位です。

## 開発

```bash
go test ./...
go vet ./...
go build ./...
```

## VS Code拡張

拡張はGo CLIを子プロセスとして起動し、JevのAnalysis Reportをコード背景、左端のバー、overview ruler、ホバーへ表示します。`insufficient`は着色しません。複数の意図がある場合は、`evidenceFit × confidence`を重みとして最大3色を混色します。

Explorer内の`Intent Legend`ビューに、11種類の意図、基準色、説明を表示します。コード背景は複数の基準色を混色するため、凡例の単色と完全には一致しない場合があります。左端のバーとoverview rulerは、最も重みが大きい意図の基準色です。着色箇所へマウスを置くと、意図ごとの判定、`evidenceFit`、`confidence`、根拠候補を確認できます。

### CLIバイナリが別途必要な理由

VSIXに含まれるのは、VS Code上で結果を表示するTypeScript/JavaScript部分です。Goのソース解析とJev API呼び出しは、別プロセスの`go-intent-analyzer` CLIが担当します。

```text
VS Code拡張
    └─ go-intent-analyzer CLIを起動
           ├─ GoコードとGit履歴を解析
           ├─ Jev APIを呼び出す
           └─ Analysis Reportを標準出力へ返す
```

現時点のVSIXにはOS・CPU別のGoバイナリを同梱していません。一括セットアップで既定位置へCLIを配置するか、手動でビルドしてVS Code設定へ絶対パスを指定します。

### 一括セットアップ（推奨）

リポジトリのルートで次を実行します。

```bash
./install_setup.sh
```

このスクリプトが次を順番に実行します。

1. Go CLIを`~/.local/bin/go-intent-analyzer`へビルド・配置
2. pnpmでVS Code拡張をビルド・テストし、VSIXを生成
3. `code --install-extension`でVSIXをインストール
4. TypeSafe APIキーを非表示で入力し、`~/.config/go-intent-analyzer/env`へ権限600で保存

拡張は`~/.local/bin/go-intent-analyzer`を自動検出するため、一括セットアップを使う場合は`settings.json`の変更もPATHの設定も不要です。セットアップ後はVS Codeを再読み込みし、Goワークスペースで`Go Intent Analyzer: Analyze Workspace`を実行します。

前提コマンドは`go`、`pnpm`、`code`です。macOSで`code`がPATHにない場合は、通常の`/Applications/Visual Studio Code.app`も自動検出します。

以下は一括セットアップを使わない場合の手動手順です。

リポジトリのルートでCLIをビルドします。

```bash
mkdir -p bin
go build -o bin/go-intent-analyzer ./cmd
```

VS Codeの`settings.json`へ生成したCLIの絶対パスを設定します。

```json
{
  "goIntentAnalyzer.executablePath": "/Users/uenokensuke/Apps/jev-tools/go-intent-analyzer/bin/go-intent-analyzer"
}
```

`settings.json`はCommand Paletteの`Preferences: Open User Settings (JSON)`から開けます。絶対パスを設定する方法では、CLIをPATHへ追加する必要はありません。

拡張を開発実行する場合:

```bash
cd editors/vscode
pnpm install --frozen-lockfile
pnpm test
code .
```

VS Codeで`F5`を押すとExtension Development Hostが起動します。解析対象のGoワークスペースを開き、Command Paletteから次を実行します。

```text
Go Intent Analyzer: Set TypeSafe API Key
Go Intent Analyzer: Analyze Workspace
```

TypeSafe APIキーは`Go Intent Analyzer: Set TypeSafe API Key`でVS Code SecretStorageへ保存できます。環境変数`TYPESAFE_API_KEY`も利用できます。

着色設定:

- `goIntentAnalyzer.minimumEvidenceFit`
- `goIntentAnalyzer.minimumConfidence`
- `goIntentAnalyzer.maxVisibleIntents`

API呼び出しを意図せず繰り返さないよう、初期版では保存時の自動解析を行いません。再解析は明示的にコマンドを実行します。

VSIXを再生成する場合:

```bash
cd editors/vscode
pnpm package
```

生成先は`dist/go-intent-analyzer.vsix`です。ローカルインストール:

```bash
code --install-extension dist/go-intent-analyzer.vsix
```

現在、型・呼び出し・interface解析は同一パッケージ内です。クロスパッケージ解析は今後の拡張対象です。
