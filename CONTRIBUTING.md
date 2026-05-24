# Contributing

## Tenant Isolation Checklist

Every new host-scoped backend package must add a tenant isolation test before it is considered complete.

- Service/repository functions must take `hostID` as the first non-`ctx` parameter.
- SQL for host-owned rows must filter by `host_user_id = ?`.
- SQL for nested rows must join through the owning `groups.host_user_id`.
- Cross-tenant reads and writes must return `404` with title `Not found`, matching a truly missing row.
- Tests must cover at least one cross-tenant read or write for each new host-scoped endpoint.
