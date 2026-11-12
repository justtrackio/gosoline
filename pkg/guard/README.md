# Conditional policy writes

Use `GetPolicySnapshot` followed by `UpdatePolicyIfUnchanged` or
`DeletePolicyIfUnchanged` for read/change/write operations. The snapshot retains
the exact stored JSON, including its original formatting. Policy IDs cannot change.

A missing policy returns a nil snapshot. A conditional write returns false when
the row changed or disappeared. Updates that change no stored values can also
return false. Check errors before treating false as a conflict. Only retry after
reading a new snapshot and recomputing the intended change; do not retry a stale
replacement unchanged.

Snapshot reads bypass the request cache. Conditional writes clear that cache,
including on conflicts, so authorization checks on the next attempt read current
policies. The caller remains responsible for authorization.

The existing unconditional write methods remain available for compatibility.
Editors must use conditional writes too if stale edits must not overwrite newer
changes. Custom implementations of Guard or Manager must implement the new
snapshot and conditional-write methods.
