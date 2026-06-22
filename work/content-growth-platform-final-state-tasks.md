1. Establish the productization foundation for matrix accounts, creator collaborations, skill packages, campaigns, and monetization.
   Define the shared domain boundaries, tenant permissions, commercial entitlements, audit requirements, and admin ownership for the four product lines. This is the dependency anchor because later modules need one consistent way to express workspace ownership, platform-operated assets, paid access, review states, and compliance evidence.

2. Introduce repository and service boundaries for core business workflows.
   Move the existing handler-centered read/write paths behind stable domain services for accounts, contents, generation, publishing, marketplace resources, and admin operations. This reduces coupling before adding high-volume account metrics, creator orders, paid skill packages, and campaign analytics.

3. Add a platform connector capability model for external media platforms.
   Expand media platform definitions from basic publishing flags into a governed capability contract covering authorization, account profile sync, content publishing, metric ingestion, comment ingestion, rate limits, and manual-only fallbacks. The goal is to let each platform expose only legally and technically supported operations instead of assuming every channel can be scraped, posted to, or analyzed the same way.

4. Upgrade tenant media accounts into a managed media account matrix.
   Add first-class account grouping, ownership type, operating role, brand/persona positioning, target audience, content categories, health state, authorization scope, and workspace-level account portfolio views. This turns isolated bound accounts into an operational matrix that can be searched, filtered, governed, and selected for content strategy.

5. Add external account data ingestion and metric history.
   Persist account profile snapshots, follower metrics, content metrics, engagement metrics, audience signals, sync status, and data freshness across connected platforms. This creates the historical data layer required for account diagnosis, content attribution, campaign reporting, and AI-assisted publishing recommendations.

6. Build the media matrix operations console for tenants.
   Provide workspace users with account portfolio dashboards, account detail pages, sync status, health warnings, metric trends, content performance tables, and operational filters. This is the product surface that makes the matrix usable for daily operations rather than hiding it as backend data.

7. Add content-to-account attribution across generation, review, scheduling, publishing, and metric collection.
   Link every draft, generated output, publishing job, external URL, and performance result back to the selected account, knowledge context, writing skill, campaign, and operator. This makes later recommendations and business reporting defensible because the system can explain what produced each result.

8. Add campaign planning as the organizing layer above single contents and schedules.
   Introduce campaigns with goals, products, target audiences, channels, accounts, timelines, budget assumptions, content quotas, approval policy, and success metrics. This is the final-state workflow container for continuous content operations, not a one-off MVP scheduling wrapper.

9. Add campaign content calendars and publishing orchestration.
   Let campaigns own planned topics, draft assignments, publishing windows, target accounts, dependencies, approval gates, and schedule generation. This connects strategy to execution so the platform can manage coordinated multi-account publishing instead of individual ad hoc posts.

10. Add performance reporting and recommendation loops for campaigns and account matrices.
   Aggregate account and content metrics into campaign reports, account benchmarks, topic performance, format performance, and publishing time recommendations. This closes the loop from publishing results back into the next round of planning and generation.

11. Establish the creator collaboration domain as a separate commercial channel from tenant-owned accounts.
   Model creators, creator media accounts, public profile data, verticals, audience attributes, pricing, availability, collaboration policies, and verification state without treating creator accounts as tenant login resources. This keeps the product compliant and avoids designing around unsafe account borrowing.

12. Add creator discovery, shortlist, and qualification workflows.
   Support creator search, saved shortlists, fit scoring, historical performance review, brand safety indicators, and operator notes. This gives brands and agencies a serious sourcing workflow before any order or publishing commitment exists.

13. Add creator campaign briefs and collaboration orders.
   Allow workspace users to create structured briefs, deliverable requirements, platform targets, disclosure requirements, usage rights, review windows, deadlines, and commercial terms. Orders become the contract-like operational unit that tracks commitment, scope, status, and accountability.

14. Add creator deliverable collaboration and approval workflows.
   Support draft exchange, asset submission, brand feedback, creator revisions, final approval, publication proof, external link capture, and post-publication metric collection. This makes creator collaboration a managed delivery process rather than a messaging-only feature.

15. Add creator settlement, billing, and platform commission records.
   Track order pricing, deposits, completion milestones, refunds, service fees, creator payouts, invoices, and settlement status. This is necessary for a productized creator marketplace because collaboration value is only complete when commercial reconciliation is auditable.

16. Add creator collaboration governance and compliance evidence.
   Persist advertising disclosure requirements, prohibited claims, authorization records, content usage rights, review logs, publication proofs, and dispute records. This protects brands, creators, and the platform when commercial content must be explainable after publication.

17. Establish skill packages as versioned commercial creation products.
   Model skill packages with category, target platform, target industry, supported content formats, prompt contract, output schema, quality rules, examples, author, listing status, price, and version lifecycle. This turns writing skill into a sellable product instead of an invisible prompt setting.

18. Add skill package authoring and platform review workflows.
   Provide platform admins and approved creators with draft, submit, review, reject, approve, publish, deprecate, and version-release states for skill packages. Review is required because skill packages can affect AI behavior, customer outcomes, and compliance risk.

19. Add skill package marketplace discovery and entitlement management.
   Let tenants browse, compare, buy, subscribe to, install, renew, and uninstall skill packages while enforcing workspace-level access rights. This is the commercial bridge between platform-managed creation expertise and tenant generation workflows.

20. Integrate installed skill packages into generation, formatting, QA, and publishing preparation.
   Let generation requests select or recommend entitled skill package versions and record the chosen package in content, generation logs, traces, and downstream reports. This enables measurable product value because the platform can attribute content outcomes to a specific paid skill package.

21. Add skill package performance analytics and revenue reporting.
   Report usage volume, generated content outcomes, approval rates, publishing performance, refund signals, subscription revenue, author revenue share, and package-level quality flags. This lets the platform operate skill packages as a marketplace with accountable supply quality.

22. Add brand asset libraries and reusable content guardrails.
   Extend workspace knowledge into structured brand assets, product claims, audience definitions, visual assets, approved phrases, forbidden phrases, legal disclaimers, and channel-specific constraints. These assets give campaigns, skill packages, creator briefs, and compliance checks a shared source of truth.

23. Add enterprise review workflows and role-based approvals.
   Support configurable review stages for drafts, campaign plans, creator deliverables, publishing jobs, and high-risk claims. This is required for company and agency usage where publishing authority, brand approval, and legal review cannot be collapsed into one operator action.

24. Add compliance and risk checks across generated content, creator deliverables, and scheduled posts.
   Evaluate advertising disclosure, risky claims, sensitive categories, private data exposure, prohibited wording, platform constraints, and AI-generated content labeling requirements. The same risk layer should serve tenant-owned publishing, creator collaboration, and marketplace skill output.

25. Add agency and multi-client operating capabilities.
   Support agencies that manage multiple client workspaces, shared operator teams, reusable campaign templates, cross-client reporting, permission scoping, and client-facing exports. This broadens the final product beyond individual creators and single-brand companies.

26. Add executive reports and client delivery packages.
   Generate scheduled weekly and monthly reports covering matrix account growth, campaign delivery, creator collaboration outcomes, content performance, spend, ROI proxies, and next-cycle recommendations. This makes the platform useful as an operating system for teams that must report outcomes to clients or management.

27. Add cross-module business intelligence and AI strategy recommendations.
   Use the accumulated account metrics, campaign results, skill package attribution, creator outcomes, and brand assets to recommend topics, formats, accounts, creators, budgets, publishing cadence, and skill packages. This is the final-state intelligence layer that differentiates the product from a basic content generator.
