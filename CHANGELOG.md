# Changelog

All notable changes to this project will be documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.13.0](https://github.com/saero-ai/xcaffold/compare/v0.12.0...v0.13.0) (2026-06-12)


### Features

* **renderer:** add paths and metadata support for Cursor skills ([c201ae7](https://github.com/saero-ai/xcaffold/commit/c201ae71f1f2156297f4163b1db2f2a4d3f65245))


### Bug Fixes

* **renderer:** emit disable-model-invocation in Cursor skill output ([6d18d0c](https://github.com/saero-ai/xcaffold/commit/6d18d0cdaba1a5a59fb523339992f17361500ab6))
* **renderer:** emit disable-model-invocation in Cursor skill output ([2d9affb](https://github.com/saero-ai/xcaffold/commit/2d9affb3a88878319f24729535759074414f7083)), closes [#118](https://github.com/saero-ai/xcaffold/issues/118)

## [0.12.0](https://github.com/saero-ai/xcaffold/compare/v0.11.0...v0.12.0) (2026-06-11)


### Features

* **cli:** add --json output flag to xcaffold list ([272be85](https://github.com/saero-ai/xcaffold/commit/272be859696189fccc85a1c62d24b77ba05cb381))

## [0.11.0](https://github.com/saero-ai/xcaffold/compare/v0.10.0...v0.11.0) (2026-06-10)


### Features

* **cli:** support default targets in global scope ([2d26e17](https://github.com/saero-ai/xcaffold/commit/2d26e1794ea87cef4a703276e6e1b50a8687e6de)), closes [#108](https://github.com/saero-ai/xcaffold/issues/108)
* **compiler:** user-configurable tier mapping overrides ([2cd0984](https://github.com/saero-ai/xcaffold/commit/2cd0984fbc8defd3adc245370940667b4179c1b5)), closes [#106](https://github.com/saero-ai/xcaffold/issues/106)
* **importer:** import antigravity2 knowledge files as memory ([006d61c](https://github.com/saero-ai/xcaffold/commit/006d61ce66e947229451e2e9bbda4105c28f4fa5))
* **renderer:** add alias resolution and active-provider selection ([3874ea1](https://github.com/saero-ai/xcaffold/commit/3874ea1d8f579d2eceba8529ae85cafeabbf4480))


### Bug Fixes

* **cli:** collect canonical provider names in multi-provider import ([5bbb386](https://github.com/saero-ai/xcaffold/commit/5bbb38625f59101d81b75814eaf3e6e75df23555))
* **cli:** fall back to base toolkit agent when no provider variant ([df7b6ff](https://github.com/saero-ai/xcaffold/commit/df7b6ffcf0316e186fcafef00ada8cc76e1e3f7f))
* **cli:** normalize provider aliases for apply and import targets ([9e372b8](https://github.com/saero-ai/xcaffold/commit/9e372b82de3025f9fcc54a6e2834287dd82e7967))
* **cli:** prefer active providers when import detects a shared directory ([8689092](https://github.com/saero-ai/xcaffold/commit/868909287881ea4b13fee541c3d6a05e35b68212))
* **importer:** remove redundant RootMCPPath from antigravity2 ([2a9f80d](https://github.com/saero-ai/xcaffold/commit/2a9f80df8654a0a321e4860080a79733be93043f))
* **parser:** wire target into variable loading ([df801df](https://github.com/saero-ai/xcaffold/commit/df801df902bd0959b2eeec9372ae79b5da94ccdb)), closes [#107](https://github.com/saero-ai/xcaffold/issues/107)
* **providers:** remove internal reference from Status field comment ([20c20bd](https://github.com/saero-ai/xcaffold/commit/20c20bd125715f56be405e4af621bedf51ac4466))

## [0.10.0](https://github.com/saero-ai/xcaffold/compare/v0.9.4...v0.10.0) (2026-06-09)


### Features

* **cursor:** refresh tier mappings and add model docs ([db366aa](https://github.com/saero-ai/xcaffold/commit/db366aafd4cb74520a24d1b60733c28f2daa8ec6))
* **cursor:** support native model slugs and tier mapping ([171217d](https://github.com/saero-ai/xcaffold/commit/171217d9a372beced8c7f3bd3ff8d33223b9f10f))
* **cursor:** support native model slugs and tier mapping ([c277d33](https://github.com/saero-ai/xcaffold/commit/c277d33f210607433e9fa328772d8409583d369b)), closes [#98](https://github.com/saero-ai/xcaffold/issues/98)
* **providers:** refresh model tier mappings across all providers ([27160c2](https://github.com/saero-ai/xcaffold/commit/27160c2cc8d6727c16fff4dc76a7ba648cc86c8a))


### Bug Fixes

* **cli:** include non-xcaf source files in status drift scan ([899928d](https://github.com/saero-ai/xcaffold/commit/899928da755d8d1e2a9c909ca142f551b3460f7c))
* **cli:** include non-xcaf source files in status drift scan ([e4d5455](https://github.com/saero-ai/xcaffold/commit/e4d545508b958017681fa985e54325d8d9ff4f24)), closes [#99](https://github.com/saero-ai/xcaffold/issues/99)

## [0.9.4](https://github.com/saero-ai/xcaffold/compare/v0.9.3...v0.9.4) (2026-06-06)


### Bug Fixes

* **compiler:** restore context rendering in blueprint-filtered applies ([936a196](https://github.com/saero-ai/xcaffold/commit/936a1965d7b2336d061298bbda5082ef245f0dac))
* **compiler:** restore context rendering in blueprint-filtered applies ([b24ed8b](https://github.com/saero-ai/xcaffold/commit/b24ed8bbc815dc95a8a6f1f02f4950df16fb3a02))
* **compiler:** restore context rendering in blueprint-filtered applies ([c8a9e92](https://github.com/saero-ai/xcaffold/commit/c8a9e924fdc7c605f35c4b485886b16fa77cfcee)), closes [#94](https://github.com/saero-ai/xcaffold/issues/94)

## [0.9.3](https://github.com/saero-ai/xcaffold/compare/v0.9.2...v0.9.3) (2026-05-26)


### Bug Fixes

* **import:** use outputRoot for split file write paths ([ede5338](https://github.com/saero-ai/xcaffold/commit/ede53381909b7acb62e013179da6255abaee1784))
* **import:** write split files to global dir instead of CWD ([8a7d65d](https://github.com/saero-ai/xcaffold/commit/8a7d65d72dc4c65e769629c8e231a9285a6b8c1e))
* **renderer:** exclude contexts with default: false from bare apply ([0b9811d](https://github.com/saero-ai/xcaffold/commit/0b9811d6b7b760c099c93f69a0f70200bef963ae))
* **renderer:** exclude default:false contexts from bare apply ([cb0d310](https://github.com/saero-ai/xcaffold/commit/cb0d310565adb74dbaee0c30698c3dab0780dd49))
* **renderer:** exclude default:false contexts from bare apply ([ab969e2](https://github.com/saero-ai/xcaffold/commit/ab969e26ff389cab3c14ff3f460dd79f782ba5aa))

## [0.9.2](https://github.com/saero-ai/xcaffold/compare/v0.9.1...v0.9.2) (2026-05-24)


### Bug Fixes

* **cli:** per-output-dir state isolation and cross-scope guard ([f37dbc6](https://github.com/saero-ai/xcaffold/commit/f37dbc677a107e777c3d0aad23bcf3bef7bed359))
* **cli:** prevent cross-scope cleanup across different output dirs ([29e3def](https://github.com/saero-ai/xcaffold/commit/29e3def6d85d3fb44ecb1b0b451d761d1e2a0920))
* **cli:** prevent cross-scope cleanup of different output-dir artifacts ([05fa170](https://github.com/saero-ai/xcaffold/commit/05fa170bb6e8effc71d67ce20cd7a9fb958836fe))

## [0.9.1](https://github.com/saero-ai/xcaffold/compare/v0.9.0...v0.9.1) (2026-05-23)


### Bug Fixes

* **cli:** correct global docs, help text, and registry file extension ([99162dc](https://github.com/saero-ai/xcaffold/commit/99162dc10d23e20bac5024576b7978e191456bd4))
* **cli:** correct global docs, help text, and registry file extension ([af06068](https://github.com/saero-ai/xcaffold/commit/af06068a4aa3a6af8a5460f7105d7f5f6541c2ab))
* **cli:** global docs, help text, and registry extension ([2bd65e9](https://github.com/saero-ai/xcaffold/commit/2bd65e96922266567262548da01638cdb1d6fa98))

## [0.9.0](https://github.com/saero-ai/xcaffold/compare/v0.8.0...v0.9.0) (2026-05-22)


### Features

* **cli:** add registry subcommands for project management ([1330a3f](https://github.com/saero-ai/xcaffold/commit/1330a3f5f932d0b6e1d497410207336088f004b3))
* **cli:** add registry subcommands for project management ([3821001](https://github.com/saero-ai/xcaffold/commit/38210011f89a3310f9be9a3e283141ab648d8250))
* **cli:** unblock --global, implement init bootstrap, wire global state ([92bc6b9](https://github.com/saero-ai/xcaffold/commit/92bc6b9d241823b146188e35447b0f72e7842727))
* **compiler:** complete global scope in parser, compiler, and AST ([337ef5b](https://github.com/saero-ai/xcaffold/commit/337ef5b58738a1f4cb9e35fe86b0c088e74db492))
* complete global scope support ([5be7f31](https://github.com/saero-ai/xcaffold/commit/5be7f3172914b4ffbdf907ca45501ea86651979d))
* registry CLI and global scope completion ([0400a42](https://github.com/saero-ai/xcaffold/commit/0400a42aa71b9d4dae352ab2f2d09bd5457c678a))
* **registry:** expand GlobalScanResult with policies and contexts ([1dd16b3](https://github.com/saero-ai/xcaffold/commit/1dd16b3498b085f4fdf707fcebaa71b444d1c949))
* **renderer:** add SupportsGlobalScope to all providers ([1daa6ca](https://github.com/saero-ai/xcaffold/commit/1daa6ca41dbd69ac63d5af9a0965d6e437b636e5))

## [0.8.0](https://github.com/saero-ai/xcaffold/compare/v0.7.2...v0.8.0) (2026-05-20)


### Features

* **cli:** add --output-dir flag and path resolution for apply command ([5d45594](https://github.com/saero-ai/xcaffold/commit/5d45594d5b7a512e3615964ddd902c004b56bb4d))
* **cli:** add --output-dir flag for alternate output directory ([8ed3c26](https://github.com/saero-ai/xcaffold/commit/8ed3c26e836bf1a2f102831ada75c56c0caad42f))
* **cli:** add --output-dir flag to status command ([c209d06](https://github.com/saero-ai/xcaffold/commit/c209d06e3725dc7edaa5b95c3d8510bd8aad9ec9))
* **cli:** wire --output-dir flag through apply flow ([d7403b3](https://github.com/saero-ai/xcaffold/commit/d7403b3f0d3bb7fcd61eff362bc4cf7aef84410d))
* **state:** add OutputDir field to TargetState ([782646f](https://github.com/saero-ai/xcaffold/commit/782646fec6e93e4075886580fed8f0f35450691c))


### Bug Fixes

* **cli:** use stored output-dir for drift detection in apply safeguard ([9e0d3d3](https://github.com/saero-ai/xcaffold/commit/9e0d3d36b05b52451a8efeec8f15bf372a367238))

## [0.7.2](https://github.com/saero-ai/xcaffold/compare/v0.7.1...v0.7.2) (2026-05-20)


### Bug Fixes

* **parser:** add blueprint name inference and mismatch warnings ([0039915](https://github.com/saero-ai/xcaffold/commit/003991587e0c9dff8c9f1752e5a4f981aedbfa59))
* **parser:** handle nested flat-file resource name inference ([63b4f73](https://github.com/saero-ai/xcaffold/commit/63b4f73b15c2af4b1ff33dd6b7783e82d854418b))
* **parser:** handle nested flat-file resource name inference ([1733ba2](https://github.com/saero-ai/xcaffold/commit/1733ba25ac1001926ab45b4740b11babbe42f14a))
* **parser:** handle nested flat-file resource name inference correctly ([bff12d9](https://github.com/saero-ai/xcaffold/commit/bff12d9fc4e9b226a4175b347421c23a33f5ad77))
* **parser:** remove rule carve-out and fix skill directory validator ([101bbe4](https://github.com/saero-ai/xcaffold/commit/101bbe4ccaa36978358cd87238bd6dbc0bc71106))

## [0.7.1](https://github.com/saero-ai/xcaffold/compare/v0.7.0...v0.7.1) (2026-05-19)


### Bug Fixes

* **cli:** include file path in xcaffold validate error messages ([53378c2](https://github.com/saero-ai/xcaffold/commit/53378c23accd5cd920687f7163caafddfcc22ac8))
* **parser:** use strings.TrimPrefix to satisfy gosimple lint ([b394362](https://github.com/saero-ai/xcaffold/commit/b394362fae418bfbc6dbf59ed88b10c8faef8235))
* **validate:** loop configured targets and omit empty policy line ([3a35677](https://github.com/saero-ai/xcaffold/commit/3a35677d403ea6387f1f6e69edf3160095112833))
* **validate:** loop configured targets and omit empty policy line ([1873288](https://github.com/saero-ai/xcaffold/commit/1873288bf44353f6777921bfb5962518fd640cc6)), closes [#56](https://github.com/saero-ai/xcaffold/issues/56)

## [0.7.0](https://github.com/saero-ai/xcaffold/compare/v0.6.1...v0.7.0) (2026-05-19)


### Features

* **cli:** auto-detect blueprint state in xcaffold status ([f6b60c1](https://github.com/saero-ai/xcaffold/commit/f6b60c14c0b3d52535af6726db933eb61aae17e5))
* **cli:** auto-detect blueprint state in xcaffold status ([caa3d68](https://github.com/saero-ai/xcaffold/commit/caa3d68f32e188f9d2533ae3c651ab3c04f04ed9)), closes [#45](https://github.com/saero-ai/xcaffold/issues/45)
* **parser:** add content-based monorepo boundary detection ([fcc2933](https://github.com/saero-ai/xcaffold/commit/fcc2933fc7852b4e6213cf9511c08e7bfd6d6af7))
* **parser:** add content-based monorepo boundary detection ([b354aff](https://github.com/saero-ai/xcaffold/commit/b354aff8f7520474aac0b18b0ef2b9a839f266c0))

## [0.6.1](https://github.com/saero-ai/xcaffold/compare/v0.6.0...v0.6.1) (2026-05-18)


### Bug Fixes

* **cli:** resolve actual installed version from build info ([37c69f5](https://github.com/saero-ai/xcaffold/commit/37c69f5bb06d61fabf962d2d14eea24bebae3571))
* **cli:** resolve version from build info instead of hardcoded string ([345b8f2](https://github.com/saero-ai/xcaffold/commit/345b8f28a432b88026b24f67e5b62c2a815b81ef))

## [0.6.0](https://github.com/saero-ai/xcaffold/compare/v0.5.0...v0.6.0) (2026-05-18)


### Features

* **compiler:** relax blueprint transitive dep resolution ([26d8a5d](https://github.com/saero-ai/xcaffold/commit/26d8a5d5c052e05c19f19f4844d05484b6826e13))
* **compiler:** relax blueprint transitive dep resolution ([0b430b7](https://github.com/saero-ai/xcaffold/commit/0b430b7f3c95cc43d609bd3ef3eaafe8ca0dd5a6)), closes [#36](https://github.com/saero-ai/xcaffold/issues/36)

## [0.5.0](https://github.com/saero-ai/xcaffold/compare/v0.4.2...v0.5.0) (2026-05-17)


### Features

* **import:** add layout detection and override routing ([faa46d0](https://github.com/saero-ai/xcaffold/commit/faa46d007b4e315b0253012567fe5b5508cd21cc))
* **import:** add multi-provider conflict detection ([3472441](https://github.com/saero-ai/xcaffold/commit/3472441bc6afe2be381ecd704a1891d1269c6834))


### Bug Fixes

* **cli:** multi-target smart-skip and remove backward-compat code ([b022baa](https://github.com/saero-ai/xcaffold/commit/b022baa6de213689237bc53e51c4a5fcab210b6b))
* **cli:** refresh state after incremental import and clarify messages ([b432664](https://github.com/saero-ai/xcaffold/commit/b4326642f2e3bbab1a6beb6de9a9481052214dbe))
* **cli:** replace "mcp" literals with kindMCP constant ([97e4996](https://github.com/saero-ai/xcaffold/commit/97e4996767b5732d18266dddb305584e031acebb))
* **cli:** resolve multi-target smart-skip false positive ([897eb2e](https://github.com/saero-ai/xcaffold/commit/897eb2eac114f2def1e38223abb4d88221e68141))
* **cli:** rewrite changed resources in place during incremental import ([9372dc6](https://github.com/saero-ai/xcaffold/commit/9372dc68bc4e439dc79b9bfc7dabc83e8d97415b))
* **cli:** thread scanned config through incremental import merge ([d92e22f](https://github.com/saero-ai/xcaffold/commit/d92e22f93437b41cad9dbc0f2dc30e3e32a41f73))
* **importer:** normalize provider field formats for round-trip fidelity ([6d24607](https://github.com/saero-ai/xcaffold/commit/6d246077330ee40b28d2bb929e626e8d57026e03))
* **parser:** populate SourceFile on parsed resources ([bda5dd8](https://github.com/saero-ai/xcaffold/commit/bda5dd83b20b66a03daf90d9799c91319e085ace))

## [0.4.2](https://github.com/saero-ai/xcaffold/compare/v0.4.1...v0.4.2) (2026-05-15)


### Bug Fixes

* **ci:** use PAT for release-please and add manual release trigger ([002a36f](https://github.com/saero-ai/xcaffold/commit/002a36ffda7f033cd3b69e57148f30ad57ec7552))
* **ci:** use RELEASE_PAT for release-please workflow ([dbec6b6](https://github.com/saero-ai/xcaffold/commit/dbec6b6de0a559a631eed5a372208b8b71765bd9))
* **schema:** remove leftover api-version from translator output ([ecb2203](https://github.com/saero-ai/xcaffold/commit/ecb2203c1ea1564ad2d0d7b33e8e936195fd65ef))

## [0.4.1](https://github.com/saero-ai/xcaffold/compare/v0.4.0...v0.4.1) (2026-05-15)


### Bug Fixes

* **ci:** trigger release on GitHub release events ([53675ef](https://github.com/saero-ai/xcaffold/commit/53675ef87228b00e5b58f8f7f98ff224b3cba5f9))
* **importer:** body-priority base selection and import pipeline fixes ([72eec68](https://github.com/saero-ai/xcaffold/commit/72eec685984aeb679f2ffca6ccdd47c782afd742))

## [0.4.0](https://github.com/saero-ai/xcaffold/compare/v0.3.0...v0.4.0) (2026-05-15)


### Features

* **cli:** show conditional status for provider-required optional fields ([e002eaf](https://github.com/saero-ai/xcaffold/commit/e002eaf36eaf81c33709c56ccedcfe179c6b3d49))
* Codex provider, schema enforcement, and blueprint improvements ([0d4e9e5](https://github.com/saero-ai/xcaffold/commit/0d4e9e5d5ac769c5af0fbbf4daf205abf612ff20))
* **parser:** reject agents without description field ([6336c10](https://github.com/saero-ai/xcaffold/commit/6336c10aa98c082b062aaed967f21ae9b1d0736f))
* **schema:** make agent description required and improve help display ([cb8998e](https://github.com/saero-ai/xcaffold/commit/cb8998e22505d21b617a73d3816dc5729c33c5db))
* **schema:** make agent description required at xcaffold level ([57194ca](https://github.com/saero-ai/xcaffold/commit/57194cab9b8c9539fa91fca2686c01711fc0e771))


### Bug Fixes

* **blueprint:** increase max extends depth to 10 ([a284dd0](https://github.com/saero-ai/xcaffold/commit/a284dd07a150eba5a726079f26f54dff0bd4b917))
* **blueprint:** resolve 5 implementation gaps ([dd9dc85](https://github.com/saero-ai/xcaffold/commit/dd9dc850d09a69754ae6ca9a2e29177c16a11d76))
* **blueprint:** use ClearableList for resource selectors ([12576fa](https://github.com/saero-ai/xcaffold/commit/12576fa76082e58bbb1d673961624a478d2cf39a))
* **cli:** unhide blueprint discovery flags ([7261f59](https://github.com/saero-ai/xcaffold/commit/7261f59062089f3336f98d06aacc7bdb55c74a91))
* **renderer:** resolve settings and hooks dynamically ([eac3280](https://github.com/saero-ai/xcaffold/commit/eac3280f61de38ff636b5f99f1aaa0c1bc5bea7a))
* **schema:** address code review findings for agent description ([f1462e8](https://github.com/saero-ai/xcaffold/commit/f1462e83fa4f042d9a50e1b0a01f8c877dc70486))

## [Unreleased]

### Breaking Changes

* **schema:** `description` is now required on `kind: agent` resources — existing `.xcaf` files that omit `description` will fail validation. (parser, schema)

### Features

* **cli:** `xcaffold help --xcaf` now shows `optional*` for fields that are optional at xcaffold level but required by specific providers, with a note explaining which providers require them. (cli)

## [0.3.0](https://github.com/saero-ai/xcaffold/compare/v0.2.0...v0.3.0) (2026-05-14)


### Features

* add Codex provider, schema cleanup, and doc improvements ([4e7c11a](https://github.com/saero-ai/xcaffold/commit/4e7c11ae905af4d893ce39db51f09272f0fb300b))
* **codex:** add Codex provider ([0256987](https://github.com/saero-ai/xcaffold/commit/0256987c3a37619e014f603712ff8ea78012a1ba))
* **codex:** add Codex provider core ([1137631](https://github.com/saero-ai/xcaffold/commit/1137631bb81ba92c4c3742fc087bdbb39678eb25))
* **codex:** add renderer unit tests ([21250cc](https://github.com/saero-ai/xcaffold/commit/21250cc5f7d2a4cde591a3807691c37ac6011593))


### Bug Fixes

* **ci:** stabilize lint config and clean up unused version vars ([6dcb5d8](https://github.com/saero-ai/xcaffold/commit/6dcb5d8279f38eead00e19d4aed529df1513d935))
* **ci:** use default govet analyzers and tune lint thresholds ([46876cd](https://github.com/saero-ai/xcaffold/commit/46876cdf0f55463c5726816593f1d22b8dd00c14))
* **cli:** simplify version output to match industry standard ([d2a6c5e](https://github.com/saero-ai/xcaffold/commit/d2a6c5ee0e91fb138ad8c0c8be11afcc3f29c08c))
* **schema:** remove unnecessary --- delimiters from pure YAML kinds ([fd0f384](https://github.com/saero-ai/xcaffold/commit/fd0f3843a30c948492fd870b60918c5e8355d764))

## [0.2.0](https://github.com/saero-ai/xcaffold/releases/tag/v0.2.0) (2026-05-14)

### Breaking Changes

- `.xcf` → `.xcaf` file extension — all resource files must be renamed; the parser no longer accepts `.xcf`. (parser)
- `kind: config` removed — use `kind: project` with individual resource documents. Files with an empty or missing `kind:` field now produce a descriptive error with migration guidance. (parser)
- `tools:` renamed to `allowed-tools:` under `kind: skill` — `AgentConfig.tools` is unchanged; the rename applies only to skills. (ast, renderer)
- `xcaffold apply` no longer defaults to `--target claude` when no target is configured — set `targets:` in `project.xcaf` or pass `--target`. (cli)
- `xcaffold init --yes` requires `--target` when no known provider CLI is detected on `$PATH`. (cli)
- `graph --format json` uses snake_case field names (`config_path`, `disk_entries`, `blocked_tools`) — breaks existing JSON consumers. (graph)

### Added

**Providers**

- **Codex provider (Preview)** — compile `.xcaf` manifests to OpenAI Codex output (`.codex/`). Supports agents (TOML), skills (shared `.agents/skills/`), hooks (JSON), MCP (TOML), and project instructions (`AGENTS.md`). Rules and memory unsupported with fidelity notes. ([providers/codex/](providers/codex/))

**Resource kinds**

- `kind: global` — resource kind for `~/.xcaffold/global.xcaf`; holds shared resources and settings without project metadata. (ast, parser)
- `kind: policy` — declarative constraint engine with `require` and `deny` rules evaluated during `apply` and `validate`. Four built-in policies ship with the binary: `path-safety`, `settings-schema`, `agent-has-description`, `no-empty-skills`. Projects reference policies via a `policies:` list in `kind: project`. Create a same-name `kind: policy` file with `severity: off` to disable a built-in. (ast, compiler, policy)
- `kind: context` — shared prompt context blocks composable into agents and blueprints; defined in `xcaf/contexts/`. (ast, parser, renderer)

**Schema features**

- `targets` field on `kind: blueprint` — blueprints can declare independent compilation targets. (ast)
- `ClearableList` type for list fields — setting a list field to `[]` explicitly clears inherited values; absent continues to inherit, empty clears, populated replaces. (ast)
- Two-layer field classification with `+xcaf:role=` markers on all config struct fields (`identity`, `rendering`, `composition`, `metadata`, `filtering`). (ast)
- `disable-model-invocation` (`*bool`) and `user-invocable` (`*bool`) on `AgentConfig`. (ast)
- `provider` pass-through map (`map[string]any`) on `TargetOverride` for provider-native fields. (ast)
- `whenToUse`, `license`, `disableModelInvocation` (`*bool`), `userInvocable` (`*bool`), `argumentHint` on `SkillConfig`. (ast)
- `targets` (`map[string]TargetOverride`) on `SkillConfig` for per-provider overrides and provider pass-through. (ast)
- `AllowedEnvVars` on `ProjectConfig` — security filtering for env var injection via `${env.NAME}`. (ast)
- `task` and `max_turns` fields on `TestConfig` (schema `project.test`). (ast)

**Variable resolution system**

- `--var-file` flag on `apply`, `validate`, and related commands. (cli)
- Variable expansion in `.xcaf` files: `${var.name}` for project variables, `${env.NAME}` for environment variables. (parser, compiler)
- Variable stack loading from `project.xcaf`, `vars.xcaf`, and `--var-file` sources. (compiler)

**CLI commands and flags**

- `xcaffold status` command — sync and drift metrics across all applied targets with inline file status reporting, replacing `xcaffold diff`. (cli)
- `xcaffold list` — adaptive 3-column output displaying all managed projects with path, targets, resource counts, and last-applied timestamp. (cli)
- `xcaffold graph` — deep hierarchical topology visualization; segments global components, renders blocked and allowed tools, separates inherited skills from rules. (cli)
- `xcaffold graph --project <name>` — queries any registered project's topology from any location. (cli)
- `xcaffold graph --all` — combined global and registered projects view. (cli)
- `xcaffold help <kind>` — shows per-provider field annotations for a resource kind and generates annotated templates. (cli)
- `--target <provider>` flag on `xcaffold validate` — compile-time field validation per provider, with provider name in the header and a field validation summary in the footer. (cli)
- `--blueprint` flag on `xcaffold validate`. (cli)
- `--json` flag on `xcaffold init` — machine-readable manifest output. (cli)
- `--target` string-slice flag on `xcaffold init` — multi-select provider targeting. (cli)
- `--force` and `--backup` flags on `xcaffold apply` — drift circumvention and timestamped backup. (cli)
- `--check` flag on `xcaffold apply` — fail-fast schema validation without writing artifacts. (cli)
- `--global / -g` boolean flag replaces `--scope global|project|all` across all commands. (cli)
- `--target` flag on `apply` and `import` for isolating platform outputs. (cli)
- `xcaffold apply --dry-run` — preview and orphan detection without writing. (cli)
- Idempotent `xcaffold init` — re-running updates rather than overwrites. (cli)
- Incremental `xcaffold import` — imports only new or changed resources. (importer)
- Apply preview — `xcaffold apply` shows a diff preview before writing. (cli)
- `xcaffold import --source` — semantic cross-provider translation during import. (importer)

**Compiler and optimizer**

- Multi-target compilation support. (compiler)
- `TargetRenderer` registry — pluggable compiler architecture; all provider dispatch goes through `resolveRenderer()` + `renderer.Orchestrate()`. (compiler, renderer)
- Smart compilation skipping via multi-file source hashing. (compiler)
- Deterministic orphan purge with `--dry-run` preview. (compiler)
- Walk-up configuration search from project subdirectories, bounded by `$HOME`. (compiler)
- Lockfile standardization with per-target naming (`scaffold.claude.lock`, `scaffold.cursor.lock`). (compiler)
- Skill artifact auto-discovery by compiler. (compiler)
- `xcaffold apply` runs optimizer passes after compilation and before policy evaluation. (compiler)
- Security invariant policies: output path confinement, settings schema, hook URL validation. (policy)

**Renderer — provider-agnostic surface**

- `CapabilitySet` type declaring per-resource support for each renderer. (renderer)
- `Orchestrate()` function dispatching compilation to per-resource methods based on `CapabilitySet`. (renderer)
- `TargetRenderer` per-resource methods: `CompileAgents`, `CompileSkills`, `CompileRules`, `CompileWorkflows`, `CompileHooks`, `CompileSettings`, `CompileMCP`, `CompileProjectInstructions`, plus `Capabilities()` and `Finalize()`. (renderer)
- Cross-provider invariant test suite: render-or-note, no raw aliases, no Claude env var leakage, reference fidelity, code catalog completeness. (renderer)
- `provider_features_test.go` — ground truth assertions for all five providers' capability sets, target names, and output directories. (renderer)
- Shared renderer helpers: `CompileSkillSubdir`, `SortedKeys`, `YAMLScalar`, `StripAllFrontmatter`. (renderer)
- `LowerWorkflows` in `renderer/shared/` to avoid import cycles. (renderer)
- `FidelityNote` struct with FidelityLevel (`info` / `warning` / `error`) and `NewNote()` constructor. (renderer)
- Stable fidelity code catalog in `fidelity_codes.go` — 16 codes including SKILL_SCRIPTS_DROPPED, SKILL_ASSETS_DROPPED, SETTINGS_FIELD_UNSUPPORTED, `AGENT_MODEL_UNMAPPED`, `AGENT_SECURITY_FIELDS_DROPPED`, HOOK_INTERPOLATION_REQUIRES_ENV_SYNTAX. (renderer)
- `AllCodes()` enumeration for tooling introspection. (renderer)
- `cmd/xcaffold/fidelity.go` with `printFidelityNotes()` and `buildSuppressedResourcesMap()` for command-layer suppression. (cmd)
- Antigravity renderer — agents rendered as specialist notes. (renderer)
- Gemini CLI renderer (`--target gemini`) — instructions to `GEMINI.md`, rules to `.gemini/rules/`, skills to `.gemini/skills/`, agents to `.gemini/agents/`, hooks and MCP to `.gemini/settings.json`. (renderer)
- `ProviderManifest` registry — replaces hardcoded provider switches. (renderer)
- gen-schema tooling for `+xcaf:` marker extraction and schema registry. (ast)
- Override parsing expanded to 9 resource kinds. (parser)

**Importer**

- `ProviderImporter` interface with per-provider implementations for claude, cursor, gemini, copilot, and antigravity. (importer)
- `ProviderExtras` catchall for genuinely unclassified provider-specific artifacts. (ast, importer)
- `SourceProvider` annotation on all AST resource types for import provenance tracking. (ast)
- `ReclassifyExtras` — auto-graduates `ProviderExtras` entries when an importer recognizes them. (parser)
- `KindHookScript` and canonical hook-file routing across claude, cursor, copilot, and gemini. (importer)
- Shared importer helpers: `ParseFrontmatter`, `ParseFrontmatterLenient`, `MatchGlob`, `AppendUnique`. (importer)
- `import --global` scans all provider directories and merges all discovered resources. (importer)

**Schema and golden files**

- Golden manifest reference files in `schema/golden/` for every resource kind. (schema)
- CI test validating all golden manifests parse without error. (schema)
- Per-kind reference guides (`agent-reference.md`, `skill-reference.md`, etc.) generated inside the xcaffold skill during `xcaffold init`. (init)

**Other**

- `xcaffold init` generates a self-referential `/xcaffold` skill (`xcaf/skills/xcaffold/skill.xcaf`) teaching AI assistants local schema constraints. (init)
- `xcaffold init` multi-file generator scaffolds a full `xcaf/` directory, replacing the legacy single-file builder. (init)
- `instructions-file:` directive on agents, skills, and rules for sourcing prompts from external markdown files. (ast)
- `references:` directive on skills for copying supplementary context files (glob patterns). (ast)
- Provider override list merge with tri-state `cleared` signal — `cleared: true` empties an inherited list field. (ast)
- Claude provider pass-through for skills — keys under `targets.claude.provider:` emitted into SKILL.md frontmatter. (renderer)
- File-origin error reporting for duplicate resource IDs across multiple `.xcaf` files. (parser)
- Walk-up `EnsureGlobalHome()` migrates or initializes `~/.xcaffold/` automatically on first run. (cli)
- Project auto-registration into global registry on `init`, `import`, and `apply`. (cli)
- `xcaffold apply --project <name>` resolves project paths from the global registry. (cli)
- `hooks` and `workflows` included in `xcaffold graph` topology output. (graph)
- `review project.xcaf` displays skills, rules, hooks, MCP servers, and workflows in addition to agents. (review)
- `knownTools` validation extended with Task, Computer, AskUserQuestion, Agent, ExitPlanMode, EnterPlanMode. (parser)
- GoReleaser — pre-built binaries for Linux (amd64/arm64), macOS (amd64/arm64), Windows (amd64) with Homebrew tap. (release)
- `AGENTS.md` following the [agents.txt](https://agentstext.com) convention. (docs)
- `llms.txt` AI discovery index at repository root. (docs)
- `docs/concepts/architecture/overview.md` — system architecture documentation with Mermaid diagrams. (docs)
- Shared `internal/auth` package eliminating `AuthMode` type duplication. (internal)
- `make install` target with `LDFLAGS` injection for version propagation. (build)

### Changed

- `TargetRenderer` interface: monolithic `Compile()`/`Render()` replaced by per-resource methods with `Capabilities()` and `Finalize()`. (renderer)
- `compiler.Compile()` signature: `(*Output, []FidelityNote, error)` — second return carries fidelity notes. (compiler)
- `compiler.Compile()` uses `resolveRenderer()` + `renderer.Orchestrate()` instead of a direct target switch. (compiler)
- `compiler.OutputDir()` returns empty string for unknown targets instead of `.claude`. (compiler)
- `suppress-fidelity-warnings` enforcement moved from individual renderers to the command layer; renderers emit notes unconditionally. (cmd, renderer)
- `xcaffold apply`, `xcaffold export`, and `xcaffold validate` receive and print fidelity notes via the shared helper. (cmd)
- Cursor renderer: 12 stderr writes replaced with typed fidelity notes. (renderer)
- Antigravity renderer: 4 stderr writes replaced with typed fidelity notes. (renderer)
- `AgentConfig` struct fields reordered to canonical grouping: identity, model and execution, tool access, permissions and invocation, lifecycle, memory and context, composition references, inline composition, targets, instructions last. (ast)
- `SkillConfig` struct fields reordered to canonical six-group layout: identity, tool access, permissions and invocation, composition files, targets, instructions last. (ast)
- Claude renderer emits new agent frontmatter fields: `disable-model-invocation`, `user-invocable`, `memory` (after `isolation`). (renderer)
- Claude renderer emits skill frontmatter fields: `when_to_use`, `license`, `allowed-tools`, `disable-model-invocation`, `user-invocable`, `argument-hint`. (renderer)
- Attribute resolver regex broadened to accept kebab-case field names (e.g. `${skill.tdd.allowed-tools}`). (resolver)
- `fields.yaml` entries reclassified from `xcaffold-only` to `unsupported`. (renderer)
- Parser name/kind mismatch warnings collected in `XcaffoldConfig.ParseWarnings` instead of printing to stderr. (parser)
- `xcaffold apply` output: header breadcrumb, glyph helpers, file count summary, import hint footer. (cli)
- `xcaffold apply` lists each drifted file with path and status before aborting. (cli)
- `xcaffold graph` dependency rendering overhauled — rules grouped by folder prefix, agent memory nested dynamically. (graph)
- Import pipeline unified on `ProviderImporter` interface — multi-directory import now uses `ProviderImporter.Import()` per directory; memory, MCP, settings, hooks, and project instructions no longer dropped in multi-dir mode. (importer)
- `isConfigFile()` renamed to `isParseableFile()` — now rejects empty and `config` kind values. (parser)
- `WriteSplitFiles()` emits separate files with frontmatter for body-bearing kinds. (compiler)
- `~/.xcaffold/global.xcaf` uses `kind: global` instead of `kind: config`. (ast)
- Project manifest relocated from `./project.xcaf` to `.xcaffold/project.xcaf`. (compiler, init, importer)
- `--scope global|project|all` replaced with `--global / -g` boolean flag. (cli)
- `xcaffold test` rewrites compilation to send the compiled system prompt directly to the LLM API via `internal/llmclient`; trace records declared tool calls from the response. `test.task` in `project.xcaf` sets the task prompt. (test)
- `xcaffold test --claude-path` renamed to `--cli-path` for provider-agnostic binary resolution. (cli)
- Memory rendering transitioned to convention-based `.md` files in `xcaf/agents/<id>/memory/` — discovered by the compiler at compile time. (compiler, renderer)
- Lockfile format standardized with per-target naming; V1 lock files upgraded automatically. (compiler)
- `validate --target` includes provider name in header and appends a field validation summary. (cli)
- README rewritten with badge row, "Why xcaffold?" section, Homebrew install target, expanded schema documentation, and multi-platform output tables. (docs)
- Diátaxis `index.md` files standardized with unified cross-navigation sections. (docs)

### Fixed

- `tagResourcesWithProvider` skipping MCP, hooks, and settings during multi-provider import; all 7 resource kinds now receive provider-scoped `targets` entries. (importer)
- `xcaffold apply --backup` skipping backup for 2nd and subsequent targets in multi-target projects. (cli)
- `xcaffold status --all` silently ignored without `--target`; now appends per-provider grouped file listing in overview mode. (cli)
- `xcaffold status` exits with code 1 on drift detection, enabling scriptable CI checks. (cli)
- Copilot renderer path-doubling: `OutputDir()` returns `.github`, all emitted paths are relative. (renderer)
- Global-scope memory file leakage during `xcaffold import` — orphaned files not owned by declared project agents are now pruned. (importer)
- Model alias resolution for gemini, copilot, and cursor — raw aliases like `sonnet-4` now map to provider-specific identifiers. (renderer)
- Antigravity renderer silently dropping agents without emitting a `RENDERER_KIND_UNSUPPORTED` fidelity note. (renderer)
- Copilot `InstructionsFile` rendering and model resolution. (renderer)
- Copilot MCP config layout — correctly emits `.vscode/mcp.json`. (renderer)
- `graph` hardcoded `.claude` fallback replaced with `compiler.OutputDir()`. (graph)
- `graph` excluding inherited global resources from project-scope topology output. (graph)
- `diff` surfacing `FindXCAFFiles` errors instead of reporting false-positive `SRC DELETED`. (diff)
- `apply` excluding `registry.xcaf` from source file tracking. (apply)
- Memory import path-safe slugification and compounding `project_` prefixes during recursive import. (importer)
- Project root derivation with nested `.xcaffold/` namespace. (compiler)
- `analyze` no longer errors when no `project.xcaf` is present. (analyze)
- `export --output` flag correctly sets the destination path. (export)
- `init --global` with a local `project.xcaf` present. (init)
- `apply --check` returns non-zero exit code on validation errors. (apply)
- `apply --check-permissions --global` reads the global config directory. (apply)
- `init` generating stale `version: "1.0"` templates and incorrect `agents:` indentation. (init)
- Schema versions and YAML structure in README examples. (docs)
- Unmapped `model` declarations failing string resolution in `settings.json` renderer. (renderer)
- Compiler silently discarding `skills`, `rules`, `hooks`, and `mcp` blocks. (compiler)
- `statusLine` and `enabledPlugins` strict typing in settings renderer. (renderer)
- `trace.Recorder` data race — added `sync.Mutex` for concurrent HTTP handler writes. (internal)
- SSRF in `internal/proxy` — replaced `strings.HasSuffix` host check with strict equality. (internal)
- `os.Exit(1)` in `diff.go` and `validate.go` replaced with `return fmt.Errorf(...)`. (cli)

### Removed

- `xcaffold plan` command — use `apply --dry-run`. (cli)
- `xcaffold diff` command — replaced by `xcaffold status`. (cli)
- `xcaffold translate` command — translation via `import --source` and cross-provider `apply`. (cli)
- `xcaffold migrate` command — had no consumers. (cli)
- `--target agentsmd` compilation target — AGENTS.md is generated by the cursor and copilot renderers. (renderer)
- `--scope all` compilation mode. (cli)
- `kind: memory` from parser — memory entries are now plain `.md` files discovered by the compiler. (parser)
- `MemoryConfig.Instructions`, `MemoryConfig.InstructionsFile`, `MemoryConfig.Inherited` fields. (ast)
- `MemorySeed.Lifecycle` field and seed-once lifecycle with `--reseed` flag. (state, cli)
- `resolveMemoryBody`, `renderMemoryMarkdown`, `CompileWithPriorSeeds`, `WithReseed` from Claude renderer. (renderer)
- `CodeMemorySeedSkipped` and `CodeMemoryBodyEmpty` fidelity codes. (renderer)
- `MemoryOptions.Reseed` and `MemoryOptions.PriorHashes` from renderer interface. (renderer)
- `memoryDoc` struct and `WriteSplitFiles` memory block. (renderer)
- `internal/mascot` package. (internal)
- `renderer.Register()`, `renderer.Get()`, `renderer.Registered()` dead-code functions. (renderer)
- `bir.Analyze()` unused function. (bir)
- `buildConfigFromDir` and 10 provider-specific extraction functions from `import.go`. (importer)
- `extractAgents`, `extractSkills`, `extractRules`, `extractWorkflows` legacy functions. (importer)
- Unreachable fallback branch in `importScope`. (importer)
- Duplicate `rendererForTarget` in `apply.go`. (compiler)
- Duplicate `detectAllGlobalPlatformDirs` / `detectAllPlatformDirs` merged into parameterized `detectPlatformDirs`. (cli)
- `wazero` WASM runtime and `golang.org/x/sys` transitive dependency. (internal)
- `--tokens` flag on `xcaffold graph`. (graph)

## [0.1.0] - 2026-04-02
### Added
- Complete rewrite of the CLI compiler replacing the deprecated TypeScript prototype with a robust Go binary.
- One-Way Compilation architecture targeting Anthropic Claude Code configurations natively.
- Automatic creation and formatting of `.claude/agents/*.md` and `.claude/settings.json`.
- `.xcaffold/project.xcaf.state` manifest generation tracking SHA-256 state blobs of output configurations.
- `xcaffold plan` command for static parsing and pre-deployment analysis.
- `xcaffold diff` command to enforce GitOps strictness and identify shadow configuration modifications (drift).
- Support for `tools`, `skills`, `blocked_tools`, `effort`, `model`, and `mcp` declarations within `project.xcaf`.

### Removed
- Support for multi-provider prompt polyfilling has been explicitly removed in V1 in favor of the strict native ecosystem.
- Support for Bi-Directional Compilation (Decompilation of `.claude/` files back to `.xcaf`).

### Security
- Replaced ambiguous degradation warnings with a fail-closed schema validator (`exit 1`) to ensure security rules are not bypassed during configuration generation.
