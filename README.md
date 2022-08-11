# hhek2sqlite
Convert Hogia Hemekonomi database to sqlite-format

I mitten av 90-talet fanns ett program som hette Hogia Hemekonomi. Det sparar data i en mdb-fil. Det är ett program för att konvertera den filen till ett modernare format kallat Sqlite. Det nya filformatet används av https://github.com/jonasgit/wHHEK

Det här programmet fungerar i Windows 10, 11 och Wine (Linux m.m.).  Åtminstone versioner aktuella i skrivande stund. För Wine behövs:
winetricks mdac28 jet40
