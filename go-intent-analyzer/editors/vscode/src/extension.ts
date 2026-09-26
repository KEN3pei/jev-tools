import { spawn, type ChildProcessWithoutNullStreams } from "node:child_process";
import { existsSync } from "node:fs";
import os from "node:os";
import path from "node:path";
import * as vscode from "vscode";
import { INTENT_VISUALS, type IntentVisual } from "./intents";
import { renderStyle, type RenderStyle } from "./render";
import type { AnalysisReport, Evidence, UnitReport } from "./types";

const secretKey = "goIntentAnalyzer.typesafeApiKey";
let report: AnalysisReport | undefined;
let running: ChildProcessWithoutNullStreams | undefined;
let decorationTypes: vscode.TextEditorDecorationType[] = [];

export function activate(context: vscode.ExtensionContext): void {
  const legend = new IntentLegendProvider();
  context.subscriptions.push(
    vscode.commands.registerCommand("goIntentAnalyzer.analyzeWorkspace", () => analyzeWorkspace(context)),
    vscode.commands.registerCommand("goIntentAnalyzer.clear", clearDecorations),
    vscode.commands.registerCommand("goIntentAnalyzer.setApiKey", () => setApiKey(context)),
    vscode.window.registerTreeDataProvider("goIntentAnalyzer.legend", legend),
    vscode.window.onDidChangeVisibleTextEditors(() => applyReport()),
    vscode.workspace.onDidChangeConfiguration((event) => {
      if (event.affectsConfiguration("goIntentAnalyzer")) applyReport();
    }),
    { dispose: () => running?.kill() },
    { dispose: clearDecorations }
  );
}

class IntentLegendProvider implements vscode.TreeDataProvider<IntentVisual> {
  getTreeItem(intent: IntentVisual): vscode.TreeItem {
    const item = new vscode.TreeItem(intent.label, vscode.TreeItemCollapsibleState.None);
    item.description = intent.id;
    item.tooltip = new vscode.MarkdownString(`**${intent.label}**\n\n${intent.description}\n\nColor: \`${intent.color}\``);
    item.iconPath = new vscode.ThemeIcon("circle-filled", new vscode.ThemeColor(`goIntentAnalyzer.intent.${intent.id}`));
    return item;
  }

  getChildren(): IntentVisual[] {
    return INTENT_VISUALS;
  }
}

export function deactivate(): void { running?.kill(); clearDecorations(); }

async function analyzeWorkspace(context: vscode.ExtensionContext): Promise<void> {
  const root = workspaceRoot();
  if (!root) { void vscode.window.showErrorMessage("Go Intent Analyzer: Open a workspace folder first."); return; }

  try {
    const nextReport = await vscode.window.withProgress({ location: vscode.ProgressLocation.Notification, title: "Go Intent Analyzer: analyzing workspace", cancellable: true }, async (progress, token) => {
      progress.report({ message: "Collecting Go evidence and asking Jev…" });
      return runAnalyzer(root, (await context.secrets.get(secretKey)) ?? process.env.TYPESAFE_API_KEY, token);
    });
    validateReport(nextReport);
    report = nextReport;
    applyReport();
    const supported = report.units.reduce((count, unit) => count + unit.intents.filter((intent) => intent.assessment !== "insufficient").length, 0);
    const suffix = report.errors?.length ? ` (${report.errors.length} analysis errors)` : "";
    void vscode.window.showInformationMessage(`Go Intent Analyzer: colored ${report.units.length} units with ${supported} inferred intents${suffix}.`);
  } catch (error) {
    if (String(error).includes("cancelled")) return;
    void vscode.window.showErrorMessage(`Go Intent Analyzer: ${error instanceof Error ? error.message : String(error)}`);
  }
}

async function setApiKey(context: vscode.ExtensionContext): Promise<boolean> {
  const value = await vscode.window.showInputBox({ title: "TypeSafe API key", prompt: "Stored in VS Code SecretStorage and passed only to go-intent-analyzer.", password: true, ignoreFocusOut: true });
  if (!value) return false;
  await context.secrets.store(secretKey, value.trim());
  void vscode.window.showInformationMessage("Go Intent Analyzer: TypeSafe API key saved.");
  return true;
}

function runAnalyzer(root: string, apiKey: string | undefined, token: vscode.CancellationToken): Promise<AnalysisReport> {
  const configuration = vscode.workspace.getConfiguration("goIntentAnalyzer");
  const configured = configuration.get<string>("executablePath", "").trim();
  const executable = resolveExecutable(configured, root);
  return new Promise((resolve, reject) => {
    const child = spawn(executable, ["-dir", root], { cwd: root, env: { ...process.env, ...(apiKey ? { TYPESAFE_API_KEY: apiKey } : {}) }, shell: false });
    running = child;
    let stdout = "";
    let stderr = "";
    const limit = 64 * 1024 * 1024;
    const cancellation = token.onCancellationRequested(() => child.kill());
    child.stdout.setEncoding("utf8"); child.stderr.setEncoding("utf8");
    child.stdout.on("data", (chunk: string) => { stdout += chunk; if (stdout.length > limit) child.kill(); });
    child.stderr.on("data", (chunk: string) => { stderr += chunk; });
    child.on("error", (error) => reject(new Error(`${error.message}. Configure goIntentAnalyzer.executablePath if the CLI is not on PATH.`)));
    child.on("close", (code) => {
      running = undefined; cancellation.dispose();
      if (token.isCancellationRequested) { reject(new Error("cancelled")); return; }
      if (stdout.length > limit) { reject(new Error("analyzer output exceeded 64 MiB")); return; }
      let parsed: AnalysisReport;
      try { parsed = JSON.parse(stdout) as AnalysisReport; } catch { reject(new Error(`could not parse analyzer JSON${stderr ? `: ${lastLine(stderr)}` : ""}`)); return; }
      if (code !== 0 && !parsed.units?.length) { reject(new Error(lastLine(stderr) || `analyzer exited with code ${code}`)); return; }
      resolve(parsed);
    });
  });
}

function applyReport(): void {
  clearDecorationTypes();
  if (!report) return;
  const configuration = vscode.workspace.getConfiguration("goIntentAnalyzer");
  const minimumFit = configuration.get<number>("minimumEvidenceFit", 0.35);
  const minimumConfidence = configuration.get<number>("minimumConfidence", 0.5);
  const maxVisible = configuration.get<number>("maxVisibleIntents", 3);

  for (const editor of vscode.window.visibleTextEditors) {
    const filename = normalized(editor.document.uri.fsPath);
    const units = report.units.filter((unit) => normalized(unit.uri) === filename);
    const groups = new Map<string, { style: RenderStyle; options: vscode.DecorationOptions[] }>();
    for (const unit of units) {
      const style = renderStyle(unit.intents, minimumFit, minimumConfidence, maxVisible);
      if (!style) continue;
      const key = `${style.background}|${style.border}`;
      const group = groups.get(key) ?? { style, options: [] };
      group.options.push({ range: toRange(unit), hoverMessage: hoverFor(unit, style) });
      groups.set(key, group);
    }
    for (const group of groups.values()) {
      const decoration = vscode.window.createTextEditorDecorationType({
        isWholeLine: true,
        backgroundColor: group.style.background,
        borderColor: group.style.border,
        borderStyle: "none none none solid",
        borderWidth: "0 0 0 3px",
        overviewRulerColor: group.style.border,
        overviewRulerLane: vscode.OverviewRulerLane.Right
      });
      decorationTypes.push(decoration);
      editor.setDecorations(decoration, group.options);
    }
  }
}

function hoverFor(unit: UnitReport, style: RenderStyle): vscode.MarkdownString {
  const markdown = new vscode.MarkdownString(undefined, true);
  markdown.isTrusted = false;
  markdown.appendMarkdown(`**${escapeMarkdown(unit.name)}** · ${unit.kind}\n\n`);
  for (const rendered of style.visible) {
    const intent = rendered.intent;
    markdown.appendMarkdown(`- **${escapeMarkdown(intent.intent)}** — ${intent.assessment}, fit ${format(intent.evidenceFit)}, confidence ${format(intent.confidence)}\n`);
    for (const evidence of intent.evidence.slice(0, 4)) markdown.appendMarkdown(`  - ${evidenceLink(evidence)}\n`);
  }
  markdown.appendMarkdown("\n_Colors are a weighted blend of the strongest visible intents._");
  return markdown;
}

function evidenceLink(evidence: Evidence): string {
  const label = escapeMarkdown(evidence.signal);
  if (!evidence.uri) return label;
  const uri = vscode.Uri.file(evidence.uri).with({ fragment: evidence.range ? `L${evidence.range.start.line + 1}` : "" });
  return `[${label}](${uri.toString()})`;
}

function toRange(unit: UnitReport): vscode.Range { return new vscode.Range(unit.range.start.line, unit.range.start.character, unit.range.end.line, unit.range.end.character); }
function resolveExecutable(configured: string, root: string): string {
  if (configured) {
    const expanded = configured === "~" || configured.startsWith(`~${path.sep}`) ? path.join(os.homedir(), configured.slice(2)) : configured;
    return expanded.includes(path.sep) && !path.isAbsolute(expanded) ? path.resolve(root, expanded) : expanded;
  }
  const installed = path.join(os.homedir(), ".local", "bin", "go-intent-analyzer");
  return existsSync(installed) ? installed : "go-intent-analyzer";
}
function validateReport(value: AnalysisReport): void {
  if (!value || value.schemaVersion !== "1.0" || !Array.isArray(value.units)) throw new Error("unsupported or malformed Analysis Report");
  for (const unit of value.units) {
    if (!unit.id || !unit.uri || !unit.range || !Array.isArray(unit.intents)) throw new Error(`malformed analysis unit: ${unit?.id ?? "unknown"}`);
  }
}
function workspaceRoot(): string | undefined { const active = vscode.window.activeTextEditor?.document.uri; return (active && vscode.workspace.getWorkspaceFolder(active)?.uri.fsPath) ?? vscode.workspace.workspaceFolders?.[0]?.uri.fsPath; }
function normalized(value: string): string { const resolved = path.resolve(value); return process.platform === "win32" ? resolved.toLowerCase() : resolved; }
function format(value: number | null): string { return value === null ? "n/a" : value.toFixed(2); }
function lastLine(value: string): string { return value.trim().split(/\r?\n/).at(-1) ?? ""; }
function escapeMarkdown(value: string): string { return value.replace(/[\\`*_{}[\]()#+\-.!]/g, "\\$&"); }
function clearDecorationTypes(): void { for (const decoration of decorationTypes) decoration.dispose(); decorationTypes = []; }
function clearDecorations(): void { report = undefined; clearDecorationTypes(); }
