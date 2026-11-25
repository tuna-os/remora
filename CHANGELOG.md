# Changelog

## [0.4.8](https://github.com/tuna-os/remora/compare/v0.4.7...v0.4.8) (2026-11-25)


### Bug Fixes

* **ci:** skip pull_request CI run on release-please branches ([#199](https://github.com/tuna-os/remora/issues/199)) ([1e39c08](https://github.com/tuna-os/remora/commit/1e39c08974081b2719e511d596909310a89f6266))
* evaluate the configured coverage threshold ([#203](https://github.com/tuna-os/remora/issues/203)) ([daeadfa](https://github.com/tuna-os/remora/commit/daeadfa05e6fcde169fb8c861c9547e9078a683d))


### Refactoring

* **buildplan:** move the apply no-op and base pin transitions out of the CLI ([#195](https://github.com/tuna-os/remora/issues/195)) ([869b8ed](https://github.com/tuna-os/remora/commit/869b8ed1d045531d000ac5b8b770865e6038628b))
* **pkgmgr:** give the package-manager family list one owner ([#198](https://github.com/tuna-os/remora/issues/198)) ([b68932b](https://github.com/tuna-os/remora/commit/b68932bef08d29959499a88e7d8c8643a4f659fa))
* **registry:** move digest pinning out of internal/host ([#196](https://github.com/tuna-os/remora/issues/196)) ([fbc9dc0](https://github.com/tuna-os/remora/commit/fbc9dc03eb6a3de36e493e57f1226ad9ed678409))


### Documentation

* add migration guide for rpm-ostree users ([#200](https://github.com/tuna-os/remora/issues/200)) ([dc36ac1](https://github.com/tuna-os/remora/commit/dc36ac112f48f3c893cb0e60c83a4f3624c3ebd2))
* **readme:** link the rpm-ostree migration guide ([#201](https://github.com/tuna-os/remora/issues/201)) ([27268f2](https://github.com/tuna-os/remora/commit/27268f2a4fb5587605233781d49ce3450a98f18a))

## [0.4.7](https://github.com/tuna-os/remora/compare/v0.4.6...v0.4.7) (2026-11-13)


### Documentation

* record golangci-lint as an enforced CI check ([#182](https://github.com/tuna-os/remora/issues/182)) ([681f478](https://github.com/tuna-os/remora/commit/681f478d71400870597351117c945f33d859b0fe))

## [0.4.6](https://github.com/tuna-os/remora/compare/v0.4.5...v0.4.6) (2026-10-17)


### Bug Fixes

* reconcile runtime units with manifest ([#172](https://github.com/tuna-os/remora/issues/172)) ([75363dd](https://github.com/tuna-os/remora/commit/75363dda2b3d1b6a62f47437132d3ec20a23e44b))


### Documentation

* guide donors to online round-robin Hive setup (#contribute) ([af33b4f](https://github.com/tuna-os/remora/commit/af33b4fe0db792ed1bb3421d1f92170e17c2b2f6))

## [0.4.5](https://github.com/tuna-os/remora/compare/v0.4.4...v0.4.5) (2026-10-16)


### Documentation

* add realistic configuration examples to README ([#170](https://github.com/tuna-os/remora/issues/170)) ([f3f5d36](https://github.com/tuna-os/remora/commit/f3f5d36e41cfa8aedb5fc12b00460f918f0eddaa))

## [0.4.4](https://github.com/tuna-os/remora/compare/v0.4.3...v0.4.4) (2026-10-15)


### Bug Fixes

* **dnf:** copy up rpmdb before install to avoid overlayfs corruption ([#98](https://github.com/tuna-os/remora/issues/98)) ([3e6d4c2](https://github.com/tuna-os/remora/commit/3e6d4c2a63bae7ae83fcf0a1eb58e6979749c009))

## [0.4.3](https://github.com/tuna-os/remora/compare/v0.4.2...v0.4.3) (2026-09-23)


### Documentation

* propose remora support for bases without a package manager ([#82](https://github.com/tuna-os/remora/issues/82)) ([cae1143](https://github.com/tuna-os/remora/commit/cae1143f05e47379388cc961b2c916800c931029))
* scope --apply and --soft-reboot to the apply command ([#65](https://github.com/tuna-os/remora/issues/65)) ([a9a1bbe](https://github.com/tuna-os/remora/commit/a9a1bbe435bd25ea1a5ae54f6e3c0197a8e054c8))

## [0.4.2](https://github.com/tuna-os/remora/compare/v0.4.1...v0.4.2) (2026-09-03)


### Documentation

* add a rollback runbook for a bad remora build ([#62](https://github.com/tuna-os/remora/issues/62)) ([52684a6](https://github.com/tuna-os/remora/commit/52684a64c796819ece832f55a72f193a8b6bb99c))

## [0.4.1](https://github.com/tuna-os/remora/compare/v0.4.0...v0.4.1) (2026-09-02)


### Documentation

* **roadmap:** plan the post-v0.4.0 period ([#58](https://github.com/tuna-os/remora/issues/58)) ([48bc05f](https://github.com/tuna-os/remora/commit/48bc05fd94dc7bdfeb72b05dbcfefe7d6b8e15f7))

## [0.4.0](https://github.com/tuna-os/remora/compare/v0.3.0...v0.4.0) (2026-08-26)


### Features

* resolve packages to a lockfile so unchanged rebuilds are free ([#34](https://github.com/tuna-os/remora/issues/34)) ([4634ee0](https://github.com/tuna-os/remora/commit/4634ee0f8610c3517ccf3ea1e5293143339d4a3b))


### Documentation

* **roadmap:** correct why the TunaOS image pin matters ([#32](https://github.com/tuna-os/remora/issues/32)) ([c671951](https://github.com/tuna-os/remora/commit/c67195198c123bae7593378079765877b3d00c5d))

## [0.3.0](https://github.com/tuna-os/remora/compare/v0.2.0...v0.3.0) (2026-08-26)


### Features

* **ci:** move CI to Blacksmith runner ([e62d069](https://github.com/tuna-os/remora/commit/e62d06964eb750b891f189a066effb9f9e059f7d))
* digest-pinned bases, reproducible layers, and no-op rebuilds ([#27](https://github.com/tuna-os/remora/issues/27)) ([404ebaf](https://github.com/tuna-os/remora/commit/404ebaf291b4bbb95f66eb1b36f4bcb20be07cc6))
