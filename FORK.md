# Viant distribution

This fork retains the upstream license and module path. File-backed SQL backend
and catalog caches now have explicit ownership: concurrent outer `sql.DB` handles
share one backend, and closing the last owner closes that backend and removes its
cache entries. The database file is preserved and can be reopened normally.

Legacy direct `Driver.Open` connections release their connector on close. Failed
connection hooks return borrowed connections before reporting the failure.
Private in-memory connectors keep their existing isolated ownership semantics.

Validate with:

```sh
go test .
go test -race . -run 'Test(FileLifecycle|RepeatedFileLifecycle|LegacyDriverOpen|FailedLegacyOpen)'
```

Consumers using the upstream import path can pin a published fork commit through
a remote Go module replacement. No local path replacement is needed in releases.
