//-*- coding: utf-8 -*-

package main

import (
	"context"
	"database/sql"
	"log"
	"os"
	"strconv"
	"testing"
	"time"
	
	"github.com/shopspring/decimal"  // MIT License
	_ "github.com/alexbrainman/odbc" // BSD-3-Clause License
	_ "github.com/mattn/go-sqlite3"
)

type person struct {
	namn string
	birth  int
	sex string
}

type Plats struct {
	Namn string      // size
	Gironummer string    // size
	Typ bool // oanvänt?
	RefKonto string  // size, != 0 betyder kontokortsföretag
}

type transaction struct {
	lopnr int
	fromAcc string
	toAcc string
	tType string
	what string
	date time.Time
	who string
	amount decimal.Decimal
	comment string
	fixed bool
}

type konto struct {
	KontoNummer string    // size 20
	Benämning string      // size 40, index
	Saldo decimal.Decimal // BCD / Decimal Precision 19
	StartSaldo decimal.Decimal  // BCD / Decimal Precision 19
	StartManad string     // size 10
	SaldoArsskifte string // BCD / Decimal Precision 19
	ArsskifteManad string // size 10
}


func openJetDB(filename string, ro bool) *sql.DB {
	readonlyCommand := ""
	if ro {
		readonlyCommand = "READONLY;"
	}

	databaseAccessCommand := "Driver={Microsoft Access Driver (*.mdb)};" +
		readonlyCommand +
		"DBQ=" + filename
	//fmt.Println("Database access command: "+databaseAccessCommand)
	db, err := sql.Open("odbc",
		databaseAccessCommand)
	if err != nil {
		log.Fatal(err)
		return nil
	}
	return db
}

func openSqlite(filename string) *sql.DB {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Fatal(err)
	}
	return db
}

func Test1(t *testing.T) {
	t.Log("Test1 begins")
	var filename string = "GOTEST1"

	err := os.Remove(filename+".db")
	if err != nil {
		t.Log(err)
	}
	err = os.Remove(filename+"b.mdb")
	if err != nil {
		t.Log(err)
	}
	
	// Open testdata
	db := openJetDB(filename+".MDB", true)
	if db != nil {
		t.Log("OpenMDB succeeded.")
	} else {
		t.Error("OpenMDB failed to open file.")
	}
	t.Log("Check original file.")
	checkDB1(t, db)
	db.Close()
	
	convert_mdb2sqlite(filename+".mdb", filename+".db")
	// Open testdata
	db = openSqlite(filename+".db")
	if db != nil {
		t.Log("OpenDB succeeded.")
	} else {
		t.Error("OpenDB failed to open file.")
	}
	t.Log("Check sqlite converted file.")
	revopt = true; // make toUtf8() work
	checkDB1(t, db)
	db.Close()

	convert_sqlite2mdb(filename+".db", filename+"b.mdb")
	// Open testdata
	db = openJetDB(filename+".mdb", true)
	db2 := openJetDB(filename+"b.mdb", true)
	if db != nil {
		t.Log("OpenDB succeeded.")
	} else {
		t.Error("OpenDB failed to open file.")
	}
	if db2 != nil {
		t.Log("OpenDB succeeded.")
	} else {
		t.Error("OpenDB failed to open file.")
	}
	t.Log("Check double converted file.")
	revopt = false; // make toUtf8() work
	checkDB1(t, db2)
	//t.Log("Check double converted file b.")
	//checkDB1b(t, db, db2)
	db.Close()
}

// Databasen skapad i Hemekonomi:
// - Ny databas
// - Lägg till person "Person Ett", kön man, född 1999
// - Lägg till person "Person Två", kön kvinna, född 2001
// - Lägg till plats "Plats Ett", ej kredit
// - Lägg till transaktion: insättning, konto:Plånboken, datum 2020-12-24, Vad: Studiestöd, Vem: Gemensamt, Summa 1,10kr, Kommentar "En transaktion"
// - Lägg till transaktion: inköp, konto:Plånboken, "Livs", datum 2020-12-24, Person Ett, Summa 0,10kr, Kommentar "Tom € räksmörgås"
// - Lägg till transaktion: inköp, konto:Plånboken, "Livs", datum 2020-12-24, Person Två, Summa 0.10kr, Kommentar "Tom € RÄKSMÖRGÅS"
func checkDB1(t *testing.T, db *sql.DB) {
	// kolla antal personer (Gemensamt skapas automatiskt)
	antal := countrows(db, "Personer")
	if antal == 3 {
		t.Log("Antal Pers OK")
	} else {
		t.Error("Antal Pers fel:" + strconv.Itoa(antal))
	}
	// kolla antal platser
	antal = countrows(db, "Platser")
	if antal == 1 {
		t.Log("Antal Platser OK")
	} else {
		t.Error("Antal Platser fel:" + strconv.Itoa(antal))
	}
	// kolla antal transaktioner
	antal = countrows(db, "Transaktioner")
	if antal == 3 {
		t.Log("Antal Transaktioner OK")
	} else {
		t.Error("Antal Transaktioner fel:" + strconv.Itoa(antal))
	}
	// Kolla persondata1
	data := hämtaPerson(db, 1)
	if data.namn == "Gemensamt" {
		t.Log("Person namn 1 OK")
	} else {
		t.Error("Person namn 1 fel:" + data.namn)
	}
	if data.birth == 0 {
		t.Log("Person 1 födelseår OK")
	} else {
		t.Error("Person 1 födelseår fel:" + strconv.Itoa(data.birth))
	}
	if data.sex == "Gemensamt" {
		t.Log("Person 1 kön OK")
	} else {
		t.Error("Person 1 kön fel:" + data.sex)
	}
	// Kolla persondata2
	data = hämtaPerson(db, 2)
	if data.namn == "Person Ett" {
		t.Log("Person namn 2 OK")
	} else {
		t.Error("Person namn 2 fel:" + data.namn)
	}
	if data.birth == 1999 {
		t.Log("Person 2 födelseår OK")
	} else {
		t.Error("Person 2 födelseår fel:" + strconv.Itoa(data.birth))
	}
	if data.sex == "Man" {
		t.Log("Person 2 kön OK")
	} else {
		t.Error("Person 2 kön fel:" + data.sex)
	}
	// Kolla persondata3
	data = hämtaPerson(db, 3)
	if data.namn == "Person Två" {
		t.Log("Person namn 3 OK")
	} else {
		t.Error("Person namn 3 fel:" + data.namn)
	}
	if data.birth == 2001 {
		t.Log("Person 3 födelseår OK")
	} else {
		t.Error("Person 3 födelseår fel:" + strconv.Itoa(data.birth))
	}
	if data.sex == "Kvinna" {
		t.Log("Person 3 kön OK")
	} else {
		t.Error("Person 3 kön fel:" + data.sex)
	}
	// Kolla platsdata
	data2 := hämtaPlats(db, 1)
	if data2.Namn == "Plats Ett" {
		t.Log("Plats 1 namn OK")
	} else {
		t.Error("Plats 1 namn fel:" + data2.Namn)
	}
	// kolla transaktion1: typ, frånkonto, tillkonto, datum, person, summa, kommentar
	data3 := hämtaTransaktion(db, 1)
	if data3.tType == "Insättning" {
		t.Log("Transaktion 1 typ OK")
	} else {
		t.Error("Transaktion 1 typ fel:" + data3.tType)
	}
	if data3.fromAcc == "---" {
		t.Log("Transaktion 1 frånkonto OK")
	} else {
		t.Error("Transaktion 1 frånkonto fel:" + data3.fromAcc)
	}
	if data3.toAcc == "Plånboken" {
		t.Log("Transaktion 1 tillkonto OK")
	} else {
		t.Error("Transaktion 1 tillkonto fel:" + data3.toAcc)
	}
	if data3.what == "Studiestöd" {
		t.Log("Transaktion 1 vad OK")
	} else {
		t.Error("Transaktion 1 vad fel:" + data3.what)
	}
	if data3.date.Format("2006-01-02") == "2020-12-24" {
		t.Log("Transaktion 1 datum OK")
	} else {
		t.Error("Transaktion 1 datum fel:" + data3.date.Format("2006-01-02"))
	}
	if data3.who == "Gemensamt" {
		t.Log("Transaktion 1 vem OK")
	} else {
		t.Error("Transaktion 1 vem fel:" + data3.who)
	}
	summa, _ := decimal.NewFromString("1.1")
	if data3.amount.Cmp(summa)==0 {
		t.Log("Transaktion 1 summa OK")
	} else {
		t.Error("Transaktion 1 summa fel:" + data3.amount.String())
	}
	if data3.comment == "En transaktion" {
		t.Log("Transaktion 1 text OK")
	} else {
		t.Error("Transaktion 1 text fel:" + data3.comment)
	}
	if data3.fixed == false {
		t.Log("Transaktion 1 fast transaktion OK")
	} else {
		t.Error("Transaktion 1 fast transaktion fel:" + strconv.FormatBool(data3.fixed))
	}
	// kolla transaktion2: typ, frånkonto, tillkonto, datum, person, summa, kommentar
	data3 = hämtaTransaktion(db, 2)
	if data3.tType == "Inköp" {
		t.Log("Transaktion 2 typ OK")
	} else {
		t.Error("Transaktion 2 typ fel:" + data3.tType)
	}
	if data3.fromAcc == "Plånboken" {
		t.Log("Transaktion 2 frånkonto OK")
	} else {
		t.Error("Transaktion 2 frånkonto fel:" + data3.fromAcc)
	}
	if data3.toAcc == "Plats Ett" {
		t.Log("Transaktion 2 tillkonto OK")
	} else {
		t.Error("Transaktion 2 tillkonto fel:" + data3.toAcc)
	}
	if data3.what == "Livsmedel" {
		t.Log("Transaktion 2 vad OK")
	} else {
		t.Error("Transaktion 2 vad fel:" + data3.what)
	}
	if data3.date.Format("2006-01-02") == "2020-12-24" {
		t.Log("Transaktion 2 datum OK")
	} else {
		t.Error("Transaktion 2 datum fel:" + data3.date.Format("2006-01-02"))
	}
	if data3.who == "Person Ett" {
		t.Log("Transaktion 2 vem OK")
	} else {
		t.Error("Transaktion 2 vem fel:" + data3.who)
	}
	summa, _ = decimal.NewFromString("0.1")
	if data3.amount.Cmp(summa)==0 {
		t.Log("Transaktion 2 summa OK")
	} else {
		t.Error("Transaktion 2 summa fel:" + data3.amount.String())
	}
	if data3.comment == "Tom € räksmörgås" {
		t.Log("Transaktion 2 text OK")
	} else {
		t.Error("Transaktion 2 text fel:" + data3.comment)
	}
	if data3.fixed == false {
		t.Log("Transaktion 2 fast transaktion OK")
	} else {
		t.Error("Transaktion 2 fast transaktion fel:" + strconv.FormatBool(data3.fixed))
	}
	// kolla transaktion3: typ, frånkonto, tillkonto, datum, person, summa, kommentar
	data3 = hämtaTransaktion(db, 3)
	if data3.tType == "Inköp" {
		t.Log("Transaktion 3 typ OK")
	} else {
		t.Error("Transaktion 3 typ fel:" + data3.tType)
	}
	if data3.fromAcc == "Plånboken" {
		t.Log("Transaktion 3 frånkonto OK")
	} else {
		t.Error("Transaktion 3 frånkonto fel:" + data3.fromAcc)
	}
	if data3.toAcc == "Plats Ett" {
		t.Log("Transaktion 3 tillkonto OK")
	} else {
		t.Error("Transaktion 3 tillkonto fel:" + data3.toAcc)
	}
	if data3.what == "Livsmedel" {
		t.Log("Transaktion 3 vad OK")
	} else {
		t.Error("Transaktion 3 vad fel:" + data3.what)
	}
	if data3.date.Format("2006-01-02") == "2020-12-24" {
		t.Log("Transaktion 3 datum OK")
	} else {
		t.Error("Transaktion 3 datum fel:" + data3.date.Format("2006-01-02"))
	}
	if data3.who == "Person Två" {
		t.Log("Transaktion 3 vem OK")
	} else {
		t.Error("Transaktion 3 vem fel:" + data3.who)
	}
	summa, _ = decimal.NewFromString("0.1")
	if data3.amount.Cmp(summa)==0 {
		t.Log("Transaktion 3 summa OK")
	} else {
		t.Error("Transaktion 3 summa fel:" + data3.amount.String())
	}
	if data3.comment == "Tom € RÄKSMÖRGÅS" {
		t.Log("Transaktion 3 text OK")
	} else {
		t.Error("Transaktion 3 text fel:" + data3.comment)
	}
	if data3.fixed == false {
		t.Log("Transaktion 3 fast transaktion OK")
	} else {
		t.Error("Transaktion 3 fast transaktion fel:" + strconv.FormatBool(data3.fixed))
	}
	// kolla saldo konto: Plånboken
	data4 := hämtaKonto(db, 1)
	if data4.Benämning == "Plånboken" {
		t.Log("Konto 1 namn OK")
	} else {
		t.Error("Konto 1 namn fel:" + data4.Benämning)
	}
	summa, _ = decimal.NewFromString("0.9")
	if data4.Saldo.Cmp(summa)==0 {
		t.Log("Konto 1 saldo OK")
	} else {
		t.Error("Konto 1 saldo fel:" + data4.Saldo.String())
	}
}

func checkDB1b(t *testing.T, db *sql.DB, db2 *sql.DB) {
	// motsvarande som checkDB1 men jämför bytearrayer
	log.Fatal("TBD checkDB1b")
}

func convert_mdb2sqlite(filenamemdb string, filenamedb string) {
	log.Println("convert_mdb2sqlite")
	konvertera(filenamemdb, filenamedb, true, false)
}

func convert_sqlite2mdb(filenamedb string, filenamemdb string) {
	log.Println("convert_sqlite2mdb")
	konvertera(filenamemdb, filenamedb, false, true)
}

func countrows(db *sql.DB, table string) int {
	var cnt int
	_ = db.QueryRow(`select count(*) from `+table).Scan(&cnt)
	return cnt
}

func hämtaPerson(db *sql.DB, lopnr int) person {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res1 := db.QueryRowContext(ctx,
		`SELECT Namn,Född,Kön FROM Personer WHERE (Löpnr=?)`, lopnr)

	var namn []byte  // size 50
	var birth string // size 4 (år, 0 för Gemensamt)
	var sex string   // size 10 (text: Gemensamt, Man, Kvinna)

	err := res1.Scan(&namn, &birth, &sex)
	if err != nil {
		log.Fatal(err)
	}

	var retperson person

	retperson.namn = toUtf8(namn)
	retperson.birth, err = strconv.Atoi(birth)
	retperson.sex = sex

	return retperson
}

func hämtaPlats(db *sql.DB, lopnr int) Plats {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res1 := db.QueryRowContext(ctx,
	        `SELECT Namn,Gironummer,Typ,RefKonto FROM Platser WHERE (Löpnr=?)`, lopnr)

	var Namn []byte       // size 40
	var Gironummer []byte // size 20
	var Typ []byte        // size 2
	var RefKonto []byte   // size 40

	err := res1.Scan(&Namn, &Gironummer, &Typ, &RefKonto)
	if err != nil {
		log.Fatal(err)
	}

	var retplats Plats

	retplats.Namn = toUtf8(Namn)
	retplats.Gironummer = toUtf8(Gironummer)
	if toUtf8(Typ) == "true" {
		retplats.Typ = true
	} else {
		retplats.Typ = false
	}
	retplats.RefKonto = toUtf8(RefKonto)

	return retplats
}

func isobytetodate(rawdate []byte) (time.Time, error) {
	return time.Parse("2006-01-02", toUtf8(rawdate))
}

func hämtaTransaktion(db *sql.DB, lopnr int) (result transaction) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var err error
	var res *sql.Rows
	
	res, err = db.QueryContext(ctx,
		`SELECT FrånKonto,TillKonto,Typ,Datum,Vad,Vem,Belopp,Löpnr,Saldo,Fastöverföring,Text from transaktioner
  where Löpnr = ?`, lopnr)
        if err != nil {
		log.Fatal(err)
	}
	
	var fromAcc []byte // size 40
	var toAcc []byte   // size 40
	var tType []byte   // size 40
	var date []byte    // size 10
	var what []byte    // size 40
	var who []byte     // size 50
	var amount []byte  // BCD / Decimal Precision 19
	var nummer int     // Autoinc Primary Key, index
	var saldo []byte   // BCD / Decimal Precision 19
	var fixed bool     // Boolean
	var comment []byte // size 60
	
	for res.Next() {
		var record transaction
		err = res.Scan(&fromAcc, &toAcc, &tType, &date, &what, &who, &amount, &nummer, &saldo, &fixed, &comment)
		
		record.lopnr = nummer
		record.fromAcc = toUtf8(fromAcc)
		record.toAcc = toUtf8(toAcc)
		record.tType = toUtf8(tType)
		record.what = toUtf8(what)
		record.date, err = isobytetodate(date)
		record.who = toUtf8(who)
		record.amount, err = decimal.NewFromString(toUtf8(amount))
		record.comment = toUtf8(comment)
		record.fixed = fixed
		
		result = record
	}
	return result
}

func hämtaKonto(db *sql.DB, lopnr int) konto {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	res1 := db.QueryRowContext(ctx,
	        `SELECT KontoNummer,Benämning,Saldo,StartSaldo,StartManad,SaldoArsskifte,ArsskifteManad FROM Konton WHERE (Löpnr=?)`, lopnr)

	var KontoNummer []byte    // size 20
	var Benämning []byte      // size 40, index
	var Saldo []byte          // BCD / Decimal Precision 19
	var StartSaldo []byte     // BCD / Decimal Precision 19
	var StartManad []byte     // size 10
	var SaldoArsskifte []byte // BCD / Decimal Precision 19
	var ArsskifteManad []byte // size 10

	err := res1.Scan(&KontoNummer, &Benämning, &Saldo, &StartSaldo, &StartManad, &SaldoArsskifte, &ArsskifteManad)
	if err != nil {
		log.Fatal(err)
	}

	var retkonto konto

	retkonto.KontoNummer = toUtf8(KontoNummer)
	retkonto.Benämning = toUtf8(Benämning)
	retkonto.Saldo, err = decimal.NewFromString(toUtf8(Saldo))
	retkonto.StartSaldo, err = decimal.NewFromString(toUtf8(StartSaldo))
	retkonto.StartManad = toUtf8(StartManad)
	retkonto.SaldoArsskifte = toUtf8(SaldoArsskifte)
	retkonto.ArsskifteManad = toUtf8(ArsskifteManad)

	return retkonto
}

// TODO: testa "\/|:.,¤$%& i kommentar, person-/plats-/typ-/konto-namn
