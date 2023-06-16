[//]: # (START/LATEST)
# Latest

## Features
  * A user-friendly description of a new feature. {issue-number}

## Fixes
 * A user-friendly description of a fix. {issue-number}

## Security
 * A user-friendly description of a security fix. {issue-number}

---

[//]: # (START/v1.0.2)
# v1.0.2

## Features
  * Upgraded Go to version 1.20. {#34}
  * Upgraded 1Password Connect to version 1.5.1. {#34}

## Fixes
 * Fixed bug causing the need for the mutatingwebhookconfig object to be deleted every time the application restarts. {#32}


---

[//]: # "START/v1.0.1"

# v1.0.1

## Fixes
* Injector no longer overwrites pod `volumeMounts`. {#22}

---

[//]: # "START/v1.0.0"

# v1.0.0

Initial 1Password Kubernetes Secrets Injector release

## Features

- Fetch secrets from 1Password and inject them into the pods as environment variables.
- Webhook works with multiple configurations (i.e. different Connect hosts and tokens).
- Provide a simple deployment process (`make deploy`).

---
