# Answer Abstraction Evaluator

質問が求める抽象度と、回答が説明を開始する抽象度の不一致をJevで評価する、Go製のCLI兼パッケージです。

たとえば「認証方式にはどのような設計があるか」という設計レベルの質問に対して、回答がいきなり特定ライブラリのインストール手順から始まっていないかを検出します。

## What it evaluates

- 質問が主に求めている抽象度
- 抽象度と説明順序の評価を適用すべき依頼か
- 回答が説明を開始した抽象度
- 抽象度の不一致
- 全体像を示す前に具体論へ進んでいないか
- 段階的に詳細化できているか
- ユーザーが学ぼうとしている概念を既知として扱っていないか
- 質問の抽象度が曖昧で、確認質問が必要か

抽象度は次の選択肢で評価します。

```text
conceptual_orientation
architecture_and_design_space
implementation_mechanics
operations_and_governance
unclear
```

## Requirements

- Go 1.22以上
- TypeSafe APIキー

```bash
export TYPESAFE_API_KEY="..."
```

## CLI usage

ツールのディレクトリへ移動します。

```bash
cd ~/Apps/jev-tools/answer-abstraction-evaluator
```

入力JSONを用意します。

```json
{
  "precedingContext": [
    "直前まで具体的なSDE cascadeを説明していた",
    "ユーザーはAI routing全体を理解しようとしている"
  ],
  "userRequest": "AI routingにはどのような設計パターンがある？",
  "candidateAnswer": "まずLiteLLMをインストールして設定ファイルを作ります。"
}
```

`--input`でファイルを渡します。

```bash
go run ./cmd/answer-abstraction-evaluator --input ./examples/input.json
```

標準入力でも利用できます。

```bash
cat ./examples/input.json | go run ./cmd/answer-abstraction-evaluator
```

長い質問や回答をシェル引数へ直接入れる必要はありません。

## Output

```json
{
  "requestedLevel": "architecture_and_design_space",
  "requestedLevelConfidence": 0.74,
  "answerEntryLevel": "implementation_mechanics",
  "answerEntryLevelConfidence": 0.76,
  "scores": {
    "evaluationApplicable": 0.94,
    "abstractionMismatch": 0.55,
    "prematureSpecificity": 0.74,
    "progressiveDisclosure": 0.42,
    "prerequisiteFit": 0.38,
    "clarificationNeeded": 0.24
  },
  "decision": "revise_entry",
  "model": "jev-1.13.0"
}
```

`decision`は次のいずれかです。

| Decision | Meaning |
|---|---|
| `not_applicable` | 直接的なコマンド、コード、事実、変換、状況報告などで、この評価を適用しない |
| `pass` | 抽象度と説明順序に重大な問題がない |
| `revise_entry` | 内容を保ちつつ、冒頭と説明順序を修正する |
| `restructure` | 抽象度または前提知識の扱いに大きな不一致があり、構成を組み直す |
| `ask_clarifying_question` | 求められる抽象度が曖昧なため、先に確認する |

既定の閾値は`evaluator/evaluator.go`の`DefaultThresholds`にあります。実運用では、自分の評価データを使って調整してください。

## Library usage

`state`をソースコード内で毎回書き換える必要はありません。質問、回答、前後文脈を関数引数で渡すと、内部で固定されたJev用`state`へ変換します。

```go
client := evaluator.Client{APIKey: os.Getenv("TYPESAFE_API_KEY")}
result, err := client.Evaluate(context.Background(), evaluator.Input{
    PrecedingContext: []string{"認証に関する基本用語は説明済み"},
    UserRequest:     "認証方式にはどのような設計があり、どう選ぶべき？",
    CandidateAnswer: "まず特定ライブラリをインストールします。",
})
```

APIキー、モデル、エンドポイント、閾値、`fetch`実装は第2引数で変更できます。

```go
client := evaluator.Client{
    APIKey:  os.Getenv("TYPESAFE_API_KEY"),
    Model:   "jev-latest",
    BaseURL: "https://api.typesafe.ai/v1/systemone",
    Thresholds: evaluator.DefaultThresholds,
}
```

## Input contract

| Field | Required | Description |
|---|---:|---|
| `userRequest` | Yes | 評価対象の質問・依頼 |
| `candidateAnswer` | Yes | 評価する回答の全文 |
| `precedingContext` | No | 質問の解釈に必要な直前の会話や前提。文字列配列 |

`precedingContext`には必要な文脈だけを渡してください。正解として期待する抽象度や事後的に判明したユーザー意図を入れると、評価結果への情報漏洩になります。

## How the state is built

関数は引数から次の`state`を作ります。

```json
{
  "evaluation_task": "Infer the requested abstraction level and evaluate whether the answer begins and progresses at an appropriate level. Do not answer the user request.",
  "preceding_context": [],
  "user_request": "...",
  "candidate_answer": "..."
}
```

固定された`QUESTIONS`と、このリクエストごとの`state`をTypeSafe System One APIへ送信します。

## Testing

ユニットテストはTypeSafe APIを呼ばず、通信部分を差し替えて実行します。

```bash
go test ./...
```

## Codex Stop hook

`codex-hook`は、Codexの`UserPromptSubmit`フックで質問を一時保存し、`Stop`フックで最終回答をJevへ送ります。判定が`pass`以外なら、Codexへ具体的な修正理由を返して1回だけ再生成させます。

```bash
go build -o ./bin/codex-hook ./cmd/codex-hook
```

設定例は[`examples/hooks.json`](./examples/hooks.json)にあります。`/absolute/path/to/codex-hook`を、ビルドしたバイナリの絶対パスへ置き換えてください。

フックの一時状態は、既定ではOSのユーザー設定ディレクトリ以下に保存されます。`JEV_HOOK_DATA_DIR`を設定すると保存先を変更できます。

```bash
export JEV_HOOK_DATA_DIR="$HOME/.local/share/jev-tools/answer-abstraction-evaluator"
```

評価結果はCodexの`systemMessage`として毎回表示され、ファイルには保存されません。質問本文はStopフックへ引き渡すため、権限`0600`の状態ファイルに一時保存され、評価完了後に削除されます。

Jev APIが失敗した場合は回答をブロックしません。`stop_hook_active`が真の場合も追加の再生成を要求しないため、修正ループは最大1回です。

実際のモデル品質を確認するには、質問・回答・期待判定を含む評価ケースを別途蓄積してください。モデルや質問、閾値を変更した場合は、同じ評価セットで回帰評価することを推奨します。

## Operational notes

- APIキーをリポジトリへコミットしないでください。
- 評価対象の回答全文がTypeSafe APIへ送信されます。機密情報や個人情報の取り扱いを確認してください。
- Jevの確率は正解の証明ではありません。閾値は実際の人間評価との対応を測って調整してください。
- 自動修正へ組み込む場合は、無限ループを避けるため修正回数に上限を設けてください。
