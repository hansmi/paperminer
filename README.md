# Amend Paperless documents with extracted information

[![Latest release](https://img.shields.io/github/v/release/hansmi/paperminer)][releases]
[![CI workflow](https://github.com/hansmi/paperminer/actions/workflows/ci.yaml/badge.svg)](https://github.com/hansmi/paperminer/actions/workflows/ci.yaml)
[![Go reference](https://pkg.go.dev/badge/github.com/hansmi/paperminer.svg)](https://pkg.go.dev/github.com/hansmi/paperminer)

Paperminer is a system for amending documents stored in
[Paperless-ngx][paperless] with additional information ("facts") extracted from
the documents themselves or other sources.

The [`hansmi/dossier` package][dossier] is called to parse PDF documents (other
formats could be implemented).

The Go programming language's [`plugin` package][gopkgplugin] comes with
a number of caveats which make it unsuitable. Compile-time plugins via the
[`hansmi/staticplug` package][staticplug] are used instead. It's therefore
necessary to set up your own build. An example for a program with a plugin can
be found in the [`example/myminer` directory](./example/myminer).

Plugins may use [dossier sketches][dossiersketch] to look for specific regular
expressions at absolute or relative positions on pages. The [`sketchfacts`
package](./pkg/sketchfacts/) is often sufficient even though it ignores pages
beyond the first. Custom logic can produce document facts from the findings.

Plugins may also extract arbitrary document pages and implement their own data
extraction. External APIs may also be involved.

Normalizing extracted text before parsing it further is generally recommended,
not just for date and time: remove extraneous whitespace and separators, etc.
Regular expressions should also be written to be flexible where possible.
OCR-derived text is often not exactly the same as the original.

Useful packages for writing document facters:

* [`hansmi/zyt`][zyt]: Parse language/locale-specific date and time formats.
* [`hansmi/aurum`][aurum]: Golden tests. Used for generic document facter tests
  by the [`factertest` package](./pkg/factertest).

[aurum]: https://github.com/hansmi/aurum/
[dossier]: https://github.com/hansmi/dossier/
[dossiersketch]: https://github.com/hansmi/dossier/#sketches
[gopkgplugin]: https://pkg.go.dev/plugin@go1.22.0
[paperless]: https://docs.paperless-ngx.com/
[releases]: https://github.com/hansmi/paperminer/releases/latest
[staticplug]: https://github.com/hansmi/staticplug/
[zyt]: https://github.com/hansmi/zyt/

<!-- vim: set sw=2 sts=2 et : -->
