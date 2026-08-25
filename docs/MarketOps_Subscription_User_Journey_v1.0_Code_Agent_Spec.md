# Syncratic MarketOps
# Subscription Journey & Upgrade Experience
## Engineering Specification for Code Agent
### Version 1.0

---

## 1. Purpose

This specification defines the end-to-end subscription user journey for MarketOps, from first visit through Explorer, Professional, and Institutional adoption.

The implementation goal is not merely to place paywalls around features. The goal is to create a progressive research experience in which users discover market intelligence, build a recurring research habit, encounter contextual value gaps, upgrade when deeper intelligence becomes useful, and move to Institutional when the workflow becomes portfolio-scale, collaborative, automated, or programmatic.

The system SHALL optimize for product adoption and research habit formation rather than immediate subscription conversion.

Primary behavioral objective:

> MarketOps becomes the first application an investor opens when beginning daily market research.

Primary product metric:

`Daily Active Researchers (DAR)`

---

## 2. Product Philosophy

MarketOps is not a charting application, an AI stock picker, or merely a collection of algorithms. It is an automated investment research environment powered by SignalOps.

The product should help users answer:

- What deserves my attention?
- Why does it matter?
- What evidence supports it?
- What context surrounds it?
- What upcoming event could change the thesis?
- How have similar signals behaved historically?

The subscription journey MUST preserve this positioning.

---

## 3. Subscription Tiers

Supported customer-facing tiers:

- Explorer
- Professional
- Institutional

Canonical positioning:

- Explorer — Discover what deserves your attention.
- Professional — Understand why opportunities matter.
- Institutional — Operationalize investment intelligence at scale.

The tiers SHALL represent increasing workflow maturity and SHALL NOT be presented simply as feature bundles.

---

## 4. Customer Maturity Model

The system SHALL model five customer states:

```text
VISITOR
    ↓
EXPLORER
    ↓
RESEARCHER
    ↓
PROFESSIONAL
    ↓
INSTITUTIONAL
```

Only Explorer, Professional, and Institutional are subscription tiers. VISITOR and RESEARCHER are behavioral lifecycle states.

---

## 5. Lifecycle Definitions

### 5.1 VISITOR

Unauthenticated or newly authenticated user evaluating MarketOps.

Primary objective: demonstrate value quickly.

The visitor SHOULD see real market intelligence before being asked to pay.

### 5.2 EXPLORER

Subscriber or entitled user with discovery-oriented access.

Explorer should answer:

- What is happening?
- What deserves attention?
- Which sectors are moving?
- Which public signals matter?

Explorer SHOULD NOT expose full proprietary evidence.

### 5.3 RESEARCHER

Behavioral state representing a highly engaged Explorer user. A Researcher may still be on the Explorer subscription tier.

Typical indicators:

- Frequent login
- Repeated watchlist usage
- Multiple asset-detail visits
- Repeated sector rotation views
- Engagement with locked intelligence panels
- Upcoming-event monitoring
- Daily or near-daily research sessions

The RESEARCHER state exists to improve upgrade timing and messaging.

### 5.4 PROFESSIONAL

Full individual research tier.

Professional answers:

- Why is this opportunity important?
- How is it valued?
- Is it distressed?
- What is options positioning showing?
- What is the earnings-event setup?
- Is the sector supportive?

### 5.5 INSTITUTIONAL

Organization-scale workflow tier.

Institutional answers:

- How do we operationalize intelligence across portfolios?
- How do teams collaborate?
- How do we automate research?
- How do we validate strategies?
- How do we integrate through APIs?

---

## 6. Tier Entitlements

### Explorer

Enabled:

- Market dashboards
- Public signals
- Sector rotation discovery
- Limited watchlists
- Basic market context
- Public opportunity discovery

Locked or limited:

- Value Intelligence
- Distressed Opportunity Intelligence
- Earnings Opportunity Intelligence
- Detailed Sector Rotation Intelligence
- Options signals
- Full earnings calendar
- Research reports
- Signal Assurance analytics
- Portfolio analysis
- Batch screening
- Historical replay
- Strategy validation
- Custom universes
- APIs
- White-label deployment

### Professional

Enabled:

- All Explorer features
- Value Intelligence
- Distressed Opportunity Intelligence
- Earnings Opportunity Intelligence
- Detailed Sector Rotation Intelligence
- Options signals
- Earnings calendar
- Research reports
- Expanded watchlists
- Full research workflows

Locked:

- Full Signal Assurance analytics
- Portfolio-scale analytics
- Batch screening
- Historical replay
- Strategy validation
- Custom universes
- APIs
- White-label deployment

### Institutional

Enabled:

- All Professional features
- Signal Assurance analytics
- Portfolio analysis
- Batch screening
- Historical replay
- Strategy validation
- Custom universes
- APIs
- Multi-user / tenant controls
- White-label deployment
- Contract-specific entitlement overrides

---

## 7. North-Star Metric

Primary metric:

`Daily Active Researchers (DAR)`

A user counts as DAR if they perform one or more meaningful research actions during a market day.

Qualifying actions SHOULD include:

- Opening Market Intelligence
- Reviewing sector rotation
- Opening an asset intelligence page
- Inspecting a signal
- Reviewing an upcoming earnings event
- Adding or reviewing watchlist assets
- Opening a research report
- Reviewing options context
- Reviewing Value Intelligence
- Reviewing Distressed Opportunity Intelligence
- Reviewing Earnings Opportunity Intelligence

Simple authentication alone MUST NOT count as DAR.

---

## 8. Supporting Metrics

Track:

- Visitor → Explorer conversion
- Explorer → Researcher activation
- Researcher → Professional conversion
- Professional → Institutional lead conversion
- Daily Active Researchers
- Weekly Active Researchers
- Research sessions per user
- Assets researched per session
- Watchlist engagement
- Locked feature interaction rate
- Upgrade prompt impression rate
- Upgrade prompt click-through rate
- Checkout initiation rate
- Checkout completion rate
- Professional retention
- Institutional lead qualification rate

---

## 9. Visitor Journey

Recommended flow:

```text
Landing Page
    ↓
Current Market Context
    ↓
Top Public Opportunities
    ↓
Sector Rotation Snapshot
    ↓
Upcoming Market Events
    ↓
Market Intelligence Summary
    ↓
Create Account / Start Explorer
```

The visitor SHOULD see real product output before pricing.

---

## 10. Landing Page Requirements

The landing page SHOULD answer:

- What is happening?
- What is MarketOps seeing?
- Why should I care?
- Can I inspect something real?

Avoid:

- Large algorithm lists
- Excessive feature matrices above the fold
- Technical architecture explanations
- “AI-powered” as the primary value proposition

Preferred positioning:

> Never miss another investment opportunity.
>
> MarketOps continuously monitors the market, surfaces what deserves attention, and provides structured context for better research.

---

## 11. Explorer Activation Journey

Immediately after signup:

```text
Welcome
    ↓
Choose market interests
    ↓
Create first watchlist
    ↓
Show current market state
    ↓
Show top public opportunities
    ↓
Show sector rotation discovery
    ↓
Prompt first asset investigation
```

Do not immediately show a pricing modal.

---

## 12. Explorer Onboarding Inputs

Optional onboarding question:

“What do you primarily research?”

Options MAY include:

- Long-term investing
- Value
- Growth
- Options
- Event-driven
- Distressed opportunities
- Sector rotation
- General market research

These preferences SHOULD tailor the dashboard. Do not make onboarding mandatory unless required for core functionality.

---

## 13. Explorer Home Experience

Default dashboard:

```text
Market Intelligence Summary
    ↓
Today's Public Opportunities
    ↓
Sector Rotation Discovery
    ↓
Upcoming Events
    ↓
Watchlist Changes
    ↓
Recent Signals
```

The purpose is focus. Avoid exposing every internal model at once.

---

## 14. Progressive Disclosure

The system SHALL use progressive disclosure.

Explorer may see:

```text
Software
Rotation In
```

Professional may see:

```text
Software
Rotation In
Rotation Score
Sector Diffusion
Breadth
Internal Leadership
Relative Strength
Capital Flow Proxy
Technical State
Options Context
Evidence
```

Detailed information should appear only when contextually relevant.

---

## 15. Contextual Upgrade Principle

Upgrade prompts SHALL appear when the user demonstrates intent.

Avoid arbitrary full-screen upgrade prompts.

A paywall should communicate:

```text
You are here.
This deeper intelligence answers the next natural question.
```

---

## 16. Upgrade Trigger: Locked Asset Intelligence

Example: Explorer opens NET.

Visible:

- Price
- Public signals
- Sector
- Basic market context

Locked:

- Value Intelligence
- Distressed Opportunity Intelligence
- Options Intelligence
- Earnings Opportunity Intelligence

Upgrade copy SHOULD focus on user outcome:

> Understand why NET deserves attention.
>
> Professional adds valuation, distress, options positioning, earnings-event context, and detailed sector intelligence.

CTA: `Upgrade to Professional`

---

## 17. Upgrade Trigger: Sector Rotation Detail

After repeated Explorer engagement with Sector Rotation:

> Want to understand why Software is rotating?
>
> Professional unlocks Sector Diffusion, Breadth, Internal Leadership, Relative Strength, Capital Flow, and the Rotation Score.

Trigger only after meaningful engagement.

---

## 18. Upgrade Trigger: Watchlist Limit

When approaching limit:

```text
You are using 20 of 25 Explorer watchlist slots.
```

At limit:

> You've reached the Explorer watchlist limit.
>
> Professional expands your research workspace and unlocks full intelligence across your tracked assets.

Never silently block without explanation.

---

## 19. Upgrade Trigger: Earnings Opportunity

Explorer sees:

```text
NVDA
Earnings in 8 days
```

Locked Professional panel:

> Earnings Opportunity Intelligence
>
> Professional analyzes technical setup, options positioning, event risk/reward, valuation context, and sector context.

CTA: `Unlock Earnings Intelligence`

---

## 20. Upgrade Trigger: Options Intelligence

Explorer may see:

```text
Options activity detected
```

Locked detail:

> Professional shows how options positioning aligns or conflicts with market price behavior.

Avoid exposing raw options complexity before the user asks for it.

---

## 21. Upgrade Trigger: Research Reports

Explorer MAY see report title and executive summary. Full report requires Professional.

Example:

```text
MarketOps Research
Software Rotation: Broadening Participation
Summary available
[Unlock Full Research]
```

---

## 22. Researcher State Detection

Suggested behavioral score:

```text
ResearcherScore =
0.20 * LoginFrequency
+ 0.20 * AssetResearchDepth
+ 0.15 * WatchlistEngagement
+ 0.15 * SectorRotationEngagement
+ 0.10 * EventMonitoring
+ 0.10 * LockedFeatureInterest
+ 0.10 * SessionRecency
```

Normalize to 0–100.

Suggested threshold:

`ResearcherScore >= 60`

All weights and thresholds SHALL be configuration-driven.

---

## 23. Researcher State Rules

Potential direct qualification:

```text
>= 4 meaningful sessions in 7 days
AND
>= 5 researched assets
AND
>= 2 locked intelligence interactions
```

OR ResearcherScore threshold.

Researcher classification is internal. Do not display it as a customer-facing tier.

---

## 24. Researcher Upgrade Experience

For Researcher-state Explorer users, display a persistent but non-intrusive CTA:

`Unlock the full research workflow`

Recommended message:

> You've been actively tracking opportunities. Professional adds full valuation, distress, earnings, options, and sector intelligence to the assets you already research.

---

## 25. Professional Checkout Flow

```text
Upgrade CTA
    ↓
Plan Comparison
    ↓
Select Monthly / Annual
    ↓
Stripe Checkout
    ↓
Stripe Success
    ↓
Webhook Confirmation
    ↓
Entitlement Update
    ↓
Professional Welcome
```

Do not grant entitlements based solely on frontend redirect. Stripe webhook or another authoritative subscription state MUST confirm activation.

---

## 26. Stripe Product Mapping

Recommended Stripe products:

- MarketOps Explorer
- MarketOps Professional

Institutional is quote/contact-sales driven.

Each self-service product SHOULD support monthly and annual prices.

---

## 27. Stripe Customer Metadata

Store metadata such as:

```json
{
  "user_id": "...",
  "tenant_id": "...",
  "product": "marketops",
  "tier": "professional",
  "environment": "production"
}
```

Do not rely on Stripe metadata as the sole entitlement store.

---

## 28. Subscription State Model

Canonical billing states:

```text
NONE
TRIALING
ACTIVE
PAST_DUE
CANCELED
UNPAID
INCOMPLETE
INCOMPLETE_EXPIRED
PAUSED
```

Map Stripe states deterministically.

---

## 29. Entitlement State

Separate billing state from entitlement state.

Example:

```text
billing_status = PAST_DUE
entitlement_status = GRACE_PERIOD
```

Supported entitlement states:

```text
ACTIVE
GRACE_PERIOD
LIMITED
SUSPENDED
EXPIRED
```

---

## 30. Subscription Upgrade

Explorer → Professional:

```text
existing subscription
    ↓
select Professional
    ↓
calculate proration
    ↓
user confirms
    ↓
Stripe updates subscription
    ↓
webhook
    ↓
entitlement update
```

### 30.1 Implemented upgrade-intent gate — 2026-08-24

The first implemented production-safe slice records upgrade intent without granting access. Locked-feature prompts and `/marketops/pricing` may record authenticated, tenant-scoped prompt/click events in `subscriber_upgrade_interactions`; Administration > Subscriptions > Upgrade funnel exposes those events for analyst/operator review.

Current boundary:

- Stripe Product and Price IDs are public catalog metadata and may be shown on the pricing page.
- Checkout controls remain disabled by design until a separate Checkout Session endpoint is approved.
- No entitlement may be granted from a frontend redirect, pricing click, or upgrade-interaction row.
- Future automatic activation must be driven by a verified Stripe webhook or another authoritative subscription-state write.
- Invalid Stripe webhook signatures must fail before persistence; valid unknown subscription events may be recorded as `unmatched` without creating access.

Production evidence on 2026-08-24: Playwright verified the pilot Explorer pricing page, disabled Checkout boundary, upgrade-interaction persistence, tenant-filtered Admin Upgrade funnel visibility, Admin Webhook ledger visibility, invalid-signature fail-closed behavior, and one signed synthetic webhook canary recorded as `unmatched`.

---

## 31. Subscription Downgrade

Professional → Explorer SHOULD default to period-end downgrade.

Do not immediately remove paid access unless explicitly chosen.

Display:

> Your Professional access remains active until <date>. Explorer access begins afterward.

---

## 32. Cancellation Flow

Cancellation SHOULD:

- Ask reason
- Offer downgrade to Explorer
- Explain access end date
- Preserve research history
- Preserve watchlists where possible
- Lock rather than delete Professional-only artifacts

Suggested reasons:

- Too expensive
- Not using enough
- Missing features
- Prefer another tool
- Temporary break
- Other

---

## 33. Graceful Downgrade UX

If a Professional user has 150 watchlist assets and Explorer supports 25, do NOT delete 125 assets.

Preferred behavior:

```text
All assets remain stored.
25 remain active under Explorer.
Additional assets become read-only / inactive.
```

Offer reactivation on upgrade.

---

## 34. Professional Welcome Journey

After successful upgrade:

```text
Upgrade confirmed
    ↓
Show newly unlocked intelligence
    ↓
Prompt asset deep dive
    ↓
Offer guided research workflow
```

Recommended message:

> Professional is active. Start with an asset you already follow, or review today's highest-priority opportunities.

---

## 35. Professional Daily Workflow

Default:

```text
Market Intelligence
    ↓
Sector Rotation
    ↓
Watchlist Changes
    ↓
Upcoming Earnings
    ↓
Top Research Candidates
    ↓
Asset Intelligence
```

The workflow should reduce daily research effort.

---

## 36. Professional Asset View

Recommended hierarchy:

```text
Asset Summary
    ↓
Why It Matters
    ↓
Value Intelligence
    ↓
Distressed Opportunity Intelligence
    ↓
Technical Context
    ↓
Options Intelligence
    ↓
Earnings Opportunity
    ↓
Sector Context
    ↓
Market Context
```

Avoid presenting all scores with equal visual priority.

---

## 37. Summary Before Detail

Preferred asset summary example:

```text
NET
Opportunity Context: Supportive

Why?
✓ Valuation improving
✓ Software sector rotating in
✓ Bullish options positioning
✓ Technical structure improving
⚠ Earnings in 12 days
```

Then allow drill-down.

---

## 38. Avoid Signal Overload

Do not expose every raw signal by default.

Use layers:

```text
Summary
    ↓
Drivers
    ↓
Evidence
    ↓
Raw Signals
    ↓
Audit
```

Expert users can reach raw data. Default users start with synthesized intelligence.

---

## 39. Institutional Trigger Model

Institutional upsell SHOULD be behavior-driven.

Potential triggers:

- Multiple portfolio-like watchlists
- Repeated exports
- Large watchlists
- Repeated screening requests
- Team invitations attempted
- Historical analysis requested
- API access page visits
- Custom universe requests
- Multiple users from same company domain
- High research volume

---

## 40. Institutional CTA

Do not show public price.

Use:

> Need portfolio-scale intelligence?
>
> Institutional adds Signal Assurance analytics, portfolio analysis, batch screening, historical replay, strategy validation, custom universes, APIs, and team controls.

CTA: `Contact Sales`

---

## 41. Institutional Lead Capture

Required fields SHOULD be minimal:

- Name
- Work email
- Organization
- Role
- Primary use case
- Estimated number of users

Optional:

- Assets under management range
- API interest
- Portfolio analytics interest
- White-label interest

Do not require lengthy procurement information initially.

---

## 42. Institutional Sales State

Internal lifecycle:

```text
LEAD
QUALIFIED
DISCOVERY
PROPOSAL
QUOTE_SENT
NEGOTIATION
WON
LOST
```

Stripe becomes billing infrastructure after commercial agreement.

---

## 43. Institutional Stripe Flow

Recommended:

```text
Contact Sales
    ↓
Qualification
    ↓
Commercial Agreement
    ↓
Stripe Customer
    ↓
Stripe Quote
    ↓
Quote Acceptance
    ↓
Subscription / Invoice
    ↓
Tenant Entitlement Override
```

Institutional SHOULD NOT use generic public Checkout.

---

## 44. Institutional Policy Overrides

Example:

```json
{
  "tenant_id": "tenant_abc",
  "tier": "institutional",
  "overrides": {
    "api_requests_per_day": 500000,
    "portfolio_count": 100,
    "custom_universe_count": 75,
    "white_label_deployment": true
  }
}
```

---

## 45. Feature-Gating Architecture

Frontend SHALL NOT hardcode tier conditionals throughout the application.

Use an entitlement service:

```text
can(user, feature)
limit(user, resource)
```

---

## 46. Entitlement API

Example:

```http
GET /v1/subscriptions/entitlements
```

Response:

```json
{
  "tier": "professional",
  "features": {
    "value_intelligence": true,
    "signal_assurance_analytics": false
  },
  "limits": {
    "watchlists": 10,
    "watchlist_assets_per_list": 250
  }
}
```

---

## 47. Locked Feature Component

Create reusable UI component such as:

`<EntitlementGate>`

Suggested props:

- feature
- required_tier
- upgrade_message
- cta
- preview_mode

Behavior:

```text
entitled → render feature
not entitled → render preview / lock state
```

---

## 48. Preview Modes

Supported:

```text
NONE
SUMMARY_ONLY
BLURRED_DETAIL
PARTIAL_DATA
READ_ONLY
```

Use SUMMARY_ONLY by default. Avoid excessive blur-based paywalls.

---

## 49. Upgrade CTA Tracking

Every upgrade CTA SHALL include context metadata:

```json
{
  "source": "asset_value_intelligence",
  "asset": "NET",
  "current_tier": "explorer",
  "required_tier": "professional",
  "session_id": "...",
  "cta_variant": "contextual"
}
```

---

## 50. Funnel Events

Track:

```text
pricing_viewed
upgrade_prompt_shown
upgrade_prompt_clicked
checkout_started
checkout_completed
checkout_abandoned
subscription_activated
subscription_upgraded
subscription_downgrade_scheduled
subscription_canceled
institutional_cta_clicked
institutional_lead_submitted
```

---

## 51. Research Behavior Events

Track:

```text
market_dashboard_viewed
market_intelligence_viewed
sector_rotation_viewed
asset_researched
signal_inspected
watchlist_created
watchlist_asset_added
earnings_event_viewed
options_context_viewed
value_intelligence_viewed
distressed_intelligence_viewed
earnings_intelligence_viewed
research_report_viewed
```

---

## 52. Conversion Attribution

Professional conversion SHOULD record:

- first_upgrade_trigger
- last_upgrade_trigger
- most_frequent_locked_feature
- researcher_score_at_conversion
- days_since_signup
- DAR_count_prior_30d

This supports optimization.

---

## 53. Upgrade Timing Rules

Recommended:

```text
First locked interaction:
inline preview only.

Second / third meaningful interaction:
stronger inline CTA.

Researcher state:
persistent Professional CTA.

High-intent action:
checkout recommendation.
```

---

## 54. Pricing Page

Pricing SHOULD be accessible at all times, but not forced before product experience.

Pricing page hierarchy:

- Explorer — Discover what deserves attention.
- Professional — Understand why opportunities matter.
- Institutional — Operationalize intelligence at scale.

---

## 55. Explorer Pricing Strategy

Pricing SHOULD be configuration-driven.

Suggested current configuration:

```json
{
  "monthly": 24.99,
  "annual": 249.00
}
```

Actual Stripe Price IDs SHALL determine billing.

---

## 56. Professional Pricing Strategy

Suggested configuration:

```json
{
  "monthly": 99.00,
  "annual": 999.00
}
```

Do not expose prices from frontend constants if Stripe or a pricing service is authoritative.

---

## 57. Annual Plan Preference

UI SHOULD visually favor annual billing.

Display savings accurately based on current price configuration. Do not hardcode a savings percentage.

---

## 58. Founding User Program

Optional.

Professional founding pricing MAY be supported through:

- Stripe coupon
- Promotion code
- Special Stripe price

Entitlements remain Professional. Do not create a separate tier.

---

## 59. Product Value Messaging

Avoid customer-facing algorithm jargon as the primary upsell message.

Avoid:

```text
Unlock VC
Unlock DOSM
Unlock EEOM
```

Prefer:

```text
Understand valuation
Evaluate distress
Analyze earnings risk
Contextualize options
Understand sector rotation
```

Internal algorithm names MAY remain visible in advanced views.

---

## 60. Research Workflow Messaging

Canonical workflow:

```text
Discover
    ↓
Understand
    ↓
Validate
    ↓
Decide
```

SignalOps-native conceptual model:

```text
Signals
    ↓
Evidence
    ↓
Opportunity
```

Human conviction remains outside the system.

---

## 61. Retention UX

Professional retention SHOULD prove recurring value.

Show metrics such as:

- Opportunities reviewed this month
- Events surfaced before occurrence
- Watchlist signals surfaced
- Sector changes detected
- Research reports generated

Avoid claims implying realized investment gains unless rigorously supported.

---

## 62. Weekly Research Digest

Recommended:

```text
What changed this week?
- Market regime
- Sector rotations
- New opportunities
- Upcoming earnings
- Watchlist changes
```

Explorer receives abbreviated digest. Professional receives deeper evidence.

---

## 63. Upgrade Email Strategy

Use sparingly.

Appropriate triggers:

- Researcher state reached
- Repeated locked feature usage
- Watchlist limit reached
- Upcoming major event on watched asset

Avoid generic repeated sales emails.

---

## 64. In-App Upgrade Messaging Principles

Every message SHOULD answer:

`Why is this useful right now?`

Bad:

`Upgrade to Professional!`

Better:

> You're tracking an earnings event. Professional shows how technicals, options positioning, valuation, and sector context align before the event.

---

## 65. Institutional Messaging Principles

Institutional is not “more Professional.” It is a different workflow.

Key terms:

- portfolio-scale
- team workflows
- governance
- automation
- integration
- validation
- customization

---

## 66. Billing Portal

Users SHOULD be able to:

- View current plan
- View billing interval
- Update payment method
- Download invoices
- Cancel
- Upgrade
- Schedule downgrade

Stripe Customer Portal MAY provide these functions.

---

## 67. Billing Failure UX

If payment fails, allow a grace period.

Display:

> We couldn't process your latest payment. Your MarketOps access remains available during the grace period. Update your payment method to avoid interruption.

Avoid immediate destructive downgrade.

---

## 68. Grace Period

Recommended configurable grace period:

`3–7 days`

During grace:

- full entitlement
- billing warning visible

After grace:

- entitlement limited or downgraded

---

## 69. Subscription Service

Recommended service boundary:

`marketops-subscription-service`

Responsibilities:

- Stripe customer mapping
- Subscription state
- Tier mapping
- Entitlements
- Limits
- Overrides
- Upgrade/downgrade state
- Webhook processing
- Billing audit

---

## 70. Stripe Webhooks

Support at minimum:

```text
checkout.session.completed
customer.subscription.created
customer.subscription.updated
customer.subscription.deleted
invoice.paid
invoice.payment_failed
invoice.payment_action_required
customer.updated
```

Verify webhook signatures.

---

## 71. Webhook Idempotency

Every Stripe event SHALL be idempotent.

Persist:

- stripe_event_id
- event_type
- processed_at
- status

Enforce unique constraint on `stripe_event_id`.

---

## 72. Subscription Persistence

Recommended tables:

```text
subscriptions
subscription_entitlements
subscription_overrides
subscription_events
stripe_customers
upgrade_interactions
research_activity
institutional_leads
```

---

## 73. subscriptions Table

Suggested fields:

```text
subscription_id
user_id
tenant_id
stripe_customer_id
stripe_subscription_id
tier
billing_interval
billing_status
entitlement_status
current_period_start
current_period_end
cancel_at_period_end
created_at
updated_at
```

---

## 74. Upgrade Interaction Table

Track:

```text
interaction_id
user_id
session_id
source_feature
asset_symbol
current_tier
required_tier
prompt_variant
shown_at
clicked_at
checkout_started_at
converted_at
```

---

## 75. Research Activity Table

Track summarized behavior rather than unlimited raw frontend noise.

Example fields:

```text
user_id
activity_date
research_sessions
assets_researched
watchlist_interactions
sector_views
event_views
locked_feature_interactions
meaningful_actions
```

Use for ResearcherScore.

---

## 76. Privacy

Do not use investment research behavior for unrelated advertising.

Research behavior should be used for:

- personalization
- upgrade relevance
- platform analytics
- product improvement

Document this appropriately in the privacy policy.

---

## 77. UX State Preservation

Upgrades SHALL preserve:

- current asset
- current research page
- filters
- watchlists
- current workflow

After checkout, return the user to the context that triggered upgrade.

Example:

```text
Explorer clicks Value Intelligence on NET
    ↓
Checkout
    ↓
Professional activated
    ↓
Return directly to NET Value Intelligence
```

This is critical.

---

## 78. Checkout Recovery

If checkout is abandoned:

- Preserve trigger context
- Do not spam
- Allow user to resume
- Optionally show a soft reminder later

---

## 79. Institutional Lead Context

When a user clicks Contact Sales from a feature, include context such as:

- source_feature
- current_tier
- organization_domain
- usage_level
- requested_capability

Do not require the user to repeat known context.

---

## 80. Admin Capabilities

Internal admin SHOULD support:

- View subscription
- View Stripe IDs
- View tier
- Grant temporary entitlement
- Apply override
- Revoke override
- Inspect billing event history
- Inspect upgrade funnel
- Inspect ResearcherScore

All admin actions MUST be audited.

---

## 81. Feature Flags

Use separate feature flags for rollout.

Effective feature availability requires:

```text
feature exists
AND
tier entitled
AND
rollout enabled
```

Do not conflate entitlement with deployment availability.

---

## 82. A/B Testing

Upgrade messaging MAY be tested.

Do not A/B test core entitlement truth.

Testable elements:

- CTA wording
- placement
- preview depth
- annual vs monthly emphasis

Primary metrics:

- conversion
- retention
- DAR
- engagement

---

## 83. Avoid Dark Patterns

Do not:

- Hide cancellation
- Misstate savings
- Preselect expensive plans without clarity
- Delete user data on downgrade
- Block account access to force upgrade
- Present model outputs as guaranteed investment results

Trust is central to MarketOps.

---

## 84. Accessibility

Upgrade UI SHALL support:

- keyboard navigation
- screen readers
- sufficient contrast
- explicit locked-state labels
- semantic buttons
- accessible modal focus management

---

## 85. Mobile Behavior

Locked feature previews MUST remain understandable on small screens.

Avoid dense comparison tables as the only upgrade experience.

---

## 86. Suggested Repository Layout

```text
marketops/
  services/
    subscription/
      cmd/
      internal/
        billing/
        stripe/
        entitlements/
        policies/
        upgrades/
        webhooks/
        institutional/
        persistence/
        analytics/

  frontend/
    subscription/
      components/
        EntitlementGate/
        UpgradeCard/
        UpgradeModal/
        PlanComparison/
        BillingStatus/
        InstitutionalCTA/
      hooks/
        useEntitlements/
        useSubscription/
        useUpgradeContext/
```

---

## 87. Phase 1 Implementation

Build:

- Explorer / Professional entitlement model
- Stripe Checkout
- Stripe webhook processing
- Entitlement API
- Reusable EntitlementGate
- Basic pricing page
- Asset intelligence locks
- Watchlist limit upgrade
- Sector detail upgrade
- Upgrade analytics

---

## 88. Phase 2 Implementation

Add:

- ResearcherScore
- Contextual upgrade ranking
- Professional onboarding
- Cancellation/downgrade UX
- Billing grace state
- Weekly research digest
- Subscription retention metrics

---

## 89. Phase 3 Implementation

Add:

- Institutional lead capture
- Tenant overrides
- Stripe Quotes workflow
- Institutional admin tooling
- Portfolio-scale upgrade triggers
- API entitlement enforcement

---

## 90. Phase 4 Optimization

Add:

- Conversion attribution
- Upgrade A/B tests
- Personalized upgrade prompts
- Usage-based institutional lead scoring
- Retention intelligence

---

## 91. Acceptance Criteria

Implementation is complete when:

1. Visitor can view real public MarketOps content before purchase.
2. Explorer signup works.
3. Explorer entitlements enforce correctly.
4. Locked features show contextual previews.
5. Upgrade CTA retains source context.
6. Stripe Checkout creates or updates subscription.
7. Webhooks activate entitlements.
8. Explorer → Professional works without account recreation.
9. Professional user returns to triggering research context after checkout.
10. Downgrade can be scheduled.
11. Cancellation preserves user data.
12. Watchlist overage is handled gracefully.
13. Research activity is recorded.
14. DAR is measurable.
15. Researcher state can be calculated.
16. Professional upgrade funnel is measurable.
17. Institutional CTA exists.
18. Institutional lead is captured.
19. Institutional entitlement overrides are supported.
20. All billing events are auditable.

---

## 92. Canonical User Journey

```text
VISITOR
    ↓
Sees real market intelligence
    ↓
Creates Explorer account
    ↓
Builds watchlist
    ↓
Uses market dashboard
    ↓
Discovers sector rotation
    ↓
Investigates assets
    ↓
Encounters deeper intelligence naturally
    ↓
Becomes habitual RESEARCHER
    ↓
Upgrades to PROFESSIONAL
    ↓
MarketOps becomes daily research workspace
    ↓
Research needs become portfolio/team/programmatic
    ↓
Institutional CTA
    ↓
Sales / Quote
    ↓
INSTITUTIONAL
```

---

## 93. Final Product Principle

The subscription system must not feel like a wall between the user and MarketOps.

It should feel like a natural progression:

```text
Discovery
    ↓
Research
    ↓
Operationalization
```

Explorer must prove that MarketOps sees useful things.

Professional must prove that MarketOps explains those things.

Institutional must let organizations scale, validate, automate, and integrate that intelligence.

The platform should optimize for recurring research behavior first.

The most important question is not:

```text
Did the user subscribe?
```

It is:

```text
Did the user begin their research inside MarketOps today?
```

If MarketOps becomes part of the user's daily investment process, upgrades and retention become consequences of product value rather than aggressive monetization.

---


---

## Addendum A — Enrollment and Operational Lifecycle Workflow

Date: 2026-08-24
Status: planning addendum for the next user-journey sprint

### A.1 Current implemented baseline

MarketOps currently has a production-safe subscription foundation, but not a complete self-service enrollment-to-paid-activation flow.

Implemented baseline:

- Explorer, Professional, and Institutional tier policy records exist.
- Subscription Administration exposes tier settings, Stripe billing mappings, users/seats, upgrade funnel, webhook ledger, audit evidence, and user activity.
- The pricing page renders configured Stripe product/price metadata.
- Locked-feature prompts and pricing interactions persist authenticated, tenant-scoped upgrade-intent rows.
- Stripe webhook handling is signed and fail-closed; valid unknown subscription events are recorded without granting access.
- Admin-managed billing and subscription mapping are available for controlled provisioning.
- Subscription enforcement canaries have validated Explorer, Professional, and Institutional behavior and safe restoration.
- User activity logging exists for login, feature views, and mutating requests.

### A.2 Enrollment gap

The next sprint must close the gap between identity creation and a coherent MarketOps first-use experience.

Current gaps:

- New users do not yet have a guided first-run journey.
- New users do not automatically receive a clearly explained default tier state.
- New users do not receive a first watchlist/default-list explanation as part of onboarding.
- Tenant/user assignment still depends on Keycloak/admin/backend provisioning rather than a visible enrollment workflow.
- There is no explicit user-facing state for pending enrollment, active Explorer, active Professional, payment issue, cancellation, or downgrade scheduled.

Required workflow:

```text
New identity / first login
    ↓
Resolve tenant and subject
    ↓
Resolve subscription lifecycle state
    ↓
Create or select default watchlist context
    ↓
Show first-use MarketOps orientation
    ↓
Record enrollment milestone
    ↓
Begin normal MarketOps research workflow
```

### A.3 Subscription lifecycle gap

Checkout remains intentionally disabled until the lifecycle target state is explicit and webhook-confirmed activation is ready.

Current gaps:

- No Checkout Session endpoint exists.
- No Stripe Customer Portal endpoint exists.
- No complete return-to-context flow exists after successful checkout.
- No automatic paid entitlement activation should occur from frontend redirects.
- Payment-failure, cancellation, downgrade, renewal, and grace-period states are not fully surfaced to users.

Required lifecycle states:

- `enrollment_pending`
- `explorer_active`
- `professional_active`
- `institutional_active`
- `checkout_pending`
- `payment_action_required`
- `payment_failed`
- `cancel_scheduled`
- `downgrade_scheduled`
- `canceled`
- `admin_provisioned`
- `webhook_pending_reconciliation`

Authoritative activation rule:

> No entitlement may be activated from a pricing click, frontend redirect, or upgrade-interaction row. Entitlement changes must come from a verified webhook reconciliation or an explicit admin-governed subscription mutation.

### A.4 Operational-flow gap

The user journey must be observable by administrators from enrollment through daily usage and upgrade interest.

Admin should be able to answer:

- Which users are enrolled?
- Which tenant does each user belong to?
- Which tier and lifecycle state is active?
- Which users are blocked by entitlement limits?
- Which users are repeatedly hitting Professional-value surfaces?
- Which users have payment or webhook reconciliation issues?
- Which users are active researchers versus inactive accounts?
- Which watchlists/default lists govern each user’s MarketOps experience?

Operational flow:

```text
User enrolls or is provisioned
    ↓
Admin sees subject + tenant + tier + lifecycle state
    ↓
User activity events accumulate
    ↓
Upgrade prompts and pricing clicks are attributed
    ↓
Checkout/webhook/admin changes update lifecycle state
    ↓
Admin can search, filter, and audit the full path
```

### A.5 Value progression gap

The product should not push upgrade prompts generically. Upgrade timing should be derived from usage and research maturity.

Signals to use for future Researcher-state scoring:

- repeated logins across trading days;
- repeated watchlist usage;
- repeated locked-feature interactions;
- Syncratic explainability usage;
- Risk/Reward drill-down usage;
- Signal Assurance drill-down usage;
- Sector Rotation Intelligence follow-through;
- Review Queue/opportunity investigation;
- multiple assets monitored over time.

Researcher-state objective:

> Identify when an Explorer user has begun behaving like a serious researcher and present Professional upgrade context at the moment deeper intelligence would be useful.

### A.6 Institutional journey gap

Institutional remains Contact Sales, but the operational path needs structure.

Required future workflow:

```text
Institutional CTA
    ↓
Lead capture with organization/use-case context
    ↓
Admin review
    ↓
Tenant provisioning
    ↓
Seat allocation
    ↓
Contract/quote mapping
    ↓
Institutional entitlement activation
    ↓
Tenant administrator manages users/seats
```

Institutional should not use generic public Checkout unless a later product decision explicitly changes that boundary.

### A.7 Recommended next sprint

Sprint name: Subscriber Enrollment and Lifecycle Journey

Implementation order:

1. Enrollment state foundation
   - Add a durable enrollment/lifecycle state projection for each subject subscription.
   - Surface the state in Subscription Administration users/seats views.
   - Record first-login/first-enrollment milestone events.

2. First-use MarketOps onboarding
   - On first valid MarketOps access, resolve default watchlist context.
   - Explain the user’s tier, default list, and available research workflow.
   - Keep onboarding dismissible and non-blocking unless required data is missing.

3. Checkout readiness boundary
   - Define Checkout Session API contract but keep live Checkout disabled until explicit approval.
   - Preserve return-to-context metadata from upgrade prompts.
   - Require webhook-confirmed activation before entitlements change.

4. Operational observability
   - Add admin filters for lifecycle state, tier, activity recency, upgrade-interest source, and webhook/payment issues.
   - Connect user activity, upgrade interactions, and subscription state into one user drill-down.

### A.8 Acceptance criteria for the sprint

The sprint is complete when:

1. A newly provisioned user has a visible lifecycle state.
2. First login creates or confirms the expected enrollment milestone.
3. The user sees a coherent first-use MarketOps experience without needing admin explanation.
4. The user’s default watchlist context is selected and explained.
5. Upgrade prompts still record source context.
6. Admin can search the user and see tier, lifecycle state, watchlist/default context, recent activity, and upgrade interest.
7. Checkout remains disabled unless explicitly approved.
8. No entitlement is granted without webhook-confirmed or admin-governed activation.
9. Institutional remains Contact Sales with a structured lead/provisioning path documented.
10. All enrollment, lifecycle, and upgrade events are auditable.

### A.10 Option B enrollment-to-subscription policy — 2026-08-25

The selected production policy is Option B: verified public identity registration does not automatically activate Explorer. A B2C subject may receive tenant-scoped identity/access scaffolding, but MarketOps readiness under subscription enforcement requires an active subscription created through governed subscription administration or verified Stripe webhook reconciliation.

Implementation guardrail: `SIGNALOPS_SUBSCRIBER_B2C_AUTO_ACTIVATE_EXPLORER` defaults to `false`. Setting it to `true` re-enables legacy Explorer auto-activation only as an explicit controlled exception.

### A.9 Implemented Keycloak B2C enrollment slice — 2026-08-25

The first enrollment implementation reuses the existing Syncratic Keycloak realm and `signalops-web` OIDC client rather than creating a separate pending realm. The browser now exposes a Create account path that invokes Keycloak registration through the same Authorization Code + PKCE callback.

SignalOps adds `GET /v1/session/enrollment` as the authenticated first-use resolver. It is intentionally reachable before normal MarketOps access grants are complete, while retaining tenant-claim validation and rate limiting. The resolver auto-provisions only verified users whose signed token tenant matches the configured B2C tenant, default `tenant-b2c`.

For an eligible verified B2C user, the resolver idempotently creates or confirms:

- MarketOps read access in `tenant_user_access`;
- active Explorer subject subscription;
- readable watchlist context, creating the B2C tenant starter list only if no list exists;
- enrollment activity/audit evidence.

Unverified users receive `email_verification_required` and no access is created. Non-B2C users receive explicit pending states for administrator-managed enrollment.


# End of Specification
