package main

import (
	"strings"
	"testing"

	"github.com/thecodearcher/limen"
)

// The generator interpolates every identifier with %s, so table, column, index,
// constraint and referenced-table names all reach the emitted DDL unquoted.
//
// `user` and `order` are reserved words in both PostgreSQL and MySQL, and they
// are exactly the names a caller reaches for through WithUserTableName and
// friends — the default names (`users`, `sessions`, ...) happen to be safe,
// which is why this survives untested.
//
// Quoting is not cosmetic here: adapters/sql quotes every identifier at
// runtime (its quoteIdent doubles embedded quote chars), so DDL emitted
// unquoted disagrees with the queries that will later run against it. A
// reserved word fails loudly at migration time; a name needing case
// preservation is worse, since unquoted DDL folds to lowercase while the
// adapter's quoted query does not, and the mismatch only surfaces at runtime.

func newTestGenerator(t *testing.T, driver Driver) *sqlMigrationGenerator {
	t.Helper()

	gen, err := newSQLMigrationGenerator(driver, &cliConfig{})
	if err != nil {
		t.Fatalf("newSQLMigrationGenerator() error = %v", err)
	}

	return gen
}

// reservedWordSchema is a minimal schema whose table and one of whose columns
// are reserved words.
func reservedWordSchema() *limen.SchemaDefinition {
	return &limen.SchemaDefinition{
		TableName: "user",
		Columns: []limen.ColumnDefinition{
			{
				Name:         "id",
				LogicalField: limen.SchemaIDField,
				Type:         limen.ColumnTypeInt64,
				IsPrimaryKey: true,
			},
			{
				Name:         "order",
				LogicalField: "order",
				Type:         limen.ColumnTypeString,
			},
		},
		Indexes: []limen.IndexDefinition{
			{
				Name:    "idx_user_order",
				Columns: []limen.SchemaField{"order"},
				Unique:  true,
			},
		},
	}
}

func assertContainsAll(t *testing.T, got string, want []string) {
	t.Helper()

	for _, w := range want {
		if !strings.Contains(got, w) {
			t.Errorf("generated SQL is missing %q\ngot:\n%s", w, got)
		}
	}
}

func TestGenerateCreateTableQuotesIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		driver Driver
		want   []string
	}{
		{
			name:   "postgres",
			driver: NewPostgresDriver(),
			want: []string{
				`CREATE TABLE IF NOT EXISTS "user" (`,
				`"id" BIGINT`,
				`"order" VARCHAR(255) NOT NULL`,
				`PRIMARY KEY ("id")`,
				`CREATE UNIQUE INDEX "idx_user_order" ON "user" ("order");`,
			},
		},
		{
			name:   "mysql",
			driver: NewMySQLDriver(),
			want: []string{
				"CREATE TABLE IF NOT EXISTS `user` (",
				"`id` BIGINT",
				"PRIMARY KEY (`id`)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := newTestGenerator(t, tt.driver).generateCreateTable(reservedWordSchema())
			if err != nil {
				t.Fatalf("generateCreateTable() error = %v", err)
			}

			assertContainsAll(t, got, tt.want)
		})
	}
}

func TestGenerateDownMigrationQuotesTableName(t *testing.T) {
	t.Parallel()

	got, err := newTestGenerator(t, NewPostgresDriver()).generateDownMigration(reservedWordSchema(), nil)
	if err != nil {
		t.Fatalf("generateDownMigration() error = %v", err)
	}

	assertContainsAll(t, got, []string{`DROP TABLE IF EXISTS "user";`})
}

func TestGenerateCreateIndexStatementQuotesIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		unique bool
		want   string
	}{
		{
			name:   "unique index",
			unique: true,
			want:   `CREATE UNIQUE INDEX "idx_user_order" ON "user" ("order");`,
		},
		{
			name:   "plain index",
			unique: false,
			want:   `CREATE INDEX "idx_user_order" ON "user" ("order");`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			idx := limen.IndexDefinition{
				Name:    "idx_user_order",
				Columns: []limen.SchemaField{"order"},
				Unique:  tt.unique,
			}

			got := newTestGenerator(t, NewPostgresDriver()).generateCreateIndexStatement(&idx, "user")

			assertContainsAll(t, got, []string{tt.want})
		})
	}
}

func TestGenerateForeignKeyStatementQuotesIdentifiers(t *testing.T) {
	t.Parallel()

	fk := limen.ForeignKeyDefinition{
		Name:             "fk_session_user",
		Column:           "user_id",
		ReferencedSchema: "user",
		ReferencedField:  "id",
		OnDelete:         "CASCADE",
	}

	tests := []struct {
		name       string
		alterTable bool
		want       []string
	}{
		{
			name:       "inside CREATE TABLE",
			alterTable: false,
			want: []string{
				`CONSTRAINT "fk_session_user" FOREIGN KEY ("user_id") REFERENCES "user" ("id")`,
			},
		},
		{
			name:       "inside ALTER TABLE",
			alterTable: true,
			want: []string{
				`ADD CONSTRAINT "fk_session_user" FOREIGN KEY ("user_id") REFERENCES "user" ("id")`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := newTestGenerator(t, NewPostgresDriver()).generateForeignKeyStatement(&fk, tt.alterTable)

			assertContainsAll(t, got, tt.want)
		})
	}
}

// reservedWordDiff adds a reserved-word column, an index and a foreign key to
// an existing table, which is the path taken when the table already exists.
func reservedWordDiff() *schemaDiff {
	return &schemaDiff{
		AddedColumns: []limen.ColumnDefinition{
			{Name: "order", LogicalField: "order", Type: limen.ColumnTypeString},
		},
		AddedIndexes: []limen.IndexDefinition{
			{Name: "idx_user_order", Columns: []limen.SchemaField{"order"}, Unique: true},
		},
		AddedForeignKeys: []limen.ForeignKeyDefinition{
			{
				Name:             "fk_user_order",
				Column:           "order",
				ReferencedSchema: "order",
				ReferencedField:  "id",
			},
		},
	}
}

func TestGenerateUpMigrationForExistingTableQuotesIdentifiers(t *testing.T) {
	t.Parallel()

	got, err := newTestGenerator(t, NewPostgresDriver()).
		generateUpMigration(reservedWordSchema(), reservedWordDiff())
	if err != nil {
		t.Fatalf("generateUpMigration() error = %v", err)
	}

	assertContainsAll(t, got, []string{
		`ALTER TABLE "user"`,
		`ADD COLUMN "order" VARCHAR(255) NOT NULL`,
		`ADD CONSTRAINT "fk_user_order" FOREIGN KEY ("order") REFERENCES "order" ("id")`,
		`CREATE UNIQUE INDEX "idx_user_order" ON "user" ("order");`,
	})
}

func TestGenerateDownMigrationForExistingTableQuotesIdentifiers(t *testing.T) {
	t.Parallel()

	got, err := newTestGenerator(t, NewPostgresDriver()).
		generateDownMigration(reservedWordSchema(), reservedWordDiff())
	if err != nil {
		t.Fatalf("generateDownMigration() error = %v", err)
	}

	assertContainsAll(t, got, []string{
		`DROP INDEX IF EXISTS "idx_user_order"`,
		`ALTER TABLE "user"`,
		`DROP CONSTRAINT "fk_user_order"`,
		`DROP COLUMN "order"`,
	})
}

// A down migration that drops an index and alters the table emits both, and
// the two must not run together into one statement. The driver's DropIndexSQL
// carries no terminator of its own, unlike generateCreateIndexStatement on the
// up path, so the caller has to supply it.
func TestGenerateDownMigrationTerminatesEachStatement(t *testing.T) {
	t.Parallel()

	got, err := newTestGenerator(t, NewPostgresDriver()).
		generateDownMigration(reservedWordSchema(), reservedWordDiff())
	if err != nil {
		t.Fatalf("generateDownMigration() error = %v", err)
	}

	assertContainsAll(t, got, []string{`DROP INDEX IF EXISTS "idx_user_order";`})

	for _, stmt := range strings.Split(got, ";") {
		if strings.Contains(stmt, "DROP INDEX") && strings.Contains(stmt, "ALTER TABLE") {
			t.Errorf("DROP INDEX and ALTER TABLE share one statement\ngot:\n%s", got)
		}
	}
}

func TestDriverDropStatementsQuoteIdentifiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  string
		want string
	}{
		{
			name: "postgres drop index",
			got:  NewPostgresDriver().DropIndexSQL("user", "idx_user_order"),
			want: `DROP INDEX IF EXISTS "idx_user_order"`,
		},
		{
			name: "postgres drop foreign key",
			got:  NewPostgresDriver().DropForeignKeySQL("user", "fk_user_order"),
			want: `DROP CONSTRAINT "fk_user_order"`,
		},
		{
			name: "postgres drop column",
			got:  NewPostgresDriver().DropColumnSQL("user", "order"),
			want: `DROP COLUMN "order"`,
		},
		{
			name: "mysql drop index",
			got:  NewMySQLDriver().DropIndexSQL("user", "idx_user_order"),
			want: "DROP INDEX `idx_user_order` ON `user`",
		},
		{
			name: "mysql drop column",
			got:  NewMySQLDriver().DropColumnSQL("user", "order"),
			want: "DROP COLUMN `order`",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if !strings.Contains(tt.got, tt.want) {
				t.Errorf("got %q, want it to contain %q", tt.got, tt.want)
			}
		})
	}
}
