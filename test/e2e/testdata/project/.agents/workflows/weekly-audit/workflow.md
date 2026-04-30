---
name: weekly-audit
description: "Weekly code audit workflow that analyzes code quality and produces a report."
steps:
  - name: analyze
    description: "Scan the codebase for code quality issues, complexity violations, and security concerns."
  - name: report
    description: "Generate a structured audit report with findings, severity levels, and remediation suggestions."
---
## Overview

Run this workflow weekly to maintain code quality standards. The workflow scans the codebase for violations, generates a structured report, and tracks trends over time.

## Step 1: Analyze

Scan the following dimensions:
- Function length violations (> 80 lines)
- Cyclomatic complexity violations (> 10)
- Unused exports and dead code
- Missing error handling (unchecked error returns)
- Security anti-patterns (hardcoded secrets, unsafe string concatenation)

## Step 2: Report

Produce a Markdown report with:
- Summary statistics (total files scanned, violations found, severity breakdown)
- Top 10 most complex functions
- New violations since last audit
- Resolved violations since last audit
- Trend chart (improving/stable/degrading)

Save the report to `docs/audit/weekly-YYYY-MM-DD.md`.
