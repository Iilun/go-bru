# go-bru, a bru parsing tool

This package is meant to ease the parsing of [bru files](https://docs.usebruno.com/bru-language-design.html), the language to store [bruno](https://www.usebruno.com/) content.

As it is meant to allow manipulation of files without knowing all contents beforehand, it does not automatically map to a struct, but rather sends back each block of the bru file in an array.

Note that the bru documentation does not seem to be up to date so some block types may be missing.