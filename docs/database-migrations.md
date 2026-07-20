# Database Migrations

Enduro uses [golang-migrate] to manage its MySQL schema. Migration files are
embedded in the Enduro binary, and the current migration version is stored in
the `schema_migrations` table.

## Choose a migration mode

By default, `database.autoMigrate` is `true` and Enduro applies pending up
migrations automatically when it starts. This is the recommended mode for
most installations and preserves the behavior of earlier Enduro releases.

Set `database.autoMigrate` to `false` to manage the schema separately from the
Enduro process:

```toml
[database]
dsn = "enduro:enduro123@tcp(127.0.0.1:7450)/enduro"
autoMigrate = false
```

Every Enduro instance connected to the database must use the same setting.
[Install the golang-migrate CLI] with the `file` source and `mysql` database
drivers. When automatic migrations are disabled, apply the migrations for a
new release before starting that release:

```sh
MIGRATIONS=./internal/db/migrations
DATABASE_URL='mysql://enduro:enduro123@tcp(127.0.0.1:7450)/enduro'

migrate -path "$MIGRATIONS" -database "$DATABASE_URL" up
migrate -path "$MIGRATIONS" -database "$DATABASE_URL" version
```

The standalone CLI needs migration files on disk; it cannot read the files
embedded in the Enduro binary. Enduro release archives therefore include the
`internal/db/migrations` directory.

## Roll back migrations

Enduro never rolls back migrations automatically, regardless of the migration
mode. Use the `migrate` command from golang-migrate before installing an older
Enduro release.

Rolling back a migration can discard data. Before continuing:

1. Stop every Enduro instance connected to the database. A running instance
   may write data during the rollback or apply the migrations again.
2. Back up the Enduro database and verify that the backup can be restored.
3. Obtain the `internal/db/migrations` directory from the currently installed
   release. It contains the down migrations needed to leave that release. The
   directory is included in Enduro release archives; for older archives that
   do not include it, download the matching source archive from the
   [release page].
4. Inspect `internal/db/migrations` in the target release. The numeric prefix
   of its latest `.up.sql` file is the target migration version.
5. [Install the golang-migrate CLI] with the `file` source and `mysql` database
   drivers.

The golang-migrate database URL differs slightly from the `database.dsn` value
in `enduro.toml`: prefix the Enduro DSN with `mysql://`. URL-encode reserved
characters in the username or password.

For example, the following commands migrate the development database from
Enduro v0.22.0, whose schema is at `1580747285`, to the schema used by v0.21.0,
`1579535702`:

```sh
MIGRATIONS=./internal/db/migrations
DATABASE_URL='mysql://enduro:enduro123@tcp(127.0.0.1:7450)/enduro'

migrate -path "$MIGRATIONS" -database "$DATABASE_URL" version
migrate -path "$MIGRATIONS" -database "$DATABASE_URL" goto 1579535702
migrate -path "$MIGRATIONS" -database "$DATABASE_URL" version
```

The `goto` command applies every required down migration and leaves the
database at the requested version. After the final `version` command reports
the expected target, install the older Enduro release and restart Enduro.

Always use the migration directory from the release being left, not the target
release: the target may not contain down migrations introduced by later
releases. Do not use `force` as a rollback command. It changes the recorded
version without running any migration SQL. If a migration fails or the
database is reported as dirty, stop and restore the backup or diagnose the
failed statement before proceeding.

[golang-migrate]: https://github.com/golang-migrate/migrate
[Install the golang-migrate CLI]: https://github.com/golang-migrate/migrate/tree/master/cmd/migrate
[release page]: https://github.com/artefactual-labs/enduro/releases
