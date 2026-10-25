# AGENTS.md

## Language Policy

Respond in the language the user writes in, per message.

The repository's documentation language is declared in the Project section of this file (e.g. `Documentation language: Russian`). If it is not declared, it is English.

* English repos: every tracked documentation file ends with `_en.md`, except the reserved names `AGENTS.md`, `CLAUDE.md` and `README.md`. A Russian counterpart, if produced, uses the same stem with `_ru.md` and is gitignored (`*_ru.md`); the English file remains the source of truth.
* Russian repos: documentation is written in Russian, without a language suffix.

The shared rules block, code comments, docstrings and commit messages are always in English. Translate non-English comments to English whenever you encounter them in code you are reading or editing.

## Agent Behavior

Judge proposals on their merits and say plainly when one is weak. When a developer proposes a feature, require justification before writing any spec. A valid justification lets you estimate how the feature fits the existing codebase, what it might break, and why it is needed. If the justification is thin, keep asking until it is sufficient. Refuse to proceed if the developer cannot provide one. If the developer explicitly asks for advice, give it. Chores (see Workflow) need no justification.

## Workflow

Classify each task before starting; if unsure which tier applies, ask.

Chore &mdash; typo and wording fixes, formatting, dependency or version bumps, renames, config value changes with no behavior change, regenerating generated files. Make the change, run existing tests and checks if code was touched, and show the diff for review. No justification, spec or plan.

Feature or behavior change &mdash; follow this sequence. Explicit user approval is required at each step before proceeding.

1. Feature proposal &mdash; developer describes what they want to build
2. Justification &mdash; ask about purpose; developer must justify before work begins
3. Specification &mdash; write the spec
4. Spec review &mdash; user approves spec before continuing
5. Plan &mdash; write implementation plan
6. Plan review &mdash; user approves plan before continuing
7. Tests &mdash; write tests first (test-driven development is the primary approach)
8. Code &mdash; implement to make tests pass
9. Full review &mdash; user and agent review the work together before committing

Documentation work (deliverable is Markdown or prose, no code written) follows the same sequence without steps 7 and 8.

## Documentation

Docs record facts. Explain only what a competent reader cannot infer: non-obvious decisions, constraints, rejected alternatives.

* Open with a 2–3 sentence preamble stating what the doc covers and who it is for; no closing summary.
* One fact, one place. Link to the source of truth (code, spec, another doc) instead of restating it.
* Facts a reader looks up (settings, commands, endpoints, paths) go in tables or lists; prose is for reasoning.
* Editing means rewriting to the current state: delete outdated text; don't append corrections, "Update:" notes or history.
* When adding text, look for text to remove. A doc that only grows is a defect.
* No filler: no hedging, no marketing adjectives, no "simply", "just", "easily", "it is important to note".

## Markdown Formatting

* Never hard-wrap prose: each paragraph, list item and table row is one line, however long. Renderers and editors wrap it.
* Separate blocks (headings, paragraphs, lists, code blocks) with one blank line; never more than one in a row.
* Keep lists tight: no blank lines between list items or sub-items (a blank line makes the list "loose" and renders extra spacing).
* Use a numbered list when order matters (steps, sequences, priorities), otherwise an unordered list with `*` markers.
* Write long dashes as `&mdash;`.
* Don't wrap headings in bold or italics; headings are already emphasized.

Code blocks and diagrams keep their formatting. Apply these rules to new and edited text; don't reformat untouched text.

## Code Comments

Comments state facts, not explanations. Record what is not derivable from the code itself: units, invariants, protocol constraints, why a non-obvious value was chosen, upstream bug references. Do not restate what the next line does, narrate control flow, or write tutorial prose.

Keep them short &mdash; a phrase or a sentence. If a block's comments outweigh its code, the comments are wrong: cut them, or make the code self-explanatory instead.

## Commit Messages

Format: `<subject>[(<environment>)]: <description>`. Square brackets mark the optional part; they are not written.

* `subject` &mdash; the component or area changed (`backend`, `helm`, `k8s`), or the kind of work when no single area fits (`chore`, `docs`).
* `environment` &mdash; the target environment for infrastructure and deployment changes (`prod`, `stage`); omit otherwise.
* `description` &mdash; short, lowercase, no trailing period.

Examples:

    backend: update openapi.yaml schema
    chore: bump go-lint version
    helm(prod): deployed cloudnative-pg chart
    k8s(stage): add gateway traefik-stage

## Project

