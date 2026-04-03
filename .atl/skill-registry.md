# Skill Registry

**Orchestrator use only.** Read this registry once per session to resolve skill paths, then pass pre-resolved paths directly to each sub-agent's launch prompt. Sub-agents receive the path and load the skill directly — they do NOT read this registry.

## User Skills

| Trigger | Skill | Path |
|---------|-------|------|
| When creating a pull request, opening a PR, or preparing changes for review. | branch-pr | /home/meridian/.config/opencode/skills/branch-pr/SKILL.md |
| When writing Go tests, using teatest, or adding test coverage. | go-testing | /home/meridian/.config/opencode/skills/go-testing/SKILL.md |
| When creating a GitHub issue, reporting a bug, or requesting a feature. | issue-creation | /home/meridian/.config/opencode/skills/issue-creation/SKILL.md |
| When user says "judgment day", "judgment-day", "review adversarial", "dual review", "doble review", "juzgar", "que lo juzguen". | judgment-day | /home/meridian/.config/opencode/skills/judgment-day/SKILL.md |
| When user asks to create a new skill, add agent instructions, or document patterns for AI. | skill-creator | /home/meridian/.config/opencode/skills/skill-creator/SKILL.md |
| When the user is looking for functionality that might exist as an installable skill. | find-skills | /home/meridian/.agents/skills/find-skills/SKILL.md |
| Build 1-click launchers and launcher-enabled apps with Pinokio. | gepeto | /home/meridian/.agents/skills/gepeto/SKILL.md |
| Discover, launch, and use apps and tools for the current task. | pinokio | /home/meridian/.agents/skills/pinokio/SKILL.md |
| Writing new Go code, reviewing Go code, refactoring existing Go code, or designing Go packages/modules. | golang-patterns | /home/meridian/Documentos/Proyectos/Personal/dev-forge/celador/.agents/skills/golang-patterns/SKILL.md |
| Go, Golang, goroutines, channels, gRPC, microservices Go, Go generics, concurrent programming, Go interfaces. | golang-pro | /home/meridian/Documentos/Proyectos/Personal/dev-forge/celador/.agents/skills/golang-pro/SKILL.md |

## Project Conventions

| File | Path | Notes |
|------|------|-------|
| — | — | No project convention files detected in the repository root (`AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.cursorrules`, `copilot-instructions.md`). |

Read the convention files listed above for project-specific patterns and rules. All referenced paths have been extracted — no need to read index files to discover more.
