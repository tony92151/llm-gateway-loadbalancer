# Graph Report - llm-gateway-loadbalancer  (2026-06-06)

## Corpus Check
- 33 files · ~18,173 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 299 nodes · 350 edges · 28 communities (20 shown, 8 thin omitted)
- Extraction: 90% EXTRACTED · 10% INFERRED · 0% AMBIGUOUS · INFERRED: 36 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `4d0904a0`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- [[_COMMUNITY_Community 0|Community 0]]
- [[_COMMUNITY_Community 1|Community 1]]
- [[_COMMUNITY_Community 2|Community 2]]
- [[_COMMUNITY_Community 3|Community 3]]
- [[_COMMUNITY_Community 4|Community 4]]
- [[_COMMUNITY_Community 5|Community 5]]
- [[_COMMUNITY_Community 6|Community 6]]
- [[_COMMUNITY_Community 7|Community 7]]
- [[_COMMUNITY_Community 8|Community 8]]
- [[_COMMUNITY_Community 9|Community 9]]
- [[_COMMUNITY_Community 10|Community 10]]
- [[_COMMUNITY_Community 11|Community 11]]
- [[_COMMUNITY_Community 12|Community 12]]
- [[_COMMUNITY_Community 13|Community 13]]
- [[_COMMUNITY_Community 14|Community 14]]
- [[_COMMUNITY_Community 15|Community 15]]
- [[_COMMUNITY_Community 16|Community 16]]
- [[_COMMUNITY_Community 17|Community 17]]
- [[_COMMUNITY_Community 18|Community 18]]
- [[_COMMUNITY_Community 19|Community 19]]
- [[_COMMUNITY_Community 20|Community 20]]
- [[_COMMUNITY_Community 21|Community 21]]
- [[_COMMUNITY_Community 22|Community 22]]
- [[_COMMUNITY_Community 25|Community 25]]
- [[_COMMUNITY_Community 26|Community 26]]
- [[_COMMUNITY_Community 27|Community 27]]

## God Nodes (most connected - your core abstractions)
1. `/graphify` - 14 edges
2. `What You Must Do When Invoked` - 14 edges
3. `Pool` - 13 edges
4. `New()` - 13 edges
5. `NewAdminHandler()` - 11 edges
6. `Part B - Semantic extraction (parallel subagents)` - 8 edges
7. `NewPool()` - 7 edges
8. `Handler` - 6 edges
9. `NewHandler()` - 6 edges
10. `For --update (incremental re-extraction)` - 6 edges

## Surprising Connections (you probably didn't know these)
- `TestLeastLoadSkipsCooldownAndChoosesLowestInFlight()` --calls--> `NewPool()`  [INFERRED]
  internal/selector/selector_test.go → internal/selector/selector.go
- `TestRoundRobinCyclesAvailableKeys()` --calls--> `NewPool()`  [INFERRED]
  internal/selector/selector_test.go → internal/selector/selector.go
- `TestSelectSkipsKeyAtRPMLimit()` --calls--> `NewPool()`  [INFERRED]
  internal/selector/selector_test.go → internal/selector/selector.go
- `TestSelectSkipsKeyAtTPMLimitAfterUsageRecorded()` --calls--> `NewPool()`  [INFERRED]
  internal/selector/selector_test.go → internal/selector/selector.go
- `NewHandlerWithError()` --calls--> `NewPool()`  [INFERRED]
  internal/proxy/handler.go → internal/selector/selector.go

## Communities (28 total, 8 thin omitted)

### Community 0 - "Community 0"
Cohesion: 0.06
Nodes (34): code:block1 (/graphify                                             # full), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash (if [ ! -f graphify-out/.graphify_extract.json ]; then), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c ") (+26 more)

### Community 1 - "Community 1"
Cohesion: 0.07
Nodes (30): code:bash (mkdir -p graphify-out), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash (# Detect the correct Python interpreter (handles pipx, venv,), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c ") (+22 more)

### Community 2 - "Community 2"
Cohesion: 0.14
Nodes (16): Config, Handler, copyHeaders(), copyStream(), errorString(), extractModel(), isEventStream(), isHopByHopHeader() (+8 more)

### Community 3 - "Community 3"
Cohesion: 0.11
Nodes (17): Check for context, code:block1 (┌─────────────────────────────────────────┐), code:bash (openspec list --json), code:block3 (User: I'm thinking about adding real-time collaboration), code:block4 (User: The auth system is a mess), code:block5 (User: /opsx:explore add-auth-system), code:block6 (User: Should we use Postgres or SQLite?), code:block7 (## What We Figured Out) (+9 more)

### Community 4 - "Community 4"
Cohesion: 0.13
Nodes (8): New(), TestLoadExpandsEnvironmentAndValidates(), TestLoadRejectsTokenBalanceInMVP(), Load(), Validate(), Key, Pool, max()

### Community 5 - "Community 5"
Cohesion: 0.13
Nodes (15): NewAdminHandler(), parseDashboardWindow(), parseWindow(), TestAdminDashboardReturnsMergedDashboard(), TestAdminKeysReturnsRuntimeState(), TestAdminRecentRequestsAndSummary(), TestMonitorRouteServesHTMLWhenEnabled(), writeError() (+7 more)

### Community 6 - "Community 6"
Cohesion: 0.13
Nodes (14): models, name, npm, options, Authorization, name, options, lightning-ai/minimax-m2.5 (+6 more)

### Community 7 - "Community 7"
Cohesion: 0.15
Nodes (13): code:block10 (You are a graphify extraction subagent. Read the files liste), code:bash ($(cat graphify-out/.graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:bash ($(cat .graphify_python) -c "), code:block8 (spawn_agent(agent_type="worker", message="Your task is to pe) (+5 more)

### Community 8 - "Community 8"
Cohesion: 0.17
Nodes (11): Config, CryptoConfig, DatabaseConfig, KeyConfig, LoggingConfig, ModelConfig, MonitorConfig, PricingConfig (+3 more)

### Community 9 - "Community 9"
Cohesion: 0.20
Nodes (5): adminProvider, App, BuildProxyConfig(), serve(), TestBuildProxyConfigMapsKeysAndPrices()

### Community 10 - "Community 10"
Cohesion: 0.44
Nodes (8): NewHandler(), response(), roundTripClient(), TestProxyInjectsSelectedKeyAndRecordsUsage(), TestProxyRetries429WithNextKey(), TestProxySkipsKeyAfterTPMUsageLimit(), TestProxyStreamsSSE(), roundTripFunc

### Community 11 - "Community 11"
Cohesion: 0.29
Nodes (6): TestCalculateCostUsesCachedInputPricing(), TestExtractUsageFromOpenAIResponse(), Pricing, Usage, CalculateCost(), ExtractUsage()

### Community 12 - "Community 12"
Cohesion: 0.28
Nodes (7): DB, Open(), RequestLog, TestAggregateHourlyUpsertsRequestsByHourKeyAndModel(), TestDashboardSinceAggregatesOverviewKeysAndRecentErrors(), TestMigrateAndInsertRequestLog(), Summary

### Community 13 - "Community 13"
Cohesion: 0.33
Nodes (5): code:bash (openspec status --change "<name>" --json), code:bash (openspec instructions apply --change "<name>" --json), code:block3 (## Implementing: <change-name> (schema: <schema-name>)), code:block4 (## Implementation Complete), code:block5 (## Implementation Paused)

### Community 14 - "Community 14"
Cohesion: 0.16
Nodes (11): Checker, NewChecker(), response(), TestCheckOnceMarksUnhealthyAndKeepsHealthyKeys(), Config, roundTripFunc, NewPool(), TestLeastLoadSkipsCooldownAndChoosesLowestInFlight() (+3 more)

### Community 15 - "Community 15"
Cohesion: 0.40
Nodes (4): code:bash (openspec new change "<name>"), code:bash (openspec status --change "<name>" --json), code:bash (openspec instructions <artifact-id> --change "<name>" --json), code:bash (openspec status --change "<name>")

### Community 16 - "Community 16"
Cohesion: 0.50
Nodes (3): code:bash (mkdir -p openspec/changes/archive), code:bash (mv openspec/changes/<name> openspec/changes/archive/YYYY-MM-), code:block3 (## Archive Complete)

### Community 27 - "Community 27"
Cohesion: 0.29
Nodes (5): DB, round4(), DashboardKeyStats, DashboardOverviewStats, DashboardStats

## Knowledge Gaps
- **113 isolated node(s):** `$schema`, `npm`, `name`, `baseURL`, `Authorization` (+108 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **8 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `New()` connect `Community 4` to `Community 2`, `Community 5`, `Community 9`, `Community 12`, `Community 14`?**
  _High betweenness centrality (0.128) - this node is a cross-community bridge._
- **Why does `NewHandlerWithError()` connect `Community 2` to `Community 10`, `Community 4`, `Community 14`?**
  _High betweenness centrality (0.088) - this node is a cross-community bridge._
- **Why does `NewAdminHandler()` connect `Community 5` to `Community 4`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `New()` (e.g. with `.selectRoundRobin()` and `.selectLeastLoad()`) actually correct?**
  _`New()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **Are the 6 inferred relationships involving `NewAdminHandler()` (e.g. with `New()` and `TestAdminKeysReturnsRuntimeState()`) actually correct?**
  _`NewAdminHandler()` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `$schema`, `npm`, `name` to the rest of the system?**
  _113 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Community 0` be split into smaller, more focused modules?**
  _Cohesion score 0.05714285714285714 - nodes in this community are weakly interconnected._