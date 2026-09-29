# jevgen

`jevgen`は、JEV Question Setを契約定義としてGoのquestion定義と型付きanswerを生成するGo package兼コマンドです。検証規則はGo packageに実装されており、外部のJSON Schemaや同梱データファイルには依存しません。

## Installation

```bash
go install github.com/KEN3pei/jev-tools/jevgen/cmd/jevgen@latest
```

Go packageとして利用する場合:

```bash
go get github.com/KEN3pei/jev-tools/jevgen@latest
```

```go
set, err := jevgen.LoadQuestionSet("questions.json")
if err != nil {
    return err
}

source, err := jevgen.GenerateGo(set, "jevschema")
```

公開APIには`QuestionSet`、`Question`、`Config`、`LoadQuestionSet`、`GenerateGo`、`Compare`が含まれます。

## Configuration

デフォルトではカレントディレクトリの`jevgen.json`を読みます。

```json
{
  "version": "1",
  "input": "contract/questions.json",
  "output": "contract/questions_gen.go",
  "package": "contract"
}
```

設定ファイル内の相対パスは、設定ファイルが置かれたディレクトリを基準に解決されます。`jevgen.json`が存在しない場合は以下の組み込みデフォルトを使います。

- input: `questions.json`
- output: `questions_gen.go`
- package: `jevschema`

## Commands

```bash
jevgen generate

jevgen check

jevgen generate --config tools/jevgen.json

jevgen diff \
  --old path/to/old-questions.json \
  --new path/to/new-questions.json
```

`generate`と`check`では`--input`、`--output`、`--package`で設定値を上書きできます。CLIで指定した相対パスはカレントディレクトリを基準に解決されます。

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
