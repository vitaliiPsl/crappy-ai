# Skills

Skills are reusable markdown workflows the assistant can load when they match a task.

Use skills for prompts you repeat often, such as reviews, explanations, release notes, or debugging checklists.

## Layout

Crappy discovers skills from:

```text
<skills_path>/<name>/SKILL.md
```

`skills_path` defaults to `~/.crappy-ai/skills` and can be changed in settings.

## Skill File

A skill is a `SKILL.md` file with optional YAML frontmatter and markdown instructions.

```md
---
name: review
description: Review code changes for correctness issues.
---

# Review

Prioritize bugs, regressions, and missing tests.
Return findings first with file and line references.
Avoid style-only comments unless they hide a real risk.
```

If `name` is omitted, Crappy uses the skill directory name.

## Usage

You can invoke a skill by typing its slash command:

```text
/review
/review the auth changes
```

Crappy also tells the model which skills are available. When a request matches a skill, the model should call `use_skill` before working. Startup context includes only skill names and descriptions; the full `SKILL.md` content is loaded into the conversation only when `use_skill` is called.

## Limits

The first version is intentionally small:

- no project-local skill directories
- no supporting files or scripts
- no tool permission overrides
- `SKILL.md` files over 64 KiB are skipped
