# jev-tools

Jevを利用した小さな評価・ルーティングツールをまとめるリポジトリです。

各ツールはルート直下の独立したディレクトリに配置します。

## Tools

- [`answer-abstraction-evaluator`](./answer-abstraction-evaluator/README.md) — 質問が求める抽象度と、回答が説明を開始する抽象度の不一致を評価します。

JEV Question SetからGo契約を生成する`jevgen`は、独立した公開リポジトリへ移動しました。

- [`KEN3pei/jevgen`](https://github.com/KEN3pei/jevgen)

## Repository layout

```text
jev-tools/
├── README.md
└── answer-abstraction-evaluator/
    ├── README.md
    ├── go.mod
    ├── cmd/
    └── evaluator/
```
