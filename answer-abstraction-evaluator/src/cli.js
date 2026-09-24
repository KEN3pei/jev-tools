#!/usr/bin/env node

import { readFile } from "node:fs/promises";
import process from "node:process";
import { evaluateAnswer } from "./evaluator.js";

function usage() {
  return `Usage:
  npm run evaluate -- --input <input.json>
  cat input.json | npm run evaluate

Input JSON:
  {
    "precedingContext": ["optional context"],
    "userRequest": "question to evaluate against",
    "candidateAnswer": "candidate answer"
  }`;
}

async function readStdin() {
  const chunks = [];
  for await (const chunk of process.stdin) chunks.push(chunk);
  return Buffer.concat(chunks).toString("utf8");
}

function inputPath(args) {
  const index = args.indexOf("--input");
  if (index === -1) return undefined;
  if (!args[index + 1]) throw new Error("--input requires a file path");
  return args[index + 1];
}

async function main() {
  if (process.argv.includes("--help") || process.argv.includes("-h")) {
    console.log(usage());
    return;
  }

  const path = inputPath(process.argv.slice(2));
  if (!path && process.stdin.isTTY) {
    throw new Error(`No input provided.\n\n${usage()}`);
  }

  const text = path ? await readFile(path, "utf8") : await readStdin();
  const input = JSON.parse(text);
  const result = await evaluateAnswer(input);
  console.log(JSON.stringify(result, null, 2));
}

main().catch((error) => {
  console.error(error instanceof Error ? error.message : String(error));
  process.exitCode = 1;
});

