# Go Intent Analyzer for VS Code

This extension runs the `go-intent-analyzer` CLI and visualizes design intent inferred by Jev.

## Requirements

The VSIX contains the VS Code visualization code, but not the platform-specific Go CLI binary. The extension starts that CLI as a child process; the CLI analyzes Go source and Git history, calls Jev, and returns the Analysis Report.

Build the CLI from the repository root:

```bash
mkdir -p bin
go build -o bin/go-intent-analyzer ./cmd
```

Then configure its absolute path in VS Code `settings.json`:

```json
{
  "goIntentAnalyzer.executablePath": "/absolute/path/to/go-intent-analyzer/bin/go-intent-analyzer"
}
```

With an absolute path configured, the CLI does not need to be on `PATH`.

When installing from the source repository, the recommended setup is:

```bash
./install_setup.sh
```

The installer builds the CLI into `~/.local/bin/go-intent-analyzer`, packages and installs this extension, and securely prompts for the TypeSafe API key. The extension automatically detects that CLI location, so no `settings.json` change is needed in this mode.

## Commands

- `Go Intent Analyzer: Set TypeSafe API Key`
- `Go Intent Analyzer: Analyze Workspace`
- `Go Intent Analyzer: Clear Decorations`

Analysis runs only when explicitly requested. Code with insufficient evidence is not colored.

## Intent legend

Open the **Intent Legend** view in Explorer to see the color assigned to each of the 11 intent categories. Hover over an entry to see its description and hexadecimal color value.

The background of a code unit is a weighted blend of up to three visible intents. The left border and overview ruler use the strongest intent's color. Therefore, a blended background does not necessarily match one legend color exactly. Hover over colored code to inspect the intents and evidence used for that unit.
