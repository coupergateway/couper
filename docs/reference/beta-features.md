# Configuration Reference ~ Beta Features

We use Beta Features to have the possibility to develop new, complex features for
You while still being able to maintain our compatibility promise.

You can see beta features as a feature preview. We will announce new Beta Features
in the Changelog and document them on this page.

We will keep features in beta as long as we collect feedback and are actively working
on it. You can expect Beta Features to evolve.

```diff
! Beta Features can change with every release.
```

We recommended You to pin Couper to a specific [docker tag](https://hub.docker.com/r/avenga/couper/tags)
to avoid unintended updates. (Add tests to Your code to make sure that nothing is
going to break when You want to update to a newer version).

To make You and Your colleagues aware that a beta feature is used, we will prefix
all configuration items such as config blocks or functions with `beta_`.

## Feedback more than welcome

We need Your feedback to make Beta Features to proper features. Please leave Your
feedback and questions [here](https://github.com/avenga/couper/discussions), or open
an [issue](https://github.com/avenga/couper/issues). Thank You! :)

## Current Beta Features

* [OAuth2 AC Block](blocks/beta-oauth2-ac.md)
* [OIDC Block](blocks/beta-oidc.md)
* [`beta_oauth_authorization_url()` Function](functions.md)
* [`beta_oauth_verifier()` Function](functions.md)

-----

## Navigation

* &#8673; [Configuration Reference](README.md)
* &#8672; [Attributes](attributes.md)
* &#8674; [Blocks](blocks.md)
