Hidden Path Tests
=================

The tests in this directory test the examples from the [path
README](../README.md). They live in this directory because they're not package
examples, and if they lived in the `path` directory they would be picked up by
[godoc] or [pkgsite] and displayed in the main path documentation, which would
be rather confusing.

  [godoc]: https://pkg.go.dev/golang.org/x/tools/cmd/godoc
  [pkgsite]: https://pkg.go.dev/golang.org/x/pkgsite/cmd/pkgsite
