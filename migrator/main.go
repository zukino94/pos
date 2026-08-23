// Migrator skema database POS.
//
// Membaca koneksi dari config.json lalu:
//   - migrate: menjalankan skema SQL (mysql-scheme/pos.sql) ke server MySQL
//   - seed:    memasukkan data CSV dari folder seed ke tabel dengan nama
//     yang sama dengan nama file (kasir.csv -> tabel kasir)
//
// Pemakaian:
//
//	go run main.go migrate
//	go run main.go seed
package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
)

// config berisi koneksi MySQL yang dibaca dari config.json.
type config struct {
	MySQLHost     string `json:"mysql_host"`
	MySQLPort     string `json:"mysql_port"`
	MySQLUsername string `json:"mysql_username"`
	MySQLPassword string `json:"mysql_password"`
	MySQLDatabase string `json:"mysql_database"`
}

// identPattern mencocokkan nama identifier SQL, dengan atau tanpa backtick.
const identPattern = "`?([a-zA-Z0-9_]+)`?"

var (
	reCreateDatabase = regexp.MustCompile(`(?i)^CREATE\s+DATABASE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + identPattern)
	reCreateTable    = regexp.MustCompile(`(?i)^CREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?` + identPattern)
	reUse            = regexp.MustCompile(`(?i)^USE\s+` + identPattern)
	reIdent          = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
)

func main() {
	if len(os.Args) != 2 {
		usage()
	}

	var err error
	switch os.Args[1] {
	case "migrate":
		err = migrate()
	case "seed":
		err = seed()
	default:
		usage()
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "Gagal:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "Pemakaian:\n  go run main.go migrate\n  go run main.go seed")
	os.Exit(1)
}

// baseDir mengembalikan direktori tempat main.go berada sehingga
// `go run main.go` bisa dijalankan dari direktori mana pun.
func baseDir() string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Dir(thisFile)
}

func migrate() error {
	configPath := filepath.Join(baseDir(), "config.json")
	schemaPath := filepath.Clean(filepath.Join(baseDir(), "..", "mysql-scheme", "pos.sql"))

	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}
	fmt.Println("Konfigurasi  :", configPath)

	script, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("membaca skema: %w", err)
	}
	stmts := splitStatements(string(script))
	if len(stmts) == 0 {
		return fmt.Errorf("skema kosong: %s", schemaPath)
	}
	fmt.Println("Skema        :", schemaPath)

	addr := net.JoinHostPort(cfg.MySQLHost, cfg.MySQLPort)
	fmt.Printf("Koneksi      : %s@tcp(%s)\n\n", cfg.MySQLUsername, addr)

	db, err := sql.Open("mysql", dsn(cfg, ""))
	if err != nil {
		return fmt.Errorf("menyiapkan koneksi: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Semua statement dijalankan pada satu koneksi yang sama agar
	// `USE pos` berlaku juga untuk statement setelahnya.
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("menghubungkan ke %s: %w", addr, err)
	}
	defer conn.Close()

	var migratedDB string
	for _, stmt := range stmts {
		if _, err := conn.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("mengeksekusi %s: %w", describe(stmt), err)
		}
		if name, ok := usedDatabase(stmt); ok {
			migratedDB = name
		}
		fmt.Printf("  ok  %s\n", describe(stmt))
	}

	if cfg.MySQLDatabase != "" && migratedDB != "" && cfg.MySQLDatabase != migratedDB {
		fmt.Fprintf(os.Stderr, "\nPeringatan: mysql_database=%q di config.json, tetapi skema dibuat pada database `%s`.\n", cfg.MySQLDatabase, migratedDB)
	}

	fmt.Println("\nMigrasi selesai.")
	if migratedDB != "" {
		listTables(ctx, conn, migratedDB)
	}
	return nil
}

// seed memasukkan setiap file CSV di folder seed ke tabel dengan nama
// yang sama dengan nama filenya: kasir.csv -> tabel `kasir`.
func seed() error {
	configPath := filepath.Join(baseDir(), "config.json")
	seedDir := filepath.Join(baseDir(), "seed")

	cfg, err := loadConfig(configPath)
	if err != nil {
		return err
	}

	database := cfg.MySQLDatabase
	if database == "" {
		// Nama database yang dibuat oleh mysql-scheme/pos.sql.
		database = "pos"
	}

	files, err := filepath.Glob(filepath.Join(seedDir, "*.csv"))
	if err != nil {
		return fmt.Errorf("mencari file CSV: %w", err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		return fmt.Errorf("tidak ada file CSV di %s", seedDir)
	}

	fmt.Println("Konfigurasi  :", configPath)
	fmt.Println("Folder seed  :", seedDir)
	addr := net.JoinHostPort(cfg.MySQLHost, cfg.MySQLPort)
	fmt.Printf("Koneksi      : %s@tcp(%s)/%s\n\n", cfg.MySQLUsername, addr, database)

	db, err := sql.Open("mysql", dsn(cfg, database))
	if err != nil {
		return fmt.Errorf("menyiapkan koneksi: %w", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("menghubungkan ke %s: %w", addr, err)
	}

	for _, file := range files {
		table := strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
		n, err := seedTable(ctx, db, table, file)
		if err != nil {
			return err
		}
		fmt.Printf("  ok  %-18s %d baris dari %s\n", table, n, filepath.Base(file))
	}

	fmt.Println("\nSeed selesai.")
	return nil
}

// seedTable memasukkan isi satu file CSV ke tabel. Baris pertama CSV
// berisi nama kolom, setiap baris berikutnya menjadi satu INSERT.
// Semua baris dieksekusi dalam satu transaksi: jika ada yang gagal,
// seluruh isi file tersebut dibatalkan.
func seedTable(ctx context.Context, db *sql.DB, table, path string) (int, error) {
	name := filepath.Base(path)
	if !reIdent.MatchString(table) {
		return 0, fmt.Errorf("nama file %s bukan nama tabel yang valid", name)
	}

	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("membuka %s: %w", name, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	header, err := r.Read()
	if err != nil {
		return 0, fmt.Errorf("membaca header %s: %w", name, err)
	}

	quoted := make([]string, len(header))
	for i, h := range header {
		col := strings.TrimSpace(h)
		if !reIdent.MatchString(col) {
			return 0, fmt.Errorf("header %s: nama kolom %q tidak valid", name, h)
		}
		quoted[i] = "`" + col + "`"
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(header)), ",")
	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)", table, strings.Join(quoted, ", "), placeholders)

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		tx.Rollback()
		return 0, withHint(err)
	}
	defer stmt.Close()

	inserted := 0
	line := 1 // header ada di baris 1
	for {
		rec, err := r.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line++
		if err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("%s baris %d: %w", name, line, err)
		}
		if _, err := stmt.ExecContext(ctx, toAny(rec)...); err != nil {
			tx.Rollback()
			return 0, withHint(fmt.Errorf("menyisipkan %s baris %d: %w", name, line, err))
		}
		inserted++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return inserted, nil
}

// toAny mengubah record CSV menjadi argumen kueri.
func toAny(rec []string) []any {
	args := make([]any, len(rec))
	for i, v := range rec {
		args[i] = v
	}
	return args
}

// withHint menambahkan saran perbaikan untuk error MySQL yang umum.
func withHint(err error) error {
	var me *mysql.MySQLError
	if !errors.As(err, &me) {
		return err
	}
	switch me.Number {
	case 1049: // ER_BAD_DB_ERROR
		return fmt.Errorf("%w\n  Perbaikan: jalankan `go run main.go migrate` dulu untuk membuat database", err)
	case 1146: // ER_NO_SUCH_TABLE
		return fmt.Errorf("%w\n  Perbaikan: jalankan `go run main.go migrate` dulu untuk membuat tabel", err)
	}
	return err
}

// loadConfig membaca config.json dan memastikan field wajib terisi.
func loadConfig(path string) (config, error) {
	var cfg config
	raw, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("membaca %s: %w", path, err)
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, fmt.Errorf("parsing %s: %w", path, err)
	}

	cfg.MySQLHost = strings.TrimSpace(cfg.MySQLHost)
	cfg.MySQLPort = strings.TrimSpace(cfg.MySQLPort)
	cfg.MySQLUsername = strings.TrimSpace(cfg.MySQLUsername)

	var missing []string
	if cfg.MySQLHost == "" {
		missing = append(missing, "mysql_host")
	}
	if cfg.MySQLUsername == "" {
		missing = append(missing, "mysql_username")
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("lengkapi %s di %s", strings.Join(missing, ", "), path)
	}
	if cfg.MySQLPort == "" {
		cfg.MySQLPort = "3306"
	}
	return cfg, nil
}

// dsn menyusun DSN untuk koneksi. Saat database kosong, koneksi tidak
// memilih database karena skema sendiri yang menjalankan CREATE DATABASE
// dan USE.
func dsn(cfg config, database string) string {
	c := mysql.NewConfig()
	c.User = cfg.MySQLUsername
	c.Passwd = cfg.MySQLPassword
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(cfg.MySQLHost, cfg.MySQLPort)
	c.DBName = database
	c.Timeout = 5 * time.Second
	c.AllowNativePasswords = true
	return c.FormatDSN()
}

// splitStatements memecah skrip SQL menjadi daftar statement:
// baris kosong dan komentar (-- ...) dibuang, statement dipisah
// berdasarkan titik koma di luar string/identifier berkutip.
func splitStatements(script string) []string {
	var (
		stmts []string
		cur   strings.Builder
	)
	flush := func() {
		if s := strings.TrimSpace(cur.String()); s != "" {
			stmts = append(stmts, s)
		}
		cur.Reset()
	}
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		cur.WriteString(line)
		cur.WriteString("\n")
		if statementComplete(line) {
			flush()
		}
	}
	flush()
	return stmts
}

// statementComplete melaporkan apakah baris menutup statement,
// yaitu diakhiri titik koma di luar kutip.
func statementComplete(line string) bool {
	var quote byte
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			}
		case c == '\'' || c == '"' || c == '`':
			quote = c
		case c == ';':
			if strings.TrimSpace(line[i+1:]) == "" {
				return true
			}
		}
	}
	return false
}

// describe menghasilkan label singkat untuk sebuah statement.
func describe(stmt string) string {
	if m := reCreateDatabase.FindStringSubmatch(stmt); m != nil {
		return "membuat database `" + m[1] + "`"
	}
	if m := reCreateTable.FindStringSubmatch(stmt); m != nil {
		return "membuat tabel `" + m[1] + "`"
	}
	if name, ok := usedDatabase(stmt); ok {
		return "menggunakan database `" + name + "`"
	}
	first := strings.TrimSpace(strings.SplitN(stmt, "\n", 2)[0])
	if len(first) > 60 {
		first = first[:60] + "..."
	}
	return first
}

// usedDatabase mengembalikan nama database dari statement USE.
func usedDatabase(stmt string) (string, bool) {
	m := reUse.FindStringSubmatch(stmt)
	if m == nil {
		return "", false
	}
	return m[1], true
}

// listTables menampilkan tabel pada database hasil migrasi sebagai
// verifikasi; kegagalan hanya jadi peringatan karena migrasi sudah sukses.
func listTables(ctx context.Context, conn *sql.Conn, database string) {
	rows, err := conn.QueryContext(ctx, "SHOW TABLES")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Peringatan: gagal verifikasi tabel:", err)
		return
	}
	defer rows.Close()

	fmt.Printf("Tabel pada database `%s`:\n", database)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			fmt.Fprintln(os.Stderr, "Peringatan: gagal membaca daftar tabel:", err)
			return
		}
		fmt.Println("  -", name)
	}
	if err := rows.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Peringatan:", err)
	}
}
