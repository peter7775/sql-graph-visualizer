# PostgreSQL Support (Issue #7)

SQL Graph Visualizer nyní podporuje PostgreSQL databáze vedle původní MySQL podpory. Tato funkce umožňuje připojení k PostgreSQL databázím a transformaci jejich dat do Neo4j graf databáze.

## Nové funkce

### Multi-databázová architektura
- **Podpora PostgreSQL i MySQL** z jednoho nástroje
- **Abstraktní databázová vrstva** pro snadné přidání dalších databází
- **Jednotné rozhraní** pro všechny typy databází
- **Zachování zpětné kompatibility** se stávajícími MySQL konfiguracemi

### PostgreSQL-specifické funkce
- **Pokročilá SSL konfigurace** s podporou různých módů
- **Schema-aware připojení** s podporou PostgreSQL schémat
- **Optimalizované dotazy** využívající PostgreSQL information_schema
- **Podpora PostgreSQL-specifických typů dat**

## Požadavky

### Systémové požadavky
- Go 1.19+
- PostgreSQL 10+ (doporučuje se 13+)
- Neo4j 4.0+

### Go závislosti
```bash
go get github.com/lib/pq  # PostgreSQL driver
```

## Konfigurace

### Nová multi-databázová konfigurace

```yaml
# Nová konfigurace s výběrem databáze
database:
  type: "postgresql"  # nebo "mysql"
  
  postgresql:
    host: "localhost"
    port: 5432
    user: "postgres"
    password: "password"
    database: "sample_db"
    schema: "public"  # PostgreSQL-specifické
    
    # PostgreSQL SSL konfigurace
    ssl:
      mode: "prefer"  # disable, allow, prefer, require, verify-ca, verify-full
      cert_file: "/path/to/client-cert.pem"
      key_file: "/path/to/client-key.pem"
      ca_file: "/path/to/ca-cert.pem"
    
    # PostgreSQL-specifické nastavení
    application_name: "sql-graph-visualizer"
    statement_timeout: 30
    search_path: ["public", "analytics"]
    
    # Standardní nastavení
    connection_mode: existing
    data_filtering: { ... }
    security: { ... }
```

### Zachování zpětné kompatibility

Stávající MySQL konfigurace fungují bez změn:

```yaml
# Stará konfigurace (stále funguje)
mysql:
  host: "localhost"
  port: 3306
  user: "user"
  password: "password"
  database: "sakila"
```

## Testování s veřejnými databázemi

### 1. Chinook Sample Database

Nejlepší pro testování - obsahuje komplexní relační strukturu:

```bash
# Stažení a instalace
wget https://github.com/lerocha/chinook-database/raw/master/ChinookDatabase/DataSources/Chinook_PostgreSql.sql
createdb chinook
psql -d chinook -f Chinook_PostgreSql.sql
```

**Tabulky v Chinook databázi:**
- `artist` (275 umělců)
- `album` (347 alb)
- `track` (3,503 skladeb)
- `customer` (59 zákazníků)
- `invoice` + `invoiceline` (2,240 faktur, 2,240 položek)
- `employee` (8 zaměstnanců)
- `genre` (25 žánrů)
- `playlist` + `playlisttrack` (playlisty a jejich skladby)

### 2. Cloud PostgreSQL služby

**Bezplatné služby pro testování:**
- [Neon.tech](https://neon.tech/) - 3GB zdarma
- [ElephantSQL](https://www.elephantsql.com/) - 20MB zdarma
- [Supabase](https://supabase.com/) - 500MB zdarma

### 3. Ukázkový test s Chinook

```yaml
# examples/postgresql-chinook-test.yaml
database:
  type: "postgresql"
  postgresql:
    host: "localhost"
    port: 5432
    user: "postgres"
    password: "password"
    database: "chinook"
    
    data_filtering:
      table_whitelist: [
        "album", "artist", "customer", "employee", 
        "genre", "track", "playlist"
      ]
      row_limit_per_table: 0  # Žádný limit - Chinook je malá
```

## Architektura

### Databázová abstrakce

```go
// Jednotné rozhraní pro všechny databáze
type DatabaseRepository interface {
    Connect(ctx context.Context, config DatabaseConfig) (*sql.DB, error)
    GetTables(ctx context.Context, filters DataFilteringConfig) ([]string, error)
    GetColumns(ctx context.Context, tableName string) ([]*ColumnInfo, error)
    // ... další metody
}

// Factory pro vytváření správných implementací
factory := factories.NewDatabaseRepositoryFactory()
repo, err := factory.CreateRepository(models.DatabaseTypePostgreSQL)
```

### Konfigurace interface

```go
type DatabaseConfig interface {
    GetDatabaseType() DatabaseType
    GetHost() string
    GetPort() int
    GetUsername() string
    GetPassword() string
    GetDatabase() string
    // ... další gettery
}
```

## Příklady použití

### 1. Základní PostgreSQL test

```bash
# Nastavení environment proměnných
export POSTGRES_HOST=localhost
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=your_password
export POSTGRES_DB=chinook

# Spuštění testu
go run cmd/postgresql_test/main.go
```

### 2. Transformace Chinook → Neo4j

```bash
./sql-graph-visualizer -config examples/postgresql-chinook-test.yaml
```

### 3. Programatické použití

```go
// Vytvoření PostgreSQL konfigurace
config := &models.PostgreSQLConfig{
    Host:     "localhost",
    Port:     5432,
    User:     "postgres",
    Password: "password", 
    Database: "chinook",
    Schema:   "public",
}

// Vytvoření repository
factory := factories.NewDatabaseRepositoryFactory()
repo, err := factory.CreateRepository(models.DatabaseTypePostgreSQL)

// Připojení a použití
db, err := repo.Connect(ctx, config)
tables, err := repo.GetTables(ctx, filters)
```

## Bezpečnost

### SSL konfigurace pro produkční použití

```yaml
postgresql:
  ssl:
    mode: "require"  # Vynutit SSL pro produkci
    ca_file: "/path/to/ca-cert.pem"
    cert_file: "/path/to/client-cert.pem"
    key_file: "/path/to/client-key.pem"
    insecure_skip_verify: false
```

### Security nastavení

```yaml
security:
  read_only: true
  connection_timeout: 30
  query_timeout: 60
  max_connections: 5
  allowed_hosts: ["your-db-host.com"]
  forbidden_patterns: ["DROP", "DELETE", "UPDATE", "INSERT"]
```

## Výkon a optimalizace

### PostgreSQL-specifické optimalizace

1. **Rychlé odhady řádků** pomocí `pg_stat_user_tables`
2. **Schema-aware dotazy** využívající `information_schema`
3. **Efektivní introspekce** foreign keys a indexů
4. **Batch processing** optimalizované pro PostgreSQL

### Doporučené nastavení pro velké databáze

```yaml
data_filtering:
  row_limit_per_table: 10000
  query_timeout: 120
  where_conditions:
    large_table: "created_at >= CURRENT_DATE - INTERVAL '1 year'"

security:
  max_connections: 3  # Konzervativní pro cloud služby
  
neo4j:
  batch_processing:
    batch_size: 500
    commit_frequency: 2500
```

## Testování a validace

### Automatické testy

```bash
# Kompilace všech modulů
go build ./...

# Unit testy
go test ./internal/...

# Integrační test s PostgreSQL
go run cmd/postgresql_test/main.go
```

### Manuální validace

1. **Připojení**: Test konektivity k PostgreSQL
2. **Schema discovery**: Načtení seznamu tabulek a sloupců
3. **Data extraction**: Extrakce dat s filtry
4. **Neo4j transformace**: Import do graf databáze
5. **Ověření výsledků**: Kontrola v Neo4j Browser

## Roadmapa

### Budoucí rozšíření
- 🔄 SQLite podpora
- 🔄 Microsoft SQL Server podpora
- 🔄 Oracle Database podpora
- 🔄 Pokročilé PostgreSQL funkce (arrays, JSON, custom types)
- 🔄 Pokročilé optimalizace pro cloud databáze

## 📞 Podpora a troubleshooting

### Časté problémy
1. **SSL connection failed**
   ```yaml
   ssl:
     mode: "disable"  # Pro lokální testování
   ```

2. **Permission denied**
   - Zkontrolujte `pg_hba.conf`
   - Přidejte `read_only: true` do security nastavení

3. **Connection timeout**
   - Zvyšte `connection_timeout` hodnotu
   - Zkontrolujte firewall nastavení

### Logy a debugging

```bash
# Zapnutí debug logů
export LOG_LEVEL=debug
go run cmd/postgresql_test/main.go
```

---

**Implementováno:** Issue #7 - PostgreSQL podpora  
**Autor:** Petr Miroslav Stepanek  
**Verze:** 1.1.0  
**Datum:** 2025-01-06
