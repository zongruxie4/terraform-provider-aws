# AGENTS Guide

This repository is the Go-based Terraform AWS Provider. Prefer the repo's own Makefile, docs, and local package patterns over generic Go advice.

## Authoritative Sources

- `GNUmakefile`
- `docs/makefile-cheat-sheet.md`, `docs/continuous-integration.md`, `docs/development-environment.md`
- `docs/running-and-writing-acceptance-tests.md`, `docs/unit-tests.md`
- `docs/naming.md`, `docs/error-handling.md`, `docs/data-handling-and-conversion.md`, `docs/retries-and-waiters.md`
- `docs/add-a-new-resource.md`, `docs/skaff.md`, `docs/ai-usage.md`
- `names/caps.md`, `.ci/.golangci*.yml`, `.ci/providerlint/README.md`

## Repo Rule Files

- No `.cursor/rules/`, `.cursorrules`, or `.github/copilot-instructions.md` were found.
- Treat this file plus the docs above as the agent instructions for the repo.

## Environment

- Primary language: Go; module path: `github.com/hashicorp/terraform-provider-aws`.
- Required Go version: `1.25.8`; acceptance tests require Terraform CLI `0.12.26+`.
- Use `make prereq-go` or any target that depends on it to install the repo Go toolchain.
- Run `make tools` once to install linters and helper binaries.
- Use `make skaff` when adding a new resource, data source, or function.

## Common Commands

- Build provider: `make build`; CI-style build: `make go-build`; install dev tools: `make tools`.
- Format and imports: `make fmt`, `make fmt-check`, `make fix-imports`.
- Unit tests: `make test`; one package: `make test TEST=./internal/create`; one unit test: `make test TEST=./internal/create TESTARGS='-run TestUniqueId$'`.
- Service-scoped unit tests: `make test PKG=apprunner`; one service test: `make test PKG=apprunner TESTARGS='-run TestExpand.*'`; compile-only: `make test-compile TEST=./internal/service/apprunner`.
- Acceptance tests: `make testacc PKG=cloudwatch TESTS=TestAccCloudWatchDashboard_`; one test: `make testacc PKG=cloudwatch TESTS=TestAccCloudWatchDashboard_updateName`; short runs: `make testacc PKG=ecs TESTS='TestAccECSTaskDefinition_' TESTARGS=-short`; alias: `make t PKG=iam T=TestAccIAMRole_basic`.
- Acceptance test linting: `make testacc-lint PKG=cloudwatch`; `make testacc-tflint PKG=cloudwatch`.
- Linting: `make import-lint PKG=cloudwatch`; `make provider-lint PKG=cloudwatch`; `make golangci-lint PKG=cloudwatch`; faster shard: `make golangci-lint1 PKG=cloudwatch`.
- Generation and dependency checks: `make gen`; `make gen-check`; `make deps-check`.
- Autofix and CI sweeps: `make quick-fix PKG=cloudwatch`; `make ci-quick`; `make ci`.

## Test Selection Notes

- `PKG=<service>` and `K=<service>` are interchangeable service scoping variables.
- `TESTS=<regex>` and `T=<regex>` are the preferred ways to select acceptance tests.
- `TESTARGS` passes raw `go test` flags and is the easiest way to run a single unit test.
- `make test` excludes acceptance tests by default with a `-run` pattern; it is for unit tests only.
- `make testacc` sets `TF_ACC=1` for you.
- Acceptance tests need AWS credentials in the environment, usually `AWS_PROFILE` or `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY`, plus `AWS_DEFAULT_REGION`.
- The documented default acceptance-test region is `us-west-2`.
- Cross-account tests may need `AWS_ALTERNATE_PROFILE` or `AWS_ALTERNATE_ACCESS_KEY_ID`.
- Cross-region tests may need `AWS_ALTERNATE_REGION` and `AWS_THIRD_REGION`.
- Acceptance tests create real AWS resources and can cost money; keep runs tightly scoped.
- On macOS, `make test` uses temporary `GOCACHE` and `GOTMPDIR` automatically; do not fight that behavior.

## Workflow Rules

- Prefer the smallest correct change and follow the local style of the package you touch.
- Scope commands to the affected package or service before running broad repo-wide targets.
- Prefer explicit targets over legacy aliases. `make lint` exists but is marked legacy; use `golangci-lint`, `provider-lint`, and `import-lint` directly.
- If you add or change annotations like `@FrameworkResource` or `@SDKResource`, run `make gen`.
- If you edit docs or website pages, run the relevant doc or website format and lint targets, not just Go tooling.
- If you add a new resource, data source, or function, scaffold it with `skaff` instead of copying an older implementation.

## Framework and SDK Choice

- Net-new resources should use Terraform Plugin Framework; existing SDKv2 resources usually stay SDKv2 unless the task is explicitly a migration.
- Framework resources self-register with `@FrameworkResource("aws_service_name", name="Name")`; SDKv2 resources use `@SDKResource("aws_service_name", name="Name")`.
- Prefer Framework strong typing and provider helpers for new code; do not copy old resources verbatim. Use `skaff`, then adapt generated code.

## Formatting and Imports

- `gofmt -s` is the authoritative formatter; `make fmt` and `make fmt-check` wrap it.
- `goimports` is the authoritative import organizer; `make fix-imports` wraps it.
- Import order is enforced as standard library, third-party, then local imports.
- The repo uses `impi --local . --scheme stdThirdPartyLocal` for import linting.
- Keep code ASCII unless the file already requires non-ASCII content.
- Preserve the standard file header on source and doc files: `// Copyright IBM Corp. 2014, 2026` and `// SPDX-License-Identifier: MPL-2.0`.
- Struct tag order is linted. Keep tags ordered as `json`, `tfsdk`, `autoflex`.
- `nolint` comments must name a specific linter and explain why the suppression is needed.
- For Terraform snippets and acceptance-test HCL, use `terrafmt` or `terraform fmt`, not manual alignment.

## Import Alias Conventions

- Required aliases: SDK `helper/id` as `sdkid`, SDK `helper/retry` as `sdkretry`, plugin-testing `helper/acctest` as `sdkacctest`, and `internal/types` as `inttypes`.
- `internal/retry` should not be aliased. Preserve existing local aliases like `tftags` or service aliases when they improve clarity.

## Naming

- Service packages live under `internal/service/<serviceidentifier>`; identifiers are lowercase, have no underscores, and prefer the shorter AWS SDK/CLI name when they differ.
- Go filenames are `snake_case`; data source files end with `_data_source.go`; test files end with `_test.go`; docs are `<service>_<name>.html.markdown` under `website/docs/r/` or `website/docs/d/`.
- Main constructors are `Resource<ResourceName>()` and `DataSource<ResourceName>()`; CRUD helpers follow `resource<ResourceName>Create`, `Read`, `Update`, `Delete`.
- Do not include the service name in function names unless needed locally. Use Go MixedCaps for identifiers and `snake_case` for Terraform schema names.
- Preserve AWS-preferred capitalization and initialisms like `ARN`, `IAM`, `VPC`, `ID`, `API`, `APIGateway`, `CloudWatch`, `DynamoDB`, `FSx`, and `URL`. Never introduce `Id`.

## Data Handling and Schema Rules

- For net-new Framework resources, prefer typed `types.*` model fields and `internal/framework/flex` AutoFlex before writing custom conversion code.
- In SDKv2 resources, work with root attributes and blocks through `d.Get`, `d.GetOk`, `d.Set`, and `d.SetId`; avoid deep piecemeal writes when setting the parent block once is clearer.
- If AWS supplies a server-side default, prefer `Optional: true` plus `Computed: true` instead of a provider default. Avoid provider defaults unless required for correct behavior.
- Use AWS SDK constants instead of duplicating strings. AutoFlex ignores `Tags` by default, so keep tagging logic separate unless you intentionally opt in.

## Error Handling and Retries

- Name the error variable `err`, prefer early returns, and wrap returned errors with context using `fmt.Errorf("...: %w", err)`.
- Use `errors.As` to inspect Smithy and AWS error types; prefer `tfawserr.ErrCodeEquals` and `tfawserr.ErrMessageContains` for AWS API matching.
- Use shared retry and waiter helpers instead of custom polling. For net-new code, prefer `internal/retry` over deprecated `tfresource` aliases and reuse existing timeout constants, especially IAM propagation timeouts.
- Framework resources should add diagnostics with helpers like `create.ProblemStandardMessage`; SDKv2 resources should use helpers like `create.AppendDiagError`.
- In SDKv2 `Read` paths, only clear state with `d.SetId("")` when the resource is not new; guard with `!d.IsNewResource()`.

## Testing Conventions

- Unit tests start with `Test`; acceptance tests start with `TestAcc`; serialized acceptance helpers start with `testAcc`.
- Acceptance test names should follow `TestAcc<Service><Resource>_<case>` with concise suffixes like `basic`, `tags`, or `disappears`. Unit tests usually should not use underscores unless there is a strong reason.
- Unit tests should usually call `t.Parallel()`, and subtests usually should too. Acceptance tests do not need `t.Parallel()`; the harness handles concurrency.
- Follow existing service-test patterns: `acctest.Context(t)`, `acctest.PreCheck`, `acctest.ErrorCheck`, `acctest.ProtoV5ProviderFactories`, destroy checks, and import-state verification.
- New resources should at minimum have basic, disappears, and argument behavior coverage. Avoid hardcoded AMI IDs, regions, partitions, DNS suffixes, and TypeSet hashes; prefer ARN and region helpers.
- Place unit tests before acceptance tests when both live in the same file.

## Lint-Driven Style

- Staticcheck initialism rules mirror `names/caps.md`; check that file when naming is contentious.
- `paralleltest` and `tparallel` are enforced for unit tests.
- Magic numbers are linted; extract constants when a value has meaning and is reused.
- New code should not add avoidable lint debt.
- Generated files are treated strictly by lint tooling; do not hand-edit generated outputs unless the generator expects it.

## AI-Specific Rules

- No Cursor rule files or Copilot instruction files were found in this repository.
- Follow `docs/ai-usage.md`: disclose AI use in the PR description, and include `🤖🤖🤖` in the PR title if an LLM agent is directly involved in submitting it.
- Human reviewers and the human PR author own the final code and must understand it fully.
- Task-specific AI guides live under `docs/ai-agent-guides/`; consult them for resource identity, list resources, and similar focused tasks.

## Practical Pre-PR Sequence

- Run `make quick-fix PKG=<service>` if you changed a service package.
- Run the narrowest relevant unit tests.
- Run the narrowest relevant acceptance tests if AWS access and cost permit.
- Run `make gen` if annotations or generators were affected.
- Run `make golangci-lint PKG=<service>`, `make import-lint PKG=<service>`, and `make provider-lint PKG=<service>` before proposing a larger change.
- Escalate to `make ci-quick` only when the smaller checks are clean or when the change is broad.
