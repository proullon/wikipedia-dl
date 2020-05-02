CRDB_HOST=`cat .dev.conf | grep CRDB_HOST | cut -d '=' -f 2`
DB_MAX_CONN=`cat .dev.conf | grep DB_MAX_CONN | cut -d '=' -f 2`
FOLDER=`cat .dev.conf | grep FOLDER | cut -d '=' -f 2`
LANGUAGE=`cat .dev.conf | grep LANGUAGE | cut -d '=' -f 2`
DB_NAME=`cat .dev.conf | grep DB_NAME | cut -d '=' -f 2`

all: help

help:
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST) | sort

re: install ## rebuild binaries

install: ## install binaries
	clear
	go install ./...

run.full: ## run importer on target defined in .dev.conf
	rm -f /tmp/wikipediatocrdb.log
	importerctl --host=$(CRDB_HOST) --db-max-conn=$(DB_MAX_CONN) --dump-folder=$(FOLDER) --dbname=$(DB_NAME) \
					--language=$(LANGUAGE) \
					--interactive \
					--with-page-references \
					--with-page-content

run.light: ## run importer on target defined in .dev.conf
	rm -f /tmp/wikipediatocrdb.log
	importerctl --host=$(CRDB_HOST) --db-max-conn=$(DB_MAX_CONN) --dump-folder=$(FOLDER) --dbname=$(DB_NAME) \
					--language=$(LANGUAGE) \
					--interactive

run.ref: ## run importer on target defined in .dev.conf
	rm -f /tmp/wikipediatocrdb.log
	importerctl --host=$(CRDB_HOST) --db-max-conn=$(DB_MAX_CONN) --dump-folder=$(FOLDER) --dbname=$(DB_NAME) \
					--language=$(LANGUAGE) \
					--interactive \
					--with-page-references

run.tight: ## run importer on target defined in .dev.conf
	rm -f /tmp/wikipediatocrdb.log
	importerctl --host=$(CRDB_HOST) --db-max-conn=$(DB_MAX_CONN) --dump-folder=$(FOLDER) --dbname=$(DB_NAME) \
					--language=$(LANGUAGE) \
					--tight \
					--interactive \
					--with-page-references \
					--with-page-content
