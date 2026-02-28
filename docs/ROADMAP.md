# Aegis AI ASM — MVP Roadmap

## Purpose
Deliver a minimum viable cyber defense assistant capable of monitoring attack surface metrics, enforcing guardrails, and producing actionable intelligence for the security team.

## Guiding Principles
- **Security-first defaults:** ship hardened configurations and zero-trust assumptions.
- **Observability everywhere:** every service must emit audit-grade telemetry.
- **Automate the boring:** repeatable workflows become automated runbooks.
- **Human-in-the-loop:** analysts get clear context, override hooks, and approval checkpoints.

## MVP Scope
1. **Asset Discovery Pipeline**
   - Integrate with cloud inventory + CMDB as data sources.
   - Normalize resources into a single asset graph (id, owner, exposure profile).

   **Asset Discovery Source Backlog (priority order)**
   1. **crt.sh** — Free CT feed; baseline subdomain visibility (in progress now).
   2. **Subfinder** — OSS aggregator pulling from dozens of free sources (no cost).
   3. **SecurityTrails API** — Paid tier; comprehensive DNS + historical WHOIS context.
   4. **Shodan InternetDB** — Free endpoint for quick exposure hints (ports/services).
   5. **Shodan Full API** — Paid; exhaustive host telemetry + banners for triage.
   6. **Project Sonar (Rapid7)** — Free; periodic DNS + SSL datasets for enrichment.

2. **Attack Surface Scoring Engine**
   - Define scoring dimensions (exposure, vulnerability, detectability, blast radius).
   - Provide REST + CLI access to latest scores per asset.
3. **Policy Guardrails**
   - YAML-based policy packs for high-risk misconfigurations.
   - On violation: create ticket + optional auto-remediation script stub.
4. **Security Telemetry Bus**
   - gRPC/HTTP event intake with signed payloads.
   - Stream into internal queue (NATS or Kafka) + persist for 30 days.
5. **Operator Console (CLI)**
   - Read-only views for assets, policies, alerts.
   - `aegisctl` command surfaces top risks and policy drift summary.

## Milestones & Owners
| Milestone | Description | Owner | Target |
|-----------|-------------|-------|--------|
| M1 | Repo bootstrap, Go module, CI lint/tests | Aegis-PO (you) | Week 0 |
| M2 | Data models + telemetry contracts | Aegis-TL | Week 1 |
| M3 | Asset discovery PoC + persistence adapters | Dev Team Alpha | Week 2 |
| M4 | Scoring engine baseline + CLI output | Dev Team Beta | Week 3 |
| M5 | Policy guardrails + alert pipeline | Dev Team Gamma | Week 4 |
| M6 | MVP hardening review + pilot runbook | Security Guild | Week 5 |

## Success Criteria
- 95% of critical cloud assets discovered and scored.
- Policy engine detects and reports top 10 CIS violations in staging.
- Telemetry bus sustains 2k events/sec with <1% loss for 1 hour soak.
- CLI delivers prioritized risk report in <5 seconds from cache.
- MVP reviewed + signed off by Security Guild + Product Council.

## Risks & Mitigations
- **Data freshness gaps:** add per-source SLA monitors + retry queues.
- **Policy noise:** include severity thresholds + suppression rules.
- **Operator overload:** default to top findings, progressive disclosure for details.
- **Integration blockers:** document mocks + fallback JSON fixtures for partner teams.

## Next Steps
1. Lock technical requirements for M1-M3.
2. Spin up Aegis-TL charter + responsibility matrix.
3. Align developer squads with backlog derived from this roadmap.
