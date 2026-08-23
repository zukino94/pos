// Migrator skema database POS.
//
// Membaca koneksi dari config.json lalu menjalankan skema SQL
// (mysql-scheme/pos.sql) ke server MySQL.
//
// Pemakaian:
//
//	go run main.go migrate
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
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
)

func main() {

	if len(os.Args) != 2 || os.Args[1] != "migrate" {

		fmt.Fprintln(os.Stderr, "Pemakaian:\n  go run main.go migrate")
		os.Exit(1)
	}

	err := migrate()

	if err != nil {

		fmt.Fprintln(os.Stderr, "Migrasi gagal:", err)
		os.Exit(1)
	}
}

func migrate() error {

	// Path diresolve relatif terhadap lokasi main.go sehingga
	// `go run main.go` bisa dijalankan dari direktori mana pun.
	_, thisFile, _, _ := runtime.Caller(0)
	baseDir := filepath.Dir(thisFile)
	configPath := filepath.Join(baseDir, "config.json")
	schemaPath := filepath.Clean(filepath.Join(baseDir, "..", "mysql-scheme", "pos.sql"))

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

	db, err := sql.Open("mysql", dsn(cfg))

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

		_, err := conn.ExecContext(ctx, stmt)

		if err != nil {

			return fmt.Errorf("mengeksekusi %s: %w", describe(stmt), err)
		}

		name, ok := usedDatabase(stmt)

		if ok {

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

// loadConfig membaca config.json dan memastikan field wajib terisi.
func loadConfig(path string) (config, error) {

	var cfg config
	raw, err := os.ReadFile(path)

	if err != nil {

		return cfg, fmt.Errorf("membaca %s: %w", path, err)
	}

	err = json.Unmarshal(raw, &cfg)

	if err != nil {

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

// dsn menyusun DSN tanpa memilih database karena skema sendiri yang
// menjalankan CREATE DATABASE dan USE.
func dsn(cfg config) string {
	c := mysql.NewConfig()
	c.User = cfg.MySQLUsername
	c.Passwd = cfg.MySQLPassword
	c.Net = "tcp"
	c.Addr = net.JoinHostPort(cfg.MySQLHost, cfg.MySQLPort)
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

		s := strings.TrimSpace(cur.String())

		if s != "" {

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

	m := reCreateDatabase.FindStringSubmatch(stmt)

	if m != nil {
		return "membuat database `" + m[1] + "`"
	}

	m = reCreateTable.FindStringSubmatch(stmt)

	if m != nil {

		return "membuat tabel `" + m[1] + "`"
	}

	name, ok := usedDatabase(stmt)

	if ok {

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

	err = rows.Err()

	if err != nil {

		fmt.Fprintln(os.Stderr, "Peringatan:", err)
	}
}
