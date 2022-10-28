//-*- coding: utf-8 -*-

// About: A converter from MS Access/Jet database created by
// Hogia Hemekonomi (mid 1990's) to SQLite database.

// System Requirements: Windows 10 (any)

// Prepare: install gnu emacs: emacs-26.3-x64_64 (optional)
// Prepare: TDM-GCC from https://jmeubank.github.io/tdm-gcc/
//https://github.com/jmeubank/tdm-gcc/releases/download/v9.2.0-tdm-1/tdm-gcc-9.2.0.exe

// Prepare: install git: Git-2.23.0-64-bit
// Prepare: install golang 32-bits (can't access access/jet driver using 64-bits)
//   go1.16.3.windows-386.msi
// Prepare: go get github.com/alexbrainman/odbc
// Prepare: go get github.com/mattn/go-sqlite3
// Build: go build hhek2sqlite.go
// Build release: go build -ldflags="-s -w" hhek2sqlite.go
// Run: ./hhek2sqlite.exe -help
// Run: ./hhek2sqlite.exe -optin=hemekonomi.mdb -optout ekonomi.db
// System requirements for hhek2sqlite.exe is Windows XP or later

// Schema of mdb-file can be found using jet-tool by himselfv
// https://github.com/himselfv/jet-tool/releases/tag/0.6b
// \tmp\Jet.exe schema -f foo.mdb >schema.txt
// \tmp\Jet.exe dump -f foo.mdb >dump.txt

// Dump existing (empty/new) database to txt and back (tom.mdb -> dumptom.sql -> jette.mdb):
// \tmp\Jet.exe dump -f tom.mdb --all >dumptom.sql
// \tmp\Jet.exe exec -f jette.mdb -new --db-format jet20 <dumptom.sql

// Or https://sourceforge.net/projects/jetsqlconsole/
// More tools may be available via
// https://en.wikibooks.org/wiki/JET_Database/Manipulation_tools

package main

import (
	_ "embed"
	"fmt"
	"flag"
	"log"
	"os"
	"strings"
	"strconv"
	"golang.org/x/text/encoding/charmap"
	"context"
	"database/sql"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	
	_ "github.com/alexbrainman/odbc"
	_ "github.com/mattn/go-sqlite3"
	//  _ "github.com/bvinc/go-sqlite-lite/sqlite3"
)

var revopt bool

func toUtf8(in_buf []byte) string {
	var buf []byte
	
	if revopt {
		buf = in_buf
	} else {
		buf, _ = charmap.Windows1252.NewDecoder().Bytes(in_buf)
	}
	// Escape chars for SQL
	stringVal := string(buf)
	stringVal2 := strings.ReplaceAll(stringVal, "'", "''");
	stringVal3 := strings.ReplaceAll(stringVal2, "\"", "\"\"");
	return stringVal3
}

func copyPersoner(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Personer\".")

	// column fodd is Född (år 4 siffor, 0 för Gemensamt)
	// column kon is Kön (text: Gemensamt, Man, Kvinna)

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Personer (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY, Namn Text(50), Född Text(4), Kön Text(10));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE UNIQUE INDEX idxNamn ON Personer (Namn);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Personer;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Personer (Löpnr integer not null primary key AUTOINCREMENT, Namn text, Född INTEGER, Kön text);
  delete from Personer;
  `
	}
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Personer")
	
	res, err := db.Query("SELECT Namn,Född,Kön,Löpnr FROM Personer")
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var namn []byte   // size 50
	var birth string  // size 4 (år, 0 för Gemensamt)
	var sex string    // size 10 (text: Gemensamt, Man, Kvinna)
	var nummer int    // autoinc Primary Key, index
	var rownum int    // counter for showing stats

	rownum = 0
	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&namn, &birth, &sex, &nummer)

		sqlStmt:="insert into "
		sqlStmt+="Personer(Löpnr, Namn, Född, Kön) "
		sqlStmt+="values("
		sqlStmt+="'" + strconv.Itoa(nummer) + "', "
		sqlStmt+="'" + toUtf8(namn) + "', "
		sqlStmt+="'" + birth + "', "
		sqlStmt+="'" + sex + "')"

		//fmt.Println("EXEC: ", sqlStmt)

		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
	}
}

var (
	ctx context.Context
)

func comma2point(value string) string {
	if revopt {
		return "'" + strings.ReplaceAll(value, ".", ",") + "'"
	} else {
		return strings.ReplaceAll(value, ",", ".")
	}
}

func copyTransaktioner(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Transaktioner\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Transaktioner (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY,FrånKonto Text(40),TillKonto Text(40),Typ Text(40),Datum Text(10),Vad Text(40),Vem Text(50),Belopp Currency,Saldo Currency,Fastöverföring Bit,"Text" Text(60));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE INDEX idxFrånKontoTillKonto ON Transaktioner (FrånKonto, TillKonto);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Transaktioner;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Transaktioner (Löpnr integer not null primary key AUTOINCREMENT,FrånKonto TEXT,TillKonto TEXT,Typ TEXT,Datum TEXT,Vad TEXT,Vem TEXT,Belopp DECIMAL(19,4),Saldo DECIMAL(19,4),Fastöverföring BOOLEAN,Text TEXT);
  delete from Transaktioner;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Transaktioner")

	res, err := db.Query("SELECT FrånKonto,TillKonto,Typ,Datum,Vad,Vem,Belopp,Löpnr,Saldo,Fastöverföring,Text FROM Transaktioner")
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var fromAcc []byte  // size 40
	var toAcc []byte    // size 40
	var tType []byte    // size 40
	var date []byte     // size 10
	var what []byte     // size 40
	var who []byte      // size 50
	var amount []byte   // BCD / Decimal Precision 19
	var nummer int      // Autoinc Primary Key, index
	var saldo []byte    // BCD / Decimal Precision 19
	var fixed bool      // Boolean
	var comment []byte  // size 60
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&fromAcc, &toAcc, &tType, &date, &what, &who, &amount, &nummer, &saldo, &fixed, &comment)

		//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
		tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("Läst OK.")
		var dbamount = comma2point(toUtf8(amount))
		
		sqlStmt:="INSERT INTO "
		sqlStmt+="Transaktioner (FrånKonto,TillKonto,Typ,Datum,Vad,Vem,Belopp,Löpnr,Saldo,Fastöverföring,`Text`) "
		sqlStmt+="VALUES ("
		sqlStmt+="'" + toUtf8(fromAcc) + "', "
		sqlStmt+="'" + toUtf8(toAcc) + "', "
		sqlStmt+="'" + toUtf8(tType) + "', "
		sqlStmt+="'" + toUtf8(date) + "', "
		sqlStmt+="'" + toUtf8(what) + "', "
		sqlStmt+="'" + toUtf8(who) + "', "
		sqlStmt+="" + dbamount + ", "
		sqlStmt+="" + strconv.Itoa(nummer) + ", "
		sqlStmt+="" + "NULL" + ", "
		sqlStmt+="" + strconv.FormatBool(fixed) + ", "
		sqlStmt+="'" + toUtf8(comment) + "')"

		//fmt.Println("EXEC: ", sqlStmt)

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			_ = tx.Rollback()
			log.Fatal(execErr)
		}
		if err := tx.Commit(); err != nil {
			log.Fatal(err)
		}
	}
}

func copyDtbVer(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"DtbVer\".")

	var sqlStmt string
	sqlStmt = `
  create table DtbVer (VerNum Text(4) CONSTRAINT idxVerNum PRIMARY KEY,Benämning Text(80),Losenord Text(8));
  `
	//  delete from DtbVer;
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		os.Exit(1)
	}
	// Empty table
	sqlStmt = `
  delete from DtbVer;
  `
	_, err = outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		os.Exit(1)
	}

	// Läs och kopiera data

	res, err := db.Query("SELECT VerNum,Benämning,Losenord FROM DtbVer")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()
	
	var VerNum []byte  // size 4 Primary Key, index
	var Ben []byte     // size 80
	var Losenord []byte  // size 8
	for res.Next() {
		err = res.Scan(&VerNum, &Ben, &Losenord)

		sqlStmt:="insert into "
		sqlStmt+="DtbVer(VerNum, Benämning, Losenord) "
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(VerNum) + "', "
		sqlStmt+="'" + toUtf8(Ben) + "', "
		sqlStmt+="'" + toUtf8(Losenord) + "')"

		//fmt.Println("EXEC: ", sqlStmt)

		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
	}
}

func copyBetalKonton(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"BetalKonton\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table BetalKonton (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY, Konto Text(40), Kontonummer Text(40), Kundnummer Text(40) , Sigillnummer Text(40));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE UNIQUE INDEX idxKonto ON BetalKonton (Konto);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from BetalKonton;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table BetalKonton (Löpnr integer not null primary key AUTOINCREMENT, Konto TEXT, Kontonummer TEXT, Kundnummer TEXT , Sigillnummer TEXT);
  delete from BetalKonton;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "BetalKonton")

	res, err := db.Query("SELECT Konto,Kontonummer,Kundnummer,Sigillnummer,Löpnr FROM BetalKonton")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()
	
	var Konto []byte         // size 40, index
	var Kontonummer []byte   // size 40
	var Kundnummer []byte    // size 40
	var Sigillnummer []byte  // size 40
	var nummer int           // autoinc Primary Key
	var rownum int    // counter for showing stats

	rownum = 0
	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&Konto, &Kontonummer, &Kundnummer, &Sigillnummer, &nummer)

		sqlStmt:="insert into "
		sqlStmt+="BetalKonton(Löpnr, Konto,Kontonummer,Kundnummer,Sigillnummer) "
		sqlStmt+="values("
		sqlStmt+="'" + strconv.Itoa(nummer) + "', "
		sqlStmt+="'" + toUtf8(Konto) + "', "
		sqlStmt+="'" + toUtf8(Kontonummer) + "', "
		sqlStmt+="'" + toUtf8(Kundnummer) + "', "
		sqlStmt+="'" + toUtf8(Sigillnummer) + "')"

		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
	}
}

func copyBetalningar(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Betalningar\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Betalningar (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY,FrånKonto Text(40),TillPlats Text(40),Typ Text(40),Datum Text(10),Vad Text(40),Vem Text(50),Belopp Currency,'Text' Text(60),Ranta Currency,FastAmort Currency,RorligAmort Currency,OvrUtg Currency,LanLopnr INTEGER,Grey Text(2));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE INDEX idxDatum ON Betalningar (Datum);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE INDEX idxFrånKonto ON Betalningar (FrånKonto);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE INDEX idxLanLopnr ON Betalningar (LanLopnr);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE INDEX idxTillPlats ON Betalningar (TillPlats);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Betalningar;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Betalningar (Löpnr integer not null primary key AUTOINCREMENT,FrånKonto TEXT,TillPlats TEXT,Typ TEXT,Datum TEXT,Vad TEXT,Vem TEXT,Belopp DECIMAL(19,4),Text TEXT,Ranta DECIMAL(19,4),FastAmort DECIMAL(19,4),RorligAmort DECIMAL(19,4),OvrUtg DECIMAL(19,4),LanLopnr INTEGER,Grey TEXT);
  delete from Betalningar;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Betalningar")

	res, err := db.Query("SELECT FrånKonto,TillPlats,Typ,Datum,Vad,Vem,Belopp,Text,Löpnr,Ranta,FastAmort,RorligAmort,OvrUtg,LanLopnr,Grey FROM Betalningar")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var FrånKonto  []byte  // size 40
	var TillPlats []byte  // size 40
	var Typ []byte  // size 40
	var Datum []byte  // size 10
	var Vad []byte  // size 40
	var Vem []byte  // size 50
	var Belopp []byte  // BCD / Decimal Precision 19
	var Text []byte  // size 60
	var Löpnr []byte  // Autoinc Primary Key, index
	var Ranta []byte  // BCD / Decimal Precision 19
	var FastAmort []byte  // BCD / Decimal Precision 19
	var RorligAmort []byte  // BCD / Decimal Precision 19
	var OvrUtg []byte  // BCD / Decimal Precision 19
	var LanLopnr []byte  // Integer
	var Grey   []byte  // size 2
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&FrånKonto, &TillPlats, &Typ, &Datum, &Vad, &Vem, &Belopp, &Text, &Löpnr, &Ranta, &FastAmort, &RorligAmort, &OvrUtg, &LanLopnr, &Grey)

		sqlStmt:="insert into "
		sqlStmt+="Betalningar(Löpnr,FrånKonto,TillPlats,Typ,Datum,Vad,Vem,Belopp,Text,Ranta,FastAmort,RorligAmort,OvrUtg,LanLopnr,Grey) "
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(Löpnr) + "', "
		sqlStmt+="'" + toUtf8(FrånKonto) + "', "
		sqlStmt+="'" + toUtf8(TillPlats) + "', "
		sqlStmt+="'" + toUtf8(Typ) + "', "
		sqlStmt+="'" + toUtf8(Datum) + "', "
		sqlStmt+="'" + toUtf8(Vad) + "', "
		sqlStmt+="'" + toUtf8(Vem) + "', "
		sqlStmt+="" + comma2point(toUtf8(Belopp)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Ranta)) + ", "
		sqlStmt+="" + comma2point(toUtf8(FastAmort)) + ", "
		sqlStmt+="" + comma2point(toUtf8(RorligAmort)) + ", "
		sqlStmt+="'" + toUtf8(OvrUtg) + "', "
		sqlStmt+="'" + toUtf8(LanLopnr) + "', "
		sqlStmt+="'" + toUtf8(Grey) + "')"

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			_ = tx.Rollback()
			log.Fatal(execErr)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

// Överföringar
func copyTransfers(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Överföringar\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Överföringar (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY,FrånKonto Text(40),TillKonto Text(40),Belopp Currency,Datum Text(10),HurOfta Text(15),Vad Text(40),Vem Text(40),Kontrollnr INTEGER,TillDatum Text(10),Rakning Text(1));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE INDEX idxFrånKontoTillKonto ON Överföringar (FrånKonto, TillKonto);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Överföringar;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Överföringar (Löpnr integer not null primary key AUTOINCREMENT,FrånKonto TEXT,TillKonto TEXT,Belopp DECIMAL(19,4),Datum TEXT,HurOfta TEXT,Vad TEXT,Vem TEXT,Kontrollnr INTEGER,TillDatum TEXT,Rakning TEXT);
  delete from Överföringar;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Överföringar")

	res, err := db.Query("SELECT FrånKonto,TillKonto,Belopp,Datum,HurOfta,Vad,Vem,Löpnr,Kontrollnr,TillDatum,Rakning FROM Överföringar")
	
	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var FrånKonto []byte  // size 40
	var TillKonto []byte  // size 40
	var Belopp []byte  // BCD / Decimal Precision 19
	var Datum []byte  // size 10
	var HurOfta []byte  // size 15
	var Vad []byte  // size 40
	var Vem []byte  // size 40
	var Löpnr []byte  // Autoinc Primary Key, index
	var Kontrollnr []byte  //int  // Integer
	var TillDatum []byte  // size 10
	var Rakning []byte  // size 1
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&FrånKonto, &TillKonto, &Belopp, &Datum, &HurOfta, &Vad, &Vem, &Löpnr, &Kontrollnr, &TillDatum, &Rakning)

		sqlStmt:="insert into "
		sqlStmt+="Överföringar(Löpnr,FrånKonto,TillKonto,Belopp,Datum,HurOfta,Vad,Vem,Kontrollnr,TillDatum,Rakning) "
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(Löpnr) + "', "
		sqlStmt+="'" + toUtf8(FrånKonto) + "', "
		sqlStmt+="'" + toUtf8(TillKonto) + "', "
		sqlStmt+="" + comma2point(toUtf8(Belopp)) + ", "
		sqlStmt+="'" + toUtf8(Datum) + "', "
		sqlStmt+="'" + toUtf8(HurOfta) + "', "
		sqlStmt+="'" + toUtf8(Vad) + "', "
		sqlStmt+="'" + toUtf8(Vem) + "', "
		if len(Kontrollnr) < 1 {
			sqlStmt+="null, "
		} else {
			sqlStmt+="'" + toUtf8(Kontrollnr) + "', "
		}
		sqlStmt+="'" + toUtf8(TillDatum) + "', "
		sqlStmt+="'" + toUtf8(Rakning) + "')"

		//fmt.Println("EXEC: ", sqlStmt)

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			log.Printf("%q: %s\n", execErr, sqlStmt)
			_ = tx.Rollback()
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}

}

func copyKonton(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Konton\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Konton (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY, KontoNummer Text(20),Benämning Text(40),Saldo Currency,StartSaldo Currency,StartManad Text(10),SaldoArsskifte Currency,ArsskifteManad Text(10));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE UNIQUE INDEX idxBenämning ON Konton (Benämning);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Konton;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Konton (Löpnr integer not null primary key AUTOINCREMENT, KontoNummer TEXT,Benämning TEXT,Saldo DECIMAL(19,4),StartSaldo DECIMAL(19,4),StartManad TEXT,SaldoArsskifte DECIMAL(19,4),ArsskifteManad text);
  delete from Konton;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Konton")

	res, err := db.Query("SELECT KontoNummer,Benämning,Saldo,StartSaldo,StartManad,Löpnr,SaldoArsskifte,ArsskifteManad FROM Konton")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var KontoNummer []byte  // size 20
	var Benämning  []byte  // size 40, index
	var Saldo []byte  // BCD / Decimal Precision 19
	var StartSaldo []byte  // BCD / Decimal Precision 19
	var StartManad []byte  // size 10
	var Löpnr  []byte  // autoinc Primary Key
	var SaldoArsskifte []byte  // BCD / Decimal Precision 19
	var ArsskifteManad []byte  // size 10
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&KontoNummer,&Benämning,&Saldo,&StartSaldo,&StartManad,&Löpnr,&SaldoArsskifte,&ArsskifteManad)

		sqlStmt:="insert into "
		sqlStmt+="Konton(Löpnr, KontoNummer, Benämning, Saldo, StartSaldo, StartManad, SaldoArsskifte, ArsskifteManad) "
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(Löpnr) + "', "
		sqlStmt+="'" + toUtf8(KontoNummer) + "', "
		sqlStmt+="'" + toUtf8(Benämning) + "', "
		sqlStmt+="" + comma2point(toUtf8(Saldo)) + ", "
		sqlStmt+="" + comma2point(toUtf8(StartSaldo)) + ", "
		sqlStmt+="'" + toUtf8(StartManad) + "', "
		sqlStmt+="" + comma2point(toUtf8(SaldoArsskifte)) + ", "
		sqlStmt+="'" + toUtf8(ArsskifteManad) + "')"

		//fmt.Println("EXEC: ", sqlStmt)

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			_ = tx.Rollback()
			log.Fatal(execErr)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func copyLoan(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"LÅN\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table LÅN (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY,Langivare Text(40),EgenBeskrivn Text(40),LanNummer Text(25),TotLanebelopp Currency,StartDatum Text(10),RegDatum Text(10),RantJustDatum Text(10),SlutBetDatum Text(10),AktLaneskuld Currency,RorligDel Currency,FastDel Currency,FastRanta Single,RorligRanta Single,HurOfta Text(2),Ranta Currency,FastAmort Currency,RorligAmort Currency,OvrUtg Currency,Rakning Text(1),Vem Text(40),FrånKonto Text(40),Grey Text(2),Anteckningar Memo,BudgetRanta Text(40),BudgetAmort Text(40),BudgetOvriga Text(40));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE INDEX idxLangiv ON LÅN (Langivare);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from LÅN;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table LÅN (Löpnr integer not null primary key AUTOINCREMENT,Langivare TEXT,EgenBeskrivn TEXT,LanNummer TEXT,TotLanebelopp DECIMAL(19,4),StartDatum TEXT,RegDatum TEXT,RantJustDatum TEXT,SlutBetDatum TEXT,AktLaneskuld DECIMAL(19,4),RorligDel DECIMAL(19,4),FastDel DECIMAL(19,4),FastRanta REAL,RorligRanta REAL,HurOfta TEXT,Ranta DECIMAL(19,4),FastAmort DECIMAL(19,4),RorligAmort DECIMAL(19,4),OvrUtg DECIMAL(19,4),Rakning TEXT,Vem TEXT,FrånKonto TEXT,Grey TEXT,Anteckningar TEXT,BudgetRanta TEXT,BudgetAmort TEXT,BudgetOvriga TEXT);
  delete from LÅN;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "LÅN")

	res, err := db.Query("SELECT Langivare,EgenBeskrivn,LanNummer,TotLanebelopp,StartDatum,RegDatum,RantJustDatum,SlutBetDatum,AktLaneskuld,RorligDel,FastDel,FastRanta,RorligRanta,HurOfta,Ranta,FastAmort,RorligAmort,OvrUtg,Löpnr,Rakning,Vem,FrånKonto,Grey,Anteckningar,BudgetRanta,BudgetAmort,BudgetOvriga FROM LÅN")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var Langivare []byte  // size 40
	var EgenBeskrivn []byte  // size 40
	var LanNummer []byte  // size 25
	var TotLanebelopp []byte  // BCD / Decimal Precision 19
	var StartDatum []byte  // size 10
	var RegDatum []byte  // size 10
	var RantJustDatum []byte  // size 10
	var SlutBetDatum []byte  // size 10
	var AktLaneskuld []byte  // BCD / Decimal Precision 19
	var RorligDel []byte  // BCD / Decimal Precision 19
	var FastDel []byte  // BCD / Decimal Precision 19
	var FastRanta float32
	var RorligRanta float32
	var HurOfta []byte  // size 2
	var Ranta []byte  // BCD / Decimal Precision 19
	var FastAmort []byte  // BCD / Decimal Precision 19
	var RorligAmort []byte  // BCD / Decimal Precision 19
	var OvrUtg []byte  // BCD / Decimal Precision 19
	var Löpnr []byte  // autoinc Primary Key, index
	var Rakning []byte  // size 1
	var Vem []byte  // size 40
	var FrånKonto []byte  // size 40
	var Grey []byte  // size 2
	var Anteckningar []byte  // Memo
	var BudgetRanta []byte  // size 40
	var BudgetAmort []byte  // size 40
	var BudgetOvriga []byte  // size 40
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatal(err)
	}

	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&Langivare,&EgenBeskrivn,&LanNummer,&TotLanebelopp,&StartDatum,&RegDatum,&RantJustDatum,&SlutBetDatum,&AktLaneskuld,&RorligDel,&FastDel,&FastRanta,&RorligRanta,&HurOfta,&Ranta,&FastAmort,&RorligAmort,&OvrUtg,&Löpnr,&Rakning,&Vem,&FrånKonto,&Grey,&Anteckningar,&BudgetRanta,&BudgetAmort,&BudgetOvriga)

		sqlStmt:="insert into "
		sqlStmt+="LÅN(Löpnr,Langivare,EgenBeskrivn,LanNummer,TotLanebelopp,StartDatum,RegDatum,RantJustDatum,SlutBetDatum,AktLaneskuld,RorligDel,FastDel,FastRanta,RorligRanta,HurOfta,Ranta,FastAmort,RorligAmort,OvrUtg,Rakning,Vem,FrånKonto,Grey,Anteckningar,BudgetRanta,BudgetAmort,BudgetOvriga)"
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(Löpnr) + "', "
		sqlStmt+="'" + toUtf8(Langivare) + "', "
		sqlStmt+="'" + toUtf8(EgenBeskrivn) + "', "
		sqlStmt+="'" + toUtf8(LanNummer) + "', "
		sqlStmt+="" + comma2point(toUtf8(TotLanebelopp)) + ", "
		sqlStmt+="'" + toUtf8(StartDatum) + "', "
		sqlStmt+="'" + toUtf8(RegDatum) + "', "
		sqlStmt+="'" + toUtf8(RantJustDatum) + "', "
		sqlStmt+="'" + toUtf8(SlutBetDatum) + "', "
		sqlStmt+="" + comma2point(toUtf8(AktLaneskuld)) + ", "
		sqlStmt+="" + comma2point(toUtf8(RorligDel)) + ", "
		sqlStmt+="" + comma2point(toUtf8(FastDel)) + ", "
		sqlStmt+="" + comma2point(fmt.Sprintf("%g", FastRanta)) + ", "
		sqlStmt+="" + comma2point(fmt.Sprintf("%g", RorligRanta)) + ", "
		sqlStmt+="'" + toUtf8(HurOfta) + "', "
		sqlStmt+="" + comma2point(toUtf8(Ranta)) + ", "
		sqlStmt+="" + comma2point(toUtf8(FastAmort)) + ", "
		sqlStmt+="" + comma2point(toUtf8(RorligAmort)) + ", "
		sqlStmt+="'" + toUtf8(OvrUtg) + "', "
		sqlStmt+="'" + toUtf8(Rakning) + "', "
		sqlStmt+="'" + toUtf8(Vem) + "', "
		sqlStmt+="'" + toUtf8(FrånKonto) + "', "
		sqlStmt+="'" + toUtf8(Grey) + "', "
		sqlStmt+="'" + toUtf8(Anteckningar) + "', "
		sqlStmt+="" + comma2point(toUtf8(BudgetRanta)) + ", "
		sqlStmt+="" + comma2point(toUtf8(BudgetAmort)) + ", "
		sqlStmt+="" + comma2point(toUtf8(BudgetOvriga)) + ")"
		fmt.Println(sqlStmt)

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			_ = tx.Rollback()
			log.Fatal(execErr)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func copyPlatser(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Platser\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Platser (Löpnr AutoIncrement CONSTRAINT idxLöpnr PRIMARY KEY, Namn Text(40), Gironummer Text(20), Typ Text(2), RefKonto Text(40));
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE UNIQUE INDEX idxNamn ON Platser (Namn);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Platser;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Platser (Löpnr integer not null primary key AUTOINCREMENT, Namn text, Gironummer text, Typ text, RefKonto Text);
  delete from Platser;
  `
	}
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Platser")

	res, err := db.Query("SELECT Namn,Gironummer,Typ,RefKonto,Löpnr FROM Platser")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var Namn []byte  // size 40
	var Gironummer []byte  // size 20
	var Typ []byte  // size 2
	var RefKonto []byte  // size 40
	var Löpnr []byte  // autoinc Primary Key, index
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
	//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")
		err = res.Scan(&Namn,&Gironummer,&Typ,&RefKonto,&Löpnr)

		sqlStmt:="insert into "
		sqlStmt+="Platser(Löpnr, Namn, Gironummer, Typ, RefKonto) "
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(Löpnr) + "', "
		sqlStmt+="'" + toUtf8(Namn) + "', "
		sqlStmt+="'" + toUtf8(Gironummer) + "', "
		sqlStmt+="'" + toUtf8(Typ) + "', "
		sqlStmt+="'" + toUtf8(RefKonto) + "')"

		//fmt.Println("EXEC: ", sqlStmt)

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			_ = tx.Rollback()
			log.Fatal(execErr)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func copyBudget(db *sql.DB, outdb *sql.DB) {
	fmt.Println("Kopierar över \"Budget\".")

	var sqlStmt string
	if revopt {
		sqlStmt = `
  create table Budget (Löpnr AUTOINCREMENT CONSTRAINT idxLöpnr PRIMARY KEY,Typ Text(40),Inkomst Text(1),HurOfta Byte,StartMånad Text(10),Jan Currency,Feb Currency,Mar Currency,Apr Currency,Maj Currency,Jun Currency,Jul Currency,Aug Currency,Sep Currency,Okt Currency,Nov Currency,Dec Currency,Kontrollnr INTEGER);
  `
		_, err := outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			return
		}
		sqlStmt = `
  CREATE UNIQUE INDEX idxTyp ON Budget (Typ);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE INDEX idxInkomst ON Budget (Inkomst);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE INDEX idxInkomstTyp ON Budget (Inkomst,Typ);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		sqlStmt = `
  CREATE INDEX idxKontrollnr ON Budget (Kontrollnr);
  `
		_, err = outdb.Exec(sqlStmt)
		if err != nil {
			log.Printf("%q: %s\n", err, sqlStmt)
			os.Exit(1)
		}
		// Töm tabellen
		sqlStmt = `
  delete from Budget;
  `
	} else {
		// Skapa tabellen
		sqlStmt = `
  create table Budget (Löpnr integer not null primary key AUTOINCREMENT,Typ TEXT,Inkomst TEXT,HurOfta INTEGER,StartMånad TEXT,Jan DECIMAL(19,4),Feb DECIMAL(19,4),Mar DECIMAL(19,4),Apr DECIMAL(19,4),Maj DECIMAL(19,4),Jun DECIMAL(19,4),Jul DECIMAL(19,4),Aug DECIMAL(19,4),Sep DECIMAL(19,4),Okt DECIMAL(19,4),Nov DECIMAL(19,4),Dec DECIMAL(19,4),Kontrollnr INTEGER);
  delete from Budget;
  `
	}
	
	_, err := outdb.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}

	// Läs och kopiera data
	var count int;
	count = number_of_rows(db, "Budget")

	res, err := db.Query("SELECT Löpnr,Typ,Inkomst,HurOfta,StartMånad,Jan,Feb,Mar,Apr,Maj,Jun,Jul,Aug,Sep,Okt,Nov,Dec,Kontrollnr FROM Budget")

	if err != nil {
		log.Fatal(err)
		os.Exit(2)
	}
	defer res.Close()

	var Typ []byte  // size 40
	var Inkomst []byte  // size 1
	var HurOfta int16 // SmallInt
	var StartMånad []byte  // size 10
	var Jan []byte  // BCD / Decimal Precision 19
	var Feb []byte  // BCD / Decimal Precision 19
	var Mar []byte  // BCD / Decimal Precision 19
	var Apr []byte  // BCD / Decimal Precision 19
	var Maj []byte  // BCD / Decimal Precision 19
	var Jun []byte  // BCD / Decimal Precision 19
	var Jul []byte  // BCD / Decimal Precision 19
	var Aug []byte  // BCD / Decimal Precision 19
	var Sep []byte  // BCD / Decimal Precision 19
	var Okt []byte  // BCD / Decimal Precision 19
	var Nov []byte  // BCD / Decimal Precision 19
	var Dec []byte  // BCD / Decimal Precision 19
	var Kontrollnr []byte  //int32 // Integer
	var Löpnr []byte  // autoinc Primary Key, index
	var rownum int    // counter for showing stats

	rownum = 0
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	//tx, err := outdb.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	tx, err := outdb.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		log.Fatal(err)
	}
	for res.Next() {
		rownum+=1
		fmt.Println("Kopierar rad", rownum, "av", count, ".")

		err = res.Scan(&Löpnr,&Typ,&Inkomst,&HurOfta,&StartMånad,&Jan,&Feb,&Mar,&Apr,&Maj,&Jun,&Jul,&Aug,&Sep,&Okt,&Nov,&Dec,&Kontrollnr)

		sqlStmt:="insert into "
		sqlStmt+="Budget(Löpnr,Typ,Inkomst,HurOfta,StartMånad,Jan,Feb,Mar,Apr,Maj,Jun,Jul,Aug,Sep,Okt,Nov,Dec,Kontrollnr) "
		sqlStmt+="values("
		sqlStmt+="'" + toUtf8(Löpnr) + "', "
		sqlStmt+="'" + toUtf8(Typ) + "', "
		sqlStmt+="'" + toUtf8(Inkomst) + "', "
		sqlStmt+="" + strconv.Itoa(int(HurOfta)) + ", "
		sqlStmt+="'" + toUtf8(StartMånad) + "', "
		sqlStmt+="" + comma2point(toUtf8(Jan)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Feb)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Mar)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Apr)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Maj)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Jun)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Jul)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Aug)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Sep)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Okt)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Nov)) + ", "
		sqlStmt+="" + comma2point(toUtf8(Dec)) + ", "
		if Kontrollnr != nil {
			sqlStmt+="'" + toUtf8(Kontrollnr) + "')"
		} else {
			sqlStmt+="null)"
		}

		//fmt.Println("EXEC: ", sqlStmt)

		_, execErr := tx.Exec(sqlStmt)
		if execErr != nil {
			_ = tx.Rollback()
			log.Fatal(execErr)
		}
	}
	if err := tx.Commit(); err != nil {
		log.Fatal(err)
	}
}

func number_of_rows(db *sql.DB, tablename string) int {
	var count int;
	row := db.QueryRow("SELECT COUNT(*) FROM "+tablename)
	err := row.Scan(&count);
	if err != nil {
		log.Fatal(err)
		os.Exit(3)
	}
	return count
}

func sqlite_init(filename string) *sql.DB {
	if !revopt {
		err := os.Remove(filename)
		if err != nil {
			if !os.IsNotExist(err) {
				log.Println(err)
				os.Exit(2)
			}
		}
	}
	
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Fatal(err)
	}

	return db
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func CreateMDBFile(filepath string) error {
	fmt.Println("OLE skapar databas: ")

	err := ole.CoInitialize(0)
	if err != nil {
		fmt.Println("MDB create file: " + err.Error())
	}
	defer ole.CoUninitialize()
	
	unk, err := oleutil.CreateObject("adox.catalog")
	if err != nil {
		fmt.Println("MDB create file: " + err.Error())
	}
	cat, err := unk.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		fmt.Println("MDB create file: " + err.Error())
	}
	_, err = oleutil.CallMethod(cat, "create", fmt.Sprintf("provider=microsoft.jet.oledb.4.0;" +
		"Jet OLEDB:Engine Type=3;" +
		"data source=%s;", filepath))
	if err != nil {
		fmt.Println("MDB create file: " + err.Error())
	}

	return err
}


func main() {
	optinPtr := flag.String("optin", "", "Hogia Hemekonomi database filename (*.mdb)")
	optoutPtr := flag.String("optout", "", "sqlite3 database filename (*.db)")
	readonlyoptPtr := flag.Bool("readonly", true, "Öppna mdb skrivskyddat.")
	backaPtr := flag.Bool("backa", false, "Konvertera från sqlite till mdb.")
	
	flag.Parse()
	
	if *optinPtr == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *optoutPtr == "" {
		flag.Usage()
		os.Exit(1)
	}
	if *backaPtr {
		fmt.Println("Konverterar från sqlite till MDB")
	} else {
		fmt.Println("Konverterar från MDB till sqlite")
	}
	
	konvertera(*optinPtr, *optoutPtr, *readonlyoptPtr, *backaPtr)
}

func konvertera(mdbfilename string, dbfilename string, readonly bool, backa bool) {
	revopt = backa
	filename := mdbfilename;
	if !backa && !fileExists(filename) {
		fmt.Println(mdbfilename, " file does not exist (or is a directory)")
		flag.Usage()
		os.Exit(1)
	}
	if backa && fileExists(filename) {
		fmt.Println(mdbfilename, " file exists (or is a directory)")
		flag.Usage()
		os.Exit(1)
	}

	// Download base file structure
	if backa {
		err := CreateMDBFile(filename)
		if err != nil {
			panic(err)
		}
		fmt.Println("Downloaded to " + filename)
	}
	
	//   powershell show available:  get-odbcdriver -name "*mdb*"
	// ODBC options see https://docs.microsoft.com/en-us/sql/odbc/microsoft/setting-options-programmatically-for-the-access-driver?view=sql-server-ver15
	readonlyCommand := ""
	if (!backa) && readonly {
		readonlyCommand = "READONLY;"
		fmt.Println("Setting Readonly")
	}

	var err error
	var db *sql.DB
	var outdb *sql.DB
	
	databaseAccessCommand := "Driver={Microsoft Access Driver (*.mdb)};"+
		readonlyCommand +
		"DBQ="+filename
	//fmt.Println("Database access command: "+databaseAccessCommand)
	if backa {
		outdb, err = sql.Open("odbc",
			databaseAccessCommand)
	} else {
		db, err = sql.Open("odbc",
			databaseAccessCommand)
	}
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	if backa {
		db = sqlite_init(dbfilename)
	} else {
		outdb = sqlite_init(dbfilename)
	}

	copyDtbVer(db, outdb)
	copyPlatser(db, outdb)
	copyPersoner(db, outdb)
	copyKonton(db, outdb)
	copyBetalKonton(db, outdb)
	copyTransfers(db, outdb)
	copyBetalningar(db, outdb)
	copyLoan(db, outdb)
	copyBudget(db, outdb)
	copyTransaktioner(db, outdb)
	outdb.Close()
	db.Close()
}
