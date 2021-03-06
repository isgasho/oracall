#!/bin/sh
set -e
dsn=${DSN:-$(grep -v '^#' .dsn | head -n1)}
echo "dsn=$dsn"
if [ -z "$dsn" ]; then
    exit 3
fi

set -x
set +e
go test $TESTARGS -connect=${dsn} ./... $TESTARGS
go install

{
if echo "$dsn" | fgrep -q '@XE'; then
	if [ -x "$ORACLE_HOME/bin/sqlplus" -a -e testdata/db_web.sql ]; then
		$ORACLE_HOME/bin/sqlplus "$dsn" <<EOF
@testdata/db_web.pck
EXIT
EOF
	fi
fi
set -x
find ./examples/minimal -name 'generated*' -delete
oracall -connect="$dsn" -pb-out=examples/minimal:main ${1:-DB_WEB.SENDPREOFFER_31101}
} >examples/minimal/generated_functions.go
go build -o /tmp/minimal ./examples/minimal
echo
echo '-----------------------------------------------'
CMD='/tmp/minimal -connect='${dsn}" ${1:-DB_web.sendpreoffer_31101}"
echo "$CMD"
#$CMD '{"p_lang":"hu", "p_sessionid": "123", "p_kotveny_vagyon":{"teaor": "1233", "forgalom": 0}, "p_telep":[{"telep_azon":"A", "telep_kod":"C"},{"telep_azon":"z", "telep_kod":"x"}]}'
time $CMD '{"p_lang":"hu", "p_sessionid": "123", "p_kotveny_vagyon":{"teaor": "1233", "forgalom": 0}, "p_telep":[{"telep_azon":1, "telep_kod":0}], "p_kotveny": {"dijfizgyak":"N"}, "p_kedvezmenyek": ["KEDV01"]}'
