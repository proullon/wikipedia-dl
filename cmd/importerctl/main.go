package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/proullon/wikipedia-to-cockroachdb/pkg/importer"
)

func main() {
	app := cli.NewApp()
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:   "host",
			Value:  "crdb.example.com",
			Usage:  "CockroachDB host",
			EnvVar: "CRDB_HOST",
		},
		cli.StringFlag{
			Name:   "dbname",
			Value:  "wikipedia",
			Usage:  "Database name",
			EnvVar: "DB_NAME",
		},
		cli.StringFlag{
			Name:   "user",
			Value:  "wikipedia",
			Usage:  "Database user",
			EnvVar: "DB_USER",
		},
		cli.StringFlag{
			Name:   "ssl-root-cert",
			Value:  "certs/ca.crt",
			Usage:  "Root SSL certificate",
			EnvVar: "SSL_ROOT_CERT",
		},
		cli.StringFlag{
			Name:   "ssl-client-key",
			Value:  "certs/client.wikipedia.key",
			Usage:  "Client SSL key",
			EnvVar: "SSL_CLIENT_KEY",
		},
		cli.StringFlag{
			Name:   "ssl-client-cert",
			Value:  "certs/client.wikipedia.crt",
			Usage:  "Client SSL certificate",
			EnvVar: "SSL_CLIENT_CERT",
		},
		cli.IntFlag{
			Name:   "db-max-conn",
			Value:  100,
			Usage:  "Maximum number of open connection to database",
			EnvVar: "DB_MAX_CONN",
		},
		cli.StringFlag{
			Name:   "dump-folder",
			Value:  "./dumps",
			Usage:  "Folder containing xml dumps",
			EnvVar: "DUMP_FOLDER",
		},
		cli.StringFlag{
			Name:   "logfile",
			Value:  "/tmp/wikipediatocrdb.log",
			Usage:  "Log destination",
			EnvVar: "LOGFILE",
		},
		cli.BoolFlag{
			Name:   "tight",
			Usage:  "Remove dumps from disk after import",
			EnvVar: "TIGHT",
		},
		cli.BoolFlag{
			Name:   "with-page-content",
			Usage:  "Import page content",
			EnvVar: "WITH_PAGE_CONTENT",
		},
		cli.BoolFlag{
			Name:   "with-page-references",
			Usage:  "Import page references",
			EnvVar: "WITH_PAGE_REFERENCES",
		},
		cli.BoolFlag{
			Name:   "interactive",
			Usage:  "Select dump manually",
			EnvVar: "INTERACTIVE",
		},
		cli.StringFlag{
			Name:   "language",
			Value:  "en",
			Usage:  "Language to import (ie 'en', 'fr')",
			EnvVar: "LANGUAGE",
		},
	}
	app.Action = start
	err := app.Run(os.Args)
	if err != nil {
		fmt.Printf("Fatal error: %s", err)
	}
}

func start(c *cli.Context) error {

	host := c.String("host")
	dbname := c.String("dbname")
	usr := c.String("user")
	sslRootCert := c.String("ssl-root-cert")
	sslClientKey := c.String("ssl-client-key")
	sslClientCert := c.String("ssl-client-cert")

	dsn := fmt.Sprintf("postgresql://%s@%s:26257/%s?ssl=true&sslmode=require&sslrootcert=%s&sslkey=%s&sslcert=%s",
		usr,
		host,
		dbname,
		sslRootCert,
		sslClientKey,
		sslClientCert,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}
	err = db.Ping()
	if err != nil {
		return err
	}
	db.SetMaxOpenConns(c.Int("db-max-conn"))
	db.SetMaxIdleConns(c.Int("db-max-conn"))
	fmt.Printf("Connected to %s/%s\n", host, dbname)

	f, err := os.OpenFile(c.String("logfile"), os.O_WRONLY|os.O_CREATE, 0755)
	if err != nil {
		return err
	}
	log.SetOutput(f)

	err = importer.Import(db, c.String("dump-folder"), c.Int("db-max-conn"), c.Bool("tight"), c.Bool("with-page-content"), c.Bool("with-page-references"), c.Bool("interactive"), c.String("language"))
	if err != nil {
		return err
	}
	return nil
}
