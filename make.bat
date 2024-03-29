@echo off
REM F�rsta g�ngen      : ./make.bat setup
REM K�r tester         : ./make.bat test
REM K�r tester pratsamt: ./make.bat test verbose
REM Bygg f�r anv�ndning: ./make.sh release
REM Bygg f�r utveckling: ./make.sh

SET ARG1=%1
SET ARG2=%2

FOR /F %%I IN ('go env GOOS') DO SET GOOS=%%I
FOR /F %%I IN ('go env GOARCH') DO SET GOARCH=%%I

IF NOT "%GOOS%"=="windows" (
  ECHO "Only intended for windows."
  EXIT
)

DEL HHEK2sqlite.exe

IF "%ARG1%"=="setup" (
  ECHO H�mtar beroenden...

  go get github.com/alexbrainman/odbc
  go get github.com/mattn/go-sqlite3
  go get golang.org/x/text/encoding/charmap
  ECHO Klar.
  EXIT
)

IF "%ARG1%"=="clean" (
  del *~
  del #*#
  del .#*
  EXIT
)

IF "%ARG1%"=="test" (
  IF "%ARG2%"=="verbose" (
    SET BUILDCMD=test -v
  ) else (
    SET BUILDCMD=test
  )
) else (
  SET BUILDCMD=build -o HHEK2sqlite.exe
)

IF "%ARG1%"=="release" (
  SET LINKCMD=-ldflags="-s -w"
) else (
  SET LINKCMD=
)

ECHO Bygger...

@echo on
go %BUILDCMD% %LINKCMD%
@echo off

ECHO Klar.
