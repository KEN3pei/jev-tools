# Language analyzers

Each directory primarily owns question content for one programming language:

- question wording and choice criteria
- evidence keywords used by the shared extractor

Shared question construction, Jev transport, state construction, answer validation, and report mapping stay in the parent packages. Add a language-specific mapper or extractor only when its behavior actually differs.

Question wording may intentionally be duplicated between languages instead of forcing a shared cross-language taxonomy.
