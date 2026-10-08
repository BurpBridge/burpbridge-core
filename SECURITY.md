# Security policy

The current development branch, `main`, is the supported version. BurpBridge is
still under development; review its documented platform and protocol limitations
before using it with sensitive traffic.

## Report a vulnerability privately

Use [GitHub private vulnerability reporting](https://github.com/BurpBridge/burpbridge-core/security/advisories/new).
Do not post exploit details, credentials, or intercepted traffic in public issues.

Include the affected commit or version, platform, reproduction steps, expected
and actual behavior, and potential impact. Redact personal data and secrets.
Maintainers will assess the report, coordinate fixes and disclosure with you,
and ask permission before publicly crediting you. Response times depend on
maintainer availability.

## Contribute securely

Keep dependencies current, use the required DCO sign-off and GPG signature,
and review changes that affect packet handling, proxy connections, and workflow
permissions carefully. CodeQL scans Go, Python, and GitHub Actions. Secret
scanning and push protection help prevent accidental credential publication.
