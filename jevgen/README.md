# jevgen

`jevgen`は、JEV Question Setを契約定義としてGoのquestion定義と型付きanswerを生成するコマンドです。

## Commands

```bash
go run ./cmd/jevgen check --input testdata/questions.json

go run ./cmd/jevgen check \
  --input ../answer-abstraction-evaluator/contract/questions.json \
  --output ../answer-abstraction-evaluator/contract/questions_gen.go \
  --package contract

go run ./cmd/jevgen generate \
  --input testdata/questions.json \
  --output /tmp/jevschema/questions_gen.go \
  --package jevschema

go run ./cmd/jevgen diff \
  --old testdata/questions.json \
  --new path/to/new-questions.json
```

`diff`は変更を次のように分類します。

- `additive`: questionの追加
- `compatible`: 型と値の集合に影響しない変更
- `semantic`: instructionsやcriteriaの説明など、再評価を検討すべき変更
- `breaking`: question削除、type変更、choice値の追加・削除

## Generated contract

生成ファイルには以下が含まれます。

- `QuestionSetName`、`SchemaVersion`、`SchemaHash`
- `Question`と`Questions()`
- `ChoiceAnswer[T]`、`ScoreAnswer`、`NoulAnswer`
- choiceごとの型と定数
- question IDに対応した`Answers`構造体

生成ファイルはコミットし、CIで再生成後の差分を検査する運用を想定しています。
