# Database Query Monitor

Execute a database query and validate that the result meets expected criteria.

## Use Cases

- **Data Freshness**: Check that data sync jobs have updated recently
- **Application State**: Verify application has expected data (e.g., unprocessed jobs, user count)
- **Database Health**: Simple database connectivity test with meaningful query
- **Business Logic Validation**: Monitor application-specific conditions via SQL

## Configuration

```yaml
monitors:
  - id: database-query-freshness
    name: Database Query Freshness
    type: database_query
    interval: 5m
    timeout: 15s
    config:
      engine: postgres
      host: postgres.internal.example
      port: 5432
      user: monitor
      password: replace-me
      database: app
      ssl_mode: disable
      query: "SELECT COUNT(*) FROM data_sync WHERE updated_at > NOW() - INTERVAL '5 minutes'"
      expected_value: "1"
      comparison: gt
```

## Configuration Fields

| Field | Required | Default | Description |
|-------|----------|---------|-------------|
| `engine` | ✓ | — | Database engine: `postgres` or `mysql` |
| `host` | ✗* | localhost (postgres), localhost (mysql) | Database hostname or IP address |
| `port` | ✗* | 5432 (postgres), 3306 (mysql) | Database port |
| `user` | ✗* | — | Database user |
| `password` | ✗* | — | Database password |
| `database` | ✗* | — | Database name |
| `ssl_mode` | ✗ | disable | PostgreSQL SSL mode: `disable`, `require`, `verify-ca`, `verify-full` (PostgreSQL only) |
| `connection_string` | ✗* | — | Full connection string (alternative to individual host/port/user/password fields) |
| `query` | ✓ | — | SQL query to execute (must return a single value) |
| `expected_value` | ✓ | — | Expected value for comparison (numeric or string) |
| `comparison` | ✗ | eq | Comparison operator: `eq`, `gt`, `lt`, `gte`, `lte` |

*Note: Either use `connection_string` OR individual connection fields (`host`, `port`, `user`, `password`, `database`).

## Supported Engines

### PostgreSQL

Connection via individual fields:
```yaml
config:
  engine: postgres
  host: db.internal.example
  port: 5432
  user: monitor
  password: secret
  database: myapp
  ssl_mode: require
  query: "SELECT 1"
  expected_value: "1"
```

Connection via connection string:
```yaml
config:
  engine: postgres
  connection_string: "postgresql://monitor:secret@db.internal.example:5432/myapp?sslmode=require"
  query: "SELECT 1"
  expected_value: "1"
```

### MySQL

Connection via individual fields:
```yaml
config:
  engine: mysql
  host: db.internal.example
  port: 3306
  user: monitor
  password: secret
  database: myapp
  query: "SELECT 1"
  expected_value: "1"
```

Connection via connection string:
```yaml
config:
  engine: mysql
  connection_string: "monitor:secret@tcp(db.internal.example:3306)/myapp"
  query: "SELECT 1"
  expected_value: "1"
```

## Query Guidelines

- Queries must return a single value (scalar result)
- The query result is compared as a string unless the comparison operator is numeric (in which case numeric comparison is used)
- Use `COUNT(*)`, `COUNT(id)`, or similar aggregate functions for cardinality checks
- Use `CAST` or `::` (PostgreSQL) to ensure expected data types

## Comparison Operators

- **eq**: Value equals expected (default)
- **gt**: Value greater than expected
- **lt**: Value less than expected
- **gte**: Value greater than or equal to expected
- **lte**: Value less than or equal to expected

## Examples

### Check Data Sync Freshness (PostgreSQL)

```yaml
monitors:
  - id: data-sync-freshness
    name: Data Sync Freshness
    type: database_query
    groups: [Data]
    interval: 10m
    timeout: 15s
    config:
      engine: postgres
      host: db.internal.example
      port: 5432
      user: monitor
      password: ${DB_MONITOR_PASSWORD}
      database: production
      ssl_mode: require
      query: |
        SELECT COUNT(*) FROM data_sync
        WHERE updated_at > NOW() - INTERVAL '10 minutes'
        AND status = 'success'
      expected_value: "1"
      comparison: gt
```

### Check Pending Jobs (MySQL)

```yaml
monitors:
  - id: job-queue-depth
    name: Job Queue Depth
    type: database_query
    groups: [Jobs]
    interval: 5m
    timeout: 10s
    config:
      engine: mysql
      connection_string: "monitor:${DB_PASSWORD}@tcp(db.internal.example:3306)/queue"
      query: "SELECT COUNT(*) FROM jobs WHERE status = 'pending' LIMIT 1"
      expected_value: "100"
      comparison: lt
```

This alerts if there are more than 100 pending jobs (assuming you want fewer than 100).

### Check Active User Sessions

```yaml
monitors:
  - id: active-sessions
    name: Active User Sessions
    type: database_query
    groups: [Application]
    interval: 5m
    timeout: 10s
    config:
      engine: postgres
      host: db.internal.example
      port: 5432
      user: monitor
      password: ${DB_MONITOR_PASSWORD}
      database: app
      ssl_mode: require
      query: "SELECT COUNT(*) FROM user_sessions WHERE logged_out_at IS NULL"
      expected_value: "0"
      comparison: gt
```

This ensures at least one user is logged in during business hours.

## Metadata

The monitor captures the following metadata in check results:

```json
{
  "engine": "postgres",
  "value": "42",
  "expected": "10",
  "comparison": "gt"
}
```

## Error Handling

- **Connection failed**: Check database connectivity, host/port, user/password
- **Query failed**: Verify SQL syntax and that the user has query permissions
- **Query returned no rows**: Ensure the query can handle empty result sets gracefully
- **Query returned multiple values**: Use `LIMIT 1` or aggregate functions to return a single value

## Security Considerations

- Use environment variables or secrets management for database passwords
- Create a dedicated read-only database user for monitoring
- Ensure database connections use appropriate SSL modes in production
- Avoid queries that might lock tables or consume significant resources
- Use connection pooling for production deployments
- Limit query execution time with the `timeout` setting

## Performance Tips

- Index columns used in WHERE clauses for faster query execution
- Keep queries simple and efficient
- Use appropriate `timeout` values (15-30 seconds for typical queries)
- Consider caching query results at the application level if needed
- Monitor long-running queries to optimize them

## See Also

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [MySQL Documentation](https://dev.mysql.com/doc/)
- [SQL Best Practices](https://en.wikipedia.org/wiki/SQL)
